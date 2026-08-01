package cache

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func newMockManager(t *testing.T) (*Manager, *mock.Client) {
	t.Helper()

	client := mock.NewClient(gomock.NewController(t))

	m := NewManager()
	m.client = client

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
