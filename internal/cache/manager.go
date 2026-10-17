package cache

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"umineko_city_of_books/internal/cache/engine"

	"github.com/prometheus/client_golang/prometheus"
)

type (
	Manager struct {
		engines []engine.Engine

		mu         sync.RWMutex
		dependents map[string][]string
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

func (m *Manager) Invalidates(dependent Namespace, feeds ...Namespace) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dependents == nil {
		m.dependents = make(map[string][]string, len(feeds))
	}

	key := dependent.Key().String()
	for _, feed := range feeds {
		if slices.Contains(m.dependents[feed.Prefix], key) {
			continue
		}

		m.dependents[feed.Prefix] = append(m.dependents[feed.Prefix], key)
	}
}

func (m *Manager) Del(ctx context.Context, keys ...Key) error {
	if m == nil || len(keys) == 0 {
		return nil
	}

	names := make([]string, 0, len(keys))
	for _, key := range keys {
		names = append(names, key.name)
	}

	names = m.withDependents(names)

	var firstErr error
	for _, candidate := range m.engines {
		err := candidate.Del(ctx, names...)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *Manager) withDependents(keys []string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := slices.Clone(keys)
	for prefix, dependents := range m.dependents {
		matches := slices.ContainsFunc(keys, func(key string) bool {
			return strings.HasPrefix(key, prefix)
		})
		if !matches {
			continue
		}

		for _, dependent := range dependents {
			if !slices.Contains(result, dependent) {
				result = append(result, dependent)
			}
		}
	}

	return result
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
	for _, candidate := range m.engines {
		err := candidate.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *Manager) getBytes(ctx context.Context, key Key) ([]byte, error) {
	active := m.current()
	if active == nil {
		return nil, engine.ErrMiss
	}

	data, err := active.Get(ctx, key.name)
	if err != nil {
		if errorIsMiss(err) {
			cacheMisses.Inc()
		}

		return nil, err
	}

	cacheHits.Inc()

	return data, nil
}

func (m *Manager) setBytes(ctx context.Context, key Key, data []byte) error {
	active := m.current()
	if active == nil {
		return nil
	}

	return active.Set(ctx, key.name, data, key.ttl)
}

func (m *Manager) setManyBytes(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	active := m.current()
	if active == nil || len(entries) == 0 {
		return nil
	}

	return active.SetMany(ctx, entries, ttl)
}

func (m *Manager) Allow(ctx context.Context, key Key, limit int, window time.Duration) (engine.Allowance, error) {
	if limit < 1 || window < time.Millisecond {
		return engine.Allowance{}, fmt.Errorf("%w: limit %d, window %s", ErrInvalidAllowance, limit, window)
	}

	active := m.current()
	if active == nil {
		return engine.Allowance{}, ErrUnavailable
	}

	return active.Allow(ctx, key.name, limit, window)
}

func (m *Manager) Reset(ctx context.Context, key Key) error {
	active := m.current()
	if active == nil {
		return ErrUnavailable
	}

	return active.Reset(ctx, key.name)
}

func (m *Manager) current() engine.Engine {
	if m == nil {
		return nil
	}

	for _, candidate := range m.engines {
		if candidate.Enabled() {
			return candidate
		}
	}

	return nil
}
