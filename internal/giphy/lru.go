package giphy

import (
	"container/list"
	"sync"
	"time"
)

type (
	lruEntry[V any] struct {
		key       string
		value     V
		expiresAt time.Time
	}

	lru[V any] struct {
		mu       sync.Mutex
		items    map[string]*list.Element
		order    *list.List
		maxItems int
		now      func() time.Time
	}
)

func newLRU[V any](maxItems int) *lru[V] {
	return &lru[V]{
		items:    make(map[string]*list.Element),
		order:    list.New(),
		maxItems: maxItems,
		now:      time.Now,
	}
}

func (c *lru[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	entry := el.Value.(*lruEntry[V])
	if c.now().After(entry.expiresAt) {
		c.order.Remove(el)
		delete(c.items, key)
		return zero, false
	}
	c.order.MoveToFront(el)
	return entry.value, true
}

func (c *lru[V]) set(key string, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		entry := el.Value.(*lruEntry[V])
		entry.value = value
		entry.expiresAt = c.now().Add(ttl)
		c.order.MoveToFront(el)
		return
	}
	entry := &lruEntry[V]{key: key, value: value, expiresAt: c.now().Add(ttl)}
	el := c.order.PushFront(entry)
	c.items[key] = el
	for c.order.Len() > c.maxItems {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		oldestEntry := oldest.Value.(*lruEntry[V])
		c.order.Remove(oldest)
		delete(c.items, oldestEntry.key)
	}
}

func (c *lru[V]) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
