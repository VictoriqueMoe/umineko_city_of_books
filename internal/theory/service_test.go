package theory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	repo        *repository.MockTheoryRepository
	userRepo    *repository.MockUserRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	settingsSvc *settings.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	repo := repository.NewMockTheoryRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	credSvc := credibility.NewService(repo)
	quoteClient := quotefinder.NewClient()
	svc := NewService(repo, userRepo, authzSvc, blockSvc, notifSvc, settingsSvc, credSvc, quoteClient).(*service)
	return svc, &testMocks{
		repo:        repo,
		userRepo:    userRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		settingsSvc: settingsSvc,
	}
}

func waitOrFail(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("timed out waiting for goroutine")
	}
}

func validCreateTheoryReq() dto.CreateTheoryRequest {
	return dto.CreateTheoryRequest{
		Title:   "test theory",
		Body:    "body",
		Episode: 1,
		Series:  "umineko",
	}
}

// --- CreateTheory ---

func TestCreateTheory_NoLimit_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxTheoriesPerDay).Return(0)
	m.repo.EXPECT().Create(mock.Anything, userID, mock.Anything).Return(theoryID, nil)

	// when
	got, err := svc.CreateTheory(context.Background(), userID, validCreateTheoryReq())

	// then
	require.NoError(t, err)
	assert.Equal(t, theoryID, got)
}

func TestCreateTheory_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxTheoriesPerDay).Return(5)
	m.repo.EXPECT().CountUserTheoriesToday(mock.Anything, userID).Return(0, errors.New("db down"))

	// when
	_, err := svc.CreateTheory(context.Background(), userID, validCreateTheoryReq())

	// then
	require.Error(t, err)
}

func TestCreateTheory_RateLimited(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxTheoriesPerDay).Return(3)
	m.repo.EXPECT().CountUserTheoriesToday(mock.Anything, userID).Return(3, nil)

	// when
	_, err := svc.CreateTheory(context.Background(), userID, validCreateTheoryReq())

	// then
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestCreateTheory_UnderLimit_Creates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxTheoriesPerDay).Return(5)
	m.repo.EXPECT().CountUserTheoriesToday(mock.Anything, userID).Return(2, nil)
	m.repo.EXPECT().Create(mock.Anything, userID, mock.Anything).Return(theoryID, nil)

	// when
	got, err := svc.CreateTheory(context.Background(), userID, validCreateTheoryReq())

	// then
	require.NoError(t, err)
	assert.Equal(t, theoryID, got)
}

func TestCreateTheory_RepoCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxTheoriesPerDay).Return(0)
	m.repo.EXPECT().Create(mock.Anything, userID, mock.Anything).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateTheory(context.Background(), userID, validCreateTheoryReq())

	// then
	require.Error(t, err)
}

// --- GetTheoryDetail ---

func TestGetTheoryDetail_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, errors.New("boom"))

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, uuid.Nil)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestGetTheoryDetail_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, nil)

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetTheoryDetail_EvidenceError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	detail := &dto.TheoryDetailResponse{ID: id}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(detail, nil)
	m.repo.EXPECT().GetEvidence(mock.Anything, id).Return(nil, errors.New("boom"))

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, uuid.Nil)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestGetTheoryDetail_ResponsesError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	detail := &dto.TheoryDetailResponse{ID: id}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(detail, nil)
	m.repo.EXPECT().GetEvidence(mock.Anything, id).Return([]dto.EvidenceResponse{}, nil)
	m.repo.EXPECT().GetResponses(mock.Anything, id, userID).Return(nil, errors.New("boom"))

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, userID)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestGetTheoryDetail_AnonymousOK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	detail := &dto.TheoryDetailResponse{ID: id}
	evidence := []dto.EvidenceResponse{{ID: 1}}
	responses := []dto.ResponseResponse{{ID: uuid.New()}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(detail, nil)
	m.repo.EXPECT().GetEvidence(mock.Anything, id).Return(evidence, nil)
	m.repo.EXPECT().GetResponses(mock.Anything, id, uuid.Nil).Return(responses, nil)

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, uuid.Nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, evidence, got.Evidence)
	assert.Equal(t, responses, got.Responses)
	assert.Equal(t, 0, got.UserVote)
}

func TestGetTheoryDetail_WithUserVote(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	detail := &dto.TheoryDetailResponse{ID: id}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(detail, nil)
	m.repo.EXPECT().GetEvidence(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetResponses(mock.Anything, id, userID).Return(nil, nil)
	m.repo.EXPECT().GetUserTheoryVote(mock.Anything, userID, id).Return(1, nil)

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.UserVote)
}

func TestGetTheoryDetail_UserVoteErrorSwallowed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	detail := &dto.TheoryDetailResponse{ID: id}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(detail, nil)
	m.repo.EXPECT().GetEvidence(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetResponses(mock.Anything, id, userID).Return(nil, nil)
	m.repo.EXPECT().GetUserTheoryVote(mock.Anything, userID, id).Return(0, errors.New("boom"))

	// when
	got, err := svc.GetTheoryDetail(context.Background(), id, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

// --- ListTheories ---

func TestListTheories_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	p := params.ListParams{Limit: 20, Offset: 0}
	blocked := []uuid.UUID{uuid.New()}
	theories := []dto.TheoryResponse{{ID: uuid.New()}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(blocked, nil)
	m.repo.EXPECT().List(mock.Anything, p, userID, blocked).Return(theories, 1, nil)

	// when
	got, err := svc.ListTheories(context.Background(), p, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, theories, got.Theories)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 20, got.Limit)
	assert.Equal(t, 0, got.Offset)
}

func TestListTheories_BlockedLookupErrorIgnored(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	p := params.ListParams{Limit: 10}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(nil, errors.New("boom"))
	m.repo.EXPECT().List(mock.Anything, p, userID, []uuid.UUID(nil)).Return(nil, 0, nil)

	// when
	got, err := svc.ListTheories(context.Background(), p, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestListTheories_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	p := params.ListParams{Limit: 10}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, p, userID, []uuid.UUID(nil)).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListTheories(context.Background(), p, userID)

	// then
	require.Error(t, err)
}

// --- UpdateTheory ---

func TestUpdateTheory_NonAdmin_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)
	m.repo.EXPECT().Update(mock.Anything, id, userID, mock.Anything).Return(nil)

	// when
	err := svc.UpdateTheory(context.Background(), id, userID, validCreateTheoryReq())

	// then
	require.NoError(t, err)
}

func TestUpdateTheory_NonAdmin_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)
	m.repo.EXPECT().Update(mock.Anything, id, userID, mock.Anything).Return(errors.New("nope"))

	// when
	err := svc.UpdateTheory(context.Background(), id, userID, validCreateTheoryReq())

	// then
	require.Error(t, err)
}

func TestUpdateTheory_Admin_UpdateAsAdminError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, mock.Anything).Return(errors.New("nope"))

	// when
	err := svc.UpdateTheory(context.Background(), id, userID, validCreateTheoryReq())

	// then
	require.Error(t, err)
}

func TestUpdateTheory_Admin_OK_TriggersNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, mock.Anything).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).Return(authorID, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Mod"}, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == authorID && p.ActorID == userID && p.Type == dto.NotifContentEdited
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil)

	// when
	err := svc.UpdateTheory(context.Background(), id, userID, validCreateTheoryReq())

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestUpdateTheory_Admin_OK_AuthorLookupErrorSwallowed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, mock.Anything).Return(nil)

	done := make(chan struct{})
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, id).
		Run(func(_ context.Context, _ uuid.UUID) { close(done) }).
		Return(uuid.Nil, errors.New("missing"))

	// when
	err := svc.UpdateTheory(context.Background(), id, userID, validCreateTheoryReq())

	// then
	require.NoError(t, err)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not run")
	}
}

// --- DeleteTheory ---

func TestDeleteTheory_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(true)
	m.repo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteTheory(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteTheory_NonAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(false)
	m.repo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeleteTheory(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteTheory_NonAdmin_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(false)
	m.repo.EXPECT().Delete(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteTheory(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

// --- CreateResponse ---

func TestCreateResponse_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(5)
	m.repo.EXPECT().CountUserResponsesToday(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	_, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.Error(t, err)
}

func TestCreateResponse_RateLimited(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(3)
	m.repo.EXPECT().CountUserResponsesToday(mock.Anything, userID).Return(3, nil)

	// when
	_, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestCreateResponse_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.Error(t, err)
}

func TestCreateResponse_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateResponse_CannotRespondToOwnTheory_TopLevel(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(userID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)

	// when
	_, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.ErrorIs(t, err, ErrCannotRespondToOwnTheory)
}

func TestCreateResponse_OwnTheoryAllowedAsReply(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	responseID := uuid.New()
	parentID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(userID, nil).Maybe()
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	m.repo.EXPECT().CreateResponse(mock.Anything, theoryID, userID, mock.Anything).Return(responseID, nil)

	m.repo.EXPECT().GetTheorySeries(mock.Anything, theoryID).Return("umineko", nil).Maybe()
	m.repo.EXPECT().GetResponseEvidence(mock.Anything, responseID).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(0, 0, nil).Maybe()
	m.repo.EXPECT().UpdateCredibilityScore(mock.Anything, theoryID, mock.Anything).Return(nil).Maybe()
	m.repo.EXPECT().GetTheoryTitle(mock.Anything, theoryID).Return("t", nil).Maybe()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, parentID).Return(uuid.Nil, uuid.Nil, errors.New("x")).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{DisplayName: "Me"}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	req := dto.CreateResponseRequest{Side: "with_love", ParentID: &parentID}
	got, err := svc.CreateResponse(context.Background(), theoryID, userID, req)

	// then
	require.NoError(t, err)
	assert.Equal(t, responseID, got)
}

func TestCreateResponse_RepoCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateResponse(mock.Anything, theoryID, userID, mock.Anything).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.Error(t, err)
}

func TestCreateResponse_OK_SendsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	responseID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxResponsesPerDay).Return(0)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateResponse(mock.Anything, theoryID, userID, mock.Anything).Return(responseID, nil)

	m.repo.EXPECT().GetTheorySeries(mock.Anything, theoryID).Return("umineko", nil).Maybe()
	m.repo.EXPECT().GetResponseEvidence(mock.Anything, responseID).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(0, 0, nil).Maybe()
	m.repo.EXPECT().UpdateCredibilityScore(mock.Anything, theoryID, mock.Anything).Return(nil).Maybe()

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetTheoryTitle(mock.Anything, theoryID).Return("t", nil).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{DisplayName: "R"}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifTheoryResponse && p.RecipientID == authorID
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil).Maybe()

	// when
	got, err := svc.CreateResponse(context.Background(), theoryID, userID, dto.CreateResponseRequest{Side: "with_love"})

	// then
	require.NoError(t, err)
	assert.Equal(t, responseID, got)
	waitOrFail(t, &wg, 2*time.Second)
}

// --- DeleteResponse ---

func TestDeleteResponse_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	theoryID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, id).Return(uuid.New(), theoryID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyResponse).Return(true)
	m.repo.EXPECT().DeleteResponseAsAdmin(mock.Anything, id).Return(nil)
	m.repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(0, 0, nil).Maybe()
	m.repo.EXPECT().UpdateCredibilityScore(mock.Anything, theoryID, mock.Anything).Return(nil).Maybe()

	// when
	err := svc.DeleteResponse(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteResponse_NonAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	theoryID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, id).Return(uuid.New(), theoryID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyResponse).Return(false)
	m.repo.EXPECT().DeleteResponse(mock.Anything, id, userID).Return(nil)
	m.repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(0, 0, nil).Maybe()
	m.repo.EXPECT().UpdateCredibilityScore(mock.Anything, theoryID, mock.Anything).Return(nil).Maybe()

	// when
	err := svc.DeleteResponse(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteResponse_NonAdmin_RepoError_NoRecalc(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	theoryID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, id).Return(uuid.New(), theoryID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyResponse).Return(false)
	m.repo.EXPECT().DeleteResponse(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteResponse(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestDeleteResponse_ResponseInfoFailure_NoRecalc(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, id).Return(uuid.Nil, uuid.Nil, errors.New("boom"))
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyResponse).Return(false)
	m.repo.EXPECT().DeleteResponse(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeleteResponse(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

// --- VoteTheory ---

func TestVoteTheory_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.VoteTheory(context.Background(), userID, theoryID, 1)

	// then
	require.Error(t, err)
}

func TestVoteTheory_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.VoteTheory(context.Background(), userID, theoryID, 1)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestVoteTheory_VoteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteTheory(mock.Anything, userID, theoryID, 1).Return(errors.New("boom"))

	// when
	err := svc.VoteTheory(context.Background(), userID, theoryID, 1)

	// then
	require.Error(t, err)
}

func TestVoteTheory_Downvote_NoNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteTheory(mock.Anything, userID, theoryID, -1).Return(nil)

	// when
	err := svc.VoteTheory(context.Background(), userID, theoryID, -1)

	// then
	require.NoError(t, err)
}

func TestVoteTheory_Upvote_SendsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	theoryID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil).Times(1)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteTheory(mock.Anything, userID, theoryID, 1).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetTheoryAuthorID(mock.Anything, theoryID).Return(authorID, nil).Maybe()
	m.repo.EXPECT().GetTheoryTitle(mock.Anything, theoryID).Return("t", nil).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{DisplayName: "V"}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifTheoryUpvote && p.RecipientID == authorID
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil).Maybe()

	// when
	err := svc.VoteTheory(context.Background(), userID, theoryID, 1)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, 2*time.Second)
}

// --- VoteResponse ---

func TestVoteResponse_ResponseInfoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	responseID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(uuid.Nil, uuid.Nil, errors.New("boom"))

	// when
	err := svc.VoteResponse(context.Background(), userID, responseID, 1)

	// then
	require.Error(t, err)
}

func TestVoteResponse_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	responseID := uuid.New()
	respAuthorID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(respAuthorID, uuid.New(), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, respAuthorID).Return(true, nil)

	// when
	err := svc.VoteResponse(context.Background(), userID, responseID, 1)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestVoteResponse_VoteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	responseID := uuid.New()
	respAuthorID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(respAuthorID, uuid.New(), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, respAuthorID).Return(false, nil)
	m.repo.EXPECT().VoteResponse(mock.Anything, userID, responseID, 1).Return(errors.New("boom"))

	// when
	err := svc.VoteResponse(context.Background(), userID, responseID, 1)

	// then
	require.Error(t, err)
}

func TestVoteResponse_Downvote_NoNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	responseID := uuid.New()
	respAuthorID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(respAuthorID, uuid.New(), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, respAuthorID).Return(false, nil)
	m.repo.EXPECT().VoteResponse(mock.Anything, userID, responseID, -1).Return(nil)

	// when
	err := svc.VoteResponse(context.Background(), userID, responseID, -1)

	// then
	require.NoError(t, err)
}

func TestVoteResponse_Upvote_SendsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	responseID := uuid.New()
	respAuthorID := uuid.New()
	theoryID := uuid.New()
	m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(respAuthorID, theoryID, nil).Times(1)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, respAuthorID).Return(false, nil)
	m.repo.EXPECT().VoteResponse(mock.Anything, userID, responseID, 1).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetResponseInfo(mock.Anything, responseID).Return(respAuthorID, theoryID, nil).Maybe()
	m.repo.EXPECT().GetTheoryTitle(mock.Anything, theoryID).Return("t", nil).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{DisplayName: "V"}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifResponseUpvote && p.RecipientID == respAuthorID
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil).Maybe()

	// when
	err := svc.VoteResponse(context.Background(), userID, responseID, 1)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, 2*time.Second)
}
