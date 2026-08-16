package engines

import (
	"container/list"
	"context"
	"sync"
	"time"

	"umineko_city_of_books/internal/cache/engine"
)

const (
	defaultInMemoryMaxEntries = 8192
	defaultInMemoryMaxBytes   = 64 << 20
	unboundedTTLCeiling       = time.Minute
)

type (
	inMemoryEntry struct {
		key       string
		data      []byte
		expiresAt time.Time
	}

	InMemory struct {
		mu         sync.Mutex
		items      map[string]*list.Element
		order      *list.List
		maxEntries int
		maxBytes   int
		bytes      int
		now        func() time.Time
	}
)

func NewInMemory(maxEntries int) *InMemory {
	if maxEntries <= 0 {
		maxEntries = defaultInMemoryMaxEntries
	}

	return &InMemory{
		items:      make(map[string]*list.Element),
		order:      list.New(),
		maxEntries: maxEntries,
		maxBytes:   defaultInMemoryMaxBytes,
		now:        time.Now,
	}
}

func (c *InMemory) Name() string {
	return "in-memory"
}

func (c *InMemory) Enabled() bool {
	return true
}

func (c *InMemory) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, engine.ErrMiss
	}

	entry := el.Value.(*inMemoryEntry)
	if c.now().After(entry.expiresAt) {
		c.drop(el)

		return nil, engine.ErrMiss
	}

	c.order.MoveToFront(el)

	return entry.data, nil
}

func (c *InMemory) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store(key, data, ttl)

	return nil
}

func (c *InMemory) SetMany(_ context.Context, entries map[string][]byte, ttl time.Duration) error {
	if len(entries) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, data := range entries {
		c.store(key, data, ttl)
	}

	return nil
}

func (c *InMemory) Del(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range keys {
		el, ok := c.items[keys[i]]
		if !ok {
			continue
		}

		c.drop(el)
	}

	return nil
}

func (c *InMemory) Ping(_ context.Context) error {
	return nil
}

func (c *InMemory) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
	c.bytes = 0

	return nil
}

func (c *InMemory) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}

func (c *InMemory) Bytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.bytes
}

func (c *InMemory) drop(el *list.Element) {
	entry := el.Value.(*inMemoryEntry)
	c.order.Remove(el)
	delete(c.items, entry.key)
	c.bytes -= len(entry.data)
}

func (c *InMemory) store(key string, data []byte, ttl time.Duration) {
	if ttl <= 0 {
		ttl = unboundedTTLCeiling
	}

	expiresAt := c.now().Add(ttl)

	if el, ok := c.items[key]; ok {
		entry := el.Value.(*inMemoryEntry)
		c.bytes += len(data) - len(entry.data)
		entry.data = data
		entry.expiresAt = expiresAt
		c.order.MoveToFront(el)

		return
	}

	c.items[key] = c.order.PushFront(&inMemoryEntry{key: key, data: data, expiresAt: expiresAt})
	c.bytes += len(data)

	for c.order.Len() > c.maxEntries || c.bytes > c.maxBytes {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}

		c.drop(oldest)
	}
}
