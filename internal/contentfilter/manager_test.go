package contentfilter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheck_NoRules(t *testing.T) {
	// given
	e := New()

	// when
	err := e.Check(context.Background(), "body")

	// then
	require.NoError(t, err)
}

func TestCheck_EmptyTextsSkipAllRules(t *testing.T) {
	// given
	r := NewMockRule(t)
	e := New(r)

	// when
	err := e.Check(context.Background(), "", "", "")

	// then
	require.NoError(t, err)
	r.AssertExpectations(t)
}

func TestCheck_AllowsWhenNoRejection(t *testing.T) {
	// given
	r := NewMockRule(t)
	r.EXPECT().Check(mock.Anything, []string{"hello"}).Return(nil, nil).Once()

	e := New(r)

	// when
	err := e.Check(context.Background(), "hello", "")

	// then
	require.NoError(t, err)
	r.AssertExpectations(t)
}

func TestCheck_ShortCircuitsOnFirstRejection(t *testing.T) {
	// given
	first := NewMockRule(t)
	first.EXPECT().Check(mock.Anything, []string{"hello"}).
		Return(&Rejection{Rule: "first", Reason: "nope", Detail: "xyz"}, nil).Once()

	second := NewMockRule(t)

	e := New(first, second)

	// when
	err := e.Check(context.Background(), "hello")

	// then
	var rej *RejectedError
	require.ErrorAs(t, err, &rej)
	assert.Equal(t, RuleName("first"), rej.Rejection.Rule)
	assert.Equal(t, "xyz", rej.Rejection.Detail)
	first.AssertExpectations(t)
	second.AssertExpectations(t)
}

func TestCheck_PropagatesInfraError(t *testing.T) {
	// given
	boom := errors.New("boom")
	r := NewMockRule(t)
	r.EXPECT().Check(mock.Anything, []string{"text"}).Return(nil, boom).Once()

	e := New(r)

	// when
	err := e.Check(context.Background(), "text")

	// then
	require.ErrorIs(t, err, boom)
	var rej *RejectedError
	assert.False(t, errors.As(err, &rej))
}
