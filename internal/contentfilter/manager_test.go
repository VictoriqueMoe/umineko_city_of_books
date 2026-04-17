package contentfilter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRule struct {
	name    RuleName
	reject  *Rejection
	ruleErr error
	called  int
	seen    []string
}

func (r *fakeRule) Name() RuleName { return r.name }

func (r *fakeRule) Check(_ context.Context, texts []string) (*Rejection, error) {
	r.called++
	r.seen = texts
	return r.reject, r.ruleErr
}

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
	r := &fakeRule{name: "r"}
	e := New(r)

	// when
	err := e.Check(context.Background(), "", "", "")

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, r.called)
}

func TestCheck_AllowsWhenNoRejection(t *testing.T) {
	// given
	r := &fakeRule{name: "r"}
	e := New(r)

	// when
	err := e.Check(context.Background(), "hello", "")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, r.called)
	assert.Equal(t, []string{"hello"}, r.seen)
}

func TestCheck_ShortCircuitsOnFirstRejection(t *testing.T) {
	// given
	first := &fakeRule{name: "first", reject: &Rejection{Rule: "first", Reason: "nope", Detail: "xyz"}}
	second := &fakeRule{name: "second"}
	e := New(first, second)

	// when
	err := e.Check(context.Background(), "hello")

	// then
	var rej *RejectedError
	require.ErrorAs(t, err, &rej)
	assert.Equal(t, RuleName("first"), rej.Rejection.Rule)
	assert.Equal(t, "xyz", rej.Rejection.Detail)
	assert.Equal(t, 1, first.called)
	assert.Equal(t, 0, second.called)
}

func TestCheck_PropagatesInfraError(t *testing.T) {
	// given
	boom := errors.New("boom")
	r := &fakeRule{name: "r", ruleErr: boom}
	e := New(r)

	// when
	err := e.Check(context.Background(), "text")

	// then
	require.ErrorIs(t, err, boom)
	var rej *RejectedError
	assert.False(t, errors.As(err, &rej))
}
