package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceTokenDAO_DeleteIsScopedToOwner(t *testing.T) {
	tests := []struct {
		name          string
		deleteAsOwner bool
		wantRemaining []string
	}{
		{name: "another user cannot delete the owner's token", deleteAsOwner: false, wantRemaining: []string{"tok_owned"}},
		{name: "the owner can delete their own token", deleteAsOwner: true, wantRemaining: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			require.NoError(t, repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: owner.ID, Token: "tok_owned", Platform: "android"}))

			deleter := other.ID
			if tc.deleteAsOwner {
				deleter = owner.ID
			}

			// when
			err := repos.DeviceToken.Delete(ctx, spec.DeviceTokenDeletion{UserID: deleter, Token: "tok_owned"})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantRemaining, remainingTokens(t, repos, owner.ID))
		})
	}
}

func TestDeviceTokenDAO_DeleteManyIsScopedToOwner(t *testing.T) {
	tests := []struct {
		name          string
		deleteAsOwner bool
		wantRemaining []string
	}{
		{name: "another user cannot prune the owner's tokens", deleteAsOwner: false, wantRemaining: []string{"tok_one", "tok_two"}},
		{name: "the owner can prune their own tokens", deleteAsOwner: true, wantRemaining: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			require.NoError(t, repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: owner.ID, Token: "tok_one", Platform: "android"}))
			require.NoError(t, repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: owner.ID, Token: "tok_two", Platform: "android"}))

			deleter := other.ID
			if tc.deleteAsOwner {
				deleter = owner.ID
			}

			// when
			err := repos.DeviceToken.DeleteMany(ctx, spec.DeviceTokenBulkDeletion{UserID: deleter, Tokens: []string{"tok_one", "tok_two"}})

			// then
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantRemaining, remainingTokens(t, repos, owner.ID))
		})
	}
}

func TestDeviceTokenDAO_UpsertRebindsTokenOnDeviceHandover(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	previousOwner := daotest.CreateUser(t, repos)
	newOwner := daotest.CreateUser(t, repos)
	require.NoError(t, repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: previousOwner.ID, Token: "tok_handover", Platform: "android"}))

	// when
	err := repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: newOwner.ID, Token: "tok_handover", Platform: "android"})

	// then
	require.NoError(t, err)
	assert.Empty(t, remainingTokens(t, repos, previousOwner.ID))
	assert.Equal(t, []string{"tok_handover"}, remainingTokens(t, repos, newOwner.ID))
}

func remainingTokens(t *testing.T, repos *repository.Repositories, userID uuid.UUID) []string {
	t.Helper()

	registrations, err := repos.DeviceToken.RegistrationsForUser(context.Background(), userID)
	require.NoError(t, err)

	tokens := make([]string, 0, len(registrations))
	for _, reg := range registrations {
		tokens = append(tokens, reg.Token)
	}

	if len(tokens) == 0 {
		return nil
	}

	return tokens
}

func TestDeviceTokenDAO_RegistrationsCarryPlatform(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: user.ID, Token: "tok_native", Platform: "android"}))
	require.NoError(t, repos.DeviceToken.Upsert(ctx, spec.NewDeviceToken{UserID: user.ID, Token: "fid_web", Platform: "web"}))

	// when
	registrations, err := repos.DeviceToken.RegistrationsForUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []model.DeviceRegistration{
		{Token: "tok_native", Platform: "android"},
		{Token: "fid_web", Platform: "web"},
	}, registrations)
}
