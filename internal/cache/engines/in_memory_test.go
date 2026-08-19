package engines

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/config"

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

func smallEntries(n int) int {
	return n * (len("a") + len("1") + entryOverheadBytes)
}

func newClockedInMemory(t *testing.T, maxBytes int) (*InMemory, *fakeClock) {
	t.Helper()

	clock := &fakeClock{now: time.Date(1986, time.October, 4, 12, 0, 0, 0, time.UTC)}

	c := NewInMemory(maxBytes)
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

func TestInMemoryDefaultsMaxBytes(t *testing.T) {
	tests := []struct {
		name     string
		maxBytes int
		want     int
	}{
		{name: "zero falls back to default", maxBytes: 0, want: defaultInMemoryMaxBytes},
		{name: "negative falls back to default", maxBytes: -5, want: defaultInMemoryMaxBytes},
		{name: "explicit value is kept", maxBytes: 4096, want: 4096},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewInMemory(tt.maxBytes).maxBytes)
		})
	}
}

func TestInMemoryEvictsOldestBeyondTheByteBudget(t *testing.T) {
	c, _ := newClockedInMemory(t, smallEntries(2))
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
	c, _ := newClockedInMemory(t, smallEntries(2))
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
	// given a budget with room for exactly two entries once overhead is counted
	c, _ := newClockedInMemory(t, 0)
	blob := make([]byte, 400)
	budget := 2 * entrySize("a", blob)
	c.maxBytes = budget
	ctx := context.Background()

	// when a third arrives
	for _, key := range []string{"a", "b", "c"} {
		require.NoError(t, c.Set(ctx, key, blob, time.Minute))
	}

	// then the oldest is evicted and the budget is respected
	assert.LessOrEqual(t, c.Bytes(), budget)
	assert.Equal(t, 2, c.Len())

	_, err := c.Get(ctx, "a")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryCountsKeyAndPerEntryOverheadNotJustThePayload(t *testing.T) {
	// given
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	// when
	require.NoError(t, c.Set(ctx, "beatrice", make([]byte, 100), time.Minute))

	// then the budget reflects real footprint, so tiny values cannot blow past it unseen
	assert.Equal(t, len("beatrice")+100+entryOverheadBytes, c.Bytes())
}

func TestInMemoryResizeMBEvictsImmediately(t *testing.T) {
	// given a cache holding more than the new budget will allow
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	for _, key := range []string{"a", "b", "c"} {
		require.NoError(t, c.Set(ctx, key, make([]byte, 400*1024), time.Minute))
	}
	require.Equal(t, 3, c.Len())

	// when the limit is shrunk to one megabyte
	c.ResizeMB(1)

	// then it evicts on the spot rather than waiting for the next write
	assert.LessOrEqual(t, c.Bytes(), bytesPerMB)
	assert.Less(t, c.Len(), 3)
}

func TestInMemoryResizeMBClampsOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name string
		mb   int
		want int
	}{
		{name: "zero falls back to the default", mb: 0, want: defaultInMemoryMaxMB * bytesPerMB},
		{name: "negative falls back to the default", mb: -3, want: defaultInMemoryMaxMB * bytesPerMB},
		{name: "absurdly large is clamped", mb: 1 << 20, want: maxInMemoryMaxMB * bytesPerMB},
		{name: "a sane value is kept", mb: 64, want: 64 * bytesPerMB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c, _ := newClockedInMemory(t, 0)

			// when
			c.ResizeMB(tt.mb)

			// then
			assert.Equal(t, tt.want, c.maxBytes)
		})
	}
}

func TestInMemoryOnSettingChangedOnlyReactsToItsOwnKey(t *testing.T) {
	// given
	c, _ := newClockedInMemory(t, 0)
	before := c.maxBytes

	// when an unrelated setting changes
	c.OnSettingChanged(config.SettingValkeyURL.Key, "redis://localhost:6379")

	// then
	assert.Equal(t, before, c.maxBytes)
}

func TestInMemoryOnSettingChangedKeepsTheLimitWhenTheValueIsNotANumber(t *testing.T) {
	// given
	c, _ := newClockedInMemory(t, 0)
	c.ResizeMB(64)
	before := c.maxBytes

	// when
	c.OnSettingChanged(config.SettingCacheInMemoryMaxMB.Key, "not-a-number")

	// then a bad value must never shrink the cache to nothing
	assert.Equal(t, before, c.maxBytes)
}

func TestInMemoryOnSettingChangedAppliesTheNewLimit(t *testing.T) {
	// given
	c, _ := newClockedInMemory(t, 0)

	// when
	c.OnSettingChanged(config.SettingCacheInMemoryMaxMB.Key, "32")

	// then
	assert.Equal(t, 32*bytesPerMB, c.maxBytes)
}

func TestInMemoryBytesTracksOverwriteAndDelete(t *testing.T) {
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", make([]byte, 500), time.Minute))
	assert.Equal(t, entrySize("k", make([]byte, 500)), c.Bytes())

	require.NoError(t, c.Set(ctx, "k", make([]byte, 100), time.Minute))
	assert.Equal(t, entrySize("k", make([]byte, 100)), c.Bytes())

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
	c := NewInMemory(0)
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
