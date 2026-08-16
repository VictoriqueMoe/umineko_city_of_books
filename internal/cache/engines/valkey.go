package engines

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/cache/engines/observability"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"

	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeyhook"
)

const (
	probeTimeout = 5 * time.Second

	recoveryInterval = 30 * time.Second
)

type (
	Valkey struct {
		mu     sync.RWMutex
		client valkey.Client
		url    string

		healthy atomic.Bool
		downAt  atomic.Int64
	}
)

func NewValkey() *Valkey {
	v := new(Valkey)
	observability.RegisterStats(v.Client)

	return v
}

func NewValkeyWithClient(client valkey.Client) *Valkey {
	v := &Valkey{client: client}
	v.healthy.Store(true)

	return v
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

func (v *Valkey) Name() string {
	return "valkey"
}

func (v *Valkey) Enabled() bool {
	if v.current() == nil {
		return false
	}

	if v.healthy.Load() {
		return true
	}

	downAt := v.downAt.Load()
	if time.Since(time.Unix(0, downAt)) < recoveryInterval {
		return false
	}

	return v.downAt.CompareAndSwap(downAt, time.Now().UnixNano())
}

func (v *Valkey) observe(err error) error {
	if err == nil || valkey.IsValkeyNil(err) {
		v.healthy.Store(true)

		return err
	}

	if v.healthy.Swap(false) {
		v.downAt.Store(time.Now().UnixNano())
		logger.Log.Warn().Err(err).Msg("valkey cache unhealthy, falling back to next engine")
	}

	return err
}

func (v *Valkey) Reconfigure(rawURL string) error {
	client, err := v.swapClient(rawURL)
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

	v.healthy.Store(true)

	logger.Log.Info().Msg("valkey cache enabled")

	return nil
}

func (v *Valkey) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingValkeyURL.Key {
		return
	}

	err := v.Reconfigure(value)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("valkey cache reconfigure failed")
	}
}

func (v *Valkey) Get(ctx context.Context, key string) ([]byte, error) {
	client := v.current()
	if client == nil {
		return nil, engine.ErrMiss
	}

	data, err := client.Do(ctx, client.B().Get().Key(key).Build()).AsBytes()

	if err := v.observe(err); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, engine.ErrMiss
		}

		return nil, err
	}

	return data, nil
}

func (v *Valkey) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	client := v.current()
	if client == nil {
		return nil
	}

	value := client.B().Set().Key(key).Value(valkey.BinaryString(data))

	if ttl > 0 {
		return v.observe(client.Do(ctx, value.Px(ttl).Build()).Error())
	}

	return v.observe(client.Do(ctx, value.Build()).Error())
}

func (v *Valkey) SetMany(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	client := v.current()
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
			return v.observe(err)
		}
	}

	return v.observe(nil)
}

func (v *Valkey) Del(ctx context.Context, keys ...string) error {
	client := v.current()
	if client == nil || len(keys) == 0 {
		return nil
	}

	return v.observe(client.Do(ctx, client.B().Del().Key(keys...).Build()).Error())
}

func (v *Valkey) Ping(ctx context.Context) error {
	client := v.current()
	if client == nil {
		return nil
	}

	return v.observe(client.Do(ctx, client.B().Ping().Build()).Error())
}

func (v *Valkey) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.client == nil {
		return nil
	}

	v.client.Close()
	v.client = nil

	return nil
}

func (v *Valkey) Client() valkey.Client {
	return v.current()
}

func (v *Valkey) swapClient(rawURL string) (valkey.Client, error) {
	rawURL = strings.TrimSpace(rawURL)

	v.mu.Lock()
	defer v.mu.Unlock()

	if rawURL == v.url {
		return nil, nil
	}

	if v.client != nil {
		v.client.Close()
		v.client = nil
	}

	v.url = rawURL

	if rawURL == "" {
		logger.Log.Info().Msg("valkey cache disabled")
		return nil, nil
	}

	opt, err := valkey.ParseURL(rawURL)
	if err != nil {
		v.url = ""
		return nil, fmt.Errorf("cache: invalid valkey url: %w", err)
	}

	client, err := valkey.NewClient(opt)
	if err != nil {
		v.url = ""
		return nil, fmt.Errorf("cache: cannot reach valkey: %w", err)
	}

	client = valkeyhook.WithHook(client, observability.NewHook())
	v.client = client

	return client, nil
}

func (v *Valkey) current() valkey.Client {
	if v == nil {
		return nil
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.client
}
