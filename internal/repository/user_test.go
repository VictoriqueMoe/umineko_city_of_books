package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/secrets"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func newCachedUserRepo(t *testing.T) (UserRepository, *dao.MockUserDAO, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	userDAO := dao.NewMockUserDAO(t)

	return NewUserRepo(nil, userDAO, cache.NewManager(engines.NewValkeyWithClient(client)), nil, nil, nil, nil, nil, nil), userDAO, client
}

func TestUserRepository_ScoreAdjustmentsSurviveAFailedLeaderboardInvalidation(t *testing.T) {
	userID := uuid.New()
	cases := []struct {
		name   string
		expect func(userDAO *dao.MockUserDAO)
		call   func(repo UserRepository) error
	}{
		{
			name: "a detective score adjustment",
			expect: func(userDAO *dao.MockUserDAO) {
				userDAO.EXPECT().UpdateMysteryScoreAdjustment(mock.Anything, spec.UserMysteryScoreUpdate{UserID: userID, Adjustment: 5}).Return(nil)
			},
			call: func(repo UserRepository) error {
				return repo.UpdateMysteryScoreAdjustment(context.Background(), spec.UserMysteryScoreUpdate{UserID: userID, Adjustment: 5})
			},
		},
		{
			name: "a game master score adjustment",
			expect: func(userDAO *dao.MockUserDAO) {
				userDAO.EXPECT().UpdateGMScoreAdjustment(mock.Anything, spec.UserGMScoreUpdate{UserID: userID, Adjustment: -2}).Return(nil)
			},
			call: func(repo UserRepository) error {
				return repo.UpdateGMScoreAdjustment(context.Background(), spec.UserGMScoreUpdate{UserID: userID, Adjustment: -2})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given an adjustment that commits while the cache is unreachable
			repo, userDAO, client := newCachedUserRepo(t)
			tc.expect(userDAO)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(valkeymock.ErrorResult(errors.New("valkey down"))).Times(1)

			// when
			err := tc.call(repo)

			// then the committed adjustment is not reported as failed
			require.NoError(t, err)
		})
	}
}

func captureDel(client *valkeymock.Client, commands *[]string) {
	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			*commands = cmd.Commands()

			return valkeymock.Result(valkeymock.ValkeyInt64(1))
		}).
		Times(1)
}

func TestUserRepository_DeletePathsInvalidateCascadedCaches(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name   string
		expect func(userDAO *dao.MockUserDAO)
		call   func(repo UserRepository) error
	}{
		{
			name: "self service delete",
			expect: func(userDAO *dao.MockUserDAO) {
				userDAO.EXPECT().DeleteAccount(mock.Anything, userID).Return(nil)
			},
			call: func(repo UserRepository) error {
				return repo.DeleteAccount(context.Background(), userID)
			},
		},
		{
			name: "admin delete",
			expect: func(userDAO *dao.MockUserDAO) {
				userDAO.EXPECT().AdminDeleteAccount(mock.Anything, userID).Return(nil)
			},
			call: func(repo UserRepository) error {
				return repo.AdminDeleteAccount(context.Background(), userID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, userDAO, client := newCachedUserRepo(t)
			tc.expect(userDAO)

			var commands []string
			captureDel(client, &commands)

			// when
			err := tc.call(repo)

			// then
			require.NoError(t, err)
			require.NotEmpty(t, commands)
			assert.Equal(t, "DEL", commands[0])

			busted := commands[1:]
			required := []string{
				cache.MysteryTopDetectives.Key(),
				cache.MysteryTopGMs.Key(),
				cache.VanityAssignments.Key(),
				cache.UserVanityRoleIDs.Key(userID.String()),
				cache.UserRole.Key(userID.String()),
				cache.GameTopWinners.Key(string(dto.GameTypeChess)),
				cache.GameTopWinners.Key(string(dto.GameTypeCheckers)),
				cache.GameTopWinners.Key(string(dto.GameTypeOthello)),
				cache.GameTopWinners.Key(string(dto.GameTypeMinesweeper)),
				cache.GameTopWinners.Key(string(dto.GameTypeSnakesLadders)),
				cache.SecretHolders.Key(string(secrets.WitchHunter)),
				cache.SecretSolved.Key(string(secrets.WitchHunter)),
				cache.SecretHolders.Key(string(secrets.Piece01)),
				cache.SecretSolved.Key(string(secrets.Piece01)),
			}

			assert.Subset(t, busted, required)
			assert.Len(t, busted, 5+len(leaderboardGameTypes)+2*len(secrets.All()))
		})
	}
}

func TestUserRepository_DeletePathsSkipInvalidationOnDaoError(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name   string
		expect func(userDAO *dao.MockUserDAO)
		call   func(repo UserRepository) error
	}{
		{
			name: "self service delete fails",
			expect: func(userDAO *dao.MockUserDAO) {
				userDAO.EXPECT().DeleteAccount(mock.Anything, userID).Return(errors.New("db down"))
			},
			call: func(repo UserRepository) error {
				return repo.DeleteAccount(context.Background(), userID)
			},
		},
		{
			name: "admin delete fails",
			expect: func(userDAO *dao.MockUserDAO) {
				userDAO.EXPECT().AdminDeleteAccount(mock.Anything, userID).Return(errors.New("db down"))
			},
			call: func(repo UserRepository) error {
				return repo.AdminDeleteAccount(context.Background(), userID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, userDAO, client := newCachedUserRepo(t)
			tc.expect(userDAO)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then
			require.Error(t, err)
		})
	}
}
