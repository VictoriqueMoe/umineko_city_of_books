package mystery

import (
	"context"
	"sync"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type mysteryFixture struct {
	svc    *service
	m      *testMocks
	id     uuid.UUID
	userID uuid.UUID
}

func newMysteryFixture(t *testing.T) mysteryFixture {
	svc, m := newTestService(t)

	return mysteryFixture{svc: svc, m: m, id: uuid.New(), userID: uuid.New()}
}

func stubAuthor(m *testMocks, id, authorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
}

func stubNotAuthor(m *testMocks, id, actorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyTheory).Return(false)
}

func stubStaffOverride(m *testMocks, actorID uuid.UUID) {
	m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyTheory).Return(true)
}

type detailStubs struct {
	row      *repository.MysteryRow
	clues    []dto.MysteryClue
	attempts []repository.MysteryAttemptRow
	hasWon   bool
	role     role.Role
}

func stubMysteryDetail(m *testMocks, id, viewer uuid.UUID, d detailStubs) {
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(d.row, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, id, viewer).Return(d.hasWon, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(d.clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(d.attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return(d.role, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()
}

func stubActor(m *testMocks, userID uuid.UUID, displayName string) {
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: displayName}, nil)
}

func stubMentionOf(m *testMocks, actorID, mentionedID uuid.UUID, username string) {
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{username}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, mentionedID).Return(false, nil)
}

func captureNotification(m *testMocks) (*dto.NotifyParams, *sync.WaitGroup) {
	var wg sync.WaitGroup
	wg.Add(1)

	captured := new(dto.NotifyParams)
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			*captured = p
			wg.Done()

			return nil
		})

	return captured, &wg
}
