package engines

import (
	"container/list"
	"context"
	"strconv"
	"sync"
	"time"

	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
)

const (
	bytesPerMB              = 1 << 20
	defaultInMemoryMaxMB    = 128
	minInMemoryMaxMB        = 1
	maxInMemoryMaxMB        = 4096
	defaultInMemoryMaxBytes = defaultInMemoryMaxMB * bytesPerMB
	entryOverheadBytes      = 192
	unboundedTTLCeiling     = time.Minute
)

type (
	inMemoryEntry struct {
		key       string
		data      []byte
		expiresAt time.Time
	}

	InMemory struct {
		mu       sync.Mutex
		items    map[string]*list.Element
		order    *list.List
		maxBytes int
		bytes    int
		now      func() time.Time
	}
)

func NewInMemory(maxBytes int) *InMemory {
	if maxBytes <= 0 {
		maxBytes = defaultInMemoryMaxBytes
	}

	return &InMemory{
		items:    make(map[string]*list.Element),
		order:    list.New(),
		maxBytes: maxBytes,
		now:      time.Now,
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
	c.bytes -= entrySize(entry.key, entry.data)
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
	c.bytes += entrySize(key, data)

	c.evict()
}

func (c *InMemory) evict() {
	for c.bytes > c.maxBytes {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}

		c.drop(oldest)
	}
}

func entrySize(key string, data []byte) int {
	return len(key) + len(data) + entryOverheadBytes
}

func (c *InMemory) ResizeMB(maxMB int) {
	if maxMB <= 0 {
		maxMB = defaultInMemoryMaxMB
	}

	maxMB = min(max(maxMB, minInMemoryMaxMB), maxInMemoryMaxMB)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxBytes = maxMB * bytesPerMB
	c.evict()
}

func (c *InMemory) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingCacheInMemoryMaxMB.Key {
		return
	}

	maxMB, err := strconv.Atoi(value)
	if err != nil {
		logger.Log.Warn().Str("value", value).Msg("in-memory cache size is not a number, keeping the current limit")

		return
	}

	c.ResizeMB(maxMB)

	logger.Log.Info().Int("max_mb", maxMB).Msg("in-memory cache resized")
}
