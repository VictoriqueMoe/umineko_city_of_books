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

	err := SetMany(context.Background(), m, map[string]string{"a": "1", "b": "2", "c": "3"}, 0)
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

	err := SetMany(context.Background(), m, map[string]string{"k": "v"}, time.Minute)
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

	err := SetMany(context.Background(), m, map[string]string{"k": "v"}, 0)
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

	err := SetMany(context.Background(), m, map[string]sample{"s": {Name: "beatrice", N: 1}}, 0)
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

	err := SetMany(context.Background(), m, map[string]string{"a": "1", "b": "2"}, 0)

	require.ErrorIs(t, err, wantErr)
}

func TestDelSkipsRoundTripWhenNoKeys(t *testing.T) {
	m, _ := newMockManager(t)

	err := m.Del(context.Background())

	require.NoError(t, err)
}

type stubEngine struct {
	name     string
	enabled  bool
	values   map[string][]byte
	closeErr error

	sets   int
	dels   int
	pings  int
	closes int
}

func newStubEngine(name string, enabled bool) *stubEngine {
	return &stubEngine{name: name, enabled: enabled, values: make(map[string][]byte)}
}

func (s *stubEngine) Name() string { return s.name }

func (s *stubEngine) Enabled() bool { return s.enabled }

func (s *stubEngine) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := s.values[key]
	if !ok {
		return nil, engine.ErrMiss
	}

	return data, nil
}

func (s *stubEngine) Set(_ context.Context, key string, data []byte, _ time.Duration) error {
	s.sets++
	s.values[key] = data

	return nil
}

func (s *stubEngine) SetMany(_ context.Context, entries map[string][]byte, _ time.Duration) error {
	s.sets++

	for key, data := range entries {
		s.values[key] = data
	}

	return nil
}

func (s *stubEngine) Del(_ context.Context, keys ...string) error {
	s.dels++

	for i := range keys {
		delete(s.values, keys[i])
	}

	return nil
}

func (s *stubEngine) Ping(_ context.Context) error {
	s.pings++

	return nil
}

func (s *stubEngine) Close() error {
	s.closes++

	return s.closeErr
}

func TestManagerPicksFirstEnabledEngine(t *testing.T) {
	tests := []struct {
		name       string
		firstOn    bool
		secondOn   bool
		wantFirst  int
		wantSecond int
	}{
		{name: "first enabled wins", firstOn: true, secondOn: true, wantFirst: 1, wantSecond: 0},
		{name: "falls through to second", firstOn: false, secondOn: true, wantFirst: 0, wantSecond: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := newStubEngine("first", tt.firstOn)
			second := newStubEngine("second", tt.secondOn)

			m := NewManager(first, second)

			require.NoError(t, Set(context.Background(), m, "k", "v", 0))

			assert.Equal(t, tt.wantFirst, first.sets)
			assert.Equal(t, tt.wantSecond, second.sets)
		})
	}
}

func TestManagerFailsOverWhenEngineGoesDown(t *testing.T) {
	primary := newStubEngine("primary", true)
	fallback := newStubEngine("fallback", true)

	m := NewManager(primary, fallback)
	ctx := context.Background()

	require.NoError(t, Set(ctx, m, "k", "primary value", 0))
	require.Equal(t, 1, primary.sets)
	require.Zero(t, fallback.sets)

	primary.enabled = false

	require.NoError(t, Set(ctx, m, "k", "fallback value", 0))
	assert.Equal(t, 1, primary.sets)
	assert.Equal(t, 1, fallback.sets)

	got, err := Get[string](ctx, m, "k")
	require.NoError(t, err)
	assert.Equal(t, "fallback value", got)
}

func TestManagerWithoutAnyEnabledEngine(t *testing.T) {
	stub := newStubEngine("only", false)

	m := NewManager(stub)
	ctx := context.Background()

	_, err := Get[string](ctx, m, "k")
	require.ErrorIs(t, err, ErrMiss)

	require.NoError(t, Set(ctx, m, "k", "v", 0))
	require.NoError(t, SetMany(ctx, m, map[string]string{"k": "v"}, 0))
	require.NoError(t, m.Del(ctx, "k"))
	require.NoError(t, m.Ping(ctx))

	assert.Zero(t, stub.sets)
	assert.Zero(t, stub.dels)
	assert.Zero(t, stub.pings)
}

func TestManagerNilReceiverIsSafe(t *testing.T) {
	var m *Manager

	ctx := context.Background()

	assert.Nil(t, m.Engines())
	require.NoError(t, m.Del(ctx, "k"))
	require.NoError(t, m.Ping(ctx))
	require.NoError(t, m.Close())

	_, err := Get[string](ctx, m, "k")
	require.ErrorIs(t, err, ErrMiss)
}

func TestManagerExposesEnginesInPriorityOrder(t *testing.T) {
	first := newStubEngine("first", false)
	second := newStubEngine("second", true)

	assert.Equal(t, []string{"first", "second"}, engineNames(NewManager(first, second)))
}

func TestManagerCloseClosesEveryEngineAndReturnsFirstError(t *testing.T) {
	first := newStubEngine("first", true)
	first.closeErr = errors.New("first close failed")

	second := newStubEngine("second", true)
	second.closeErr = errors.New("second close failed")

	disabled := newStubEngine("disabled", false)

	err := NewManager(first, second, disabled).Close()

	require.ErrorIs(t, err, first.closeErr)
	assert.Equal(t, 1, first.closes)
	assert.Equal(t, 1, second.closes)
	assert.Equal(t, 1, disabled.closes)
}

func TestManagerCountsHitsAndMisses(t *testing.T) {
	stub := newStubEngine("stub", true)
	stub.values["present"] = []byte("v")

	m := NewManager(stub)
	ctx := context.Background()

	hitsBefore := testutil.ToFloat64(cacheHits)
	missesBefore := testutil.ToFloat64(cacheMisses)

	got, err := Get[string](ctx, m, "present")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	_, err = Get[string](ctx, m, "absent")
	require.ErrorIs(t, err, ErrMiss)

	assert.Equal(t, hitsBefore+1, testutil.ToFloat64(cacheHits))
	assert.Equal(t, missesBefore+1, testutil.ToFloat64(cacheMisses))
}
