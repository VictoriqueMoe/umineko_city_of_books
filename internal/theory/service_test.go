package theory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	testMocks struct {
		repo        *repository.MockTheoryRepository
		userRepo    *repository.MockUserRepository
		auditRepo   *repository.MockAuditLogRepository
		authz       *authz.MockService
		blockSvc    *block.MockService
		notifSvc    *notification.MockService
		settingsSvc *settings.MockService
		fanout      chan uuid.UUID
	}
)

var (
	errBoom       = errors.New("boom")
	errMissingRow = errors.Join(errors.New("no row"), dao.ErrNotFound)
)

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		repo:        repository.NewMockTheoryRepository(t),
		userRepo:    repository.NewMockUserRepository(t),
		auditRepo:   repository.NewMockAuditLogRepository(t),
		authz:       authz.NewMockService(t),
		blockSvc:    block.NewMockService(t),
		notifSvc:    notification.NewMockService(t),
		settingsSvc: settings.NewMockService(t),
		fanout:      make(chan uuid.UUID, 8),
	}
	followRepo := repository.NewMockFollowRepository(t)
	mentionSvc := mention.NewService(m.userRepo, m.blockSvc, m.notifSvc, dao.CommentDAOs{})
	svc := NewService(m.repo, m.userRepo, followRepo, m.auditRepo, m.authz, m.blockSvc, m.notifSvc, mentionSvc, m.settingsSvc, credibility.NewService(m.repo), quotefinder.NewClient(), contentfilter.New(), nil, nil).(*service)

	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, mock.Anything).Run(func(_ context.Context, userID uuid.UUID, _ ...*sql.Tx) {
		m.fanout <- userID
	}).Return(nil, nil).Maybe()
	m.notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
	m.repo.EXPECT().RecomputeStatus(mock.Anything, mock.Anything).Return(nil).Maybe()

	return svc, m
}

func validCreateTheoryReq() dto.CreateTheoryRequest {
	return dto.CreateTheoryRequest{
		Title:   "test theory",
		Body:    "body",
		Episode: 1,
		Series:  "umineko",
	}
}

func expectResponseGate(m *testMocks, theoryID uuid.UUID, userID uuid.UUID, authorID uuid.UUID) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
}

func expectResponseRecalc(m *testMocks, theoryID uuid.UUID, responseID uuid.UUID) {
	m.repo.EXPECT().GetTheorySeries(mock.Anything, theoryID).Return("umineko", nil).Maybe()
	m.repo.EXPECT().GetResponseEvidence(mock.Anything, responseID).Return(nil, nil).Maybe()
	expectCredibilityRecalc(m, theoryID)
}

func expectCredibilityRecalc(m *testMocks, theoryID uuid.UUID) {
	m.repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(0, 0, nil).Maybe()
	m.repo.EXPECT().UpdateCredibilityScore(mock.Anything, spec.TheoryCredibilityUpdate{TheoryID: theoryID, Score: 50}).Return(nil).Maybe()
}

func expectNotifyDetails(m *testMocks, theoryID uuid.UUID, actorID uuid.UUID) {
	m.repo.EXPECT().GetTheoryTitle(mock.Anything, theoryID).Return("t", nil).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "A"}, nil).Maybe()
}

func expectNotify(m *testMocks, match func(dto.NotifyParams) bool) <-chan struct{} {
	sent := make(chan struct{})
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(match)).Run(func(_ context.Context, _ dto.NotifyParams) {
		close(sent)
	}).Return(nil).Once()

	return sent
}

func receive[T any](t *testing.T, ch <-chan T, what string) (got T) {
	t.Helper()

	select {
	case got = <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}

	return got
}

func TestCreateTheory(t *testing.T) {
	cases := []struct {
		name          string
		limit         int
		count         int
		countErr      error
		reachesCreate bool
		createErr     error
		mention       bool
		wantErr       error
	}{
		{name: "without a limit it creates, fans out to followers and notifies the mentioned user", reachesCreate: true, mention: true},
		{name: "under the limit it creates", limit: 5, count: 2, reachesCreate: true},
		{name: "a count error is returned", limit: 5, countErr: errBoom, wantErr: errBoom},
		{name: "at the limit it is rate limited", limit: 3, count: 3, wantErr: ErrRateLimited},
		{name: "a create error skips the follower fan-out", reachesCreate: true, createErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				userID := uuid.New()
				theoryID := uuid.New()
				mentionedID := uuid.New()
				req := validCreateTheoryReq()
				if tc.mention {
					req.Body = "what do you make of this @alice"
				}

				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxTheoriesPerDay).Return(tc.limit)
				if tc.limit > 0 {
					m.repo.EXPECT().CountUserTheoriesToday(mock.Anything, userID).Return(tc.count, tc.countErr)
				}
				if tc.reachesCreate {
					m.repo.EXPECT().Create(mock.Anything, spec.NewTheory{
						UserID:   userID,
						Title:    req.Title,
						Body:     req.Body,
						Episode:  req.Episode,
						Series:   req.Series,
						Evidence: req.Evidence,
					}).Return(&dto.TheoryDetailResponse{ID: theoryID}, tc.createErr)
				}

				var mentioned <-chan struct{}
				if tc.mention {
					m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
					m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
					m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)
					mentioned = expectNotify(m, func(p dto.NotifyParams) bool {
						return p.Type == dto.NotifMention && p.RecipientID == mentionedID && p.ReferenceType == "theory" && p.ReferenceID == theoryID
					})
				}

				// when
				got, err := svc.CreateTheory(context.Background(), userID, req)

				// then
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					synctest.Wait()
					assert.Empty(t, m.fanout)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, theoryID, got)
				assert.Equal(t, userID, receive(t, m.fanout, "the follower fan-out"))
				if tc.mention {
					receive(t, mentioned, "the mention notification")
				}
			})
		})
	}
}

func TestGetTheoryDetail(t *testing.T) {
	viewerID := uuid.New()
	cases := []struct {
		name         string
		viewerID     uuid.UUID
		missing      bool
		getErr       error
		evidenceErr  error
		responsesErr error
		vote         int
		voteErr      error
		wantErr      error
	}{
		{name: "a repository error is returned", getErr: errBoom, wantErr: errBoom},
		{name: "a missing theory is nil without an error", missing: true},
		{name: "an evidence error is returned", evidenceErr: errBoom, wantErr: errBoom},
		{name: "a responses error is returned", viewerID: viewerID, responsesErr: errBoom, wantErr: errBoom},
		{name: "an anonymous viewer gets evidence and responses without a vote lookup"},
		{name: "a signed in viewer gets their vote", viewerID: viewerID, vote: 1},
		{name: "a vote lookup error is returned instead of showing no vote", viewerID: viewerID, voteErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			evidence := []dto.EvidenceResponse{{ID: 1}}
			responses := []dto.ResponseResponse{{ID: uuid.New()}}
			found := tc.getErr == nil && !tc.missing
			reachesResponses := found && tc.evidenceErr == nil
			reachesVote := reachesResponses && tc.responsesErr == nil && tc.viewerID != uuid.Nil

			var detail *dto.TheoryDetailResponse
			if found {
				detail = &dto.TheoryDetailResponse{ID: id}
			}

			m.repo.EXPECT().GetByID(mock.Anything, id).Return(detail, tc.getErr)
			if found {
				m.repo.EXPECT().GetEvidence(mock.Anything, id).Return(evidence, tc.evidenceErr)
			}
			if reachesResponses {
				m.repo.EXPECT().GetResponses(mock.Anything, spec.TheoryResponseQuery{TheoryID: id, ViewerID: tc.viewerID}).Return(responses, tc.responsesErr)
			}
			if reachesVote {
				m.repo.EXPECT().GetUserTheoryVote(mock.Anything, spec.TheoryVoteLookup{UserID: tc.viewerID, TheoryID: id}).Return(tc.vote, tc.voteErr)
			}

			// when
			got, err := svc.GetTheoryDetail(context.Background(), id, tc.viewerID)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			if tc.missing {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, evidence, got.Evidence)
			assert.Equal(t, responses, got.Responses)
			assert.Equal(t, tc.vote, got.UserVote)
		})
	}
}

func TestListTheories(t *testing.T) {
	cases := []struct {
		name       string
		blocked    []uuid.UUID
		blockedErr error
		listErr    error
	}{
		{name: "blocked users are excluded", blocked: []uuid.UUID{uuid.New()}},
		{name: "a blocked lookup error is returned instead of listing blocked authors", blockedErr: errBoom},
		{name: "a repository error is returned", listErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			p := params.ListParams{Limit: 20, Offset: 40}
			theories := []dto.TheoryResponse{{ID: uuid.New()}}
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(tc.blocked, tc.blockedErr)
			if tc.blockedErr == nil {
				m.repo.EXPECT().List(mock.Anything, spec.TheoryListFilter{Params: p, ViewerID: userID, ExcludeUserIDs: tc.blocked}).Return(theories, 1, tc.listErr)
			}

			// when
			got, err := svc.ListTheories(context.Background(), p, userID)

			// then
			if tc.listErr != nil || tc.blockedErr != nil {
				require.ErrorIs(t, err, errBoom)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, &dto.TheoryListResponse{Theories: theories, Total: 1, Limit: 20, Offset: 40}, got)
		})
	}
}

func TestUpdateTheory(t *testing.T) {
	cases := []struct {
		name              string
		canEditAny        bool
		updateErr         error
		notifierLookupErr error
	}{
		{name: "an edit does not re-notify mentions"},
		{name: "a non-admin repository error is returned", updateErr: errBoom},
		{name: "an admin repository error is returned", canEditAny: true, updateErr: errBoom},
		{name: "an admin edit is audited and notifies the author", canEditAny: true},
		{name: "an admin edit swallows an author lookup error in the notifier", canEditAny: true, notifierLookupErr: errors.New("missing")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				id := uuid.New()
				userID := uuid.New()
				authorID := uuid.New()
				req := validCreateTheoryReq()
				req.Body = "rethinking this @alice"
				adminEdit := tc.canEditAny && tc.updateErr == nil

				m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).Return(authorID, nil).Once()
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(tc.canEditAny)
				m.repo.EXPECT().Update(mock.Anything, spec.TheoryUpdate{
					ID:       id,
					UserID:   userID,
					Title:    req.Title,
					Body:     req.Body,
					Episode:  req.Episode,
					AsAdmin:  tc.canEditAny,
					Evidence: req.Evidence,
				}).Return(tc.updateErr)

				var notifierDone <-chan struct{}
				if adminEdit {
					m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
						ActorID:    userID,
						Action:     audit.ActionTheoryUpdateAdmin,
						TargetType: audit.TargetTheory,
						TargetID:   id.String(),
						Details:    fmt.Sprintf("title=%q", req.Title),
						SubjectID:  authorID,
					}).Return(nil)
				}
				if adminEdit && tc.notifierLookupErr != nil {
					lookedUp := make(chan struct{})
					m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).Run(func(_ context.Context, _ uuid.UUID, _ ...*sql.Tx) {
						close(lookedUp)
					}).Return(uuid.Nil, tc.notifierLookupErr).Once()
					notifierDone = lookedUp
				}
				if adminEdit && tc.notifierLookupErr == nil {
					m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).Return(authorID, nil).Once()
					m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Mod"}, nil)
					notifierDone = expectNotify(m, func(p dto.NotifyParams) bool {
						return p.RecipientID == authorID && p.ActorID == userID && p.Type == dto.NotifContentEdited
					})
				}

				// when
				err := svc.UpdateTheory(context.Background(), id, userID, req)

				// then
				synctest.Wait()
				m.userRepo.AssertNumberOfCalls(t, "GetByUsernames", 0)
				if tc.updateErr != nil {
					require.ErrorIs(t, err, tc.updateErr)
					return
				}

				require.NoError(t, err)
				if adminEdit {
					receive(t, notifierDone, "the edit notifier")
				}
			})
		})
	}
}

func TestDeleteTheory(t *testing.T) {
	cases := []struct {
		name         string
		ownTheory    bool
		lookupErr    error
		canDeleteAny bool
		deleteErr    error
		wantAction   audit.Action
		wantErr      error
	}{
		{name: "an admin deleting another user's theory is audited as an admin delete", canDeleteAny: true, wantAction: audit.ActionTheoryDeleteAdmin},
		{name: "a moderator deleting their own theory uses the plain action", ownTheory: true, canDeleteAny: true, wantAction: audit.ActionTheoryDelete},
		{name: "an owner delete is audited with the plain action", ownTheory: true, wantAction: audit.ActionTheoryDelete},
		{name: "an owner delete error writes no audit row", ownTheory: true, deleteErr: errBoom, wantErr: errBoom},
		{name: "an author lookup error is returned", lookupErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			if tc.ownTheory {
				authorID = userID
			}

			m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil {
				m.repo.EXPECT().GetTheoryTitle(mock.Anything, id).Return("a wild theory", nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(tc.canDeleteAny)
			}
			if tc.lookupErr == nil && tc.canDeleteAny {
				m.repo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(tc.deleteErr)
			}
			if tc.lookupErr == nil && !tc.canDeleteAny {
				m.repo.EXPECT().Delete(mock.Anything, spec.OwnedDeletion{ID: id, UserID: userID}).Return(tc.deleteErr)
			}
			if tc.wantAction != "" {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    userID,
					Action:     tc.wantAction,
					TargetType: audit.TargetTheory,
					TargetID:   id.String(),
					Details:    `title="a wild theory"`,
					SubjectID:  authorID,
				}).Return(nil)
			}

			// when
			err := svc.DeleteTheory(context.Background(), id, userID)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCreateResponse_Errors(t *testing.T) {
	cases := []struct {
		name      string
		limit     int
		count     int
		countErr  error
		lookupErr error
		ownTheory bool
		blocked   bool
		blockErr  error
		createErr error
		wantErr   error
	}{
		{name: "a count error is returned", limit: 5, countErr: errBoom, wantErr: errBoom},
		{name: "at the limit it is rate limited", limit: 3, count: 3, wantErr: ErrRateLimited},
		{name: "an author lookup error is returned", lookupErr: errBoom, wantErr: errBoom},
		{name: "a missing theory is not found", lookupErr: errMissingRow, wantErr: ErrTheoryNotFound},
		{name: "a block either way rejects the response", blocked: true, wantErr: block.ErrUserBlocked},
		{name: "a failed block check rejects the response", blockErr: errBoom, wantErr: errBoom},
		{name: "the author cannot respond to their own theory at the top level", ownTheory: true, wantErr: ErrCannotRespondToOwnTheory},
		{name: "a create error is returned", createErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			theoryID := uuid.New()
			authorID := uuid.New()
			if tc.ownTheory {
				authorID = userID
			}
			limited := tc.limit > 0

			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(tc.limit)
			if limited {
				m.repo.EXPECT().CountUserResponsesToday(mock.Anything, userID).Return(tc.count, tc.countErr)
			}
			if !limited {
				m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, tc.lookupErr)
			}
			if !limited && tc.lookupErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}
			if !limited && tc.lookupErr == nil && tc.blockErr == nil && !tc.blocked && !tc.ownTheory {
				m.repo.EXPECT().CreateResponse(mock.Anything, spec.NewTheoryResponse{TheoryID: theoryID, UserID: userID, Side: "with_love"}).Return(nil, tc.createErr)
			}

			// when
			got, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, uuid.Nil, got)
		})
	}
}

func TestCreateResponse_OwnTheoryAllowedAsReply(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		svc, m := newTestService(t)
		userID := uuid.New()
		theoryID := uuid.New()
		responseID := uuid.New()
		parentID := uuid.New()
		expectResponseGate(m, theoryID, userID, userID)
		m.repo.EXPECT().CreateResponse(mock.Anything, spec.NewTheoryResponse{TheoryID: theoryID, UserID: userID, ParentID: &parentID, Side: "with_love"}).Return(&dto.ResponseResponse{ID: responseID}, nil)

		expectResponseRecalc(m, theoryID, responseID)
		expectNotifyDetails(m, theoryID, userID)
		m.repo.EXPECT().GetResponseInfo(mock.Anything, parentID).Return(uuid.Nil, uuid.Nil, errBoom).Maybe()
		m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

		// when
		got, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love", ParentID: &parentID})

		// then
		require.NoError(t, err)
		assert.Equal(t, responseID, got)
	})
}

func TestCreateResponse_NotifiesTheAuthorAndAnchorsMentionsOnTheResponse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		svc, m := newTestService(t)
		userID := uuid.New()
		theoryID := uuid.New()
		authorID := uuid.New()
		responseID := uuid.New()
		mentionedID := uuid.New()
		expectResponseGate(m, theoryID, userID, authorID)
		m.repo.EXPECT().CreateResponse(mock.Anything, spec.NewTheoryResponse{TheoryID: theoryID, UserID: userID, Side: "with_love", Body: "agreed @alice"}).Return(&dto.ResponseResponse{ID: responseID}, nil)

		expectResponseRecalc(m, theoryID, responseID)
		expectNotifyDetails(m, theoryID, userID)
		m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
		m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)
		authorNotified := expectNotify(m, func(p dto.NotifyParams) bool {
			return p.Type == dto.NotifTheoryResponse && p.RecipientID == authorID
		})
		mentionNotified := expectNotify(m, func(p dto.NotifyParams) bool {
			return p.Type == dto.NotifMention && p.ReferenceID == theoryID && p.ReferenceType == fmt.Sprintf("theory_response:%s", responseID)
		})

		// when
		got, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love", Body: "agreed @alice"})

		// then
		require.NoError(t, err)
		assert.Equal(t, responseID, got)
		receive(t, authorNotified, "the theory response notification")
		receive(t, mentionNotified, "the mention notification")
	})
}

func TestDeleteResponse(t *testing.T) {
	cases := []struct {
		name         string
		ownResponse  bool
		infoErr      error
		canDeleteAny bool
		deleteErr    error
		wantAudit    bool
		wantRecalc   bool
	}{
		{name: "an admin deleting another user's response is audited", canDeleteAny: true, wantAudit: true, wantRecalc: true},
		{name: "a moderator deleting their own response writes no audit row", ownResponse: true, canDeleteAny: true, wantRecalc: true},
		{name: "an owner delete writes no audit row", ownResponse: true, wantRecalc: true},
		{name: "a non-admin delete error skips the recalculation", deleteErr: errBoom},
		{name: "a response info failure refuses the delete instead of skipping its audit", infoErr: errBoom},
		{name: "a missing response is not found", infoErr: errMissingRow},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				id := uuid.New()
				userID := uuid.New()
				responseAuthorID := uuid.New()
				theoryID := uuid.New()
				if tc.ownResponse {
					responseAuthorID = userID
				}
				if tc.infoErr != nil {
					responseAuthorID = uuid.Nil
					theoryID = uuid.Nil
				}

				m.repo.EXPECT().GetResponseInfo(mock.Anything, id).Return(responseAuthorID, theoryID, tc.infoErr)
				if tc.infoErr == nil {
					m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyResponse).Return(tc.canDeleteAny)
				}
				if tc.infoErr == nil && tc.canDeleteAny {
					m.repo.EXPECT().DeleteResponseAsAdmin(mock.Anything, id).Return(tc.deleteErr)
				}
				if tc.infoErr == nil && !tc.canDeleteAny {
					m.repo.EXPECT().DeleteResponse(mock.Anything, spec.OwnedDeletion{ID: id, UserID: userID}).Return(tc.deleteErr)
				}
				if tc.wantAudit {
					m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
						ActorID:    userID,
						Action:     audit.ActionTheoryResponseDeleteAdmin,
						TargetType: audit.TargetTheoryResponse,
						TargetID:   id.String(),
						Details:    "theory=" + theoryID.String(),
						SubjectID:  responseAuthorID,
					}).Return(nil)
				}
				if tc.wantRecalc {
					expectCredibilityRecalc(m, theoryID)
				}

				// when
				err := svc.DeleteResponse(context.Background(), id, userID)

				// then
				synctest.Wait()
				switch {
				case errors.Is(tc.infoErr, dao.ErrNotFound):
					require.ErrorIs(t, err, ErrResponseNotFound)
				case tc.infoErr != nil:
					require.ErrorIs(t, err, tc.infoErr)
				case tc.deleteErr != nil:
					require.ErrorIs(t, err, tc.deleteErr)
				default:
					require.NoError(t, err)
				}
				if !tc.wantAudit {
					m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				}
			})
		})
	}
}

func TestVoteTheory(t *testing.T) {
	cases := []struct {
		name      string
		lookupErr error
		blocked   bool
		blockErr  error
		value     int
		voteErr   error
		wantErr   error
	}{
		{name: "an author lookup error is returned", lookupErr: errBoom, value: 1, wantErr: errBoom},
		{name: "a missing theory is not found", lookupErr: errMissingRow, value: 1, wantErr: ErrTheoryNotFound},
		{name: "a block either way rejects the vote", blocked: true, value: 1, wantErr: block.ErrUserBlocked},
		{name: "a failed block check rejects the vote", blockErr: errBoom, value: 1, wantErr: errBoom},
		{name: "a vote error is returned", value: 1, voteErr: errBoom, wantErr: errBoom},
		{name: "a downvote sends no notification", value: -1},
		{name: "an upvote notifies the author", value: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				userID := uuid.New()
				theoryID := uuid.New()
				authorID := uuid.New()
				upvoted := tc.wantErr == nil && tc.value == 1

				m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, tc.lookupErr).Once()
				if tc.lookupErr == nil {
					m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
				}
				if tc.lookupErr == nil && tc.blockErr == nil && !tc.blocked {
					m.repo.EXPECT().VoteTheory(mock.Anything, spec.Vote{UserID: userID, TargetID: theoryID, Value: tc.value}).Return(tc.voteErr)
				}

				var notified <-chan struct{}
				if upvoted {
					expectNotifyDetails(m, theoryID, userID)
					notified = expectNotify(m, func(p dto.NotifyParams) bool {
						return p.Type == dto.NotifTheoryUpvote && p.RecipientID == authorID
					})
				}

				// when
				err := svc.VoteTheory(context.Background(), userID, theoryID, tc.value)

				// then
				synctest.Wait()
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}

				require.NoError(t, err)
				if upvoted {
					receive(t, notified, "the upvote notification")
				}
			})
		})
	}
}

func TestVoteResponse(t *testing.T) {
	cases := []struct {
		name     string
		infoErr  error
		blocked  bool
		blockErr error
		value    int
		voteErr  error
		wantErr  error
	}{
		{name: "a response info error is returned", infoErr: errBoom, value: 1, wantErr: errBoom},
		{name: "a missing response is not found", infoErr: errMissingRow, value: 1, wantErr: ErrResponseNotFound},
		{name: "a block either way rejects the vote", blocked: true, value: 1, wantErr: block.ErrUserBlocked},
		{name: "a failed block check rejects the vote", blockErr: errBoom, value: 1, wantErr: errBoom},
		{name: "a vote error is returned", value: 1, voteErr: errBoom, wantErr: errBoom},
		{name: "a downvote sends no notification", value: -1},
		{name: "an upvote notifies the response author", value: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				userID := uuid.New()
				responseID := uuid.New()
				respAuthorID := uuid.New()
				theoryID := uuid.New()
				upvoted := tc.wantErr == nil && tc.value == 1

				m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(respAuthorID, theoryID, tc.infoErr).Once()
				if tc.infoErr == nil {
					m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, respAuthorID).Return(tc.blocked, tc.blockErr)
				}
				if tc.infoErr == nil && tc.blockErr == nil && !tc.blocked {
					m.repo.EXPECT().VoteResponse(mock.Anything, spec.Vote{UserID: userID, TargetID: responseID, Value: tc.value}).Return(tc.voteErr)
				}

				var notified <-chan struct{}
				if upvoted {
					expectNotifyDetails(m, theoryID, userID)
					notified = expectNotify(m, func(p dto.NotifyParams) bool {
						return p.Type == dto.NotifResponseUpvote && p.RecipientID == respAuthorID
					})
				}

				// when
				err := svc.VoteResponse(context.Background(), userID, responseID, tc.value)

				// then
				synctest.Wait()
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}

				require.NoError(t, err)
				if upvoted {
					receive(t, notified, "the upvote notification")
				}
			})
		})
	}
}

func TestRefuteTheory_ResponseLookupFailures(t *testing.T) {
	cases := []struct {
		name    string
		metaErr error
		wantErr error
	}{
		{name: "a missing response is not on the theory", metaErr: errMissingRow, wantErr: ErrResponseNotOnTheory},
		{name: "a failed response lookup is surfaced, not reported as a wrong response", metaErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			authorID := uuid.New()
			theoryID := uuid.New()
			responseID := uuid.New()
			m.repo.EXPECT().GetByID(mock.Anything, theoryID).Return(&dto.TheoryDetailResponse{ID: theoryID, Author: dto.UserResponse{ID: authorID}, Status: dto.TheoryStatusContested}, nil)
			m.repo.EXPECT().GetResponseMeta(mock.Anything, responseID).Return(model.ResponseMeta{}, tc.metaErr)

			// when
			err := svc.RefuteTheory(context.Background(), theoryID, authorID, responseID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			m.repo.AssertNotCalled(t, "MarkRefuted", mock.Anything, mock.Anything)
		})
	}
}

func TestTheoryWrites_AMissingTheoryIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		call func(s *service, id, userID uuid.UUID) error
	}{
		{name: "update", call: func(s *service, id, userID uuid.UUID) error {
			return s.UpdateTheory(context.Background(), id, userID, validCreateTheoryReq())
		}},
		{name: "delete", call: func(s *service, id, userID uuid.UUID) error {
			return s.DeleteTheory(context.Background(), id, userID)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).Return(uuid.Nil, errMissingRow)

			// when
			err := tc.call(svc, id, userID)

			// then
			require.ErrorIs(t, err, ErrTheoryNotFound)
		})
	}
}

func TestRefuteTheory_Guards(t *testing.T) {
	authorID := uuid.New()
	otherID := uuid.New()
	theoryID := uuid.New()
	responseID := uuid.New()
	parentID := uuid.New()
	opposing := model.ResponseMeta{AuthorID: otherID, TheoryID: theoryID, Side: "without_love"}

	cases := []struct {
		name        string
		actorID     uuid.UUID
		isAdmin     bool
		status      dto.TheoryStatus
		meta        model.ResponseMeta
		wantAuditBy string
		wantErr     error
	}{
		{name: "author refutes with an opposing top level response", actorID: authorID, status: dto.TheoryStatusContested, meta: opposing, wantAuditBy: "author"},
		{name: "a moderator may also refute", actorID: otherID, isAdmin: true, status: dto.TheoryStatusContested, meta: opposing, wantAuditBy: "staff"},
		{name: "a bystander may not refute", actorID: otherID, status: dto.TheoryStatusContested, meta: opposing, wantErr: ErrNotAuthor},
		{name: "an already refuted theory is terminal", actorID: authorID, status: dto.TheoryStatusRefuted, meta: opposing, wantErr: ErrAlreadyRefuted},
		{name: "a response from another theory is rejected", actorID: authorID, status: dto.TheoryStatusContested, meta: model.ResponseMeta{AuthorID: otherID, TheoryID: uuid.New(), Side: "without_love"}, wantErr: ErrResponseNotOnTheory},
		{name: "a threaded reply cannot be the refutation", actorID: authorID, status: dto.TheoryStatusContested, meta: model.ResponseMeta{AuthorID: otherID, TheoryID: theoryID, Side: "without_love", ParentID: &parentID}, wantErr: ErrRefutationMustBeTopLevel},
		{name: "a supporting response cannot be the refutation", actorID: authorID, status: dto.TheoryStatusContested, meta: model.ResponseMeta{AuthorID: otherID, TheoryID: theoryID, Side: "with_love"}, wantErr: ErrRefutationMustOppose},
		{name: "the author cannot refute themselves", actorID: authorID, status: dto.TheoryStatusContested, meta: model.ResponseMeta{AuthorID: authorID, TheoryID: theoryID, Side: "without_love"}, wantErr: ErrCannotRefuteWithOwn},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				m.repo.EXPECT().GetByID(mock.Anything, theoryID).Return(&dto.TheoryDetailResponse{
					ID:     theoryID,
					Author: dto.UserResponse{ID: authorID},
					Status: tc.status,
				}, nil)
				m.authz.EXPECT().Can(mock.Anything, tc.actorID, authz.PermEditAnyTheory).Return(tc.isAdmin).Maybe()
				m.repo.EXPECT().GetResponseMeta(mock.Anything, responseID).Return(tc.meta, nil).Maybe()
				if tc.wantErr == nil {
					m.repo.EXPECT().MarkRefuted(mock.Anything, spec.TheoryRefutation{TheoryID: theoryID, ResponseID: responseID}).Return(nil)
					m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
						ActorID:    tc.actorID,
						Action:     audit.ActionTheoryRefuted,
						TargetType: audit.TargetTheory,
						TargetID:   theoryID.String(),
						Details:    fmt.Sprintf("response=%s by=%s", responseID, tc.wantAuditBy),
						SubjectID:  tc.meta.AuthorID,
					}).Return(nil)
					expectNotifyDetails(m, theoryID, tc.actorID)
					m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
				}

				// when
				err := svc.RefuteTheory(context.Background(), theoryID, tc.actorID, responseID)

				// then
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}

				require.NoError(t, err)
			})
		})
	}
}
