package cache

import (
	"context"
	"encoding/json"
	"time"

	"umineko_city_of_books/internal/logger"
)

func (m *Manager) Get[T any](ctx context.Context, key Key) (T, error) {
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

func (m *Manager) Set[T any](ctx context.Context, key Key, value T) error {
	if m == nil {
		return nil
	}

	data, err := encode(value)
	if err != nil {
		return err
	}

	return m.setBytes(ctx, key, data)
}

func (m *Manager) SetMany[T any](ctx context.Context, values map[Key]T) error {
	if m == nil || len(values) == 0 {
		return nil
	}

	byTTL := make(map[time.Duration]map[string][]byte)
	for key, value := range values {
		data, err := encode(value)
		if err != nil {
			return err
		}

		entries, ok := byTTL[key.ttl]
		if !ok {
			entries = make(map[string][]byte)
			byTTL[key.ttl] = entries
		}

		entries[key.name] = data
	}

	for ttl, entries := range byTTL {
		if err := m.setManyBytes(ctx, entries, ttl); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Load[T any](ctx context.Context, key Key, load func(context.Context) (T, error)) (T, error) {
	if cached, err := m.Get[T](ctx, key); err == nil {
		return cached, nil
	}

	value, err := load(ctx)
	if err != nil {
		var zero T

		return zero, err
	}

	if err := m.Set(ctx, key, value); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("key", key.name).Msg("cache could not store a loaded value, the next read loads it again")
	}

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
