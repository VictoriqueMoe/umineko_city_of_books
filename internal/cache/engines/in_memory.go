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
	if !entry.expiresAt.IsZero() && c.now().After(entry.expiresAt) {
		c.order.Remove(el)
		delete(c.items, key)

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

		c.order.Remove(el)
		delete(c.items, keys[i])
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

	return nil
}

func (c *InMemory) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}

func (c *InMemory) store(key string, data []byte, ttl time.Duration) {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = c.now().Add(ttl)
	}

	if el, ok := c.items[key]; ok {
		entry := el.Value.(*inMemoryEntry)
		entry.data = data
		entry.expiresAt = expiresAt
		c.order.MoveToFront(el)

		return
	}

	c.items[key] = c.order.PushFront(&inMemoryEntry{key: key, data: data, expiresAt: expiresAt})

	for c.order.Len() > c.maxEntries {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}

		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*inMemoryEntry).key)
	}
}
