package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func engineNames(m *Manager) []string {
	list := m.Engines()

	names := make([]string, 0, len(list))
	for i := range list {
		names = append(names, list[i].Name())
	}

	return names
}

func TestNewEnginesOrdersValkeyAheadOfInMemory(t *testing.T) {
	list := newEngines()

	require.Len(t, list, 2)
	assert.Equal(t, "valkey", list[0].Name())
	assert.Equal(t, "in-memory", list[1].Name())
}

func TestNewBuildsManagerWithDefaultEngines(t *testing.T) {
	assert.Equal(t, []string{"valkey", "in-memory"}, engineNames(New()))
}

func TestNewLeavesValkeyDisabledUntilConfigured(t *testing.T) {
	list := New().Engines()

	require.Len(t, list, 2)
	assert.False(t, list[0].Enabled())
	assert.True(t, list[1].Enabled())
}

func TestNewFallsBackToInMemoryWithoutValkey(t *testing.T) {
	m := New()
	ctx := context.Background()

	want := sample{Name: "beatrice", N: 1}

	require.NoError(t, m.Set(ctx, "witch", want, 0))

	got, err := m.Get[sample](ctx, "witch")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	require.NoError(t, m.Del(ctx, "witch"))

	_, err = m.Get[sample](ctx, "witch")
	require.ErrorIs(t, err, ErrMiss)
}
