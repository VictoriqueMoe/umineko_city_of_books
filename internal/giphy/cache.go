package giphy

import (
	"container/list"
	"sync"
	"time"
)

type (
	cacheEntry struct {
		key       string
		data      *Response
		expiresAt time.Time
	}

	cache struct {
		mu       sync.Mutex
		items    map[string]*list.Element
		order    *list.List
		maxItems int
		now      func() time.Time
	}
)

func newCache(maxItems int) *cache {
	return &cache{
		items:    make(map[string]*list.Element),
		order:    list.New(),
		maxItems: maxItems,
		now:      time.Now,
	}
}

func (c *cache) get(key string) (*Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*cacheEntry)
	if c.now().After(entry.expiresAt) {
		c.order.Remove(el)
		delete(c.items, key)
		return nil, false
	}
	c.order.MoveToFront(el)
	return entry.data, true
}

func (c *cache) set(key string, data *Response, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.data = data
		entry.expiresAt = c.now().Add(ttl)
		c.order.MoveToFront(el)
		return
	}
	entry := &cacheEntry{key: key, data: data, expiresAt: c.now().Add(ttl)}
	el := c.order.PushFront(entry)
	c.items[key] = el
	for c.order.Len() > c.maxItems {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		oldestEntry := oldest.Value.(*cacheEntry)
		c.order.Remove(oldest)
		delete(c.items, oldestEntry.key)
	}
}

func (c *cache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
