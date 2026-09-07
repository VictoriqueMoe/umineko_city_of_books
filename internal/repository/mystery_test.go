package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model/spec"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

type (
	mysteryWriterCase struct {
		name   string
		expect func(mysteryDAO *dao.MockMysteryDAO, err error)
		call   func(repo MysteryRepository) error
	}
)

func newCachedMysteryRepo(t *testing.T) (MysteryRepository, *dao.MockMysteryDAO, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	mysteryDAO := dao.NewMockMysteryDAO(t)

	return NewMysteryRepo(nil, mysteryDAO, NewMockAuditLogRepository(t), cache.NewManager(engines.NewValkeyWithClient(client))), mysteryDAO, client
}

func mysteryLeaderboardWriters(mysteryID, attemptID, userID uuid.UUID) []mysteryWriterCase {
	return []mysteryWriterCase{
		{
			name: "update",
			expect: func(mysteryDAO *dao.MockMysteryDAO, err error) {
				mysteryDAO.EXPECT().Update(mock.Anything, spec.MysteryOwnerUpdate{ID: mysteryID, UserID: userID, Title: "t", Body: "b", Difficulty: "hard"}).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.Update(context.Background(), spec.MysteryOwnerUpdate{ID: mysteryID, UserID: userID, Title: "t", Body: "b", Difficulty: "hard"})
			},
		},
		{
			name: "update as admin",
			expect: func(mysteryDAO *dao.MockMysteryDAO, err error) {
				mysteryDAO.EXPECT().UpdateAsAdmin(mock.Anything, spec.MysteryUpdate{ID: mysteryID, Title: "t", Body: "b", Difficulty: "nightmare", FreeForAll: true, KeepOpenAfterSolve: false, Knox: dto.DefaultKnoxContract()}).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.UpdateAsAdmin(context.Background(), spec.MysteryUpdate{ID: mysteryID, Title: "t", Body: "b", Difficulty: "nightmare", FreeForAll: true, KeepOpenAfterSolve: false, Knox: dto.DefaultKnoxContract()})
			},
		},
		{
			name: "delete",
			expect: func(mysteryDAO *dao.MockMysteryDAO, err error) {
				mysteryDAO.EXPECT().Delete(mock.Anything, spec.OwnedDeletion{ID: mysteryID, UserID: userID}).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.Delete(context.Background(), spec.OwnedDeletion{ID: mysteryID, UserID: userID})
			},
		},
		{
			name: "delete as admin",
			expect: func(mysteryDAO *dao.MockMysteryDAO, err error) {
				mysteryDAO.EXPECT().DeleteAsAdmin(mock.Anything, mysteryID).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.DeleteAsAdmin(context.Background(), mysteryID)
			},
		},
		{
			name: "delete attempt",
			expect: func(mysteryDAO *dao.MockMysteryDAO, err error) {
				mysteryDAO.EXPECT().DeleteAttempt(mock.Anything, spec.MysteryAttemptDeletion{ID: attemptID, UserID: userID}).Return(err)
			},
			call: func(repo MysteryRepository) error {
				return repo.DeleteAttempt(context.Background(), spec.MysteryAttemptDeletion{ID: attemptID, UserID: userID})
			},
		},
		{
			name: "delete attempt as admin",
			expect: func(mysteryDAO *dao.MockMysteryDAO, err error) {
				mysteryDAO.EXPECT().DeleteAttemptAsAdmin(mock.Anything, attemptID).Return(err)
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
			repo, mysteryDAO, client := newCachedMysteryRepo(t)
			tc.expect(mysteryDAO, nil)

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
			repo, mysteryDAO, client := newCachedMysteryRepo(t)
			tc.expect(mysteryDAO, errors.New("db down"))
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then
			require.Error(t, err)
		})
	}
}
