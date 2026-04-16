package giphy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_HitAndMiss(t *testing.T) {
	c := newCache(10)
	resp := &Response{Data: []Gif{{ID: "a"}}}

	_, ok := c.get("k1")
	assert.False(t, ok)

	c.set("k1", resp, time.Hour)
	got, ok := c.get("k1")
	require.True(t, ok)
	assert.Equal(t, resp, got)
}

func TestCache_TTLExpiry(t *testing.T) {
	c := newCache(10)
	now := time.Unix(1000, 0)
	c.now = func() time.Time { return now }

	c.set("k1", &Response{}, time.Minute)
	_, ok := c.get("k1")
	require.True(t, ok)

	now = now.Add(61 * time.Second)
	_, ok = c.get("k1")
	assert.False(t, ok)
	assert.Equal(t, 0, c.len())
}

func TestCache_UpdatesExistingEntry(t *testing.T) {
	c := newCache(10)
	first := &Response{Data: []Gif{{ID: "first"}}}
	second := &Response{Data: []Gif{{ID: "second"}}}

	c.set("k1", first, time.Hour)
	c.set("k1", second, time.Hour)

	got, ok := c.get("k1")
	require.True(t, ok)
	assert.Equal(t, "second", got.Data[0].ID)
	assert.Equal(t, 1, c.len())
}

func TestCache_EvictsOldestBeyondMax(t *testing.T) {
	c := newCache(3)
	for i, key := range []string{"a", "b", "c"} {
		c.set(key, &Response{Data: []Gif{{ID: key}}}, time.Hour)
		_ = i
	}
	assert.Equal(t, 3, c.len())

	c.set("d", &Response{Data: []Gif{{ID: "d"}}}, time.Hour)
	assert.Equal(t, 3, c.len())

	_, ok := c.get("a")
	assert.False(t, ok, "oldest entry should be evicted")

	for _, key := range []string{"b", "c", "d"} {
		_, ok := c.get(key)
		assert.True(t, ok, "key %s should still be cached", key)
	}
}

func TestCache_LRU_RecentAccessSurvivesEviction(t *testing.T) {
	c := newCache(3)
	c.set("a", &Response{}, time.Hour)
	c.set("b", &Response{}, time.Hour)
	c.set("c", &Response{}, time.Hour)

	_, _ = c.get("a")
	c.set("d", &Response{}, time.Hour)

	_, ok := c.get("a")
	assert.True(t, ok, "a was recently accessed so should survive")
	_, ok = c.get("b")
	assert.False(t, ok, "b is least recently used so should be evicted")
}
