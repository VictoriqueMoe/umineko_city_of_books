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

var (
	untimed = Namespace{Prefix: "untimed:"}
	timed   = Namespace{Prefix: "timed:", TTL: time.Minute}
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

	err := m.SetMany(context.Background(), map[Key]string{untimed.Key("a"): "1", untimed.Key("b"): "2", untimed.Key("c"): "3"})
	require.NoError(t, err)

	slices.SortFunc(captured, slices.Compare)

	assert.Equal(t, [][]string{
		{"SET", "untimed:a", "1"},
		{"SET", "untimed:b", "2"},
		{"SET", "untimed:c", "3"},
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

	err := m.SetMany(context.Background(), map[Key]string{timed.Key("k"): "v"})
	require.NoError(t, err)

	assert.Equal(t, []string{"SET", "timed:k", "v", "PX", "60000"}, captured)
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

	err := m.SetMany(context.Background(), map[Key]string{untimed.Key("k"): "v"})
	require.NoError(t, err)

	assert.Equal(t, []string{"SET", "untimed:k", "v"}, captured)
	assert.NotContains(t, captured, "PX")
}

func TestSetManyWritesOneBatchPerNamespaceTTL(t *testing.T) {
	// given
	m, client := newMockManager(t)

	var captured [][]string

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			for _, cmd := range cmds {
				captured = append(captured, cmd.Commands())
			}

			return okResults(len(cmds))
		}).
		Times(2)

	// when
	err := m.SetMany(context.Background(), map[Key]string{untimed.Key("a"): "1", untimed.Key("b"): "2", timed.Key("c"): "3"})

	// then each key is written with the ttl of its own namespace
	require.NoError(t, err)

	slices.SortFunc(captured, slices.Compare)

	assert.Equal(t, [][]string{
		{"SET", "timed:c", "3", "PX", "60000"},
		{"SET", "untimed:a", "1"},
		{"SET", "untimed:b", "2"},
	}, captured)
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

	err := m.SetMany(context.Background(), map[Key]sample{untimed.Key("s"): {Name: "beatrice", N: 1}})
	require.NoError(t, err)

	assert.Equal(t, []string{"SET", "untimed:s", `{"name":"beatrice","n":1}`}, captured)
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

	err := m.SetMany(context.Background(), map[Key]string{untimed.Key("a"): "1", untimed.Key("b"): "2"})

	require.ErrorIs(t, err, wantErr)
}

func TestDelSkipsRoundTripWhenNoKeys(t *testing.T) {
	m, _ := newMockManager(t)

	err := m.Del(context.Background())

	require.NoError(t, err)
}

func TestManagerSetUsesTheNamespaceTTL(t *testing.T) {
	tests := []struct {
		name    string
		key     Key
		wantKey string
		wantTTL time.Duration
	}{
		{name: "a namespace without expiry", key: untimed.Key("k"), wantKey: "untimed:k", wantTTL: 0},
		{name: "a namespace with a ttl", key: timed.Key("k"), wantKey: "timed:k", wantTTL: time.Minute},
		{name: "a real namespace", key: OGMeta.Key("/theory/abc"), wantKey: "og:meta:/theory/abc", wantTTL: OGMeta.TTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			stub := newEngineMock(t, "stub", true)
			stub.EXPECT().Set(testifymock.Anything, tt.wantKey, []byte("v"), tt.wantTTL).Return(nil).Once()

			// when
			err := NewManager(stub).Set(context.Background(), tt.key, "v")

			// then the engine saw the key's own ttl, which the mock enforces
			require.NoError(t, err)
		})
	}
}

func TestManagerDelAcrossNamespaces(t *testing.T) {
	// given
	stub := newEngineMock(t, "stub", true)
	stub.EXPECT().Del(testifymock.Anything, []string{"untimed:a", "timed:b", "role:u1"}).Return(nil).Once()

	m := NewManager(stub)

	// when
	err := m.Del(context.Background(), untimed.Key("a"), timed.Key("b"), UserRole.Key("u1"))

	// then every key reaches the engine in one call, whatever its namespace ttl
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

			chosen.EXPECT().Set(testifymock.Anything, "untimed:k", []byte("v"), time.Duration(0)).Return(nil).Once()

			m := NewManager(first, second)

			// when
			err := m.Set(context.Background(), untimed.Key("k"), "v")

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
	primary.EXPECT().Set(testifymock.Anything, "untimed:k", []byte("primary value"), time.Duration(0)).Return(nil).Once()

	fallback := newEngineMock(t, "fallback", true)
	fallback.EXPECT().Set(testifymock.Anything, "untimed:k", []byte("fallback value"), time.Duration(0)).Return(nil).Once()
	fallback.EXPECT().Get(testifymock.Anything, "untimed:k").Return([]byte("fallback value"), nil).Once()

	m := NewManager(primary, fallback)
	ctx := context.Background()
	key := untimed.Key("k")

	// when
	require.NoError(t, m.Set(ctx, key, "primary value"))
	require.NoError(t, m.Set(ctx, key, "fallback value"))
	got, err := m.Get[string](ctx, key)

	// then
	require.NoError(t, err)
	assert.Equal(t, "fallback value", got)
}

func TestManagerWithoutAnyEnabledEngine(t *testing.T) {
	// given
	off := newEngineMock(t, "only", false)
	off.EXPECT().Del(testifymock.Anything, []string{"untimed:k"}).Return(nil).Once()

	m := NewManager(off)
	ctx := context.Background()
	key := untimed.Key("k")

	// when
	_, getErr := m.Get[string](ctx, key)
	setErr := m.Set(ctx, key, "v")
	setManyErr := m.SetMany(ctx, map[Key]string{key: "v"})
	pingErr := m.Ping(ctx)
	delErr := m.Del(ctx, key)

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
	err := m.Del(context.Background(), UserRole.Key("u1"))

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
	err := NewManager(primary, fallback).Del(context.Background(), UserRole.Key("u1"))

	// then
	require.ErrorIs(t, err, delErr)
}

func TestManagerNilReceiverIsSafe(t *testing.T) {
	var m *Manager

	ctx := context.Background()
	key := untimed.Key("k")

	assert.Nil(t, m.Engines())
	require.NoError(t, m.Del(ctx, key))
	require.NoError(t, m.Ping(ctx))
	require.NoError(t, m.Close())

	_, err := m.Get[string](ctx, key)
	require.ErrorIs(t, err, ErrMiss)

	_, err = m.Allow(ctx, key, 1, time.Minute)
	require.ErrorIs(t, err, ErrUnavailable)

	require.ErrorIs(t, m.Reset(ctx, key), ErrUnavailable)
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
	stub.EXPECT().Get(testifymock.Anything, "untimed:present").Return([]byte("v"), nil).Once()
	stub.EXPECT().Get(testifymock.Anything, "untimed:absent").Return(nil, engine.ErrMiss).Once()

	m := NewManager(stub)
	ctx := context.Background()

	hitsBefore := testutil.ToFloat64(cacheHits)
	missesBefore := testutil.ToFloat64(cacheMisses)

	// when
	got, hitErr := m.Get[string](ctx, untimed.Key("present"))
	_, missErr := m.Get[string](ctx, untimed.Key("absent"))

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
		key         Key
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
			require.NoError(t, m.Set(ctx, SiteInfo.Key(), "cached"))
			require.NoError(t, m.Set(ctx, tc.key, "value"))

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
	keys := []string{Setting.Key("a").String(), Setting.Key("b").String(), Setting.Key("c").String(), VanityAssignments.Key().String()}

	// when
	got := m.withDependents(keys)

	// then
	assert.Equal(t, append(slices.Clone(keys), SiteInfo.Key().String()), got)
}

func TestManagerRejectsAnInvalidAllowance(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		window time.Duration
	}{
		{name: "zero limit", limit: 0, window: time.Minute},
		{name: "negative limit", limit: -1, window: time.Minute},
		{name: "zero window", limit: 1, window: 0},
		{name: "negative window", limit: 1, window: -time.Second},
		{name: "sub millisecond window", limit: 1, window: 500 * time.Microsecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an engine that fails the test on any call
			m := NewManager(engine.NewMockEngine(t))

			// when
			got, err := m.Allow(context.Background(), untimed.Key("k"), tt.limit, tt.window)

			// then the valkey limiter would quietly swap in its own defaults, so it never reaches an engine
			require.ErrorIs(t, err, ErrInvalidAllowance)
			assert.Zero(t, got)
		})
	}
}

func TestManagerAllowanceWithoutAnyEngine(t *testing.T) {
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
			key := untimed.Key("k")

			// when
			got, allowErr := m.Allow(ctx, key, 1, time.Minute)
			resetErr := m.Reset(ctx, key)

			// then
			require.ErrorIs(t, allowErr, ErrUnavailable)
			require.ErrorIs(t, resetErr, ErrUnavailable)
			assert.Zero(t, got)
		})
	}
}

func TestManagerAllowanceGoesToTheFirstEnabledEngine(t *testing.T) {
	// given
	off := newEngineMock(t, "off", false)

	on := newEngineMock(t, "on", true)
	on.EXPECT().Allow(testifymock.Anything, "untimed:k", 2, time.Minute).Return(engine.Allowance{Allowed: true}, nil).Once()
	on.EXPECT().Reset(testifymock.Anything, "untimed:k").Return(nil).Once()

	m := NewManager(off, on)
	ctx := context.Background()
	key := untimed.Key("k")

	// when
	got, allowErr := m.Allow(ctx, key, 2, time.Minute)
	resetErr := m.Reset(ctx, key)

	// then the disabled engine saw nothing, which the mock enforces
	require.NoError(t, allowErr)
	require.NoError(t, resetErr)
	assert.True(t, got.Allowed)
}

func TestManagerAllowSpendsTheBudgetOnTheActiveEngine(t *testing.T) {
	// given
	m := NewManager(engines.NewInMemory(0))
	ctx := context.Background()
	key := RateLimitForgotPassword.Key("ip:203.0.113.7")

	// when
	allowed := make([]bool, 0, 3)
	for range 3 {
		got, err := m.Allow(ctx, key, 3, RateLimitForgotPassword.TTL)
		require.NoError(t, err)

		allowed = append(allowed, got.Allowed)
	}

	refused, err := m.Allow(ctx, key, 3, RateLimitForgotPassword.TTL)

	// then
	require.NoError(t, err)
	assert.Equal(t, []bool{true, true, true}, allowed)
	assert.False(t, refused.Allowed)
	assert.Positive(t, refused.RetryAfter)
	assert.LessOrEqual(t, refused.RetryAfter, RateLimitForgotPassword.TTL)
}

func TestManagerResetHandsTheAllowanceBack(t *testing.T) {
	// given a lease that is held
	m := NewManager(engines.NewInMemory(0))
	ctx := context.Background()
	lease := untimed.Key("lease")

	first, err := m.Allow(ctx, lease, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, first.Allowed)

	held, err := m.Allow(ctx, lease, 1, time.Minute)
	require.NoError(t, err)
	require.False(t, held.Allowed)

	// when
	require.NoError(t, m.Reset(ctx, lease))
	got, err := m.Allow(ctx, lease, 1, time.Minute)

	// then
	require.NoError(t, err)
	assert.True(t, got.Allowed)
}

func TestManagerAllowanceFallsBackAfterValkeyFails(t *testing.T) {
	tests := []struct {
		name string
		call func(m *Manager) (bool, error)
	}{
		{
			name: "allow",
			call: func(m *Manager) (bool, error) {
				got, err := m.Allow(context.Background(), untimed.Key("k"), 1, time.Minute)

				return got.Allowed, err
			},
		},
		{
			name: "reset",
			call: func(m *Manager) (bool, error) {
				err := m.Reset(context.Background(), untimed.Key("k"))

				return err == nil, err
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
	keys := []string{OGMeta.Key("page").String()}

	// when
	got := m.withDependents(keys)

	// then
	assert.Equal(t, keys, got)
}
