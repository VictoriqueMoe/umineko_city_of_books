package cache

import (
	"context"
	"encoding/json"
	"time"
)

func (m *Manager) Get[T any](ctx context.Context, key string) (T, error) {
	var zero T

	if m == nil {
		return zero, ErrMiss
	}

	data, err := m.getBytes(ctx, key)
	if err != nil {
		return zero, err
	}

	return decode[T](data)
}

func (m *Manager) Set[T any](ctx context.Context, key string, value T, ttl time.Duration) error {
	if m == nil {
		return nil
	}

	data, err := encode(value)
	if err != nil {
		return err
	}

	return m.setBytes(ctx, key, data, ttl)
}

func (m *Manager) SetMany[T any](ctx context.Context, values map[string]T, ttl time.Duration) error {
	if m == nil || len(values) == 0 {
		return nil
	}

	entries := make(map[string][]byte, len(values))
	for key, value := range values {
		data, err := encode(value)
		if err != nil {
			return err
		}

		entries[key] = data
	}

	return m.setManyBytes(ctx, entries, ttl)
}

func (m *Manager) Load[T any](ctx context.Context, ns Namespace, load func(context.Context) (T, error), parts ...string) (T, error) {
	key := ns.Key(parts...)

	if cached, err := m.Get[T](ctx, key); err == nil {
		return cached, nil
	}

	value, err := load(ctx)
	if err != nil {
		var zero T

		return zero, err
	}

	_ = m.Set(ctx, key, value, ns.TTL)

	return value, nil
}

func encode[T any](value T) ([]byte, error) {
	switch v := any(value).(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return json.Marshal(value)
	}
}

func decode[T any](data []byte) (T, error) {
	var zero T

	switch any(zero).(type) {
	case []byte:
		return any(data).(T), nil
	case string:
		return any(string(data)).(T), nil
	default:
		var value T
		if err := json.Unmarshal(data, &value); err != nil {
			return zero, err
		}

		return value, nil
	}
}
