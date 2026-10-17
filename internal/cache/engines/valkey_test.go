package engines

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/mock"
	"github.com/valkey-io/valkey-go/valkeylimiter"
	"go.uber.org/mock/gomock"
)

func newMockValkey(t *testing.T) (*Valkey, *mock.Client) {
	t.Helper()

	client := mock.NewClient(gomock.NewController(t))

	return NewValkeyWithClient(client), client
}

func fixedWindowReply(count int64, resetAt time.Time) valkey.ValkeyResult {
	return mock.Result(mock.ValkeyArray(mock.ValkeyInt64(count), mock.ValkeyInt64(resetAt.UnixMilli())))
}

func TestValkeyDisabledWithoutClient(t *testing.T) {
	tests := []struct {
		name   string
		engine *Valkey
	}{
		{name: "nil engine", engine: nil},
		{name: "unconfigured engine", engine: new(Valkey)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, tt.engine.Enabled())
		})
	}
}

func TestValkeyEnabledWhenHealthy(t *testing.T) {
	v, _ := newMockValkey(t)

	assert.Equal(t, "valkey", v.Name())
	assert.True(t, v.Enabled())
}

func TestValkeyUnconfiguredOperationsAreNoOps(t *testing.T) {
	v := new(Valkey)
	ctx := context.Background()

	_, err := v.Get(ctx, "k")
	require.ErrorIs(t, err, engine.ErrMiss)

	require.NoError(t, v.Set(ctx, "k", []byte("v"), 0))
	require.NoError(t, v.SetMany(ctx, map[string][]byte{"k": []byte("v")}, 0))
	require.NoError(t, v.Del(ctx, "k"))
	require.NoError(t, v.Ping(ctx))
	require.NoError(t, v.Close())
}

func TestValkeyGetReturnsStoredValue(t *testing.T) {
	v, client := newMockValkey(t)

	client.EXPECT().
		Do(gomock.Any(), mock.Match("GET", "k")).
		Return(mock.Result(mock.ValkeyString("golden"))).
		Times(1)

	got, err := v.Get(context.Background(), "k")

	require.NoError(t, err)
	assert.Equal(t, []byte("golden"), got)
}

func TestValkeyTreatsMissingKeyAsHealthyMiss(t *testing.T) {
	v, client := newMockValkey(t)

	client.EXPECT().
		Do(gomock.Any(), mock.Match("GET", "k")).
		Return(mock.Result(mock.ValkeyNil())).
		Times(1)

	_, err := v.Get(context.Background(), "k")

	require.ErrorIs(t, err, engine.ErrMiss)
	assert.True(t, v.Enabled())
}

func TestValkeyDisablesItselfAfterCommandFailure(t *testing.T) {
	v, client := newMockValkey(t)

	wantErr := errors.New("connection refused")

	client.EXPECT().
		Do(gomock.Any(), mock.Match("GET", "k")).
		Return(mock.ErrorResult(wantErr)).
		Times(1)

	_, err := v.Get(context.Background(), "k")

	require.ErrorIs(t, err, wantErr)
	assert.False(t, v.Enabled())
}

func TestValkeyObserveTracksHealth(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantHealthy bool
	}{
		{name: "success keeps it enabled", err: nil, wantHealthy: true},
		{name: "missing key keeps it enabled", err: mock.Result(mock.ValkeyNil()).Error(), wantHealthy: true},
		{name: "command failure disables it", err: errors.New("connection refused"), wantHealthy: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _ := newMockValkey(t)

			got := v.observe(tt.err)

			assert.Equal(t, tt.err, got)
			assert.Equal(t, tt.wantHealthy, v.Enabled())
		})
	}
}

func TestValkeyStaysDisabledDuringCooldown(t *testing.T) {
	v, _ := newMockValkey(t)

	v.observe(errors.New("connection refused"))

	assert.False(t, v.Enabled())
	assert.False(t, v.Enabled())
}

func TestValkeyAllowsOneProbeAfterCooldown(t *testing.T) {
	v, _ := newMockValkey(t)

	v.observe(errors.New("connection refused"))
	v.downAt.Store(time.Now().Add(-recoveryInterval - time.Second).UnixNano())

	assert.True(t, v.Enabled())
	assert.False(t, v.Enabled())
}

func TestValkeyRecoversAfterSuccessfulProbe(t *testing.T) {
	v, client := newMockValkey(t)

	v.observe(errors.New("connection refused"))
	require.False(t, v.Enabled())

	v.downAt.Store(time.Now().Add(-recoveryInterval - time.Second).UnixNano())
	require.True(t, v.Enabled())

	client.EXPECT().
		Do(gomock.Any(), mock.Match("GET", "k")).
		Return(mock.Result(mock.ValkeyString("v"))).
		Times(1)

	got, err := v.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), got)

	assert.True(t, v.Enabled())
	assert.True(t, v.Enabled())
}

func TestValkeyFailedProbeRestartsCooldown(t *testing.T) {
	v, client := newMockValkey(t)

	v.observe(errors.New("connection refused"))
	v.downAt.Store(time.Now().Add(-recoveryInterval - time.Second).UnixNano())
	require.True(t, v.Enabled())

	client.EXPECT().
		Do(gomock.Any(), mock.Match("GET", "k")).
		Return(mock.ErrorResult(errors.New("still refused"))).
		Times(1)

	_, err := v.Get(context.Background(), "k")
	require.Error(t, err)

	assert.False(t, v.Enabled())
}

func TestValkeySetAttachesTTLAsMilliseconds(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want []string
	}{
		{name: "with ttl", ttl: time.Minute, want: []string{"SET", "k", "v", "PX", "60000"}},
		{name: "without ttl", ttl: 0, want: []string{"SET", "k", "v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, client := newMockValkey(t)

			var captured []string

			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
					captured = cmd.Commands()

					return mock.Result(mock.ValkeyString("OK"))
				}).
				Times(1)

			require.NoError(t, v.Set(context.Background(), "k", []byte("v"), tt.ttl))
			assert.Equal(t, tt.want, captured)
		})
	}
}

func TestValkeySkipsRoundTripWhenNothingToDo(t *testing.T) {
	v, _ := newMockValkey(t)
	ctx := context.Background()

	require.NoError(t, v.Del(ctx))
	require.NoError(t, v.SetMany(ctx, nil, 0))
}

func TestValkeyCloseReleasesClientOnce(t *testing.T) {
	v, client := newMockValkey(t)

	client.EXPECT().Close().Times(1)

	require.NoError(t, v.Close())
	assert.False(t, v.Enabled())

	require.NoError(t, v.Close())
}

func TestValkeyReconfigureRejectsInvalidURL(t *testing.T) {
	v := new(Valkey)

	err := v.Reconfigure("://")

	require.Error(t, err)
	assert.False(t, v.Enabled())
}

func TestValkeyReconfigureWithEmptyURLStaysDisabled(t *testing.T) {
	v := new(Valkey)

	require.NoError(t, v.Reconfigure(""))

	assert.False(t, v.Enabled())
}

func TestValkeyIgnoresUnrelatedSettingChanges(t *testing.T) {
	v, _ := newMockValkey(t)

	v.OnSettingChanged(config.SettingLogLevel.Key, "debug")

	assert.True(t, v.Enabled())
}

func TestValkeyClearingURLDisablesEngine(t *testing.T) {
	v, client := newMockValkey(t)
	v.url = "valkey://localhost:6379"

	client.EXPECT().Close().Times(1)

	v.OnSettingChanged(config.SettingValkeyURL.Key, "")

	assert.False(t, v.Enabled())
	assert.Nil(t, v.Client())
}

func TestNewValkeyStartsUnconfigured(t *testing.T) {
	v := NewValkey()

	assert.False(t, v.Enabled())
	assert.Nil(t, v.Client())
}

func TestValkeySetManyDisablesEngineOnFailure(t *testing.T) {
	v, client := newMockValkey(t)

	wantErr := errors.New("connection refused")

	client.EXPECT().
		DoMulti(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmds ...valkey.Completed) []valkey.ValkeyResult {
			return []valkey.ValkeyResult{mock.ErrorResult(wantErr)}
		}).
		Times(1)

	err := v.SetMany(context.Background(), map[string][]byte{"k": []byte("v")}, 0)

	require.ErrorIs(t, err, wantErr)
	assert.False(t, v.Enabled())
}

func TestValkeyUnconfiguredAllowanceReportsUnavailable(t *testing.T) {
	// given
	v := new(Valkey)
	ctx := context.Background()

	// when
	got, allowErr := v.Allow(ctx, "k", 1, time.Minute)
	resetErr := v.Reset(ctx, "k")

	// then
	require.ErrorIs(t, allowErr, engine.ErrUnavailable)
	require.ErrorIs(t, resetErr, engine.ErrUnavailable)
	assert.Zero(t, got)
}

func TestValkeyAllowRunsTheLibraryFixedWindowOnTheCurrentClient(t *testing.T) {
	tests := []struct {
		name               string
		count              int64
		resetIn            time.Duration
		wantAllowed        bool
		wantRetryAfterLow  time.Duration
		wantRetryAfterHigh time.Duration
	}{
		{name: "within the budget is allowed and never reports a negative wait", count: 5, resetIn: 42 * time.Second, wantAllowed: true},
		{name: "over the budget waits out the rest of the window", count: 6, resetIn: 42 * time.Second, wantAllowed: false, wantRetryAfterLow: 40 * time.Second, wantRetryAfterHigh: 42 * time.Second},
		{name: "a window that has just ended still never reports a negative wait", count: 6, resetIn: -time.Second, wantAllowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			v, client := newMockValkey(t)
			before := time.Now()
			resetAt := before.Add(tt.resetIn)

			var captured []string

			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
					captured = cmd.Commands()

					return fixedWindowReply(tt.count, resetAt)
				}).
				Times(1)

			// when
			got, err := v.Allow(context.Background(), "ratelimit:k", 5, time.Minute)

			// then
			after := time.Now()

			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			assert.GreaterOrEqual(t, got.RetryAfter, tt.wantRetryAfterLow)
			assert.LessOrEqual(t, got.RetryAfter, tt.wantRetryAfterHigh)
			assert.True(t, v.Enabled())

			require.Len(t, captured, 8)
			assert.Equal(t, "EVALSHA", captured[0])
			assert.Equal(t, []string{"2", "allowance:{ratelimit:k}", "allowance:{ratelimit:k}:ex", "1"}, captured[2:6])

			windowEndMS, parseErr := strconv.ParseInt(captured[6], 10, 64)
			require.NoError(t, parseErr)
			assert.GreaterOrEqual(t, windowEndMS, before.Add(time.Minute).UnixMilli())
			assert.LessOrEqual(t, windowEndMS, after.Add(time.Minute).UnixMilli())
		})
	}
}

func TestValkeyAllowLoadsTheScriptWhenTheServerHasNotSeenIt(t *testing.T) {
	// given
	v, client := newMockValkey(t)

	var sent []string

	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			sent = append(sent, cmd.Commands()[0])

			if cmd.Commands()[0] == "EVALSHA" {
				return mock.Result(mock.ValkeyError("NOSCRIPT No matching script"))
			}

			return fixedWindowReply(1, time.Now().Add(time.Minute))
		}).
		Times(2)

	// when
	got, err := v.Allow(context.Background(), "k", 1, time.Minute)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"EVALSHA", "EVAL"}, sent)
	assert.True(t, got.Allowed)
	assert.True(t, v.Enabled())
}

func TestValkeyResetDeletesBothLibraryKeys(t *testing.T) {
	// given
	v, client := newMockValkey(t)

	client.EXPECT().
		Do(gomock.Any(), mock.Match("DEL", "allowance:{stream:thumbnail-slot:k}", "allowance:{stream:thumbnail-slot:k}:ex")).
		Return(mock.Result(mock.ValkeyInt64(2))).
		Times(1)

	// when
	err := v.Reset(context.Background(), "stream:thumbnail-slot:k")

	// then
	require.NoError(t, err)
	assert.True(t, v.Enabled())
}

func TestValkeyAllowanceFailureDisablesTheEngine(t *testing.T) {
	connectionRefused := errors.New("connection refused")

	tests := []struct {
		name    string
		reply   valkey.ValkeyResult
		call    func(v *Valkey) error
		wantErr error
	}{
		{
			name:  "allow on an unreachable server",
			reply: mock.ErrorResult(connectionRefused),
			call: func(v *Valkey) error {
				_, err := v.Allow(context.Background(), "k", 1, time.Minute)

				return err
			},
			wantErr: connectionRefused,
		},
		{
			name:  "allow answered with a malformed reply",
			reply: mock.Result(mock.ValkeyArray(mock.ValkeyInt64(1))),
			call: func(v *Valkey) error {
				_, err := v.Allow(context.Background(), "k", 1, time.Minute)

				return err
			},
			wantErr: valkeylimiter.ErrInvalidResponse,
		},
		{
			name:  "reset on an unreachable server",
			reply: mock.ErrorResult(connectionRefused),
			call: func(v *Valkey) error {
				return v.Reset(context.Background(), "k")
			},
			wantErr: connectionRefused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			v, client := newMockValkey(t)

			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				Return(tt.reply).
				Times(1)

			// when
			err := tt.call(v)

			// then
			require.ErrorIs(t, err, tt.wantErr)
			assert.False(t, v.Enabled())
		})
	}
}

func TestValkeyRebindingRebuildsTheLimiterOnTheNewClient(t *testing.T) {
	// given an engine bound to a client that is then swapped out
	v, previous := newMockValkey(t)
	next := mock.NewClient(gomock.NewController(t))

	previous.EXPECT().Close().Times(1)

	next.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(fixedWindowReply(1, time.Now().Add(time.Minute))).
		Times(1)

	v.mu.Lock()
	v.unbind()
	v.bind(next)
	v.mu.Unlock()

	// when
	got, err := v.Allow(context.Background(), "k", 1, time.Minute)

	// then the closed client sees nothing, which its mock enforces
	require.NoError(t, err)
	assert.True(t, got.Allowed)
}

func TestValkeyClearingURLDropsTheLimiterWithTheClient(t *testing.T) {
	// given
	v, client := newMockValkey(t)
	v.url = "valkey://localhost:6379"

	client.EXPECT().Close().Times(1)

	// when
	v.OnSettingChanged(config.SettingValkeyURL.Key, "")
	_, allowErr := v.Allow(context.Background(), "k", 1, time.Minute)
	resetErr := v.Reset(context.Background(), "k")

	// then no limiter is left holding the closed client, which its mock enforces
	require.ErrorIs(t, allowErr, engine.ErrUnavailable)
	require.ErrorIs(t, resetErr, engine.ErrUnavailable)
}

func TestValkeyCloseNeverClosesTheSharedClientTwice(t *testing.T) {
	// given an engine whose limiter shares its client
	v, client := newMockValkey(t)

	client.EXPECT().Close().Times(1)

	// when
	require.NoError(t, v.Close())

	// then the limiter is dropped rather than closed, so the client is closed exactly once
	_, err := v.Allow(context.Background(), "k", 1, time.Minute)
	require.ErrorIs(t, err, engine.ErrUnavailable)
}

func TestProbeURLValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "empty url is accepted", url: "", wantErr: false},
		{name: "blank url is accepted", url: "   ", wantErr: false},
		{name: "malformed url is rejected", url: "://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProbeURL(context.Background(), tt.url)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
