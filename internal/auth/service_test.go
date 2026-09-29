package auth

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	slursrule "umineko_city_of_books/internal/contentfilter/rules/slurs"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type (
	testMocks struct {
		userSvc     *user.MockService
		settingsSvc *settings.MockService
		inviteRepo  *repository.MockInviteRepository
		userRepo    *repository.MockUserRepository
		sessionRepo *repository.MockSessionRepository
		resetRepo   *repository.MockPasswordResetRepository
		verifyRepo  *repository.MockEmailVerificationRepository
		auditRepo   *repository.MockAuditLogRepository
		emailSvc    *email.MockService
	}
)

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		userSvc:     user.NewMockService(t),
		settingsSvc: settings.NewMockService(t),
		inviteRepo:  repository.NewMockInviteRepository(t),
		userRepo:    repository.NewMockUserRepository(t),
		sessionRepo: repository.NewMockSessionRepository(t),
		resetRepo:   repository.NewMockPasswordResetRepository(t),
		verifyRepo:  repository.NewMockEmailVerificationRepository(t),
		auditRepo:   repository.NewMockAuditLogRepository(t),
		emailSvc:    email.NewMockService(t),
	}

	filter := contentfilter.New(slursrule.New())
	sessionMgr := session.NewManager(m.sessionRepo, m.settingsSvc)
	svc := NewService(m.userSvc, sessionMgr, m.settingsSvc, m.inviteRepo, m.userRepo, m.resetRepo, m.verifyRepo, m.auditRepo, m.emailSvc, filter, filter).(*service)

	return svc, m
}

func hashFor(t *testing.T, password string) string {
	t.Helper()

	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)

	return string(h)
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

func accountSpec(username, email, displayName string) spec.NewAccount {
	return spec.NewAccount{
		User: spec.NewUser{
			Username:     username,
			Email:        email,
			PasswordHash: "already-hashed",
			DisplayName:  displayName,
			HomePage:     "landing",
			DMsEnabled:   true,
		},
	}
}

func matchesRegistration(account spec.NewAccount, inviteCode string) any {
	return mock.MatchedBy(func(registration spec.NewRegistration) bool {
		return registration.Account == account &&
			registration.InviteCode == inviteCode &&
			registration.VerificationHash != "" &&
			!registration.VerificationExpiresAt.IsZero() &&
			registration.SessionToken != "" &&
			!registration.SessionExpiresAt.IsZero()
	})
}

func expectAccountBuilt(m *testMocks, req dto.RegisterRequest, displayName string) spec.NewAccount {
	account := accountSpec(req.Username, "alice@example.com", displayName)
	m.userRepo.EXPECT().EmailInUse(mock.Anything, spec.UserEmailFilter{Email: "alice@example.com", ExcludeUserID: uuid.Nil}).Return(false, nil)
	m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
	m.userSvc.EXPECT().NewAccountSpec(mock.Anything, req.Username, "alice@example.com", req.Password, displayName).Return(account, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingSessionDurationDays).Return(30)

	return account
}

func expectSiteEmailSent(m *testMocks, to string) {
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://localhost:4323")
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().Send(mock.Anything, to, mock.Anything, mock.Anything).Return(nil)
}

func expectVerificationSent(m *testMocks, userID uuid.UUID, email string) {
	m.verifyRepo.EXPECT().Issue(mock.Anything, mock.MatchedBy(func(verification spec.NewEmailVerification) bool {
		return verification.UserID == userID && verification.TokenHash != "" && !verification.ExpiresAt.IsZero()
	})).Return(nil)
	expectSiteEmailSent(m, email)
}

func expectEmailReplaced(m *testMocks, userID uuid.UUID, previousEmail string) {
	m.userRepo.EXPECT().EmailInUse(mock.Anything, spec.UserEmailFilter{Email: "new@example.com", ExcludeUserID: userID}).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Email: previousEmail}, nil)
	m.userRepo.EXPECT().SetEmail(mock.Anything, spec.UserEmailUpdate{UserID: userID, Email: "new@example.com"}).Return(nil)
	expectVerificationSent(m, userID, "new@example.com")
}

func expectPreviousAddressAlerted(m *testMocks) {
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(true)
	m.emailSvc.EXPECT().Send(mock.Anything, "old@example.com", mock.Anything, mock.Anything).Return(nil)
}

func expectValidCredentials(m *testMocks, req dto.LoginRequest) uuid.UUID {
	userID := uuid.New()
	m.userSvc.EXPECT().ValidateCredentials(mock.Anything, req.Username, req.Password).Return(&dto.UserResponse{ID: userID, Username: req.Username}, nil)

	return userID
}

func expectLoginAllowed(m *testMocks, req dto.LoginRequest) uuid.UUID {
	userID := expectValidCredentials(m, req)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingSessionDurationDays).Return(30)

	return userID
}

func expectLiveResetToken(m *testMocks) uuid.UUID {
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(8)
	m.resetRepo.EXPECT().GetByTokenHash(mock.Anything, hashResetToken("sometoken")).Return(&model.PasswordResetToken{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)

	return userID
}

func expectLiveVerificationToken(m *testMocks) uuid.UUID {
	userID := uuid.New()
	m.verifyRepo.EXPECT().GetByTokenHash(mock.Anything, hashResetToken("sometoken")).Return(&model.EmailVerificationToken{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)

	return userID
}

func TestForgotPassword_Rejections(t *testing.T) {
	cases := []struct {
		name     string
		enabled  bool
		username string
		found    *model.User
		want     error
	}{
		{name: "email disabled", enabled: false, username: "alice", want: ErrEmailDisabled},
		{name: "user not found", enabled: true, username: "ghost", found: nil, want: ErrUserNotFound},
		{name: "no email set", enabled: true, username: "alice", found: &model.User{ID: uuid.New(), Email: ""}, want: ErrNoEmailAddress},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.emailSvc.EXPECT().Enabled(mock.Anything).Return(tc.enabled)

			if tc.enabled {
				m.userRepo.EXPECT().GetByUsername(mock.Anything, tc.username).Return(tc.found, nil)
			}

			// when
			err := svc.ForgotPassword(context.Background(), tc.username)

			// then
			require.ErrorIs(t, err, tc.want)
			m.resetRepo.AssertNotCalled(t, "Issue", mock.Anything, mock.Anything)
		})
	}
}

func TestForgotPassword_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.emailSvc.EXPECT().Enabled(mock.Anything).Return(true)
	m.userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: userID, Email: "alice@example.com"}, nil)
	m.resetRepo.EXPECT().Issue(mock.Anything, mock.MatchedBy(func(reset spec.NewPasswordReset) bool {
		return reset.UserID == userID && reset.TokenHash != "" && !reset.ExpiresAt.IsZero()
	})).Return(nil)
	expectSiteEmailSent(m, "alice@example.com")

	// when
	err := svc.ForgotPassword(context.Background(), "alice")

	// then
	require.NoError(t, err)
}

func TestResetPassword_Rejections(t *testing.T) {
	cases := []struct {
		name     string
		token    string
		password string
		lookup   bool
		found    *model.PasswordResetToken
		want     error
	}{
		{name: "empty token", token: "", password: "newpassword123", want: ErrInvalidResetToken},
		{name: "password too short", token: "sometoken", password: "short", want: ErrPasswordTooShort},
		{name: "unknown token", token: "sometoken", password: "newpassword123", lookup: true, found: nil, want: ErrInvalidResetToken},
		{name: "expired token", token: "sometoken", password: "newpassword123", lookup: true, found: &model.PasswordResetToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Hour)}, want: ErrInvalidResetToken},
		{name: "already used token", token: "sometoken", password: "newpassword123", lookup: true, found: &model.PasswordResetToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour), UsedAt: new(time.Now().Add(-time.Minute))}, want: ErrInvalidResetToken},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			if tc.token != "" {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(8)
			}

			if tc.lookup {
				m.resetRepo.EXPECT().GetByTokenHash(mock.Anything, hashResetToken(tc.token)).Return(tc.found, nil)
			}

			// when
			err := svc.ResetPassword(context.Background(), tc.token, tc.password)

			// then
			require.ErrorIs(t, err, tc.want)
			m.userRepo.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything)
		})
	}
}

func TestResetPassword_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := expectLiveResetToken(m)
	m.userRepo.EXPECT().ResetPassword(mock.Anything, mock.MatchedBy(func(update spec.PasswordUpdate) bool {
		return update.UserID == userID &&
			update.TokenHash == hashResetToken("sometoken") &&
			bcrypt.CompareHashAndPassword([]byte(update.PasswordHash), []byte("newpassword123")) == nil
	})).Return(nil)

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "newpassword123")

	// then
	require.NoError(t, err)
}

func TestResetPassword_RepositoryErrorBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectLiveResetToken(m)
	m.userRepo.EXPECT().ResetPassword(mock.Anything, mock.Anything).Return(errors.New("mark reset token used: boom"))

	// when
	err := svc.ResetPassword(context.Background(), "sometoken", "newpassword123")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark reset token used")
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

func TestRegister_RegistrationGate(t *testing.T) {
	dbDown := errors.New("db down")

	cases := []struct {
		name       string
		regType    string
		inviteCode string
		invite     *model.Invite
		lookupErr  error
		want       error
		wantMsg    string
	}{
		{name: "closed registration is rejected", regType: "closed", want: ErrRegistrationDisabled},
		{name: "invite required but missing", regType: "invite", want: ErrInviteRequired},
		{name: "invite lookup error", regType: "invite", inviteCode: "code123", lookupErr: dbDown, want: dbDown, wantMsg: "check invite"},
		{name: "invite not found", regType: "invite", inviteCode: "code123", invite: nil, want: ErrInvalidInvite},
		{name: "invite already used", regType: "invite", inviteCode: "code123", invite: &model.Invite{Code: "code123", UsedBy: new(uuid.New())}, want: ErrInvalidInvite},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			req := validRegisterRequest()
			req.InviteCode = tc.inviteCode
			m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return(tc.regType)

			if tc.inviteCode != "" {
				m.inviteRepo.EXPECT().GetByCode(mock.Anything, tc.inviteCode).Return(tc.invite, tc.lookupErr)
			}

			// when
			resp, token, err := svc.Register(context.Background(), req)

			// then
			require.ErrorIs(t, err, tc.want)
			assert.ErrorContains(t, err, tc.wantMsg)
			assert.Nil(t, resp)
			assert.Empty(t, token)
		})
	}
}

func TestRegister_UsernameRejections(t *testing.T) {
	cases := []struct {
		name     string
		username string
		want     error
	}{
		{name: "too short", username: "ab", want: ErrInvalidUsername},
		{name: "too long", username: "a123456789012345678901234567890", want: ErrInvalidUsername},
		{name: "bad characters", username: "alice!", want: ErrInvalidUsername},
		{name: "spaces", username: "alice bob", want: ErrInvalidUsername},
		{name: "reserved pattern featherine", username: "featherine", want: user.ErrUsernameTaken},
		{name: "reserved pattern FAA_fan", username: "FAA_fan", want: user.ErrUsernameTaken},
		{name: "reserved pattern myauauroratheory", username: "myauauroratheory", want: user.ErrUsernameTaken},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("open")
			req := validRegisterRequest()
			req.Username = tc.username

			// when
			_, _, err := svc.Register(context.Background(), req)

			// then
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestRegister_ReservedRouteSegment(t *testing.T) {
	cases := []struct {
		name     string
		username string
	}{
		{name: "the live directory", username: "live"},
		{name: "a route with a live child", username: "games"},
		{name: "the profile prefix", username: "user"},
		{name: "a static mount", username: "uploads"},
		{name: "a hyphenated route", username: "game-board"},
		{name: "casing does not get past the guard", username: "LIVE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a site route segment offered as a username
			svc, m := newTestService(t)
			m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return("open")
			req := validRegisterRequest()
			req.Username = tc.username

			// when the account is registered
			_, _, err := svc.Register(context.Background(), req)

			// then it is refused, so /{username}/live can never be shadowed by a site route
			require.ErrorIs(t, err, ErrReservedUsername)
		})
	}
}

func TestRegister_Rejections(t *testing.T) {
	accountFailure := errors.New("db down")
	sessionFailure := errors.New("create session: boom")
	inviteFailure := errors.New("mark invite as used: boom")
	emailFilter := spec.UserEmailFilter{Email: "alice@example.com", ExcludeUserID: uuid.Nil}
	noArrange := func(*testMocks, dto.RegisterRequest) {}

	cases := []struct {
		name       string
		regType    string
		inviteCode string
		password   string
		email      string
		arrange    func(m *testMocks, req dto.RegisterRequest)
		want       error
		wantMsg    string
	}{
		{name: "password too short", regType: "open", password: "short", arrange: noArrange, want: ErrPasswordTooShort},
		{name: "invalid email", regType: "open", email: "not-an-email", arrange: noArrange, want: ErrInvalidEmail},
		{
			name:    "email taken",
			regType: "open",
			arrange: func(m *testMocks, _ dto.RegisterRequest) {
				m.userRepo.EXPECT().EmailInUse(mock.Anything, emailFilter).Return(true, nil)
			},
			want: ErrEmailTaken,
		},
		{
			name:    "username taken",
			regType: "open",
			arrange: func(m *testMocks, req dto.RegisterRequest) {
				m.userRepo.EXPECT().EmailInUse(mock.Anything, emailFilter).Return(false, nil)
				m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(user.ErrUsernameTaken)
			},
			want: user.ErrUsernameTaken,
		},
		{
			name:    "create user error",
			regType: "open",
			arrange: func(m *testMocks, req dto.RegisterRequest) {
				m.userRepo.EXPECT().EmailInUse(mock.Anything, emailFilter).Return(false, nil)
				m.userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, req.Username).Return(nil)
				m.userSvc.EXPECT().NewAccountSpec(mock.Anything, req.Username, "alice@example.com", req.Password, req.DisplayName).Return(spec.NewAccount{}, accountFailure)
			},
			want:    accountFailure,
			wantMsg: "create user",
		},
		{
			name:    "session create error from the registration write",
			regType: "open",
			arrange: func(m *testMocks, req dto.RegisterRequest) {
				account := expectAccountBuilt(m, req, req.DisplayName)
				m.userRepo.EXPECT().RegisterAccount(mock.Anything, matchesRegistration(account, "")).Return(nil, sessionFailure)
			},
			want:    sessionFailure,
			wantMsg: "create session",
		},
		{
			name:       "invite mark-used failure aborts registration",
			regType:    "invite",
			inviteCode: "code123",
			arrange: func(m *testMocks, req dto.RegisterRequest) {
				m.inviteRepo.EXPECT().GetByCode(mock.Anything, "code123").Return(&model.Invite{Code: "code123"}, nil)
				account := expectAccountBuilt(m, req, req.DisplayName)
				m.userRepo.EXPECT().RegisterAccount(mock.Anything, matchesRegistration(account, "code123")).Return(nil, inviteFailure)
			},
			want:    inviteFailure,
			wantMsg: "mark invite as used",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			req := validRegisterRequest()
			req.InviteCode = tc.inviteCode
			req.Password = cmp.Or(tc.password, req.Password)
			req.Email = cmp.Or(tc.email, req.Email)
			m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return(tc.regType)
			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(8)
			tc.arrange(m, req)

			// when
			resp, token, err := svc.Register(context.Background(), req)

			// then
			require.ErrorIs(t, err, tc.want)
			assert.ErrorContains(t, err, tc.wantMsg)
			assert.Nil(t, resp)
			assert.Empty(t, token)
			m.emailSvc.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestRegister_OK(t *testing.T) {
	cases := []struct {
		name            string
		regType         string
		inviteCode      string
		displayName     string
		password        string
		minLen          int
		wantDisplayName string
	}{
		{name: "open registration defaults the display name to the username", regType: "open", displayName: "", password: "password123", minLen: 8, wantDisplayName: "alice"},
		{name: "invite registration passes the code to the repository", regType: "invite", inviteCode: "code123", displayName: "Alice", password: "password123", minLen: 8, wantDisplayName: "Alice"},
		{name: "a zero minimum password length skips the check", regType: "open", displayName: "Alice", password: "x", minLen: 0, wantDisplayName: "Alice"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			req := validRegisterRequest()
			req.InviteCode = tc.inviteCode
			req.DisplayName = tc.displayName
			req.Password = tc.password
			m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingRegistrationType).Return(tc.regType)
			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(tc.minLen)

			if tc.inviteCode != "" {
				m.inviteRepo.EXPECT().GetByCode(mock.Anything, tc.inviteCode).Return(&model.Invite{Code: tc.inviteCode}, nil)
			}

			account := expectAccountBuilt(m, req, tc.wantDisplayName)

			var registered spec.NewRegistration
			m.userRepo.EXPECT().RegisterAccount(mock.Anything, matchesRegistration(account, tc.inviteCode)).
				Run(func(_ context.Context, registration spec.NewRegistration, _ ...*sql.Tx) {
					registered = registration
				}).
				Return(&model.User{ID: userID, Username: req.Username}, nil)
			expectSiteEmailSent(m, "alice@example.com")

			// when
			resp, token, err := svc.Register(context.Background(), req)

			// then
			require.NoError(t, err)
			assert.NotEmpty(t, token)
			assert.Equal(t, registered.SessionToken, token)
			assert.Equal(t, userID, resp.ID)
			m.inviteRepo.AssertNotCalled(t, "MarkUsed", mock.Anything, mock.Anything)
		})
	}
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
	userID := expectValidCredentials(m, req)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(true, nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionLoginBanned,
		TargetType: audit.TargetUser,
		TargetID:   userID.String(),
		Details:    "username=alice",
		SubjectID:  userID,
	}).Return(nil)

	// when
	_, _, err := svc.Login(context.Background(), req)

	// then
	require.ErrorIs(t, err, ErrUserBanned)
}

func TestLogin_SessionCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "password123"}
	userID := expectLoginAllowed(m, req)
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(created spec.NewSession) bool {
		return created.UserID == userID
	})).Return(errors.New("boom"))

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
	userID := expectLoginAllowed(m, req)

	var created spec.NewSession
	m.sessionRepo.EXPECT().Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, session spec.NewSession, _ ...*sql.Tx) {
			created = session
		}).
		Return(nil)

	// when
	resp, token, err := svc.Login(context.Background(), req)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, userID, resp.ID)
	assert.Equal(t, userID, created.UserID)
	assert.Equal(t, token, created.Token)
}

func TestLogin_BannedCheckErrorRefusesTheLogin(t *testing.T) {
	// given credentials that are valid but a ban lookup that fails
	svc, m := newTestService(t)
	req := dto.LoginRequest{Username: "alice", Password: "password123"}
	userID := expectValidCredentials(m, req)
	m.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, errors.New("db down"))

	// when
	_, token, err := svc.Login(context.Background(), req)

	// then no session is issued, because an unanswerable ban check is not an absent ban
	require.ErrorIs(t, err, ErrUserBanned)
	assert.Empty(t, token)
}

func TestLogout(t *testing.T) {
	deleteFailure := errors.New("boom")

	cases := []struct {
		name      string
		token     string
		deleteErr error
	}{
		{name: "empty token is a no-op", token: ""},
		{name: "deletes the session", token: "token123", deleteErr: nil},
		{name: "delete error bubbles", token: "token123", deleteErr: deleteFailure},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			if tc.token != "" {
				m.sessionRepo.EXPECT().Delete(mock.Anything, tc.token).Return(tc.deleteErr)
			}

			// when
			err := svc.Logout(context.Background(), tc.token)

			// then
			require.ErrorIs(t, err, tc.deleteErr)
		})
	}
}

func TestSetEmail_Rejections(t *testing.T) {
	pwHash := hashFor(t, "pw")

	cases := []struct {
		name           string
		email          string
		password       string
		checksPassword bool
		emailTaken     bool
		want           error
	}{
		{name: "invalid email", email: "nope", password: "pw", want: ErrInvalidEmail},
		{name: "email taken", email: "taken@example.com", password: "pw", checksPassword: true, emailTaken: true, want: ErrEmailTaken},
		{name: "wrong password is rejected", email: "new@example.com", password: "wrong", checksPassword: true, want: ErrIncorrectPassword},
		{name: "empty password is rejected", email: "new@example.com", password: "", checksPassword: true, want: ErrIncorrectPassword},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()

			if tc.checksPassword {
				m.userRepo.EXPECT().GetPasswordHash(mock.Anything, userID).Return(pwHash, nil)
			}

			if tc.emailTaken {
				m.userRepo.EXPECT().EmailInUse(mock.Anything, spec.UserEmailFilter{Email: tc.email, ExcludeUserID: userID}).Return(true, nil)
			}

			// when
			err := svc.SetEmail(context.Background(), userID, tc.email, tc.password)

			// then
			require.ErrorIs(t, err, tc.want)
			m.userRepo.AssertNotCalled(t, "SetEmail", mock.Anything, mock.Anything)
			m.emailSvc.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestSetEmail_OK(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		previousEmail string
		wantDetails   string
	}{
		{name: "a first address is normalised and audited on its own", input: "New@Example.com", previousEmail: "", wantDetails: "new@example.com"},
		{name: "a replaced address is audited as a change and the previous address is alerted", input: "new@example.com", previousEmail: "old@example.com", wantDetails: "old@example.com -> new@example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			m.userRepo.EXPECT().GetPasswordHash(mock.Anything, userID).Return(hashFor(t, "pw"), nil)
			expectEmailReplaced(m, userID, tc.previousEmail)
			m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
				ActorID:    userID,
				Action:     audit.ActionChangeEmail,
				TargetType: audit.TargetUser,
				TargetID:   userID.String(),
				Details:    tc.wantDetails,
				SubjectID:  userID,
			}).Return(nil)

			if tc.previousEmail != "" {
				expectPreviousAddressAlerted(m)
			}

			// when
			err := svc.SetEmail(context.Background(), userID, tc.input, "pw")

			// then
			require.NoError(t, err)
		})
	}
}

func TestSetEmailForUser_OK(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		previousEmail string
	}{
		{name: "a first address is normalised and needs no password", input: "  New@Example.com  ", previousEmail: ""},
		{name: "a replaced address alerts the previous address", input: "new@example.com", previousEmail: "old@example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			expectEmailReplaced(m, userID, tc.previousEmail)

			if tc.previousEmail != "" {
				expectPreviousAddressAlerted(m)
			}

			// when
			err := svc.SetEmailForUser(context.Background(), userID, tc.input)

			// then
			require.NoError(t, err)
			m.userRepo.AssertNotCalled(t, "GetPasswordHash", mock.Anything, mock.Anything)
		})
	}
}

func TestSetEmailForUser_Rejections(t *testing.T) {
	cases := []struct {
		name    string
		email   string
		arrange func(m *testMocks, userID uuid.UUID)
		want    error
	}{
		{
			name:    "invalid email",
			email:   "nope",
			arrange: func(*testMocks, uuid.UUID) {},
			want:    ErrInvalidEmail,
		},
		{
			name:  "email already in use",
			email: "taken@example.com",
			arrange: func(m *testMocks, userID uuid.UUID) {
				m.userRepo.EXPECT().EmailInUse(mock.Anything, spec.UserEmailFilter{Email: "taken@example.com", ExcludeUserID: userID}).Return(true, nil)
			},
			want: ErrEmailTaken,
		},
		{
			name:  "user gone",
			email: "new@example.com",
			arrange: func(m *testMocks, userID uuid.UUID) {
				m.userRepo.EXPECT().EmailInUse(mock.Anything, spec.UserEmailFilter{Email: "new@example.com", ExcludeUserID: userID}).Return(false, nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, nil)
			},
			want: ErrUserNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			tc.arrange(m, userID)

			// when
			err := svc.SetEmailForUser(context.Background(), userID, tc.email)

			// then
			require.ErrorIs(t, err, tc.want)
			m.userRepo.AssertNotCalled(t, "SetEmail", mock.Anything, mock.Anything)
		})
	}
}

func TestEmailVerificationState_Rejections(t *testing.T) {
	cases := []struct {
		name string
		call func(*service, context.Context, uuid.UUID) error
		user *model.User
		want error
	}{
		{name: "mark verified: user gone", call: (*service).MarkEmailVerified, user: nil, want: ErrUserNotFound},
		{name: "mark verified: no email set", call: (*service).MarkEmailVerified, user: &model.User{}, want: ErrNoEmailAddress},
		{name: "mark verified: already verified", call: (*service).MarkEmailVerified, user: &model.User{Email: "a@example.com", EmailVerified: true}, want: ErrEmailAlreadyVerified},
		{name: "mark unverified: user gone", call: (*service).MarkEmailUnverified, user: nil, want: ErrUserNotFound},
		{name: "mark unverified: no email set", call: (*service).MarkEmailUnverified, user: &model.User{}, want: ErrNoEmailAddress},
		{name: "mark unverified: not verified yet", call: (*service).MarkEmailUnverified, user: &model.User{Email: "a@example.com"}, want: ErrEmailNotVerified},
		{name: "resend: user gone", call: (*service).ResendVerification, user: nil, want: ErrUserNotFound},
		{name: "resend: no email set", call: (*service).ResendVerification, user: &model.User{Email: ""}, want: ErrNoEmailAddress},
		{name: "resend: already verified", call: (*service).ResendVerification, user: &model.User{Email: "a@example.com", EmailVerified: true}, want: ErrEmailAlreadyVerified},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(tc.user, nil)

			// when
			err := tc.call(svc, context.Background(), userID)

			// then
			require.ErrorIs(t, err, tc.want)
			m.userRepo.AssertNotCalled(t, "SetEmailVerified", mock.Anything, mock.Anything)
			m.verifyRepo.AssertNotCalled(t, "Issue", mock.Anything, mock.Anything)
			m.emailSvc.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMarkEmailVerification_OK(t *testing.T) {
	cases := []struct {
		name        string
		call        func(*service, context.Context, uuid.UUID) error
		wasVerified bool
	}{
		{name: "mark verified", call: (*service).MarkEmailVerified, wasVerified: false},
		{name: "mark unverified sends no email", call: (*service).MarkEmailUnverified, wasVerified: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Email: "a@example.com", EmailVerified: tc.wasVerified}, nil)
			m.userRepo.EXPECT().SetEmailVerified(mock.Anything, spec.UserEmailVerification{UserID: userID, Verified: !tc.wasVerified}).Return(nil)

			// when
			err := tc.call(svc, context.Background(), userID)

			// then
			require.NoError(t, err)
			m.emailSvc.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			m.verifyRepo.AssertNotCalled(t, "Issue", mock.Anything, mock.Anything)
		})
	}
}

func TestVerifyEmail_Rejections(t *testing.T) {
	cases := []struct {
		name  string
		token string
		found *model.EmailVerificationToken
	}{
		{name: "empty token", token: ""},
		{name: "unknown token", token: "sometoken", found: nil},
		{name: "expired token", token: "sometoken", found: &model.EmailVerificationToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Hour)}},
		{name: "already used token", token: "sometoken", found: &model.EmailVerificationToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour), UsedAt: new(time.Now().Add(-time.Minute))}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			if tc.token != "" {
				m.verifyRepo.EXPECT().GetByTokenHash(mock.Anything, hashResetToken(tc.token)).Return(tc.found, nil)
			}

			// when
			err := svc.VerifyEmail(context.Background(), tc.token)

			// then
			require.ErrorIs(t, err, ErrInvalidVerificationToken)
			m.userRepo.AssertNotCalled(t, "ConfirmEmailVerification", mock.Anything, mock.Anything)
		})
	}
}

func TestVerifyEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := expectLiveVerificationToken(m)
	m.userRepo.EXPECT().ConfirmEmailVerification(mock.Anything, spec.UserEmailConfirmation{UserID: userID, TokenHash: hashResetToken("sometoken")}).Return(nil)

	// when
	err := svc.VerifyEmail(context.Background(), "sometoken")

	// then
	require.NoError(t, err)
}

func TestVerifyEmail_TokenConsumptionFailureBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := expectLiveVerificationToken(m)
	m.userRepo.EXPECT().ConfirmEmailVerification(mock.Anything, spec.UserEmailConfirmation{UserID: userID, TokenHash: hashResetToken("sometoken")}).Return(errors.New("mark verification token used: boom"))

	// when
	err := svc.VerifyEmail(context.Background(), "sometoken")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark verification token used")
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
