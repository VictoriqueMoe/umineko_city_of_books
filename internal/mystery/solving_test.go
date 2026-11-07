package mystery

import (
	"context"
	"errors"
	"testing"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	clueEditCase struct {
		name      string
		asAuthor  bool
		canEdit   bool
		authorErr error
		writeErr  error
		wantErr   error
		wantAudit bool
	}
)

func TestMarkSolved_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errMissingRow)

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
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyWork).Return(false)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestMarkSolved_LookupFailures(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name             string
		attemptAuthorErr error
		attemptMystryErr error
		rowErr           error
		wantErr          error
	}{
		{name: "a missing attempt is not found", attemptAuthorErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a failed attempt author lookup is surfaced, not reported as not found", attemptAuthorErr: boom, wantErr: boom},
		{name: "an attempt whose mystery row has gone is not found", attemptMystryErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a failed attempt mystery lookup is surfaced, not reported as not found", attemptMystryErr: boom, wantErr: boom},
		{name: "a failed mystery read is surfaced, not reported as not found", rowErr: boom, wantErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid := uuid.New()
			userID := uuid.New()
			aid := uuid.New()
			attemptAuthor := uuid.New()
			stubAuthor(m, mid, userID)
			m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, tc.attemptAuthorErr)
			if tc.attemptAuthorErr == nil {
				m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, tc.attemptMystryErr)
			}
			if tc.attemptAuthorErr == nil && tc.attemptMystryErr == nil {
				m.repo.EXPECT().GetByID(mock.Anything, mid).Return(nil, tc.rowErr)
			}

			// when
			err := svc.MarkSolved(context.Background(), mid, userID, aid)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			m.repo.AssertNotCalled(t, "MarkSolved", mock.Anything, mock.Anything)
		})
	}
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
	require.ErrorIs(t, err, ErrAttemptNotOnMystery)
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
	require.ErrorIs(t, err, ErrOwnAttempt)
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
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&model.MysteryRow{ID: mid, UserID: userID}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: attemptAuthor}).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, spec.MysterySolve{MysteryID: mid, AttemptID: aid, LockMystery: true}).Return(errors.New("boom"))

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
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&model.MysteryRow{ID: mid, UserID: userID}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: attemptAuthor}).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, spec.MysterySolve{MysteryID: mid, AttemptID: aid, LockMystery: true}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionMysterySolved,
		TargetType: audit.TargetMystery,
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
	m.authz.EXPECT().Can(mock.Anything, admin, authz.PermEditAnyWork).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&model.MysteryRow{ID: mid, UserID: author}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: attemptAuthor}).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, spec.MysterySolve{MysteryID: mid, AttemptID: aid, LockMystery: true}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    admin,
		Action:     audit.ActionMysterySolved,
		TargetType: audit.TargetMystery,
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
				m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyWork).Return(true)
			}
			stubAuthor(m, mid, authorID)
			m.repo.EXPECT().MarkPermanentlySolved(mock.Anything, mid).Return(nil)
			m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
				ActorID:    actorID,
				Action:     audit.ActionMysteryClosed,
				TargetType: audit.TargetMystery,
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errMissingRow)

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

func TestAddClue_AFailedClueCountIsSurfacedInsteadOfReusingSortPositionZero(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	boom := errors.New("boom")
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(0, boom)

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.ErrorIs(t, err, boom)
	m.repo.AssertNotCalled(t, "AddClue", mock.Anything, mock.Anything)
}

func TestAddClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(0, nil)
	m.repo.EXPECT().AddClue(mock.Anything, spec.NewMysteryClue{MysteryID: mid, Body: "c", TruthType: "red", SortOrder: 0}).Return(nil, errors.New("boom"))

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
	m.repo.EXPECT().AddClue(mock.Anything, spec.NewMysteryClue{MysteryID: mid, Body: "c", TruthType: "red", SortOrder: 3}).Return(&dto.MysteryClue{ID: 1}, nil)

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
	m.repo.EXPECT().AddClue(mock.Anything, spec.NewMysteryClue{MysteryID: mid, Body: "c", TruthType: "blue", SortOrder: 0, PlayerID: &playerID}).Return(&dto.MysteryClue{ID: 1}, nil)

	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == playerID && p.Type == dto.NotifMysteryPrivateClue
	})).Return(nil).Maybe()

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c", TruthType: "blue", PlayerID: &playerID})

	// then
	require.NoError(t, err)
}

func clueEditCases() []clueEditCase {
	boom := errors.New("boom")

	return []clueEditCase{
		{name: "the game master changes their own clue without an audit entry", asAuthor: true},
		{name: "staff holding the work permission change it and are audited against the author", canEdit: true, wantAudit: true},
		{name: "anyone else is refused before anything is written", wantErr: ErrNotAuthor},
		{name: "a missing mystery is not found", authorErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a clue that belongs to another mystery is not found", asAuthor: true, writeErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a failed write is surfaced", asAuthor: true, writeErr: boom, wantErr: boom},
	}
}

func expectClueAccess(m *testMocks, tc clueEditCase, mysteryID, userID, authorID uuid.UUID) bool {
	m.repo.EXPECT().GetAuthorID(mock.Anything, mysteryID).Return(authorID, tc.authorErr)
	if tc.authorErr != nil {
		return false
	}

	if !tc.asAuthor {
		m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyWork).Return(tc.canEdit)
	}

	return tc.asAuthor || tc.canEdit
}

func TestDeleteClue(t *testing.T) {
	for _, tc := range clueEditCases() {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			if tc.asAuthor {
				authorID = userID
			}
			if expectClueAccess(m, tc, mid, userID, authorID) {
				m.repo.EXPECT().DeleteClue(mock.Anything, spec.MysteryClueDeletion{MysteryID: mid, ClueID: 7}).Return(tc.writeErr)
			}
			if tc.wantAudit {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    userID,
					Action:     audit.ActionMysteryClueDelete,
					TargetType: audit.TargetMystery,
					TargetID:   mid.String(),
					Details:    "clue=7",
					SubjectID:  authorID,
				}).Return(nil)
			}

			// when
			err := svc.DeleteClue(context.Background(), mid, 7, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestUpdateClue_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateClue(context.Background(), uuid.New(), 1, uuid.New(), " ")

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateClue(t *testing.T) {
	for _, tc := range clueEditCases() {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			if tc.asAuthor {
				authorID = userID
			}
			if expectClueAccess(m, tc, mid, userID, authorID) {
				m.repo.EXPECT().UpdateClue(mock.Anything, spec.MysteryClueUpdate{MysteryID: mid, ClueID: 3, Body: "new"}).Return(tc.writeErr)
			}
			if tc.wantAudit {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    userID,
					Action:     audit.ActionMysteryClueUpdate,
					TargetType: audit.TargetMystery,
					TargetID:   mid.String(),
					Details:    "clue=3",
					SubjectID:  authorID,
				}).Return(nil)
			}

			// when
			err := svc.UpdateClue(context.Background(), mid, 3, userID, "  new  ")

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
