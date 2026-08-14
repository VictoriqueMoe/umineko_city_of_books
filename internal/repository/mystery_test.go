package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
	"umineko_city_of_books/internal/dto"
)

type (
	mysteryWriterCase struct {
		name   string
		expect func(dao *MockMysteryDAO, err error)
		call   func(repo MysteryRepository) error
	}
)

func newCachedMysteryRepo(t *testing.T) (MysteryRepository, *MockMysteryDAO, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	dao := NewMockMysteryDAO(t)

	return NewMysteryRepo(nil, dao, NewMockAuditLogRepository(t), cache.NewManagerWithClient(client)), dao, client
}

func mysteryLeaderboardWriters(mysteryID, attemptID, userID uuid.UUID) []mysteryWriterCase {
	return []mysteryWriterCase{
		{
			name: "update",
			expect: func(dao *MockMysteryDAO, err error) {
				dao.EXPECT().Update(mock.Anything, mysteryID, userID, "t", "b", "hard").Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.Update(context.Background(), mysteryID, userID, "t", "b", "hard")
			},
		},
		{
			name: "update as admin",
			expect: func(dao *MockMysteryDAO, err error) {
				dao.EXPECT().UpdateAsAdmin(mock.Anything, mysteryID, "t", "b", "nightmare", true, false, mock.Anything).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.UpdateAsAdmin(context.Background(), mysteryID, "t", "b", "nightmare", true, false, dto.DefaultKnoxContract())
			},
		},
		{
			name: "delete",
			expect: func(dao *MockMysteryDAO, err error) {
				dao.EXPECT().Delete(mock.Anything, mysteryID, userID).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.Delete(context.Background(), mysteryID, userID)
			},
		},
		{
			name: "delete as admin",
			expect: func(dao *MockMysteryDAO, err error) {
				dao.EXPECT().DeleteAsAdmin(mock.Anything, mysteryID).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.DeleteAsAdmin(context.Background(), mysteryID)
			},
		},
		{
			name: "delete attempt",
			expect: func(dao *MockMysteryDAO, err error) {
				dao.EXPECT().DeleteAttempt(mock.Anything, attemptID, userID).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.DeleteAttempt(context.Background(), attemptID, userID)
			},
		},
		{
			name: "delete attempt as admin",
			expect: func(dao *MockMysteryDAO, err error) {
				dao.EXPECT().DeleteAttemptAsAdmin(mock.Anything, attemptID).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.DeleteAttemptAsAdmin(context.Background(), attemptID)
			},
		},
	}
}

func TestMysteryRepository_LeaderboardWritersInvalidateBothCrowns(t *testing.T) {
	mysteryID, attemptID, userID := uuid.New(), uuid.New(), uuid.New()

	for _, tc := range mysteryLeaderboardWriters(mysteryID, attemptID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, dao, client := newCachedMysteryRepo(t)
			tc.expect(dao, nil)

			var commands []string
			captureDel(client, &commands)

			// when
			err := tc.call(repo)

			// then
			require.NoError(t, err)
			assert.Equal(t, []string{"DEL", cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key()}, commands)
		})
	}
}

func TestMysteryRepository_LeaderboardWritersSkipInvalidationOnDaoError(t *testing.T) {
	mysteryID, attemptID, userID := uuid.New(), uuid.New(), uuid.New()

	for _, tc := range mysteryLeaderboardWriters(mysteryID, attemptID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, dao, client := newCachedMysteryRepo(t)
			tc.expect(dao, errors.New("db down"))
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then
			require.Error(t, err)
		})
	}
}
