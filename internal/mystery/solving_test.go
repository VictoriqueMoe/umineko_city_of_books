package mystery

import (
	"context"
	"errors"
	"testing"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMarkSolved_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSolved_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	aid := uuid.New()
	stubAuthor(m, mid, authorID)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestMarkSolved_AttemptAuthorError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSolved_AttemptMysteryError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSolved_AttemptWrongMystery(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(uuid.New(), nil)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.Error(t, err)
}

func TestMarkSolved_OwnAttempt(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.Error(t, err)
}

func TestMarkSolved_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&repository.MysteryRow{ID: mid, UserID: userID}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, attemptAuthor).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid, true).Return(errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.Error(t, err)
}

func TestMarkSolved_OK_Broadcasts(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&repository.MysteryRow{ID: mid, UserID: userID}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, attemptAuthor).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid, true).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysterySolved,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "attempt=" + aid.String(),
		SubjectID:  attemptAuthor,
	}).Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, nil).Maybe()

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.NoError(t, err)
}

func TestMarkSolved_Admin_CanSolve(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	admin := uuid.New()
	author := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	stubAuthor(m, mid, author)
	m.authz.EXPECT().Can(mock.Anything, admin, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&repository.MysteryRow{ID: mid, UserID: author}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, attemptAuthor).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid, true).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    admin,
		Action:     repository.AuditActionMysterySolved,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "attempt=" + aid.String(),
		SubjectID:  attemptAuthor,
	}).Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, nil).Maybe()

	// when
	err := svc.MarkSolved(context.Background(), mid, admin, aid)

	// then
	require.NoError(t, err)
}

func TestMarkPermanentlySolved_AuditsTheClose(t *testing.T) {
	tests := []struct {
		name        string
		actorIsGM   bool
		wantDetails string
	}{
		{name: "the game master closes their own board", actorIsGM: true, wantDetails: "by=author"},
		{name: "staff closes someone else's board", actorIsGM: false, wantDetails: "by=staff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid := uuid.New()
			authorID := uuid.New()
			actorID := authorID
			if !tt.actorIsGM {
				actorID = uuid.New()
				m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyTheory).Return(true)
			}
			stubAuthor(m, mid, authorID)
			m.repo.EXPECT().MarkPermanentlySolved(mock.Anything, mid).Return(nil)
			m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
				ActorID:    actorID,
				Action:     repository.AuditActionMysteryClosed,
				TargetType: repository.AuditTargetMystery,
				TargetID:   mid.String(),
				Details:    tt.wantDetails,
				SubjectID:  authorID,
			}).Return(nil)
			m.repo.EXPECT().GetSolverIDs(mock.Anything, mid).Return(nil, nil).Maybe()
			m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()
			m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, nil).Maybe()

			// when
			err := svc.MarkPermanentlySolved(context.Background(), mid, actorID)

			// then
			require.NoError(t, err)
		})
	}
}

func TestAddClue_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.AddClue(context.Background(), uuid.New(), uuid.New(), dto.CreateClueRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestAddClue_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAddClue_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestAddClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(0, nil)
	m.repo.EXPECT().AddClue(mock.Anything, mid, repository.NewClue{Body: "c", TruthType: "red", SortOrder: 0}).Return(nil, errors.New("boom"))

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.Error(t, err)
}

func TestAddClue_OK_DefaultTruthType(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(3, nil)
	m.repo.EXPECT().AddClue(mock.Anything, mid, repository.NewClue{Body: "c", TruthType: "red", SortOrder: 3}).Return(&dto.MysteryClue{ID: 1}, nil)

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.NoError(t, err)
}

func TestAddClue_Private_NotifiesPlayer(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	playerID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(0, nil)
	m.repo.EXPECT().AddClue(mock.Anything, mid, repository.NewClue{Body: "c", TruthType: "blue", SortOrder: 0, PlayerID: &playerID}).Return(&dto.MysteryClue{ID: 1}, nil)

	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == playerID && p.Type == dto.NotifMysteryPrivateClue
	})).Return(nil).Maybe()

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c", TruthType: "blue", PlayerID: &playerID})

	// then
	require.NoError(t, err)
}

func TestDeleteClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().DeleteClue(mock.Anything, 7).Return(errors.New("boom"))

	// when
	err := svc.DeleteClue(context.Background(), mid, 7, userID)

	// then
	require.Error(t, err)
}

func TestDeleteClue_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().DeleteClue(mock.Anything, 7).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryClueDelete,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "clue=7",
	}).Return(nil)

	// when
	err := svc.DeleteClue(context.Background(), mid, 7, userID)

	// then
	require.NoError(t, err)
}

func TestUpdateClue_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateClue(context.Background(), uuid.New(), 1, uuid.New(), " ")

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().UpdateClue(mock.Anything, 3, "new").Return(errors.New("boom"))

	// when
	err := svc.UpdateClue(context.Background(), uuid.New(), 3, uuid.New(), "new")

	// then
	require.Error(t, err)
}

func TestUpdateClue_OK_Trims(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().UpdateClue(mock.Anything, 3, "new").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryClueUpdate,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "clue=3",
	}).Return(nil)

	// when
	err := svc.UpdateClue(context.Background(), mid, 3, userID, "  new  ")

	// then
	require.NoError(t, err)
}
