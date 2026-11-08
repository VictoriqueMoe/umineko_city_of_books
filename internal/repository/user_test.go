package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
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

type (
	userRepoMocks struct {
		users       *dao.MockUserDAO
		roles       *MockRoleRepository
		blocks      *MockBlockRepository
		permissions *MockPermissionRepository
		mysteries   *MockMysteryRepository
		posts       *MockPostRepository
		art         *MockArtRepository
		gameRooms   *MockGameRoomRepository
		userSecrets *MockUserSecretRepository
	}
)

func newUserRepoMocks(t *testing.T) userRepoMocks {
	t.Helper()

	return userRepoMocks{
		users:       dao.NewMockUserDAO(t),
		roles:       NewMockRoleRepository(t),
		blocks:      NewMockBlockRepository(t),
		permissions: NewMockPermissionRepository(t),
		mysteries:   NewMockMysteryRepository(t),
		posts:       NewMockPostRepository(t),
		art:         NewMockArtRepository(t),
		gameRooms:   NewMockGameRoomRepository(t),
		userSecrets: NewMockUserSecretRepository(t),
	}
}

func newUserRepoWithCache(m userRepoMocks, c *cache.Manager) UserRepository {
	return NewUserRepo(nil, m.users, c, m.roles, nil, nil, nil, nil, nil, m.blocks, m.permissions, m.mysteries, m.posts, m.art, m.gameRooms, m.userSecrets)
}

func newCachedUserRepo(t *testing.T) (UserRepository, userRepoMocks, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	m := newUserRepoMocks(t)

	return newUserRepoWithCache(m, cache.NewManager(engines.NewValkeyWithClient(client))), m, client
}

func newInMemoryUserRepo(t *testing.T) (UserRepository, *dao.MockUserDAO) {
	t.Helper()

	m := newUserRepoMocks(t)

	return newUserRepoWithCache(m, cache.NewManager(engines.NewInMemory(0))), m.users
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

func TestUserRepository_ScoreAdjustmentsInvalidateTheMysteryLeaderboards(t *testing.T) {
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
			// given
			repo, m, _ := newCachedUserRepo(t)
			tc.expect(m.users)
			m.mysteries.EXPECT().InvalidateLeaderboards(mock.Anything).Once()

			// when
			err := tc.call(repo)

			// then the leaderboards are cleared through the repository that owns them
			require.NoError(t, err)
		})
	}
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

func TestUserRepository_UpdateIPDropsTheCachedIPSoTheNextReadSeesTheDatabase(t *testing.T) {
	// given a cached ip
	repo, userDAO := newInMemoryUserRepo(t)
	userID := uuid.New()
	update := spec.UserIPUpdate{UserID: userID, IP: "10.0.0.2"}
	userDAO.EXPECT().GetIP(mock.Anything, userID).Return("10.0.0.1", nil).Once()
	_, err := repo.GetIP(context.Background(), userID)
	require.NoError(t, err)
	userDAO.EXPECT().UpdateIP(mock.Anything, update).Return(nil).Once()
	userDAO.EXPECT().GetIP(mock.Anything, userID).Return("10.0.0.2", nil).Once()

	// when
	writeErr := repo.UpdateIP(context.Background(), update)
	got, getErr := repo.GetIP(context.Background(), userID)

	// then the read reloads from the database, so concurrent writes can never leave the cache and the row disagreeing
	require.NoError(t, writeErr)
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

func TestUserRepository_DeletePathsInvalidateCascadedCachesThroughTheirOwners(t *testing.T) {
	userID := uuid.New()
	blocker := uuid.New()
	blocked := uuid.New()

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
			// given a user who blocked one person and was blocked by another
			repo, m, client := newCachedUserRepo(t)
			m.blocks.EXPECT().GetBlockedIDs(mock.Anything, userID).Return([]uuid.UUID{blocked, blocker}, nil).Once()
			tc.expect(m.users)

			secretIDs := make([]string, 0, len(secrets.All()))
			for _, secret := range secrets.All() {
				secretIDs = append(secretIDs, string(secret.ID))
			}

			m.blocks.EXPECT().InvalidateBlockedIDs(mock.Anything, []uuid.UUID{userID, blocked, blocker}).Once()
			m.roles.EXPECT().InvalidateRole(mock.Anything, userID).Once()
			m.permissions.EXPECT().InvalidateVanityAssignments(mock.Anything, []string{userID.String()}).Once()
			m.mysteries.EXPECT().InvalidateLeaderboards(mock.Anything).Once()
			m.posts.EXPECT().InvalidateCornerCounts(mock.Anything).Once()
			m.art.EXPECT().InvalidateCornerCounts(mock.Anything).Once()
			m.gameRooms.EXPECT().InvalidateTopWinners(mock.Anything, leaderboardGameTypes).Once()

			var invalidatedSecrets []string
			m.userSecrets.EXPECT().InvalidateSecrets(mock.Anything, mock.Anything).Run(func(_ context.Context, ids ...string) {
				invalidatedSecrets = ids
			}).Once()

			var commands []string
			captureDel(client, &commands)

			// when
			err := tc.call(repo)

			// then every owning repository clears its own keys, and only the user's ip is cleared here
			require.NoError(t, err)
			assert.ElementsMatch(t, secretIDs, invalidatedSecrets)
			assert.Equal(t, []string{"DEL", cache.UserIP.Key(userID.String()).String()}, commands)
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
			// given strict owner mocks that fail on any invalidation
			repo, m, client := newCachedUserRepo(t)
			m.blocks.EXPECT().GetBlockedIDs(mock.Anything, userID).Return([]uuid.UUID{uuid.New()}, nil).Once()
			tc.expect(m.users)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then
			require.Error(t, err)
		})
	}
}

func TestUserRepository_DeletePathsAbortWhenBlockCounterpartsCannotBeRead(t *testing.T) {
	userID := uuid.New()
	readErr := errors.New("db down")

	tests := []struct {
		name string
		call func(repo UserRepository) error
	}{
		{
			name: "self service delete",
			call: func(repo UserRepository) error {
				return repo.DeleteAccount(context.Background(), userID)
			},
		},
		{
			name: "admin delete",
			call: func(repo UserRepository) error {
				return repo.AdminDeleteAccount(context.Background(), userID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, m, client := newCachedUserRepo(t)
			m.blocks.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(nil, readErr).Once()
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then the user is never deleted, so no counterpart is left with a stale list
			require.ErrorIs(t, err, readErr)
		})
	}
}
