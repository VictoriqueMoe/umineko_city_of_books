package report

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (
	*service,
	*repository.MockReportRepository,
	*repository.MockRoleRepository,
	*repository.MockUserRepository,
	*notification.MockService,
	*settings.MockService,
) {
	reportRepo := repository.NewMockReportRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	svc := NewService(reportRepo, roleRepo, userRepo, notifSvc, settingsSvc).(*service)
	return svc, reportRepo, roleRepo, userRepo, notifSvc, settingsSvc
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

func TestCreate_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		req  CreateReportRequest
	}{
		{"missing target_type", CreateReportRequest{TargetType: "", TargetID: "x", Reason: "r"}},
		{"missing target_id", CreateReportRequest{TargetType: "post", TargetID: "", Reason: "r"}},
		{"missing reason", CreateReportRequest{TargetType: "post", TargetID: "x", Reason: ""}},
		{"all empty", CreateReportRequest{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _, _, _, _, _ := newTestService(t)
			reporterID := uuid.New()

			// when
			err := svc.Create(context.Background(), reporterID, tc.req)

			// then
			require.ErrorIs(t, err, ErrMissingFields)
		})
	}
}

func TestCreate_RepoErrorBubbles(t *testing.T) {
	// given
	svc, reportRepo, _, _, _, _ := newTestService(t)
	reporterID := uuid.New()
	req := CreateReportRequest{TargetType: "post", TargetID: uuid.NewString(), Reason: "spam"}
	reportRepo.EXPECT().Create(mock.Anything, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason).
		Return(int64(0), errors.New("db down"))

	// when
	err := svc.Create(context.Background(), reporterID, req)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create report")
	assert.Contains(t, err.Error(), "db down")
}

func TestCreate_OK_NotifiesModerators(t *testing.T) {
	// given
	svc, reportRepo, roleRepo, userRepo, notifSvc, settingsSvc := newTestService(t)
	reporterID := uuid.New()
	targetID := uuid.New()
	req := CreateReportRequest{
		TargetType: "post",
		TargetID:   targetID.String(),
		ContextID:  "ctx-123",
		Reason:     "spam",
	}
	modA := uuid.New()
	modB := uuid.New()
	reporter := &model.User{ID: reporterID, DisplayName: "Alice"}

	reportRepo.EXPECT().Create(mock.Anything, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason).
		Return(int64(42), nil)

	var wg sync.WaitGroup
	wg.Add(1)
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{
		authz.RoleSuperAdmin,
		authz.RoleAdmin,
		authz.RoleModerator,
	}).Return([]uuid.UUID{modA, modB}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, reporterID).Return(reporter, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.MatchedBy(func(params []dto.NotifyParams) bool {
		if len(params) != 2 {
			return false
		}
		for i, p := range params {
			expected := []uuid.UUID{modA, modB}[i]
			if p.RecipientID != expected {
				return false
			}
			if p.Type != dto.NotifReport {
				return false
			}
			if p.ReferenceID != targetID {
				return false
			}
			if p.ReferenceType != req.TargetType {
				return false
			}
			if p.ActorID != reporterID {
				return false
			}
			if p.EmailSubject == "" || p.EmailBody == "" {
				return false
			}
		}
		return true
	})).Run(func(_ context.Context, _ []dto.NotifyParams) {
		wg.Done()
	}).Return()

	// when
	err := svc.Create(context.Background(), reporterID, req)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestCreate_OK_InvalidTargetIDUsesNilUUID(t *testing.T) {
	// given
	svc, reportRepo, roleRepo, userRepo, notifSvc, settingsSvc := newTestService(t)
	reporterID := uuid.New()
	req := CreateReportRequest{
		TargetType: "post",
		TargetID:   "not-a-uuid",
		Reason:     "spam",
	}
	mod := uuid.New()
	reporter := &model.User{ID: reporterID, DisplayName: "Alice"}

	reportRepo.EXPECT().Create(mock.Anything, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason).
		Return(int64(1), nil)

	var wg sync.WaitGroup
	wg.Add(1)
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return([]uuid.UUID{mod}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, reporterID).Return(reporter, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.MatchedBy(func(params []dto.NotifyParams) bool {
		if len(params) != 1 {
			return false
		}
		return params[0].ReferenceID == uuid.Nil
	})).Run(func(_ context.Context, _ []dto.NotifyParams) {
		wg.Done()
	}).Return()

	// when
	err := svc.Create(context.Background(), reporterID, req)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestCreate_OK_RoleRepoErrorAbortsNotifications(t *testing.T) {
	// given
	svc, reportRepo, roleRepo, _, _, _ := newTestService(t)
	reporterID := uuid.New()
	req := CreateReportRequest{
		TargetType: "post",
		TargetID:   uuid.NewString(),
		Reason:     "spam",
	}
	reportRepo.EXPECT().Create(mock.Anything, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason).
		Return(int64(1), nil)

	done := make(chan struct{})
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ []role.Role) { close(done) }).
		Return(nil, errors.New("db down"))

	// when
	err := svc.Create(context.Background(), reporterID, req)

	// then
	require.NoError(t, err)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("role lookup goroutine did not run")
	}
}

func TestCreate_OK_UserLookupErrorFallsBackToDefaultName(t *testing.T) {
	// given
	svc, reportRepo, roleRepo, userRepo, notifSvc, settingsSvc := newTestService(t)
	reporterID := uuid.New()
	req := CreateReportRequest{
		TargetType: "post",
		TargetID:   uuid.NewString(),
		Reason:     "spam",
	}
	mod := uuid.New()

	reportRepo.EXPECT().Create(mock.Anything, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason).
		Return(int64(1), nil)

	var wg sync.WaitGroup
	wg.Add(1)
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return([]uuid.UUID{mod}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, reporterID).Return(nil, errors.New("missing"))
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ []dto.NotifyParams) { wg.Done() }).
		Return()

	// when
	err := svc.Create(context.Background(), reporterID, req)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestCreate_OK_NoModeratorsStillCallsNotifyManyWithEmptyList(t *testing.T) {
	// given
	svc, reportRepo, roleRepo, userRepo, notifSvc, settingsSvc := newTestService(t)
	reporterID := uuid.New()
	req := CreateReportRequest{
		TargetType: "post",
		TargetID:   uuid.NewString(),
		Reason:     "spam",
	}
	reporter := &model.User{ID: reporterID, DisplayName: "Alice"}

	reportRepo.EXPECT().Create(mock.Anything, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason).
		Return(int64(1), nil)

	var wg sync.WaitGroup
	wg.Add(1)
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return(nil, nil)
	userRepo.EXPECT().GetByID(mock.Anything, reporterID).Return(reporter, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.MatchedBy(func(params []dto.NotifyParams) bool {
		return len(params) == 0
	})).Run(func(_ context.Context, _ []dto.NotifyParams) { wg.Done() }).Return()

	// when
	err := svc.Create(context.Background(), reporterID, req)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestList_OK(t *testing.T) {
	// given
	svc, reportRepo, _, _, _, _ := newTestService(t)
	resolver := uuid.New()
	rows := []repository.ReportRow{
		{
			ID:             1,
			ReporterID:     uuid.New(),
			ReporterName:   "Alice",
			ReporterAvatar: "a.png",
			TargetType:     "post",
			TargetID:       "t1",
			ContextID:      "ctx1",
			Reason:         "spam",
			Status:         "open",
			CreatedAt:      "2026-01-01",
		},
		{
			ID:             2,
			ReporterID:     uuid.New(),
			ReporterName:   "Bob",
			ReporterAvatar: "b.png",
			TargetType:     "comment",
			TargetID:       "t2",
			ContextID:      "",
			Reason:         "abuse",
			Status:         "resolved",
			ResolvedByID:   &resolver,
			ResolvedByName: "Mod",
			CreatedAt:      "2026-01-02",
		},
	}
	reportRepo.EXPECT().List(mock.Anything, "open", 10, 5).Return(rows, 42, nil)

	// when
	got, err := svc.List(context.Background(), "open", 10, 5)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 42, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 5, got.Offset)
	require.Len(t, got.Reports, 2)
	assert.Equal(t, 1, got.Reports[0].ID)
	assert.Equal(t, "Alice", got.Reports[0].ReporterName)
	assert.Equal(t, "a.png", got.Reports[0].ReporterAvatar)
	assert.Equal(t, "post", got.Reports[0].TargetType)
	assert.Equal(t, "t1", got.Reports[0].TargetID)
	assert.Equal(t, "ctx1", got.Reports[0].ContextID)
	assert.Equal(t, "spam", got.Reports[0].Reason)
	assert.Equal(t, "open", got.Reports[0].Status)
	assert.Equal(t, "", got.Reports[0].ResolvedBy)
	assert.Equal(t, "2026-01-01", got.Reports[0].CreatedAt)
	assert.Equal(t, "Mod", got.Reports[1].ResolvedBy)
}

func TestList_EmptyResult(t *testing.T) {
	// given
	svc, reportRepo, _, _, _, _ := newTestService(t)
	reportRepo.EXPECT().List(mock.Anything, "", 10, 0).Return(nil, 0, nil)

	// when
	got, err := svc.List(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Empty(t, got.Reports)
	assert.Equal(t, 0, got.Total)
}

func TestList_RepoError(t *testing.T) {
	// given
	svc, reportRepo, _, _, _, _ := newTestService(t)
	reportRepo.EXPECT().List(mock.Anything, "", 10, 0).Return(nil, 0, errors.New("db down"))

	// when
	got, err := svc.List(context.Background(), "", 10, 0)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "list reports")
	assert.Contains(t, err.Error(), "db down")
}

func TestResolve_GetByIDError(t *testing.T) {
	// given
	svc, reportRepo, _, _, _, _ := newTestService(t)
	resolverID := uuid.New()
	reportRepo.EXPECT().GetByID(mock.Anything, 7).Return(nil, errors.New("not found"))

	// when
	err := svc.Resolve(context.Background(), 7, resolverID, "ok")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve report")
	assert.Contains(t, err.Error(), "not found")
}

func TestResolve_RepoResolveErrorBubbles(t *testing.T) {
	// given
	svc, reportRepo, _, _, _, _ := newTestService(t)
	resolverID := uuid.New()
	reporterID := uuid.New()
	row := &repository.ReportRow{
		ID:         7,
		ReporterID: reporterID,
		TargetType: "post",
		TargetID:   uuid.NewString(),
	}
	reportRepo.EXPECT().GetByID(mock.Anything, 7).Return(row, nil)
	reportRepo.EXPECT().Resolve(mock.Anything, 7, resolverID, "ok").Return(errors.New("db down"))

	// when
	err := svc.Resolve(context.Background(), 7, resolverID, "ok")

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "db down")
}

func TestResolve_OK_SendsNotificationWithComment(t *testing.T) {
	// given
	svc, reportRepo, _, userRepo, notifSvc, settingsSvc := newTestService(t)
	resolverID := uuid.New()
	reporterID := uuid.New()
	targetID := uuid.New()
	row := &repository.ReportRow{
		ID:         7,
		ReporterID: reporterID,
		TargetType: "post",
		TargetID:   targetID.String(),
	}
	resolver := &model.User{ID: resolverID, DisplayName: "ModUser"}

	reportRepo.EXPECT().GetByID(mock.Anything, 7).Return(row, nil)
	reportRepo.EXPECT().Resolve(mock.Anything, 7, resolverID, "handled").Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	userRepo.EXPECT().GetByID(mock.Anything, resolverID).Return(resolver, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == reporterID &&
			p.Type == dto.NotifReportResolved &&
			p.ReferenceID == targetID &&
			p.ReferenceType == "post" &&
			p.ActorID == resolverID &&
			p.Message == "resolved your report on a post: handled" &&
			p.EmailSubject != "" &&
			p.EmailBody != ""
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil)

	// when
	err := svc.Resolve(context.Background(), 7, resolverID, "handled")

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestResolve_OK_EmptyCommentOmitsCommentFromMessage(t *testing.T) {
	// given
	svc, reportRepo, _, userRepo, notifSvc, settingsSvc := newTestService(t)
	resolverID := uuid.New()
	reporterID := uuid.New()
	targetID := uuid.New()
	row := &repository.ReportRow{
		ID:         7,
		ReporterID: reporterID,
		TargetType: "comment",
		TargetID:   targetID.String(),
	}
	resolver := &model.User{ID: resolverID, DisplayName: "ModUser"}

	reportRepo.EXPECT().GetByID(mock.Anything, 7).Return(row, nil)
	reportRepo.EXPECT().Resolve(mock.Anything, 7, resolverID, "").Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	userRepo.EXPECT().GetByID(mock.Anything, resolverID).Return(resolver, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Message == "resolved your report on a comment"
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil)

	// when
	err := svc.Resolve(context.Background(), 7, resolverID, "")

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestResolve_OK_InvalidTargetIDUsesNilUUID(t *testing.T) {
	// given
	svc, reportRepo, _, userRepo, notifSvc, settingsSvc := newTestService(t)
	resolverID := uuid.New()
	reporterID := uuid.New()
	row := &repository.ReportRow{
		ID:         7,
		ReporterID: reporterID,
		TargetType: "post",
		TargetID:   "not-a-uuid",
	}
	resolver := &model.User{ID: resolverID, DisplayName: "ModUser"}

	reportRepo.EXPECT().GetByID(mock.Anything, 7).Return(row, nil)
	reportRepo.EXPECT().Resolve(mock.Anything, 7, resolverID, "ok").Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	userRepo.EXPECT().GetByID(mock.Anything, resolverID).Return(resolver, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.ReferenceID == uuid.Nil
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil)

	// when
	err := svc.Resolve(context.Background(), 7, resolverID, "ok")

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestResolve_OK_UserLookupErrorFallsBackToDefaultName(t *testing.T) {
	// given
	svc, reportRepo, _, userRepo, notifSvc, settingsSvc := newTestService(t)
	resolverID := uuid.New()
	reporterID := uuid.New()
	targetID := uuid.New()
	row := &repository.ReportRow{
		ID:         7,
		ReporterID: reporterID,
		TargetType: "post",
		TargetID:   targetID.String(),
	}

	reportRepo.EXPECT().GetByID(mock.Anything, 7).Return(row, nil)
	reportRepo.EXPECT().Resolve(mock.Anything, 7, resolverID, "ok").Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	userRepo.EXPECT().GetByID(mock.Anything, resolverID).Return(nil, errors.New("missing"))
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).
		Return(nil)

	// when
	err := svc.Resolve(context.Background(), 7, resolverID, "ok")

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}
