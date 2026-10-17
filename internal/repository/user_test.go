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

func newInMemoryUserRepo(t *testing.T) (UserRepository, *dao.MockUserDAO) {
	t.Helper()

	userDAO := dao.NewMockUserDAO(t)

	return NewUserRepo(nil, userDAO, cache.NewManager(engines.NewInMemory(0)), nil, nil, nil, nil, nil, nil), userDAO
}

func TestUserRepository_GetIPLoadsOnceThenServesTheCache(t *testing.T) {
	// given
	repo, userDAO := newInMemoryUserRepo(t)
	userID := uuid.New()
	userDAO.EXPECT().GetIP(mock.Anything, userID).Return("10.0.0.1", nil).Once()

	// when
	first, firstErr := repo.GetIP(context.Background(), userID)
	second, secondErr := repo.GetIP(context.Background(), userID)

	// then
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.Equal(t, "10.0.0.1", first)
	assert.Equal(t, "10.0.0.1", second)
}

func TestUserRepository_UpdateIPCachesTheWrittenIP(t *testing.T) {
	// given
	repo, userDAO := newInMemoryUserRepo(t)
	userID := uuid.New()
	update := spec.UserIPUpdate{UserID: userID, IP: "10.0.0.2"}
	userDAO.EXPECT().UpdateIP(mock.Anything, update).Return(nil).Once()

	// when
	err := repo.UpdateIP(context.Background(), update)
	got, getErr := repo.GetIP(context.Background(), userID)

	// then the read is answered by the cache, never the dao
	require.NoError(t, err)
	require.NoError(t, getErr)
	assert.Equal(t, "10.0.0.2", got)
}

func TestUserRepository_FailedUpdateIPLeavesTheCacheAlone(t *testing.T) {
	// given
	repo, userDAO := newInMemoryUserRepo(t)
	userID := uuid.New()
	update := spec.UserIPUpdate{UserID: userID, IP: "10.0.0.2"}
	writeErr := errors.New("db down")
	userDAO.EXPECT().UpdateIP(mock.Anything, update).Return(writeErr).Once()
	userDAO.EXPECT().GetIP(mock.Anything, userID).Return("10.0.0.1", nil).Once()

	// when
	err := repo.UpdateIP(context.Background(), update)
	got, getErr := repo.GetIP(context.Background(), userID)

	// then
	require.ErrorIs(t, err, writeErr)
	require.NoError(t, getErr)
	assert.Equal(t, "10.0.0.1", got)
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
				cache.MysteryTopDetectives.Key().String(),
				cache.MysteryTopGMs.Key().String(),
				cache.VanityAssignments.Key().String(),
				cache.UserVanityRoleIDs.Key(userID.String()).String(),
				cache.UserRole.Key(userID.String()).String(),
				cache.UserIP.Key(userID.String()).String(),
				cache.UserBlockedIDs.Key(userID.String()).String(),
				cache.PostCornerCounts.Key().String(),
				cache.ArtCornerCounts.Key().String(),
				cache.GameTopWinners.Key(string(dto.GameTypeChess)).String(),
				cache.GameTopWinners.Key(string(dto.GameTypeCheckers)).String(),
				cache.GameTopWinners.Key(string(dto.GameTypeOthello)).String(),
				cache.GameTopWinners.Key(string(dto.GameTypeMinesweeper)).String(),
				cache.GameTopWinners.Key(string(dto.GameTypeSnakesLadders)).String(),
				cache.SecretHolders.Key(string(secrets.WitchHunter)).String(),
				cache.SecretSolved.Key(string(secrets.WitchHunter)).String(),
				cache.SecretHolders.Key(string(secrets.Piece01)).String(),
				cache.SecretSolved.Key(string(secrets.Piece01)).String(),
			}

			assert.Subset(t, busted, required)
			assert.Len(t, busted, 9+len(leaderboardGameTypes)+2*len(secrets.All()))
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
