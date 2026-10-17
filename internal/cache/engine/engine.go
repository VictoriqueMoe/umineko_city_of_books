package engine

import (
	"context"
	"time"
)

type (
	Engine interface {
		Name() string
		Enabled() bool
		Get(ctx context.Context, key string) ([]byte, error)
		Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
		SetMany(ctx context.Context, entries map[string][]byte, ttl time.Duration) error
		Allow(ctx context.Context, key string, limit int, window time.Duration) (Allowance, error)
		Reset(ctx context.Context, key string) error
		Del(ctx context.Context, keys ...string) error
		Ping(ctx context.Context) error
		Close() error
	}

	Allowance struct {
		Allowed    bool
		RetryAfter time.Duration
	}
)
