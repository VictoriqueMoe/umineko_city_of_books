package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	slursrule "umineko_city_of_books/internal/contentfilter/rules/slurs"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
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
	resetRepo   *repository.MockPasswordResetRepository
	verifyRepo  *repository.MockEmailVerificationRepository
	emailSvc    *email.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	userSvc := user.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	inviteRepo := repository.NewMockInviteRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	sessionRepo := repository.NewMockSessionRepository(t)
	resetRepo := repository.NewMockPasswordResetRepository(t)
	verifyRepo := repository.NewMockEmailVerificationRepository(t)
	emailSvc := email.NewMockService(t)
	sessionMgr := session.NewManager(sessionRepo, settingsSvc)
	filter := contentfilter.New(slursrule.New())
	svc := NewService(userSvc, sessionMgr, settingsSvc, inviteRepo, userRepo, auditRepo, resetRepo, verifyRepo, emailSvc, filter).(*service)
	return svc, &testMocks{
		userSvc:     userSvc,
		settingsSvc: settingsSvc,
		inviteRepo:  inviteRepo,
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		sessionRepo: sessionRepo,
		resetRepo:   resetRepo,
		verifyRepo:  verifyRepo,
		emailSvc:    emailSvc,
	}
}

func validRegisterRequest() dto.RegisterRequest {
	return dto.RegisterRequest{
		LoginRequest: dto.LoginRequest{
			Username: "alice",
			Password: "password123",
		},
		Email:       "alice@example.com",
		DisplayName: "Alice",
	}
}

func expectVerificationSent(m *testMocks, userID uuid.UUID, email string) {
	m.verifyRepo.EXPECT().DeleteUnusedForUser(mock.Anything, userID).Return(nil)
	m.verifyRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), userID, mock.Anything).Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://localhost:4323")
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().Send(mock.Anything, email, mock.Anything, mock.Anything).Return(nil)
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

func TestForgotPassword_EmailDisabled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(false)

	// when
	err := svc.ForgotPassword(context.Background(), "alice")

	// then
	require.ErrorIs(t, err, ErrEmailDisabled)
}

func TestForgotPassword_UserNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(true)
	m.userRepo.EXPECT().GetByUsername(mock.Anything, "ghost").Return(nil, nil)

	// when
	err := svc.ForgotPassword(context.Background(), "ghost")

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestForgotPassword_NoEmailSet(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(true)
	m.userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: uuid.New(), Email: ""}, nil)

	// when
	err := svc.ForgotPassword(context.Background(), "alice")

	// then
	require.ErrorIs(t, err, ErrNoEmailAddress)
}

func TestForgotPassword_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(true)
	m.userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: userID, Email: "alice@example.com"}, nil)
	m.resetRepo.EXPECT().DeleteUnusedForUser(mock.Anything, userID).Return(nil)
	m.resetRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), userID, mock.Anything).Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://localhost:4323")
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().Send(mock.Anything, "alice@example.com", mock.Anything, mock.Anything).Return(nil)

	// when
	err := svc.ForgotPassword(context.Background(), "alice")

	// then
	require.NoError(t, err)
}

func TestResetPassword_EmptyToken(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.ResetPassword(context.Background(), "", "newpassword123")

	// then
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestResetPassword_TooShort(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectMinPasswordLength(m, 8)

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "short")

	// then
	require.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectMinPasswordLength(m, 8)
	m.resetRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "newpassword123")

	// then
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestResetPassword_Expired(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectMinPasswordLength(m, 8)
	expired := &repository.PasswordResetToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Hour)}
	m.resetRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(expired, nil)

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "newpassword123")

	// then
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestResetPassword_AlreadyUsed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectMinPasswordLength(m, 8)
	usedAt := time.Now().Add(-time.Minute)
	used := &repository.PasswordResetToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour), UsedAt: &usedAt}
	m.resetRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(used, nil)

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "newpassword123")

	// then
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestResetPassword_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	expectMinPasswordLength(m, 8)
	valid := &repository.PasswordResetToken{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}
	m.resetRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(valid, nil)
	m.userRepo.EXPECT().SetPassword(mock.Anything, userID, "newpassword123").Return(nil)
	m.resetRepo.EXPECT().MarkUsed(mock.Anything, mock.Anything).Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, userID).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "password_reset", "user", userID.String(), "").Return(nil)

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "newpassword123")

	// then
	require.NoError(t, err)
}

func TestEmailEnabled_DelegatesToEmailService(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(true)

	// when
	enabled := svc.EmailEnabled(context.Background())

	// then
	assert.True(t, enabled)
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
	m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(&repository.Invite{Code: "code123", UsedBy: new(uuid.New())}, nil)

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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, "alice@example.com", req.Password, req.DisplayName).Return(nil, errors.New("db down"))

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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, "alice@example.com", req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	expectVerificationSent(m, userID, "alice@example.com")
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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, "alice@example.com", req.Password, req.Username).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	expectVerificationSent(m, userID, "alice@example.com")
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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, "alice@example.com", req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	expectVerificationSent(m, userID, "alice@example.com")
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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, "alice@example.com", req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	expectVerificationSent(m, userID, "alice@example.com")
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
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().Create(mock.Anything, req.Username, "alice@example.com", req.Password, req.DisplayName).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)
	expectVerificationSent(m, userID, "alice@example.com")
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

func TestRegister_InvalidEmail(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	req.Email = "not-an-email"

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidEmail)
}

func TestRegister_EmailTaken(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectOpenRegistration(m)
	expectMinPasswordLength(m, 8)
	req := validRegisterRequest()
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "alice@example.com", uuid.Nil).Return(true, nil)

	// when
	_, _, err := svc.Register(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestSetEmail_InvalidEmail(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.SetEmail(context.Background(), uuid.New(), "nope")

	// then
	require.ErrorIs(t, err, ErrInvalidEmail)
}

func TestSetEmail_EmailTaken(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "taken@example.com", userID).Return(true, nil)

	// when
	err := svc.SetEmail(context.Background(), userID, "taken@example.com")

	// then
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestSetEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.userRepo.EXPECT().EmailInUse(mock.Anything, "new@example.com", userID).Return(false, nil)
	m.userRepo.EXPECT().SetEmail(mock.Anything, userID, "new@example.com").Return(nil)
	expectVerificationSent(m, userID, "new@example.com")

	// when
	err := svc.SetEmail(context.Background(), userID, "New@Example.com")

	// then
	require.NoError(t, err)
}

func TestVerifyEmail_EmptyToken(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.VerifyEmail(context.Background(), "")

	// then
	require.ErrorIs(t, err, ErrInvalidVerificationToken)
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.verifyRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	err := svc.VerifyEmail(context.Background(), "sometoken")

	// then
	require.ErrorIs(t, err, ErrInvalidVerificationToken)
}

func TestVerifyEmail_Expired(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expired := &repository.EmailVerificationToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Hour)}
	m.verifyRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(expired, nil)

	// when
	err := svc.VerifyEmail(context.Background(), "sometoken")

	// then
	require.ErrorIs(t, err, ErrInvalidVerificationToken)
}

func TestVerifyEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	rec := &repository.EmailVerificationToken{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}
	m.verifyRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(rec, nil)
	m.userRepo.EXPECT().MarkEmailVerified(mock.Anything, userID).Return(nil)
	m.verifyRepo.EXPECT().MarkUsed(mock.Anything, mock.Anything).Return(nil)

	// when
	err := svc.VerifyEmail(context.Background(), "sometoken")

	// then
	require.NoError(t, err)
}

func TestResendVerification_NoEmail(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Email: ""}, nil)

	// when
	err := svc.ResendVerification(context.Background(), userID)

	// then
	require.ErrorIs(t, err, ErrNoEmailAddress)
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Email: "a@example.com", EmailVerified: true}, nil)

	// when
	err := svc.ResendVerification(context.Background(), userID)

	// then
	require.ErrorIs(t, err, ErrEmailAlreadyVerified)
}

func TestResendVerification_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Email: "a@example.com", EmailVerified: false}, nil)
	expectVerificationSent(m, userID, "a@example.com")

	// when
	err := svc.ResendVerification(context.Background(), userID)

	// then
	require.NoError(t, err)
}
