package auth

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	slursrule "umineko_city_of_books/internal/contentfilter/rules/slurs"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	userSvc     *user.MockService
	settingsSvc *settings.MockService
	inviteRepo  *repository.MockInviteRepository
	userRepo    *repository.MockUserRepository
	auditRepo   *repository.MockAuditLogRepository
	sessionRepo *repository.MockSessionRepository
}

func newTestService(t *testing.T) (*service, *testMocks) {
	userSvc := user.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	inviteRepo := repository.NewMockInviteRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	sessionRepo := repository.NewMockSessionRepository(t)
	sessionMgr := session.NewManager(sessionRepo, settingsSvc)
	filter := contentfilter.New(slursrule.New())
	svc := NewService(userSvc, sessionMgr, settingsSvc, inviteRepo, userRepo, auditRepo, filter).(*service)
	return svc, &testMocks{
		userSvc:     userSvc,
		settingsSvc: settingsSvc,
		inviteRepo:  inviteRepo,
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		sessionRepo: sessionRepo,
	}
}

func validRegisterRequest() dto.RegisterRequest {
	return dto.RegisterRequest{
		LoginRequest: dto.LoginRequest{
			Username: "alice",
			Password: "password123",
		},
		DisplayName: "Alice",
	}
}

func expectOpenRegistration(m *testMocks) {
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("open")
}

func expectMinPasswordLength(m *testMocks, n int) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(n)
}

func expectSessionDuration(m *testMocks) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingSessionDurationDays).Return(30)
}

func TestRegister_ClosedRegistrationRejected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("closed")

	// when
	_, _, err := svc.Register(context.Background(), validRegisterRequest())

	// then
	require.ErrorIs(t, err, ErrRegistrationDisabled)
}

func TestRegister_InviteRequiredButMissing(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("invite")
	req := validRegisterRequest()

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrInviteRequired)
}

func TestRegister_InviteLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("invite")
	req := validRegisterRequest()
	req.InviteCode = "code123"
	m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(nil, errors.New("db down"))

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check invite")
}

func TestRegister_InviteNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("invite")
	req := validRegisterRequest()
	req.InviteCode = "code123"
	m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(nil, nil)

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidInvite)
}

func TestRegister_InviteAlreadyUsed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("invite")
	req := validRegisterRequest()
	req.InviteCode = "code123"
	usedBy := uuid.New()
	m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(&repository.Invite{Code: "code123", UsedBy: &usedBy}, nil)

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidInvite)
}

func TestRegister_InvalidUsername(t *testing.T) {
	cases := []struct {
		name     string
		username string
	}{
		{"too short", "ab"},
		{"too long", "a123456789012345678901234567890"},
		{"bad characters", "alice!"},
		{"spaces", "alice bob"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			expectOpenRegistration(m)
			req := validRegisterRequest()
			req.Username = tc.username

			// when
			_, _, err := svc.Register(context.Background(), req)

			// then
			require.ErrorIs(t, err, ErrInvalidUsername)
		})
	}
}

func TestRegister_ReservedUsername(t *testing.T) {
	cases := []string{"featherine", "FAA_fan", "myauauroratheory"}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			expectOpenRegistration(m)
			req := validRegisterRequest()
			req.Username = name

			// when
			_, _, err := svc.Register(context.Background(), req)

			// then
			require.ErrorIs(t, err, user.ErrUsernameTaken)
		})
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	req.Password = "short"

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestRegister_UsernameTaken(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(user.ErrUsernameTaken)

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, user.ErrUsernameTaken)
}

func TestRegister_CreateUserError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, req.Password, req.DisplayName).Return(nil, errors.New("db down"))

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create user")
}

func TestRegister_SessionCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	userID := uuid.New()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "user_created", "user", userID.String(), "username="+req.Username).Return(nil)
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(errors.New("boom"))

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
}

func TestRegister_OpenOK_DefaultsDisplayName(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	req.DisplayName = ""
	userID := uuid.New()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, req.Password, req.Username).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "user_created", "user", userID.String(), "username="+req.Username).Return(nil)
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil)

	// when
	resp, token, err := svc.Register(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, userID, resp.ID)
}

func TestRegister_InviteOK_MarksUsed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("invite")
	req := validRegisterRequest()
	req.InviteCode = "code123"
	m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(&repository.Invite{Code: "code123"}, nil)
	expectMinPasswordLength(m, 8)
	userID := uuid.New()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "user_created", "user", userID.String(), "username="+req.Username).Return(nil)
	m.inviteRepo.EXPECT().MarkUsed(mock.Anything, "code123", userID).Return(nil)
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil)

	// when
	resp, token, err := svc.Register(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, userID, resp.ID)
}

func TestRegister_InviteOK_MarkUsedErrorSwallowed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("invite")
	req := validRegisterRequest()
	req.InviteCode = "code123"
	m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(&repository.Invite{Code: "code123"}, nil)
	expectMinPasswordLength(m, 8)
	userID := uuid.New()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "user_created", "user", userID.String(), "username="+req.Username).Return(nil)
	m.inviteRepo.EXPECT().MarkUsed(mock.Anything, "code123", userID).Return(errors.New("boom"))
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil)

	// when
	resp, token, err := svc.Register(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, userID, resp.ID)
}

func TestRegister_MinPasswordLengthZeroSkipsCheck(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 0)
	req := validRegisterRequest()
	req.Password = "x"
	userID := uuid.New()
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "user_created", "user", userID.String(), "username="+req.Username).Return(nil)
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil)

	// when
	_, token, err := svc.Register(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	// given
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "wrong"}
	m.userSvc.EXPECT().ValidateCredentials(mock.Anything, req.Username, req.Password).Return(nil, user.ErrInvalidCredentials)

	// when
	_, _, err := svc.Login(context.Background(), req)

	// then
	require.ErrorIs(t, err, user.ErrInvalidCredentials)
}

func TestLogin_BannedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "password123"}
	userID := uuid.New()
	m.userSvc.EXPECT().ValidateCredentials(mock.Anything, req.Username, req.Password).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(true, nil)

	// when
	_, _, err := svc.Login(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrUserBanned)
}

func TestLogin_SessionCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "password123"}
	userID := uuid.New()
	m.userSvc.EXPECT().ValidateCredentials(mock.Anything, req.Username, req.Password).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, nil)
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(errors.New("boom"))

	// when
	_, _, err := svc.Login(context.Background(), req)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
}

func TestLogin_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "password123"}
	userID := uuid.New()
	m.userSvc.EXPECT().ValidateCredentials(mock.Anything, req.Username, req.Password).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, nil)
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil)

	// when
	resp, token, err := svc.Login(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, userID, resp.ID)
}

func TestLogin_BannedCheckErrorTreatedAsNotBanned(t *testing.T) {
	// given
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "password123"}
	userID := uuid.New()
	m.userSvc.EXPECT().ValidateCredentials(mock.Anything, req.Username, req.Password).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, errors.New("db down"))
	expectSessionDuration(m)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil)

	// when
	_, token, err := svc.Login(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLogout_EmptyTokenNoop(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.Logout(context.Background(), "")

	// then
	require.NoError(t, err)
}

func TestLogout_DeletesSession(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.sessionRepo.EXPECT().Delete(mock.Anything, "token123").Return(nil)

	// when
	err := svc.Logout(context.Background(), "token123")

	// then
	require.NoError(t, err)
}

func TestLogout_DeleteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.sessionRepo.EXPECT().Delete(mock.Anything, "token123").Return(errors.New("boom"))

	// when
	err := svc.Logout(context.Background(), "token123")

	// then
	require.Error(t, err)
}
