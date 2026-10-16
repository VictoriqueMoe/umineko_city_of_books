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
	"go.uber.org/mock/gomock"
)

func newMockValkey(t *testing.T) (*Valkey, *mock.Client) {
	t.Helper()

	client := mock.NewClient(gomock.NewController(t))

	return NewValkeyWithClient(client), client
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

func TestValkeyUnconfiguredRateStateReportsUnavailable(t *testing.T) {
	// given
	v := new(Valkey)
	ctx := context.Background()

	// when
	_, _, incrementErr := v.IncrementWindow(ctx, "k", time.Minute)
	claimed, claimErr := v.SetIfAbsent(ctx, "k", []byte("v"), time.Minute)
	inForce, deadlineErr := v.ExtendDeadline(ctx, "k", time.Now().Add(time.Minute))

	// then
	require.ErrorIs(t, incrementErr, engine.ErrUnavailable)
	require.ErrorIs(t, claimErr, engine.ErrUnavailable)
	require.ErrorIs(t, deadlineErr, engine.ErrUnavailable)
	assert.False(t, claimed)
	assert.True(t, inForce.IsZero())
}

func TestValkeyExtendDeadlineRunsTheDeadlineScript(t *testing.T) {
	requested := time.Now().Add(time.Minute)
	later := requested.Add(time.Hour)

	tests := []struct {
		name        string
		replyMS     int64
		wantInForce time.Time
	}{
		{name: "the requested deadline is stored", replyMS: requested.UnixMilli(), wantInForce: time.UnixMilli(requested.UnixMilli())},
		{name: "a later deadline already held is kept", replyMS: later.UnixMilli(), wantInForce: time.UnixMilli(later.UnixMilli())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			v, client := newMockValkey(t)

			var captured []string

			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
					captured = cmd.Commands()

					return mock.Result(mock.ValkeyInt64(tt.replyMS))
				}).
				Times(1)

			// when
			inForce, err := v.ExtendDeadline(context.Background(), "giphy:rate-limited-until", requested)

			// then the server decides which deadline is in force, so the reply is what comes back
			require.NoError(t, err)
			assert.True(t, tt.wantInForce.Equal(inForce))
			require.Len(t, captured, 5)
			assert.Equal(t, "EVALSHA", captured[0])
			assert.Equal(t, []string{"1", "giphy:rate-limited-until", strconv.FormatInt(requested.UnixMilli(), 10)}, captured[2:])
			assert.True(t, v.Enabled())
		})
	}
}

func TestValkeyExtendDeadlineRejectsADeadlineThatIsNotInTheFuture(t *testing.T) {
	// given a client that fails the test on any command
	v, _ := newMockValkey(t)

	// when
	inForce, err := v.ExtendDeadline(context.Background(), "k", time.Now().Add(-time.Minute))

	// then
	require.ErrorIs(t, err, engine.ErrDeadlinePassed)
	assert.True(t, inForce.IsZero())
	assert.True(t, v.Enabled())
}

func TestValkeyExtendDeadlineRejectsAMalformedReply(t *testing.T) {
	// given
	v, client := newMockValkey(t)

	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(mock.Result(mock.ValkeyString("golden"))).
		Times(1)

	// when
	inForce, err := v.ExtendDeadline(context.Background(), "k", time.Now().Add(time.Minute))

	// then a malformed answer is not an outage, so the engine stays enabled
	require.ErrorIs(t, err, engine.ErrUnexpectedReply)
	assert.True(t, inForce.IsZero())
	assert.True(t, v.Enabled())
}

func TestValkeyIncrementWindowRunsTheFixedWindowScript(t *testing.T) {
	// given
	v, client := newMockValkey(t)

	var captured []string

	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			captured = cmd.Commands()

			return mock.Result(mock.ValkeyArray(mock.ValkeyInt64(3), mock.ValkeyInt64(42_000)))
		}).
		Times(1)

	// when
	count, remaining, err := v.IncrementWindow(context.Background(), "ratelimit:k", time.Minute)

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.Equal(t, 42*time.Second, remaining)
	require.Len(t, captured, 5)
	assert.Equal(t, "EVALSHA", captured[0])
	assert.Equal(t, []string{"1", "ratelimit:k", "60000"}, captured[2:])
	assert.True(t, v.Enabled())
}

func TestValkeyIncrementWindowLoadsTheScriptWhenTheServerHasNotSeenIt(t *testing.T) {
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

			return mock.Result(mock.ValkeyArray(mock.ValkeyInt64(1), mock.ValkeyInt64(60_000)))
		}).
		Times(2)

	// when
	count, remaining, err := v.IncrementWindow(context.Background(), "k", time.Minute)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"EVALSHA", "EVAL"}, sent)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, time.Minute, remaining)
	assert.True(t, v.Enabled())
}

func TestValkeyIncrementWindowRejectsAMalformedReply(t *testing.T) {
	// given
	v, client := newMockValkey(t)

	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(mock.Result(mock.ValkeyArray(mock.ValkeyInt64(1)))).
		Times(1)

	// when
	_, _, err := v.IncrementWindow(context.Background(), "k", time.Minute)

	// then
	require.ErrorIs(t, err, engine.ErrUnexpectedReply)
}

func TestValkeySetIfAbsentSendsSetNXWithTTL(t *testing.T) {
	tests := []struct {
		name      string
		reply     valkey.ValkeyResult
		wantClaim bool
	}{
		{name: "key was free", reply: mock.Result(mock.ValkeyString("OK")), wantClaim: true},
		{name: "key was taken", reply: mock.Result(mock.ValkeyNil()), wantClaim: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			v, client := newMockValkey(t)

			client.EXPECT().
				Do(gomock.Any(), mock.Match("SET", "k", "v", "NX", "PX", "60000")).
				Return(tt.reply).
				Times(1)

			// when
			claimed, err := v.SetIfAbsent(context.Background(), "k", []byte("v"), time.Minute)

			// then a taken key is a normal answer, not an outage
			require.NoError(t, err)
			assert.Equal(t, tt.wantClaim, claimed)
			assert.True(t, v.Enabled())
		})
	}
}

func TestValkeyRateStateFailureDisablesTheEngine(t *testing.T) {
	tests := []struct {
		name string
		call func(v *Valkey) error
	}{
		{
			name: "increment window",
			call: func(v *Valkey) error {
				_, _, err := v.IncrementWindow(context.Background(), "k", time.Minute)

				return err
			},
		},
		{
			name: "set if absent",
			call: func(v *Valkey) error {
				_, err := v.SetIfAbsent(context.Background(), "k", []byte("v"), time.Minute)

				return err
			},
		},
		{
			name: "extend deadline",
			call: func(v *Valkey) error {
				_, err := v.ExtendDeadline(context.Background(), "k", time.Now().Add(time.Minute))

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			v, client := newMockValkey(t)
			wantErr := errors.New("connection refused")

			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				Return(mock.ErrorResult(wantErr)).
				Times(1)

			// when
			err := tt.call(v)

			// then
			require.ErrorIs(t, err, wantErr)
			assert.False(t, v.Enabled())
		})
	}
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
