package newaccountlinks

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/authctx"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/contentfilter"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newRule(t *testing.T, restricted bool) (*Rule, uuid.UUID) {
	t.Helper()

	userID := uuid.New()

	authzSvc := authz.NewMockService(t)
	authzSvc.EXPECT().IsRestrictedNewAccount(mock.Anything, userID).Return(restricted).Maybe()

	return New(authzSvc), userID
}

func TestCheck_RejectsLinksFromANewAccount(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "http url", text: "look at http://localhost:5173/game-board"},
		{name: "https url", text: "https://example.com/gore.png"},
		{name: "www without a scheme", text: "go to www.example.com now"},
		{name: "url buried in a sentence", text: "hi everyone please visit https://bad.site/x thanks"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a member who signed up an hour ago
			rule, userID := newRule(t, true)
			ctx := authctx.WithUserID(context.Background(), userID)

			// when
			rejection, err := rule.Check(ctx, []string{tc.text})

			// then
			require.NoError(t, err)
			require.NotNil(t, rejection)
			assert.Equal(t, contentfilter.RuleNewAccountLinks, rejection.Rule)
		})
	}
}

func TestCheck_AllowsOrdinaryTextFromANewAccount(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "plain sentence", text: "Beatrice did it with the letter"},
		{name: "punctuation without a domain", text: "wait... what?! no. really."},
		{name: "decimal number", text: "the answer is 1.5 or thereabouts"},
		{name: "bare domain, which the frontend never linkifies or previews", text: "check example.moe for details"},
		{name: "a filename is not a domain", text: "the fix is in build.sh and main.rs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			rule, userID := newRule(t, true)
			ctx := authctx.WithUserID(context.Background(), userID)

			// when
			rejection, err := rule.Check(ctx, []string{tc.text})

			// then a new member must still be able to talk
			require.NoError(t, err)
			assert.Nil(t, rejection)
		})
	}
}

func TestCheck_AllowsLinksFromAnUnrestrictedAccount(t *testing.T) {
	// given anyone authz does not restrict: established members, staff, or a failed lookup
	rule, userID := newRule(t, false)
	ctx := authctx.WithUserID(context.Background(), userID)

	// when
	rejection, err := rule.Check(ctx, []string{"https://example.com"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rejection)
}

func TestCheck_SkipsWhenThereIsNoAuthenticatedUser(t *testing.T) {
	// given content with no user in context, such as a system or bot write
	authzSvc := authz.NewMockService(t)
	rule := New(authzSvc)

	// when
	rejection, err := rule.Check(context.Background(), []string{"https://example.com"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rejection)
}
