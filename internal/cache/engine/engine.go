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
		SetIfAbsent(ctx context.Context, key string, data []byte, ttl time.Duration) (bool, error)
		IncrementWindow(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
		ExtendDeadline(ctx context.Context, key string, deadline time.Time) (time.Time, error)
		Del(ctx context.Context, keys ...string) error
		Ping(ctx context.Context) error
		Close() error
	}
)
