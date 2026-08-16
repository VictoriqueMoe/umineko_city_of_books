package cache

import (
	"context"
	"time"

	"umineko_city_of_books/internal/cache/engine"

	"github.com/prometheus/client_golang/prometheus"
)

type (
	Manager struct {
		engines []engine.Engine
	}
)

var (
	cacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Number of cache lookups that returned a value.",
	})
	cacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Number of cache lookups that found no value.",
	})
)

func init() {
	prometheus.MustRegister(cacheHits, cacheMisses)
}

func NewManager(candidates ...engine.Engine) *Manager {
	return &Manager{engines: candidates}
}

func (m *Manager) Engines() []engine.Engine {
	if m == nil {
		return nil
	}

	return m.engines
}

func (m *Manager) Del(ctx context.Context, keys ...string) error {
	active := m.current()
	if active == nil || len(keys) == 0 {
		return nil
	}

	return active.Del(ctx, keys...)
}

func (m *Manager) Ping(ctx context.Context) error {
	active := m.current()
	if active == nil {
		return nil
	}

	return active.Ping(ctx)
}

func (m *Manager) Close() error {
	if m == nil {
		return nil
	}

	var firstErr error
	for i := range m.engines {
		err := m.engines[i].Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *Manager) getBytes(ctx context.Context, key string) ([]byte, error) {
	active := m.current()
	if active == nil {
		return nil, engine.ErrMiss
	}

	data, err := active.Get(ctx, key)
	if err != nil {
		if errorIsMiss(err) {
			cacheMisses.Inc()
		}

		return nil, err
	}

	cacheHits.Inc()

	return data, nil
}

func (m *Manager) setBytes(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	active := m.current()
	if active == nil {
		return nil
	}

	return active.Set(ctx, key, data, ttl)
}

func (m *Manager) setManyBytes(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	active := m.current()
	if active == nil || len(entries) == 0 {
		return nil
	}

	return active.SetMany(ctx, entries, ttl)
}

func (m *Manager) current() engine.Engine {
	if m == nil {
		return nil
	}

	for i := range m.engines {
		if m.engines[i].Enabled() {
			return m.engines[i]
		}
	}

	return nil
}
