package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"

	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeyhook"
)

type (
	Manager struct {
		mu     sync.RWMutex
		client valkey.Client
		url    string
	}
)

const (
	probeTimeout = 5 * time.Second
)

func NewManager() *Manager {
	m := new(Manager)
	registerStatsCollector(m)

	return m
}

func ProbeURL(ctx context.Context, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}

	opt, err := valkey.ParseURL(rawURL)
	if err != nil {
		return fmt.Errorf("cache: invalid valkey url: %w", err)
	}

	client, err := valkey.NewClient(opt)
	if err != nil {
		return fmt.Errorf("cache: cannot reach valkey: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	err = client.Do(ctx, client.B().Ping().Build()).Error()
	if err != nil {
		return fmt.Errorf("cache: cannot reach valkey: %w", err)
	}

	return nil
}

func (m *Manager) Reconfigure(rawURL string) error {
	client, err := m.swapClient(rawURL)
	if err != nil {
		return err
	}

	if client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	err = client.Do(ctx, client.B().Ping().Build()).Error()
	if err != nil {
		return fmt.Errorf("cache: ping failed: %w", err)
	}

	logger.Log.Info().Msg("valkey cache enabled")

	return nil
}

func (m *Manager) swapClient(rawURL string) (valkey.Client, error) {
	rawURL = strings.TrimSpace(rawURL)

	m.mu.Lock()
	defer m.mu.Unlock()

	if rawURL == m.url {
		return nil, nil
	}

	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	m.url = rawURL

	if rawURL == "" {
		logger.Log.Info().Msg("valkey cache disabled")
		return nil, nil
	}

	opt, err := valkey.ParseURL(rawURL)
	if err != nil {
		m.url = ""
		return nil, fmt.Errorf("cache: invalid valkey url: %w", err)
	}

	client, err := valkey.NewClient(opt)
	if err != nil {
		m.url = ""
		return nil, fmt.Errorf("cache: cannot reach valkey: %w", err)
	}

	client = valkeyhook.WithHook(client, newObservabilityHook())
	m.client = client

	return client, nil
}

func (m *Manager) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingValkeyURL.Key {
		return
	}

	err := m.Reconfigure(value)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("valkey cache reconfigure failed")
	}
}

func (m *Manager) Enabled() bool {
	return m.current() != nil
}

func (m *Manager) getBytes(ctx context.Context, key string) ([]byte, error) {
	client := m.current()
	if client == nil {
		return nil, ErrMiss
	}

	data, err := client.Do(ctx, client.B().Get().Key(key).Build()).AsBytes()
	if valkey.IsValkeyNil(err) {
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

	value := client.B().Set().Key(key).Value(valkey.BinaryString(data))

	if ttl > 0 {
		return client.Do(ctx, value.Px(ttl).Build()).Error()
	}

	return client.Do(ctx, value.Build()).Error()
}

func (m *Manager) setManyBytes(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	client := m.current()
	if client == nil || len(entries) == 0 {
		return nil
	}

	cmds := make([]valkey.Completed, 0, len(entries))
	for key, data := range entries {
		value := client.B().Set().Key(key).Value(valkey.BinaryString(data))

		if ttl > 0 {
			cmds = append(cmds, value.Px(ttl).Build())
			continue
		}

		cmds = append(cmds, value.Build())
	}

	for _, resp := range client.DoMulti(ctx, cmds...) {
		err := resp.Error()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Del(ctx context.Context, keys ...string) error {
	client := m.current()
	if client == nil || len(keys) == 0 {
		return nil
	}

	return client.Do(ctx, client.B().Del().Key(keys...).Build()).Error()
}

func (m *Manager) Ping(ctx context.Context) error {
	client := m.current()
	if client == nil {
		return nil
	}

	return client.Do(ctx, client.B().Ping().Build()).Error()
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return nil
	}

	m.client.Close()
	m.client = nil

	return nil
}

func (m *Manager) current() valkey.Client {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.client
}
