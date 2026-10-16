package engines

import (
	"container/list"
	"context"
	"fmt"
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

	entry, ok := c.lookup(key, c.now())
	if !ok {
		return nil, engine.ErrMiss
	}

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

func (c *InMemory) SetIfAbsent(_ context.Context, key string, data []byte, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.lookup(key, c.now()); ok {
		return false, nil
	}

	c.store(key, data, ttl)

	return true, nil
}

func (c *InMemory) IncrementWindow(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()

	entry, ok := c.lookup(key, now)
	if !ok {
		c.store(key, strconv.AppendInt(nil, 1, 10), window)

		return 1, window, nil
	}

	count, err := strconv.ParseInt(string(entry.data), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %s", engine.ErrNotCounter, key)
	}

	count++
	next := strconv.AppendInt(nil, count, 10)
	c.bytes += len(next) - len(entry.data)
	entry.data = next

	if entry.expiresAt.IsZero() {
		entry.expiresAt = now.Add(window)
	}

	remaining := entry.expiresAt.Sub(now)

	c.evict()

	return count, remaining, nil
}

func (c *InMemory) ExtendDeadline(_ context.Context, key string, deadline time.Time) (time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if !deadline.After(now) {
		return time.Time{}, engine.ErrDeadlinePassed
	}

	requestedMS := deadline.UnixMilli()

	if entry, ok := c.lookup(key, now); ok {
		currentMS, err := strconv.ParseInt(string(entry.data), 10, 64)
		if err == nil && currentMS >= requestedMS {
			return time.UnixMilli(currentMS), nil
		}
	}

	inForce := time.UnixMilli(requestedMS)
	c.storeUntil(key, strconv.AppendInt(nil, requestedMS, 10), inForce)

	return inForce, nil
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

func (c *InMemory) lookup(key string, now time.Time) (*inMemoryEntry, bool) {
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}

	entry := el.Value.(*inMemoryEntry)
	if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
		c.drop(el)

		return nil, false
	}

	c.order.MoveToFront(el)

	return entry, true
}

func (c *InMemory) drop(el *list.Element) {
	entry := el.Value.(*inMemoryEntry)
	c.order.Remove(el)
	delete(c.items, entry.key)
	c.bytes -= entrySize(entry.key, entry.data)
}

func (c *InMemory) store(key string, data []byte, ttl time.Duration) {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = c.now().Add(ttl)
	}

	c.storeUntil(key, data, expiresAt)
}

func (c *InMemory) storeUntil(key string, data []byte, expiresAt time.Time) {
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
