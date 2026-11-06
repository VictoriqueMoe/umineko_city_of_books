package observability

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func TestHookStartsTheErrorCounterAtZeroForACommandThatSucceeds(t *testing.T) {
	// given
	client := mock.NewClient(gomock.NewController(t))
	client.EXPECT().
		Do(gomock.Any(), mock.Match("EXISTS", "k")).
		Return(mock.Result(mock.ValkeyInt64(1))).
		Times(1)

	seriesBefore := testutil.CollectAndCount(commandErrors)

	// when
	resp := NewHook().Do(client, t.Context(), client.B().Exists().Key("k").Build())

	// then
	require.NoError(t, resp.Error())
	assert.Equal(t, seriesBefore+1, testutil.CollectAndCount(commandErrors))
	assert.InDelta(t, 0, testutil.ToFloat64(commandErrors.WithLabelValues("EXISTS")), 0)
}

func TestHookCountsAFailedCommand(t *testing.T) {
	// given
	client := mock.NewClient(gomock.NewController(t))
	client.EXPECT().
		Do(gomock.Any(), mock.Match("GET", "k")).
		Return(mock.ErrorResult(errors.New("connection refused"))).
		Times(1)

	before := testutil.ToFloat64(commandErrors.WithLabelValues("GET"))

	// when
	resp := NewHook().Do(client, t.Context(), client.B().Get().Key("k").Build())

	// then
	require.Error(t, resp.Error())
	assert.InDelta(t, before+1, testutil.ToFloat64(commandErrors.WithLabelValues("GET")), 0)
}
