package giphy

import (
	"container/list"
	"sync"
	"time"
)

type userCacheEntry struct {
	gifID     string
	username  string
	known     bool
	expiresAt time.Time
}

type userCache struct {
	mu       sync.Mutex
	items    map[string]*list.Element
	order    *list.List
	maxItems int
	now      func() time.Time
}

func newUserCache(maxItems int) *userCache {
	return &userCache{
		items:    make(map[string]*list.Element),
		order:    list.New(),
		maxItems: maxItems,
		now:      time.Now,
	}
}

func (c *userCache) get(gifID string) (string, bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[gifID]
	if !ok {
		return "", false, false
	}
	entry := el.Value.(*userCacheEntry)
	if c.now().After(entry.expiresAt) {
		c.order.Remove(el)
		delete(c.items, gifID)
		return "", false, false
	}
	c.order.MoveToFront(el)
	return entry.username, entry.known, true
}

func (c *userCache) set(gifID, username string, known bool, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[gifID]; ok {
		entry := el.Value.(*userCacheEntry)
		entry.username = username
		entry.known = known
		entry.expiresAt = c.now().Add(ttl)
		c.order.MoveToFront(el)
		return
	}
	entry := &userCacheEntry{gifID: gifID, username: username, known: known, expiresAt: c.now().Add(ttl)}
	el := c.order.PushFront(entry)
	c.items[gifID] = el
	for c.order.Len() > c.maxItems {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		oldestEntry := oldest.Value.(*userCacheEntry)
		c.order.Remove(oldest)
		delete(c.items, oldestEntry.gifID)
	}
}
