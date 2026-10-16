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
		{name: "zero ttl never expires, matching valkey", ttl: 0, advance: 30 * time.Second, wantHit: true},
		{name: "zero ttl survives a year, because forever means forever", ttl: 0, advance: 365 * 24 * time.Hour, wantHit: true},
		{name: "negative ttl is treated as forever too", ttl: -time.Minute, advance: 365 * 24 * time.Hour, wantHit: true},
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

func TestInMemoryHoldsZeroTTLUntilEvicted(t *testing.T) {
	// given a zero ttl entry, which valkey would keep forever
	c, clock := newClockedInMemory(t, 0)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "setting:maintenance_mode", []byte("false"), 0))

	// when a long time passes
	clock.advance(24 * time.Hour)

	// then it is still there, so both engines agree on what zero means
	got, err := c.Get(ctx, "setting:maintenance_mode")
	require.NoError(t, err)
	require.Equal(t, []byte("false"), got)

	// and memory is still bounded, because the byte budget evicts rather than the clock
	blob := make([]byte, 400)
	c.maxBytes = entrySize("filler", blob)
	require.NoError(t, c.Set(ctx, "filler", blob, 0))

	_, err = c.Get(ctx, "setting:maintenance_mode")
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

func TestInMemoryIncrementWindowCountsWithinTheWindow(t *testing.T) {
	tests := []struct {
		name          string
		advance       time.Duration
		wantCount     int64
		wantRemaining time.Duration
	}{
		{name: "same instant", advance: 0, wantCount: 2, wantRemaining: time.Minute},
		{name: "part way through", advance: 20 * time.Second, wantCount: 2, wantRemaining: 40 * time.Second},
		{name: "exactly at expiry, matching get", advance: time.Minute, wantCount: 2, wantRemaining: 0},
		{name: "past expiry starts a fresh window", advance: 90 * time.Second, wantCount: 1, wantRemaining: time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c, clock := newClockedInMemory(t, 0)
			ctx := context.Background()

			first, firstRemaining, err := c.IncrementWindow(ctx, "k", time.Minute)
			require.NoError(t, err)
			require.Equal(t, int64(1), first)
			require.Equal(t, time.Minute, firstRemaining)

			// when
			clock.advance(tt.advance)
			count, remaining, err := c.IncrementWindow(ctx, "k", time.Minute)

			// then the window is fixed from the first hit and never slides
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			assert.Equal(t, tt.wantRemaining, remaining)
		})
	}
}

func TestInMemoryIncrementWindowStoresTheCountLikeValkey(t *testing.T) {
	// given
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()

	// when
	for range 3 {
		_, _, err := c.IncrementWindow(ctx, "k", time.Minute)
		require.NoError(t, err)
	}

	// then
	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("3"), got)
	assert.Equal(t, entrySize("k", []byte("3")), c.Bytes())
}

func TestInMemoryIncrementWindowAttachesTheWindowToAKeyWithoutExpiry(t *testing.T) {
	// given a counter that was stored with no expiry
	c, clock := newClockedInMemory(t, 0)
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", []byte("5"), 0))

	// when
	count, remaining, err := c.IncrementWindow(ctx, "k", time.Minute)

	// then it keeps counting and now expires with the window
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
	assert.Equal(t, time.Minute, remaining)

	clock.advance(2 * time.Minute)

	_, err = c.Get(ctx, "k")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemoryIncrementWindowRefusesAValueThatIsNotACounter(t *testing.T) {
	// given
	c, _ := newClockedInMemory(t, 0)
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", []byte("golden"), time.Minute))

	// when
	_, _, err := c.IncrementWindow(ctx, "k", time.Minute)

	// then
	require.ErrorIs(t, err, engine.ErrNotCounter)

	got, getErr := c.Get(ctx, "k")
	require.NoError(t, getErr)
	assert.Equal(t, []byte("golden"), got)
}

func TestInMemoryIncrementWindowIsAtomicUnderConcurrency(t *testing.T) {
	// given
	c := NewInMemory(0)
	ctx := context.Background()

	// when
	var wg sync.WaitGroup
	for range 64 {
		wg.Go(func() {
			_, _, err := c.IncrementWindow(ctx, "k", time.Minute)
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	// then no increment was lost
	count, _, err := c.IncrementWindow(ctx, "k", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(65), count)
}

func TestInMemorySetIfAbsent(t *testing.T) {
	tests := []struct {
		name      string
		existing  []byte
		ttl       time.Duration
		advance   time.Duration
		wantClaim bool
		wantValue []byte
	}{
		{name: "absent key is claimed", existing: nil, wantClaim: true, wantValue: []byte("new")},
		{name: "live key is kept", existing: []byte("old"), ttl: time.Minute, advance: 30 * time.Second, wantClaim: false, wantValue: []byte("old")},
		{name: "key without expiry is kept", existing: []byte("old"), ttl: 0, advance: 24 * time.Hour, wantClaim: false, wantValue: []byte("old")},
		{name: "expired key is claimed again", existing: []byte("old"), ttl: time.Minute, advance: 90 * time.Second, wantClaim: true, wantValue: []byte("new")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c, clock := newClockedInMemory(t, 0)
			ctx := context.Background()

			if tt.existing != nil {
				require.NoError(t, c.Set(ctx, "k", tt.existing, tt.ttl))
			}

			clock.advance(tt.advance)

			// when
			claimed, err := c.SetIfAbsent(ctx, "k", []byte("new"), time.Minute)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.wantClaim, claimed)

			got, getErr := c.Get(ctx, "k")
			require.NoError(t, getErr)
			assert.Equal(t, tt.wantValue, got)
		})
	}
}

func TestInMemorySetIfAbsentExpiresWithItsTTL(t *testing.T) {
	// given
	c, clock := newClockedInMemory(t, 0)
	ctx := context.Background()

	claimed, err := c.SetIfAbsent(ctx, "k", []byte("v"), time.Minute)
	require.NoError(t, err)
	require.True(t, claimed)

	// when
	clock.advance(90 * time.Second)

	// then
	_, err = c.Get(ctx, "k")
	require.ErrorIs(t, err, engine.ErrMiss)
}

func TestInMemorySetIfAbsentAdmitsExactlyOneConcurrentClaim(t *testing.T) {
	// given
	c := NewInMemory(0)
	ctx := context.Background()

	// when
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		wins int
	)
	for range 64 {
		wg.Go(func() {
			claimed, err := c.SetIfAbsent(ctx, "lease", []byte("v"), time.Minute)
			assert.NoError(t, err)

			if !claimed {
				return
			}

			mu.Lock()
			wins++
			mu.Unlock()
		})
	}
	wg.Wait()

	// then
	assert.Equal(t, 1, wins)
}

func TestInMemoryExtendDeadline(t *testing.T) {
	tests := []struct {
		name             string
		existingDeadline time.Duration
		existingRaw      []byte
		existingTTL      time.Duration
		advance          time.Duration
		requested        time.Duration
		wantInForce      time.Duration
	}{
		{name: "absent key takes the deadline", requested: time.Minute, wantInForce: time.Minute},
		{name: "a later deadline extends the one held", existingDeadline: time.Minute, existingTTL: time.Minute, requested: time.Hour, wantInForce: time.Hour},
		{name: "an earlier deadline never overwrites a later one", existingDeadline: time.Hour, existingTTL: time.Hour, requested: time.Minute, wantInForce: time.Hour},
		{name: "the same deadline holds", existingDeadline: time.Minute, existingTTL: time.Minute, requested: time.Minute, wantInForce: time.Minute},
		{name: "a lapsed deadline is replaced", existingDeadline: time.Hour, existingTTL: time.Second, advance: 2 * time.Second, requested: time.Minute, wantInForce: time.Minute},
		{name: "a value that is not a deadline is replaced", existingRaw: []byte("golden"), existingTTL: time.Hour, requested: time.Minute, wantInForce: time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c, clock := newClockedInMemory(t, 0)
			ctx := context.Background()
			start := clock.now

			existing := tt.existingRaw
			if tt.existingDeadline > 0 {
				existing = strconv.AppendInt(nil, start.Add(tt.existingDeadline).UnixMilli(), 10)
			}

			if existing != nil {
				require.NoError(t, c.Set(ctx, "k", existing, tt.existingTTL))
			}

			clock.advance(tt.advance)

			// when
			inForce, err := c.ExtendDeadline(ctx, "k", start.Add(tt.requested))

			// then
			require.NoError(t, err)
			want := start.Add(tt.wantInForce)
			assert.True(t, want.Equal(inForce))

			stored, getErr := c.Get(ctx, "k")
			require.NoError(t, getErr)
			assert.Equal(t, strconv.FormatInt(want.UnixMilli(), 10), string(stored))
		})
	}
}

func TestInMemoryExtendDeadlineExpiresAtTheDeadline(t *testing.T) {
	tests := []struct {
		name    string
		advance time.Duration
		wantHit bool
	}{
		{name: "before the deadline", advance: 30 * time.Second, wantHit: true},
		{name: "exactly at the deadline, matching get", advance: time.Minute, wantHit: true},
		{name: "past the deadline", advance: time.Minute + time.Millisecond, wantHit: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c, clock := newClockedInMemory(t, 0)
			ctx := context.Background()

			_, err := c.ExtendDeadline(ctx, "k", clock.now.Add(time.Minute))
			require.NoError(t, err)

			// when
			clock.advance(tt.advance)
			_, getErr := c.Get(ctx, "k")

			// then
			if tt.wantHit {
				require.NoError(t, getErr)

				return
			}

			require.ErrorIs(t, getErr, engine.ErrMiss)
		})
	}
}

func TestInMemoryExtendDeadlineRejectsADeadlineThatIsNotInTheFuture(t *testing.T) {
	tests := []struct {
		name   string
		offset time.Duration
	}{
		{name: "in the past", offset: -time.Minute},
		{name: "exactly now", offset: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c, clock := newClockedInMemory(t, 0)
			ctx := context.Background()

			// when
			inForce, err := c.ExtendDeadline(ctx, "k", clock.now.Add(tt.offset))

			// then nothing is stored, because a zero or negative expiry would read as forever
			require.ErrorIs(t, err, engine.ErrDeadlinePassed)
			assert.True(t, inForce.IsZero())
			assert.Zero(t, c.Len())
		})
	}
}

func TestInMemoryExtendDeadlineKeepsTheLatestUnderConcurrency(t *testing.T) {
	// given
	c, clock := newClockedInMemory(t, 0)
	ctx := context.Background()
	latest := clock.now.Add(64 * time.Minute)

	// when every caller races its own deadline in
	var wg sync.WaitGroup
	for i := range 64 {
		wg.Go(func() {
			_, err := c.ExtendDeadline(ctx, "k", clock.now.Add(time.Duration(i+1)*time.Minute))
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	// then the latest one wins whatever order they landed in
	inForce, err := c.ExtendDeadline(ctx, "k", clock.now.Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, latest.Equal(inForce))
}

func TestInMemoryHandlesConcurrentAccess(t *testing.T) {
	c := NewInMemory(0)
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := range 64 {
		wg.Go(func() {
			key := strconv.Itoa(i % 16)

			_ = c.Set(ctx, key, []byte(key), 0)
			_, _ = c.Get(ctx, key)
			_ = c.Del(ctx, key)
		})
	}

	wg.Wait()

	assert.LessOrEqual(t, c.Len(), 8)
}
