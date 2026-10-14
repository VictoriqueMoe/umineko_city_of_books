package coalesce

import (
	"context"
	"sync"
	"time"
)

type (
	Flush[K comparable, V any] func(ctx context.Context, key K, value V)

	Number interface {
		~int | ~int32 | ~int64 | ~float64
	}

	window[V any] struct {
		pending  V
		dirty    bool
		lastSent time.Time
		timer    *time.Timer
	}

	Keyed[K comparable, V any] struct {
		mu       sync.Mutex
		interval time.Duration
		merge    func(pending, next V) V
		flush    Flush[K, V]
		windows  map[K]*window[V]
	}
)

func NewKeyed[K comparable, V any](interval time.Duration, merge func(pending, next V) V, flush Flush[K, V]) *Keyed[K, V] {
	return &Keyed[K, V]{
		interval: interval,
		merge:    merge,
		flush:    flush,
		windows:  make(map[K]*window[V]),
	}
}

func Latest[V any](_, next V) V {
	return next
}

func Sum[V Number](pending, next V) V {
	return pending + next
}

func (c *Keyed[K, V]) Add(ctx context.Context, key K, value V) {
	if c.claim(key, value, time.Now()) {
		c.flush(ctx, key, value)
	}
}

func (c *Keyed[K, V]) Forget(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	w := c.windows[key]
	if w == nil {
		return
	}

	if w.timer != nil {
		w.timer.Stop()
	}
	delete(c.windows, key)
}

func (c *Keyed[K, V]) claim(key K, value V, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.forgetIdleLocked(now)

	w := c.windows[key]
	if w == nil {
		w = &window[V]{}
		c.windows[key] = w
	}

	if w.timer == nil && now.Sub(w.lastSent) >= c.interval {
		w.lastSent = now
		return true
	}

	if w.dirty {
		w.pending = c.merge(w.pending, value)
	} else {
		w.pending = value
		w.dirty = true
	}

	if w.timer == nil {
		w.timer = time.AfterFunc(c.interval-now.Sub(w.lastSent), func() {
			c.flushPending(key)
		})
	}

	return false
}

func (c *Keyed[K, V]) flushPending(key K) {
	value, ok := c.takePending(key, time.Now())
	if !ok {
		return
	}

	c.flush(context.Background(), key, value)
}

func (c *Keyed[K, V]) takePending(key K, now time.Time) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V

	w := c.windows[key]
	if w == nil || !w.dirty {
		return zero, false
	}

	value := w.pending
	w.pending = zero
	w.dirty = false
	w.timer = nil
	w.lastSent = now

	return value, true
}

func (c *Keyed[K, V]) forgetIdleLocked(now time.Time) {
	for key, w := range c.windows {
		if w.timer == nil && !w.dirty && now.Sub(w.lastSent) >= c.interval {
			delete(c.windows, key)
		}
	}
}
