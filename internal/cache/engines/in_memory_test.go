package engines

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache/engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func newClockedInMemory(t *testing.T, maxEntries int) (*InMemory, *fakeClock) {
	t.Helper()

	clock := &fakeClock{now: time.Date(1986, time.October, 4, 12, 0, 0, 0, time.UTC)}

	c := NewInMemory(maxEntries)
	c.now = clock.Now

	return c, clock
}

func TestInMemoryStoresAndReturnsValues(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "beatrice", []byte("golden"), 0))

	got, err := c.Get(ctx, "beatrice")
	require.NoError(t, err)
	assert.Equal(t, []byte("golden"), got)
}

func TestInMemoryReturnsMissForUnknownKey(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)

	_, err := c.Get(context.Background(), "absent")

	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryDefaultsMaxEntries(t *testing.T) {
	tests := []struct {
		name       string
		maxEntries int
		want       int
	}{
		{name: "zero falls back to default", maxEntries: 0, want: defaultInMemoryMaxEntries},
		{name: "negative falls back to default", maxEntries: -5, want: defaultInMemoryMaxEntries},
		{name: "explicit value is kept", maxEntries: 16, want: 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewInMemory(tt.maxEntries).maxEntries)
		})
	}
}

func TestInMemoryEvictsOldestBeyondMaxEntries(t *testing.T) {
	c, _ := newClockedInMemory(t, 2)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "a", []byte("1"), 0))
	require.NoError(t, c.Set(ctx, "b", []byte("2"), 0))
	require.NoError(t, c.Set(ctx, "c", []byte("3"), 0))

	assert.Equal(t, 2, c.Len())

	_, err := c.Get(ctx, "a")
	require.ErrorIs(t, err, engine.ErrMiss)

	for _, key := range []string{"b", "c"} {
		_, err := c.Get(ctx, key)
		require.NoError(t, err)
	}
}

func TestInMemoryReadRefreshesRecency(t *testing.T) {
	c, _ := newClockedInMemory(t, 2)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "a", []byte("1"), 0))
	require.NoError(t, c.Set(ctx, "b", []byte("2"), 0))

	_, err := c.Get(ctx, "a")
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "c", []byte("3"), 0))

	_, err = c.Get(ctx, "a")
	require.NoError(t, err)

	_, err = c.Get(ctx, "b")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryTTLExpiry(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		advance time.Duration
		wantHit bool
	}{
		{name: "before expiry", ttl: time.Minute, advance: 30 * time.Second, wantHit: true},
		{name: "exactly at expiry", ttl: time.Minute, advance: time.Minute, wantHit: true},
		{name: "past expiry", ttl: time.Minute, advance: 90 * time.Second, wantHit: false},
		{name: "zero ttl still live below the ceiling", ttl: 0, advance: 30 * time.Second, wantHit: true},
		{name: "zero ttl expires at the ceiling", ttl: 0, advance: unboundedTTLCeiling + time.Second, wantHit: false},
		{name: "negative ttl expires at the ceiling", ttl: -time.Minute, advance: unboundedTTLCeiling + time.Second, wantHit: false},
		{name: "long ttl is left alone", ttl: 24 * time.Hour, advance: 12 * time.Hour, wantHit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, clock := newClockedInMemory(t, 0)
			ctx := context.Background()

			require.NoError(t, c.Set(ctx, "k", []byte("v"), tt.ttl))

			clock.advance(tt.advance)

			got, err := c.Get(ctx, "k")

			if !tt.wantHit {
				require.ErrorIs(t, err, engine.ErrMiss)
				assert.Zero(t, c.Len())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, []byte("v"), got)
		})
	}
}

func TestInMemorySetManyStoresEveryEntry(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	entries := map[string][]byte{"a": []byte("1"), "b": []byte("2"), "c": []byte("3")}

	require.NoError(t, c.SetMany(ctx, entries, 0))
	assert.Equal(t, 3, c.Len())

	for key, want := range entries {
		got, err := c.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestInMemorySetManyIgnoresEmptyInput(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)

	require.NoError(t, c.SetMany(context.Background(), nil, 0))

	assert.Zero(t, c.Len())
}

func TestInMemoryDelRemovesOnlyNamedKeys(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "a", []byte("1"), 0))
	require.NoError(t, c.Set(ctx, "b", []byte("2"), 0))

	require.NoError(t, c.Del(ctx, "a", "never-stored"))

	assert.Equal(t, 1, c.Len())

	_, err := c.Get(ctx, "a")
	require.ErrorIs(t, err, engine.ErrMiss)

	_, err = c.Get(ctx, "b")
	require.NoError(t, err)
}

func TestInMemoryOverwriteReplacesWithoutGrowing(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", []byte("first"), 0))
	require.NoError(t, c.Set(ctx, "k", []byte("second"), 0))

	assert.Equal(t, 1, c.Len())

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("second"), got)
}

func TestInMemoryOverwriteResetsExpiry(t *testing.T) {
	c, clock := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", []byte("first"), time.Minute))

	clock.advance(30 * time.Second)
	require.NoError(t, c.Set(ctx, "k", []byte("second"), time.Minute))

	clock.advance(45 * time.Second)

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("second"), got)
}

func TestInMemoryCloseClearsEntries(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", []byte("v"), 0))
	require.NoError(t, c.Close())

	assert.Zero(t, c.Len())

	_, err := c.Get(ctx, "k")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryNeverHoldsAnEntryIndefinitely(t *testing.T) {
	c, clock := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "setting:maintenance_mode", []byte("false"), 0))

	clock.advance(24 * time.Hour)

	_, err := c.Get(ctx, "setting:maintenance_mode")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryEvictsOnByteCeiling(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	c.maxBytes = 1000
	ctx := context.Background()

	blob := make([]byte, 400)
	for _, key := range []string{"a", "b", "c"} {
		require.NoError(t, c.Set(ctx, key, blob, time.Minute))
	}

	assert.LessOrEqual(t, c.Bytes(), 1000)
	assert.Equal(t, 2, c.Len())

	_, err := c.Get(ctx, "a")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryBytesTracksOverwriteAndDelete(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", make([]byte, 500), time.Minute))
	assert.Equal(t, 500, c.Bytes())

	require.NoError(t, c.Set(ctx, "k", make([]byte, 100), time.Minute))
	assert.Equal(t, 100, c.Bytes())

	require.NoError(t, c.Del(ctx, "k"))
	assert.Zero(t, c.Bytes())

	require.NoError(t, c.Set(ctx, "k", make([]byte, 50), time.Minute))
	require.NoError(t, c.Close())
	assert.Zero(t, c.Bytes())
}

func TestInMemoryIsAlwaysAvailable(t *testing.T) {
	c := NewInMemory(0)

	assert.Equal(t, "in-memory", c.Name())
	assert.True(t, c.Enabled())
	require.NoError(t, c.Ping(context.Background()))
}

func TestInMemoryHandlesConcurrentAccess(t *testing.T) {
	c := NewInMemory(8)
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := range 64 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			key := strconv.Itoa(i % 16)

			_ = c.Set(ctx, key, []byte(key), 0)
			_, _ = c.Get(ctx, key)
			_ = c.Del(ctx, key)
		}()
	}

	wg.Wait()

	assert.LessOrEqual(t, c.Len(), 8)
}
