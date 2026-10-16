package cache

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/cache/engines"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func newMockManager(t *testing.T) (*Manager, *mock.Client) {
	t.Helper()

	client := mock.NewClient(gomock.NewController(t))

	m := NewManager(engines.NewValkeyWithClient(client))

	return m, client
}

func newEngineMock(t *testing.T, name string, enabled bool) *engine.MockEngine {
	t.Helper()

	e := engine.NewMockEngine(t)
	e.EXPECT().Name().Return(name).Maybe()
	e.EXPECT().Enabled().Return(enabled).Maybe()

	return e
}

func okResults(n int) []valkey.ValkeyResult {
	resps := make([]valkey.ValkeyResult, n)
	for i := range resps {
		resps[i] = mock.Result(mock.ValkeyString("OK"))
	}

	return resps
}

func TestSetManySendsEveryKeyInOneBatch(t *testing.T) {
	m, client := newMockManager(t)

	var captured [][]string

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			for i := range cmds {
				captured = append(captured, cmds[i].Commands())
			}

			return okResults(len(cmds))
		}).
		Times(1)

	err := m.SetMany(context.Background(), map[string]string{"a": "1", "b": "2", "c": "3"}, 0)
	require.NoError(t, err)

	slices.SortFunc(captured, slices.Compare)

	assert.Equal(t, [][]string{
		{"SET", "a", "1"},
		{"SET", "b", "2"},
		{"SET", "c", "3"},
	}, captured)
}

func TestSetManyAttachesTTLAsMilliseconds(t *testing.T) {
	m, client := newMockManager(t)

	var captured []string

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			captured = cmds[0].Commands()

			return okResults(len(cmds))
		}).
		Times(1)

	err := m.SetMany(context.Background(), map[string]string{"k": "v"}, time.Minute)
	require.NoError(t, err)

	assert.Equal(t, []string{"SET", "k", "v", "PX", "60000"}, captured)
}

func TestSetManyOmitsTTLWhenZero(t *testing.T) {
	m, client := newMockManager(t)

	var captured []string

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			captured = cmds[0].Commands()

			return okResults(len(cmds))
		}).
		Times(1)

	err := m.SetMany(context.Background(), map[string]string{"k": "v"}, 0)
	require.NoError(t, err)

	assert.Equal(t, []string{"SET", "k", "v"}, captured)
	assert.NotContains(t, captured, "PX")
}

func TestSetManyEncodesStructsAsJSON(t *testing.T) {
	m, client := newMockManager(t)

	var captured []string

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			captured = cmds[0].Commands()

			return okResults(len(cmds))
		}).
		Times(1)

	err := m.SetMany(context.Background(), map[string]sample{"s": {Name: "beatrice", N: 1}}, 0)
	require.NoError(t, err)

	assert.Equal(t, []string{"SET", "s", `{"name":"beatrice","n":1}`}, captured)
}

func TestSetManyReturnsCommandError(t *testing.T) {
	m, client := newMockManager(t)

	wantErr := errors.New("valkey unavailable")

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			resps := okResults(len(cmds))
			resps[len(resps)-1] = mock.ErrorResult(wantErr)

			return resps
		}).
		Times(1)

	err := m.SetMany(context.Background(), map[string]string{"a": "1", "b": "2"}, 0)

	require.ErrorIs(t, err, wantErr)
}

func TestDelSkipsRoundTripWhenNoKeys(t *testing.T) {
	m, _ := newMockManager(t)

	err := m.Del(context.Background())

	require.NoError(t, err)
}

func TestManagerPicksFirstEnabledEngine(t *testing.T) {
	tests := []struct {
		name      string
		firstOn   bool
		wantFirst bool
	}{
		{name: "first enabled wins", firstOn: true, wantFirst: true},
		{name: "falls through to second", firstOn: false, wantFirst: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			first := newEngineMock(t, "first", tt.firstOn)
			second := newEngineMock(t, "second", true)

			chosen := second
			if tt.wantFirst {
				chosen = first
			}

			chosen.EXPECT().Set(testifymock.Anything, "k", []byte("v"), time.Duration(0)).Return(nil).Once()

			m := NewManager(first, second)

			// when
			err := m.Set(context.Background(), "k", "v", 0)

			// then the engine that was not chosen saw no write, which the mock enforces
			require.NoError(t, err)
		})
	}
}

func TestManagerFailsOverWhenEngineGoesDown(t *testing.T) {
	// given a primary that is healthy for one call and then goes down
	primary := engine.NewMockEngine(t)
	primary.EXPECT().Enabled().Return(true).Once()
	primary.EXPECT().Enabled().Return(false)
	primary.EXPECT().Set(testifymock.Anything, "k", []byte("primary value"), time.Duration(0)).Return(nil).Once()

	fallback := newEngineMock(t, "fallback", true)
	fallback.EXPECT().Set(testifymock.Anything, "k", []byte("fallback value"), time.Duration(0)).Return(nil).Once()
	fallback.EXPECT().Get(testifymock.Anything, "k").Return([]byte("fallback value"), nil).Once()

	m := NewManager(primary, fallback)
	ctx := context.Background()

	// when
	require.NoError(t, m.Set(ctx, "k", "primary value", 0))
	require.NoError(t, m.Set(ctx, "k", "fallback value", 0))
	got, err := m.Get[string](ctx, "k")

	// then
	require.NoError(t, err)
	assert.Equal(t, "fallback value", got)
}

func TestManagerWithoutAnyEnabledEngine(t *testing.T) {
	// given
	off := newEngineMock(t, "only", false)
	off.EXPECT().Del(testifymock.Anything, []string{"k"}).Return(nil).Once()

	m := NewManager(off)
	ctx := context.Background()

	// when
	_, getErr := m.Get[string](ctx, "k")
	setErr := m.Set(ctx, "k", "v", 0)
	setManyErr := m.SetMany(ctx, map[string]string{"k": "v"}, 0)
	pingErr := m.Ping(ctx)
	delErr := m.Del(ctx, "k")

	// then only the delete reaches the disabled engine
	require.ErrorIs(t, getErr, ErrMiss)
	require.NoError(t, setErr)
	require.NoError(t, setManyErr)
	require.NoError(t, pingErr)
	require.NoError(t, delErr)
}

func TestManagerDelReachesEveryEngineIncludingDisabledOnes(t *testing.T) {
	// given
	primary := newEngineMock(t, "primary", false)
	primary.EXPECT().Del(testifymock.Anything, []string{"role:u1"}).Return(nil).Once()

	fallback := newEngineMock(t, "fallback", true)
	fallback.EXPECT().Del(testifymock.Anything, []string{"role:u1"}).Return(nil).Once()

	m := NewManager(primary, fallback)

	// when
	err := m.Del(context.Background(), "role:u1")

	// then
	require.NoError(t, err)
}

func TestManagerDelSurfacesAnEngineFailure(t *testing.T) {
	// given
	delErr := errors.New("valkey unreachable")

	primary := newEngineMock(t, "primary", false)
	primary.EXPECT().Del(testifymock.Anything, []string{"role:u1"}).Return(delErr).Once()

	fallback := newEngineMock(t, "fallback", true)
	fallback.EXPECT().Del(testifymock.Anything, []string{"role:u1"}).Return(nil).Once()

	// when
	err := NewManager(primary, fallback).Del(context.Background(), "role:u1")

	// then
	require.ErrorIs(t, err, delErr)
}

func TestManagerNilReceiverIsSafe(t *testing.T) {
	var m *Manager

	ctx := context.Background()

	assert.Nil(t, m.Engines())
	require.NoError(t, m.Del(ctx, "k"))
	require.NoError(t, m.Ping(ctx))
	require.NoError(t, m.Close())

	_, err := m.Get[string](ctx, "k")
	require.ErrorIs(t, err, ErrMiss)

	_, err = m.GetDeadline(ctx, "k")
	require.ErrorIs(t, err, ErrMiss)
}

func TestManagerExposesEnginesInPriorityOrder(t *testing.T) {
	first := newEngineMock(t, "first", false)
	second := newEngineMock(t, "second", true)

	assert.Equal(t, []string{"first", "second"}, engineNames(NewManager(first, second)))
}

func TestManagerCloseClosesEveryEngineAndReturnsFirstError(t *testing.T) {
	// given
	firstErr := errors.New("first close failed")

	first := newEngineMock(t, "first", true)
	first.EXPECT().Close().Return(firstErr).Once()

	second := newEngineMock(t, "second", true)
	second.EXPECT().Close().Return(errors.New("second close failed")).Once()

	disabled := newEngineMock(t, "disabled", false)
	disabled.EXPECT().Close().Return(nil).Once()

	// when
	err := NewManager(first, second, disabled).Close()

	// then
	require.ErrorIs(t, err, firstErr)
}

func TestManagerCountsHitsAndMisses(t *testing.T) {
	// given
	stub := newEngineMock(t, "stub", true)
	stub.EXPECT().Get(testifymock.Anything, "present").Return([]byte("v"), nil).Once()
	stub.EXPECT().Get(testifymock.Anything, "absent").Return(nil, engine.ErrMiss).Once()

	m := NewManager(stub)
	ctx := context.Background()

	hitsBefore := testutil.ToFloat64(cacheHits)
	missesBefore := testutil.ToFloat64(cacheMisses)

	// when
	got, hitErr := m.Get[string](ctx, "present")
	_, missErr := m.Get[string](ctx, "absent")

	// then
	require.NoError(t, hitErr)
	assert.Equal(t, "v", got)
	require.ErrorIs(t, missErr, ErrMiss)
	assert.Equal(t, hitsBefore+1, testutil.ToFloat64(cacheHits))
	assert.Equal(t, missesBefore+1, testutil.ToFloat64(cacheMisses))
}

func TestDelDropsDependentsOfDeletedKeys(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantDropped bool
	}{
		{name: "top detectives", key: MysteryTopDetectives.Key(), wantDropped: true},
		{name: "top winners", key: GameTopWinners.Key("chess"), wantDropped: true},
		{name: "setting", key: Setting.Key("site_name"), wantDropped: true},
		{name: "unrelated", key: OGMeta.Key("page"), wantDropped: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			m := NewManager(engines.NewInMemory(0))
			m.Invalidates(SiteInfo, MysteryTopDetectives, GameTopWinners, Setting)
			require.NoError(t, m.Set(ctx, SiteInfo.Key(), "cached", time.Minute))
			require.NoError(t, m.Set(ctx, tc.key, "value", time.Minute))

			// when
			err := m.Del(ctx, tc.key)

			// then
			require.NoError(t, err)
			_, getErr := m.Get[string](ctx, SiteInfo.Key())
			if tc.wantDropped {
				assert.ErrorIs(t, getErr, engine.ErrMiss)
			} else {
				assert.NoError(t, getErr)
			}
		})
	}
}

func TestWithDependentsAddsEachDependentOnce(t *testing.T) {
	// given
	m := NewManager()
	m.Invalidates(SiteInfo, Setting, VanityAssignments)
	keys := []string{Setting.Key("a"), Setting.Key("b"), Setting.Key("c"), VanityAssignments.Key()}

	// when
	got := m.withDependents(keys)

	// then
	assert.Equal(t, append(slices.Clone(keys), SiteInfo.Key()), got)
}

func TestManagerRejectsTTLsShorterThanAMillisecond(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{name: "zero", ttl: 0},
		{name: "negative", ttl: -time.Second},
		{name: "sub millisecond", ttl: 500 * time.Microsecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an engine that fails the test on any call
			m := NewManager(engine.NewMockEngine(t))
			ctx := context.Background()

			// when
			_, _, incrementErr := m.IncrementWindow(ctx, "k", tt.ttl)
			_, claimErr := m.SetIfAbsent(ctx, "k", "v", tt.ttl)

			// then a zero ttl would mean forever, so it never reaches an engine
			require.ErrorIs(t, incrementErr, ErrTTLTooShort)
			require.ErrorIs(t, claimErr, ErrTTLTooShort)
		})
	}
}

func TestManagerRejectsADeadlineThatIsNotInTheFuture(t *testing.T) {
	tests := []struct {
		name     string
		deadline time.Time
	}{
		{name: "zero time", deadline: time.Time{}},
		{name: "in the past", deadline: time.Now().Add(-time.Minute)},
		{name: "already now", deadline: time.Now()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an engine that fails the test on any call
			m := NewManager(engine.NewMockEngine(t))

			// when
			inForce, err := m.ExtendDeadline(context.Background(), "k", tt.deadline)

			// then a deadline that has passed would hold nothing, so it never reaches an engine
			require.ErrorIs(t, err, ErrDeadlinePassed)
			assert.True(t, inForce.IsZero())
		})
	}
}

func TestManagerRateStateWithoutAnyEngine(t *testing.T) {
	tests := []struct {
		name    string
		manager func(t *testing.T) *Manager
	}{
		{name: "nil manager", manager: func(t *testing.T) *Manager { return nil }},
		{name: "no engines", manager: func(t *testing.T) *Manager { return NewManager() }},
		{name: "every engine disabled", manager: func(t *testing.T) *Manager { return NewManager(newEngineMock(t, "off", false)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			m := tt.manager(t)

			// when
			_, _, incrementErr := m.IncrementWindow(ctx, "k", time.Minute)
			claimed, claimErr := m.SetIfAbsent(ctx, "k", "v", time.Minute)
			inForce, deadlineErr := m.ExtendDeadline(ctx, "k", time.Now().Add(time.Minute))

			// then
			require.ErrorIs(t, incrementErr, ErrUnavailable)
			require.ErrorIs(t, claimErr, ErrUnavailable)
			require.ErrorIs(t, deadlineErr, ErrUnavailable)
			assert.False(t, claimed)
			assert.True(t, inForce.IsZero())
		})
	}
}

func TestManagerIncrementWindowCountsOnTheActiveEngine(t *testing.T) {
	// given
	m := NewManager(engines.NewInMemory(0))
	ctx := context.Background()

	// when
	counts := make([]int64, 0, 3)
	for range 3 {
		count, remaining, err := m.IncrementWindow(ctx, RateLimitMail.Key("ip:203.0.113.7"), RateLimitMail.TTL)
		require.NoError(t, err)
		assert.Positive(t, remaining)
		assert.LessOrEqual(t, remaining, RateLimitMail.TTL)

		counts = append(counts, count)
	}

	// then
	assert.Equal(t, []int64{1, 2, 3}, counts)
}

func TestManagerSetIfAbsentClaimsOnce(t *testing.T) {
	// given
	m := NewManager(engines.NewInMemory(0))
	ctx := context.Background()

	// when
	first, firstErr := m.SetIfAbsent(ctx, "lease", "battler", time.Minute)
	second, secondErr := m.SetIfAbsent(ctx, "lease", "beatrice", time.Minute)

	// then
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.True(t, first)
	assert.False(t, second)

	got, err := m.Get[string](ctx, "lease")
	require.NoError(t, err)
	assert.Equal(t, "battler", got)
}

func TestManagerExtendDeadlineKeepsTheLatest(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		first       time.Time
		second      time.Time
		wantInForce time.Time
	}{
		{name: "a later deadline extends", first: now.Add(time.Minute), second: now.Add(time.Hour), wantInForce: now.Add(time.Hour)},
		{name: "an earlier deadline never overwrites a later one", first: now.Add(time.Hour), second: now.Add(time.Minute), wantInForce: now.Add(time.Hour)},
		{name: "the same deadline holds", first: now.Add(time.Minute), second: now.Add(time.Minute), wantInForce: now.Add(time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			m := NewManager(engines.NewInMemory(0))
			ctx := context.Background()
			key := GiphyRateLimitedUntil.Key()

			_, err := m.ExtendDeadline(ctx, key, tt.first)
			require.NoError(t, err)

			// when
			inForce, err := m.ExtendDeadline(ctx, key, tt.second)

			// then
			require.NoError(t, err)
			want := time.UnixMilli(tt.wantInForce.UnixMilli())
			assert.True(t, want.Equal(inForce))

			stored, getErr := m.GetDeadline(ctx, key)
			require.NoError(t, getErr)
			assert.True(t, want.Equal(stored))
		})
	}
}

func TestManagerGetDeadline(t *testing.T) {
	deadline := time.Now().Add(time.Minute)

	tests := []struct {
		name    string
		store   func(t *testing.T, m *Manager)
		want    time.Time
		wantErr error
	}{
		{
			name:    "missing key is a miss",
			store:   func(t *testing.T, m *Manager) {},
			wantErr: ErrMiss,
		},
		{
			name: "a stored deadline is read back",
			store: func(t *testing.T, m *Manager) {
				_, err := m.ExtendDeadline(context.Background(), "k", deadline)
				require.NoError(t, err)
			},
			want: time.UnixMilli(deadline.UnixMilli()),
		},
		{
			name: "a value that is not a deadline is refused",
			store: func(t *testing.T, m *Manager) {
				require.NoError(t, m.Set(context.Background(), "k", "golden", time.Minute))
			},
			wantErr: ErrNotDeadline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			m := NewManager(engines.NewInMemory(0))
			ctx := context.Background()
			tt.store(t, m)

			// when
			got, err := m.GetDeadline(ctx, "k")

			// then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got))
		})
	}
}

func TestManagerRateStateFallsBackAfterValkeyFails(t *testing.T) {
	tests := []struct {
		name string
		call func(m *Manager) (bool, error)
	}{
		{
			name: "increment window",
			call: func(m *Manager) (bool, error) {
				count, _, err := m.IncrementWindow(context.Background(), "k", time.Minute)

				return count == 1, err
			},
		},
		{
			name: "set if absent",
			call: func(m *Manager) (bool, error) {
				return m.SetIfAbsent(context.Background(), "k", "v", time.Minute)
			},
		},
		{
			name: "extend deadline",
			call: func(m *Manager) (bool, error) {
				deadline := time.Now().Add(time.Minute)

				inForce, err := m.ExtendDeadline(context.Background(), "k", deadline)

				return inForce.Equal(time.UnixMilli(deadline.UnixMilli())), err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given valkey answering with a failure once
			client := mock.NewClient(gomock.NewController(t))
			wantErr := errors.New("connection refused")

			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				Return(mock.ErrorResult(wantErr)).
				Times(1)

			m := NewManager(engines.NewValkeyWithClient(client), engines.NewInMemory(0))

			// when
			_, failedErr := tt.call(m)
			fresh, fallbackErr := tt.call(m)

			// then the failing request sees the error, and the next one is served by the fallback
			require.ErrorIs(t, failedErr, wantErr)
			require.NoError(t, fallbackErr)
			assert.True(t, fresh)
		})
	}
}

func TestWithDependentsLeavesUnrelatedKeysAlone(t *testing.T) {
	// given
	m := NewManager()
	m.Invalidates(SiteInfo, Setting)
	keys := []string{OGMeta.Key("page")}

	// when
	got := m.withDependents(keys)

	// then
	assert.Equal(t, keys, got)
}
