package admin

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/giphy/banlist"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	userRepo    *repository.MockUserRepository
	roleRepo    *repository.MockRoleRepository
	statsRepo   *repository.MockStatsRepository
	auditRepo   *repository.MockAuditLogRepository
	inviteRepo  *repository.MockInviteRepository
	vanityRepo  *repository.MockVanityRoleRepository
	permRepo    *repository.MockPermissionRepository
	sessionRepo *repository.MockSessionRepository
	bannedRepo  *repository.MockBannedGiphyRepository
	authz       *authz.MockService
	settingsSvc *settings.MockService
	uploadSvc   *upload.MockService
	hub         *ws.Hub
	chatSync    *MockSystemRoomSync
	banlist     banlist.Service
	emailSvc    *email.MockService
	authSvc     *auth.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	statsRepo := repository.NewMockStatsRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	inviteRepo := repository.NewMockInviteRepository(t)
	vanityRepo := repository.NewMockVanityRoleRepository(t)
	permRepo := repository.NewMockPermissionRepository(t)
	sessionRepo := repository.NewMockSessionRepository(t)
	bannedRepo := repository.NewMockBannedGiphyRepository(t)
	bannedRepo.EXPECT().List(mock.Anything).Return(nil, nil).Maybe()
	banlistSvc, err := banlist.NewService(context.Background(), bannedRepo)
	require.NoError(t, err)
	authzSvc := authz.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	hub := ws.NewHub()
	chatSync := NewMockSystemRoomSync(t)
	sessionMgr := session.NewManager(sessionRepo, settingsSvc)
	emailSvc := email.NewMockService(t)
	authSvc := auth.NewMockService(t)

	svc := NewService(
		userRepo,
		roleRepo,
		statsRepo,
		auditRepo,
		inviteRepo,
		vanityRepo,
		permRepo,
		banlistSvc,
		authzSvc,
		settingsSvc,
		sessionMgr,
		uploadSvc,
		hub,
		chatSync,
		emailSvc,
		authSvc,
	).(*service)

	return svc, &testMocks{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		statsRepo:   statsRepo,
		auditRepo:   auditRepo,
		inviteRepo:  inviteRepo,
		vanityRepo:  vanityRepo,
		permRepo:    permRepo,
		sessionRepo: sessionRepo,
		bannedRepo:  bannedRepo,
		authz:       authzSvc,
		settingsSvc: settingsSvc,
		uploadSvc:   uploadSvc,
		hub:         hub,
		chatSync:    chatSync,
		banlist:     banlistSvc,
		emailSvc:    emailSvc,
		authSvc:     authSvc,
	}
}

func TestGetStats_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.statsRepo.EXPECT().GetOverview(mock.Anything).Return(&repository.SiteStats{
		TotalUsers:     5,
		TotalTheories:  3,
		TotalResponses: 2,
		PostsByCorner:  map[string]int{"a": 1},
	}, nil)
	m.statsRepo.EXPECT().GetMostActiveUsers(mock.Anything, 10).Return([]repository.ActiveUser{
		{ID: userID, Username: "u", DisplayName: "U", AvatarURL: "/a.png", ActionCount: 7},
	}, nil)

	// when
	got, err := svc.GetStats(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, got.TotalUsers)
	assert.Equal(t, 3, got.TotalTheories)
	assert.Len(t, got.MostActiveUsers, 1)
	assert.Equal(t, userID, got.MostActiveUsers[0].ID)
	assert.Equal(t, 7, got.MostActiveUsers[0].ActionCount)
}

func TestGetStats_OverviewError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.statsRepo.EXPECT().GetOverview(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetStats(context.Background())

	// then
	require.Error(t, err)
}
