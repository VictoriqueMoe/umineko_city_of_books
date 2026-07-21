package cache

import (
	"context"
	"encoding/json"
	"time"
)

func Get[T any](ctx context.Context, m *Manager, key string) (T, error) {
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

func Set[T any](ctx context.Context, m *Manager, key string, value T, ttl time.Duration) error {
	if m == nil {
		return nil
	}

	data, err := encode(value)
	if err != nil {
		return err
	}

	return m.setBytes(ctx, key, data, ttl)
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
