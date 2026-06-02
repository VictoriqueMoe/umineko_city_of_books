package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
)

type (
	Service interface {
		Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, string, error)
		Login(ctx context.Context, req dto.LoginRequest) (*dto.UserResponse, string, error)
		Logout(ctx context.Context, token string) error
		ForgotPassword(ctx context.Context, username string) error
		ResetPassword(ctx context.Context, token, newPassword string) error
		EmailEnabled(ctx context.Context) bool
		SetEmail(ctx context.Context, userID uuid.UUID, email string) error
		VerifyEmail(ctx context.Context, token string) error
		ResendVerification(ctx context.Context, userID uuid.UUID) error
	}

	service struct {
		userService   user.Service
		session       *session.Manager
		settingsSvc   settings.Service
		inviteRepo    repository.InviteRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		resetRepo     repository.PasswordResetRepository
		verifyRepo    repository.EmailVerificationRepository
		emailSvc      email.Service
		contentFilter *contentfilter.Manager
	}
)

const (
	resetTokenTTL  = time.Hour
	verifyTokenTTL = 24 * time.Hour
)

var (
	validUsername    = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	reservedPatterns = []string{"featherine", "faa", "auaurora"}
)

func isReservedUsername(username string) bool {
	lower := strings.ToLower(username)
	for _, pattern := range reservedPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func NewService(userService user.Service, sessionMgr *session.Manager, settingsSvc settings.Service, inviteRepo repository.InviteRepository, userRepo repository.UserRepository, auditRepo repository.AuditLogRepository, resetRepo repository.PasswordResetRepository, verifyRepo repository.EmailVerificationRepository, emailSvc email.Service, contentFilter *contentfilter.Manager) Service {
	return &service{
		userService:   userService,
		session:       sessionMgr,
		settingsSvc:   settingsSvc,
		inviteRepo:    inviteRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		resetRepo:     resetRepo,
		verifyRepo:    verifyRepo,
		emailSvc:      emailSvc,
		contentFilter: contentFilter,
	}
}

func (s *service) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, string, error) {
	regType := s.settingsSvc.Get(ctx, config.SettingRegistrationType)

	switch regType {
	case "closed":
		return nil, "", ErrRegistrationDisabled
	case "invite":
		if req.InviteCode == "" {
			return nil, "", ErrInviteRequired
		}
		invite, err := s.inviteRepo.GetByCode(ctx, req.InviteCode)
		if err != nil {
			return nil, "", fmt.Errorf("check invite: %w", err)
		}
		if invite == nil || invite.UsedBy != nil {
			return nil, "", ErrInvalidInvite
		}
	}

	if !isValidUsername(req.Username) {
		return nil, "", ErrInvalidUsername
	}

	if isReservedUsername(req.Username) {
		return nil, "", user.ErrUsernameTaken
	}

	minLen := s.settingsSvc.GetInt(ctx, config.SettingMinPasswordLength)
	if minLen > 0 && len(req.Password) < minLen {
		return nil, "", ErrPasswordTooShort
	}

	email := normalizeEmail(req.Email)
	if !isValidEmail(email) {
		return nil, "", ErrInvalidEmail
	}

	inUse, err := s.userRepo.EmailInUse(ctx, email, uuid.Nil)
	if err != nil {
		return nil, "", fmt.Errorf("check email: %w", err)
	}
	if inUse {
		return nil, "", ErrEmailTaken
	}

	logger.Log.Debug().Str("username", req.Username).Msg("registering user")
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	if s.contentFilter != nil {
		if err := s.contentFilter.Check(ctx, req.Username, req.DisplayName); err != nil {
			return nil, "", err
		}
	}

	if err := s.userService.CheckUsernameAvailable(ctx, req.Username); err != nil {
		return nil, "", err
	}

	userResp, err := s.userService.Create(ctx, req.Username, email, req.Password, req.DisplayName)
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	s.sendVerification(ctx, userResp.ID, email)

	if s.auditRepo != nil {
		if err := s.auditRepo.Create(ctx, userResp.ID, "user_created", "user", userResp.ID.String(), fmt.Sprintf("username=%s", req.Username)); err != nil {
			logger.Log.Warn().Err(err).Str("username", req.Username).Msg("failed to write user_created audit log")
		}
	}

	if regType == "invite" {
		if err := s.inviteRepo.MarkUsed(ctx, req.InviteCode, userResp.ID); err != nil {
			logger.Log.Error().Err(err).Str("code", req.InviteCode).Msg("failed to mark invite as used")
		}
	}

	token, err := s.session.Create(ctx, userResp.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return userResp, token, nil
}

func (s *service) Login(ctx context.Context, req dto.LoginRequest) (*dto.UserResponse, string, error) {
	logger.Log.Debug().Str("username", req.Username).Msg("login attempt")
	userResp, err := s.userService.ValidateCredentials(ctx, req.Username, req.Password)
	if err != nil {
		return nil, "", err
	}

	banned, _ := s.userRepo.IsBanned(ctx, userResp.ID)
	if banned {
		return nil, "", ErrUserBanned
	}

	token, err := s.session.Create(ctx, userResp.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return userResp, token, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	if token != "" {
		return s.session.Delete(ctx, token)
	}
	return nil
}

func (s *service) EmailEnabled(ctx context.Context) bool {
	return s.emailSvc.Enabled(ctx)
}

func (s *service) ForgotPassword(ctx context.Context, username string) error {
	if !s.emailSvc.Enabled(ctx) {
		return ErrEmailDisabled
	}

	usr, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if usr == nil {
		return ErrUserNotFound
	}
	if usr.Email == "" {
		return ErrNoEmailAddress
	}

	raw, hash, err := generateResetToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	if err := s.resetRepo.DeleteUnusedForUser(ctx, usr.ID); err != nil {
		logger.Log.Warn().Err(err).Str("user_id", usr.ID.String()).Msg("failed to clear previous reset tokens")
	}

	expiresAt := time.Now().Add(resetTokenTTL)
	if err := s.resetRepo.Create(ctx, hash, usr.ID, expiresAt); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	baseURL := strings.TrimRight(s.settingsSvc.Get(ctx, config.SettingBaseURL), "/")
	siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
	link := fmt.Sprintf("%s/reset-password?token=%s", baseURL, raw)

	subject, body := notification.PasswordResetEmail(siteName, link)
	if err := s.emailSvc.Send(ctx, usr.Email, subject, body); err != nil {
		return fmt.Errorf("send reset email: %w", err)
	}

	logger.Log.Info().Str("user_id", usr.ID.String()).Msg("password reset email sent")
	return nil
}

func (s *service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token == "" {
		return ErrInvalidResetToken
	}

	minLen := s.settingsSvc.GetInt(ctx, config.SettingMinPasswordLength)
	if minLen > 0 && len(newPassword) < minLen {
		return ErrPasswordTooShort
	}

	hash := hashResetToken(token)
	rec, err := s.resetRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("get reset token: %w", err)
	}
	if rec == nil || rec.UsedAt != nil || time.Now().After(rec.ExpiresAt) {
		return ErrInvalidResetToken
	}

	if err := s.userRepo.SetPassword(ctx, rec.UserID, newPassword); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	if err := s.resetRepo.MarkUsed(ctx, hash); err != nil {
		logger.Log.Warn().Err(err).Msg("failed to mark reset token used")
	}

	if err := s.session.DeleteAllForUser(ctx, rec.UserID); err != nil {
		logger.Log.Warn().Err(err).Str("user_id", rec.UserID.String()).Msg("failed to invalidate sessions after password reset")
	}

	if s.auditRepo != nil {
		if err := s.auditRepo.Create(ctx, rec.UserID, "password_reset", "user", rec.UserID.String(), ""); err != nil {
			logger.Log.Warn().Err(err).Msg("failed to write password_reset audit log")
		}
	}

	return nil
}

func generateResetToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}

	raw = hex.EncodeToString(buf)
	return raw, hashResetToken(raw), nil
}

func hashResetToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *service) SetEmail(ctx context.Context, userID uuid.UUID, email string) error {
	email = normalizeEmail(email)
	if !isValidEmail(email) {
		return ErrInvalidEmail
	}

	inUse, err := s.userRepo.EmailInUse(ctx, email, userID)
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	if inUse {
		return ErrEmailTaken
	}

	if err := s.userRepo.SetEmail(ctx, userID, email); err != nil {
		return fmt.Errorf("set email: %w", err)
	}

	s.sendVerification(ctx, userID, email)
	return nil
}

func (s *service) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidVerificationToken
	}

	hash := hashResetToken(token)
	rec, err := s.verifyRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("get verification token: %w", err)
	}
	if rec == nil || rec.UsedAt != nil || time.Now().After(rec.ExpiresAt) {
		return ErrInvalidVerificationToken
	}

	if err := s.userRepo.MarkEmailVerified(ctx, rec.UserID); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}

	if err := s.verifyRepo.MarkUsed(ctx, hash); err != nil {
		logger.Log.Warn().Err(err).Msg("failed to mark verification token used")
	}

	return nil
}

func (s *service) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	usr, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if usr == nil {
		return ErrUserNotFound
	}
	if usr.Email == "" {
		return ErrNoEmailAddress
	}
	if usr.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	s.sendVerification(ctx, userID, usr.Email)
	return nil
}

func (s *service) sendVerification(ctx context.Context, userID uuid.UUID, email string) {
	raw, hash, err := generateResetToken()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to generate verification token")
		return
	}

	if err := s.verifyRepo.DeleteUnusedForUser(ctx, userID); err != nil {
		logger.Log.Warn().Err(err).Str("user_id", userID.String()).Msg("failed to clear previous verification tokens")
	}

	if err := s.verifyRepo.Create(ctx, hash, userID, time.Now().Add(verifyTokenTTL)); err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to store verification token")
		return
	}

	baseURL := strings.TrimRight(s.settingsSvc.Get(ctx, config.SettingBaseURL), "/")
	siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
	link := fmt.Sprintf("%s/verify-email?token=%s", baseURL, raw)

	subject, body := notification.VerificationEmail(siteName, link)
	if err := s.emailSvc.Send(ctx, email, subject, body); err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to send verification email")
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	return addr.Address == email
}

func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 30 {
		return false
	}
	return validUsername.MatchString(username)
}
