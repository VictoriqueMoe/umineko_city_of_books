package cache

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"

	"github.com/redis/go-redis/v9"
)

type (
	Manager struct {
		mu     sync.RWMutex
		client *redis.Client
		url    string
	}
)

func NewManager() *Manager {
	m := new(Manager)
	registerStatsCollector(m)

	return m
}

func (m *Manager) Reconfigure(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)

	m.mu.Lock()
	defer m.mu.Unlock()

	if rawURL == m.url {
		return
	}

	if m.client != nil {
		_ = m.client.Close()
		m.client = nil
	}

	m.url = rawURL

	if rawURL == "" {
		logger.Log.Info().Msg("valkey cache disabled")
		return
	}

	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		m.url = ""
		logger.Log.Warn().Err(err).Msg("invalid valkey url, cache disabled")
		return
	}

	client := redis.NewClient(opts)
	client.AddHook(newObservabilityHook())
	m.client = client

	logger.Log.Info().Msg("valkey cache enabled")
}

func (m *Manager) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingValkeyURL.Key {
		return
	}

	m.Reconfigure(value)
}

func (m *Manager) Enabled() bool {
	return m.current() != nil
}

func (m *Manager) getBytes(ctx context.Context, key string) ([]byte, error) {
	client := m.current()
	if client == nil {
		return nil, ErrMiss
	}

	data, err := client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		cacheMisses.Inc()
		return nil, ErrMiss
	}
	if err != nil {
		return nil, err
	}

	cacheHits.Inc()
	return data, nil
}

func (m *Manager) setBytes(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	client := m.current()
	if client == nil {
		return nil
	}

	return client.Set(ctx, key, data, ttl).Err()
}

func (m *Manager) Del(ctx context.Context, keys ...string) error {
	client := m.current()
	if client == nil {
		return nil
	}

	return client.Del(ctx, keys...).Err()
}

func (m *Manager) Ping(ctx context.Context) error {
	client := m.current()
	if client == nil {
		return nil
	}

	return client.Ping(ctx).Err()
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return nil
	}

	err := m.client.Close()
	m.client = nil

	return err
}

func (m *Manager) current() *redis.Client {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.client
}
