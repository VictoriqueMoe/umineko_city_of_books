package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/session"
	usersvc "umineko_city_of_books/internal/user"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var rulesSettings = map[string]*config.SiteSettingDef{
	"theories":             config.SettingRulesTheories,
	"theories_higurashi":   config.SettingRulesTheoriesHigurashi,
	"mysteries":            config.SettingRulesMysteries,
	"ships":                config.SettingRulesShips,
	"game_board":           config.SettingRulesGameBoard,
	"game_board_umineko":   config.SettingRulesGameBoardUmineko,
	"game_board_higurashi": config.SettingRulesGameBoardHigurashi,
	"game_board_ciconia":   config.SettingRulesGameBoardCiconia,
	"game_board_higanbana": config.SettingRulesGameBoardHiganbana,
	"game_board_roseguns":  config.SettingRulesGameBoardRoseguns,
	"gallery":              config.SettingRulesGallery,
	"gallery_umineko":      config.SettingRulesGalleryUmineko,
	"gallery_higurashi":    config.SettingRulesGalleryHigurashi,
	"gallery_ciconia":      config.SettingRulesGalleryCiconia,
	"fanfiction":           config.SettingRulesFanfiction,
	"journals":             config.SettingRulesJournals,
	"suggestions":          config.SettingRulesSuggestions,
	"chat_rooms":           config.SettingRulesChatRooms,
	"landing":              config.SettingRulesLanding,
}

func (s *Service) getAllAuthRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupRegisterRoute,
		s.setupLoginRoute,
		s.setupLogoutRoute,
		s.setupForgotPasswordRoute,
		s.setupResetPasswordRoute,
		s.setupSetEmailRoute,
		s.setupVerifyEmailRoute,
		s.setupResendVerificationRoute,
		s.setupSessionRoute,
		s.setupSiteInfoRoute,
		s.setupGetRulesRoute,
		s.setupStaffRoute,
	}
}

func (s *Service) setupRegisterRoute(r fiber.Router) {
	r.Post("/auth/register", middleware.RequireTurnstile(s.SettingsService), s.register)
}

func (s *Service) setupLoginRoute(r fiber.Router) {
	r.Post("/auth/login", middleware.RequireTurnstile(s.SettingsService), s.login)
}

func (s *Service) setupLogoutRoute(r fiber.Router) {
	r.Post("/auth/logout", s.logout)
}

func (s *Service) setupForgotPasswordRoute(r fiber.Router) {
	r.Post("/auth/forgot-password", middleware.RequireTurnstile(s.SettingsService), s.forgotPassword)
}

func (s *Service) setupResetPasswordRoute(r fiber.Router) {
	r.Post("/auth/reset-password", s.resetPassword)
}

func (s *Service) setupSetEmailRoute(r fiber.Router) {
	r.Post("/auth/set-email", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setEmail)
}

func (s *Service) setupVerifyEmailRoute(r fiber.Router) {
	r.Post("/auth/verify-email", s.verifyEmail)
}

func (s *Service) setupResendVerificationRoute(r fiber.Router) {
	r.Post("/auth/resend-verification", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.resendVerification)
}

func (s *Service) setupSessionRoute(r fiber.Router) {
	r.Get("/auth/session", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getSession)
}

func (s *Service) getSession(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	if userID == uuid.Nil {
		return ctx.JSON(fiber.Map{"authenticated": false})
	}

	user, err := s.UserService.GetByID(ctx.Context(), userID)
	if err != nil || user == nil {
		return ctx.JSON(fiber.Map{"authenticated": false})
	}

	return ctx.JSON(fiber.Map{"authenticated": true, "username": user.Username})
}

func (s *Service) setSessionCookie(ctx fiber.Ctx, token string) {
	days := s.SettingsService.GetInt(ctx.Context(), config.SettingSessionDurationDays)
	if days < 1 {
		days = 30
	}

	baseURL := s.SettingsService.Get(ctx.Context(), config.SettingBaseURL)
	secure := strings.HasPrefix(baseURL, "https://")
	ctx.Cookie(&fiber.Cookie{
		Name:     session.CookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		MaxAge:   days * 24 * 60 * 60,
		Path:     "/",
	})
}

func (s *Service) setAppSessionToken(ctx fiber.Ctx, token string) {
	if ctx.Get("X-Client-Platform") != "" {
		ctx.Set("X-Session-Token", token)
	}
}

func (s *Service) clearSessionCookie(ctx fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     session.CookieName,
		Value:    "",
		HTTPOnly: true,
		SameSite: "Lax",
		MaxAge:   -1,
		Path:     "/",
	})
}

func validateCredentials(creds dto.Credentials) error {
	if creds.GetUsername() == "" || creds.GetPassword() == "" {
		return fiber.NewError(fiber.StatusBadRequest, "username and password are required")
	}
	return nil
}

func (s *Service) register(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.RegisterRequest](ctx)
	if !ok {
		return nil
	}
	if err := validateCredentials(&req); err != nil {
		return utils.BadRequest(ctx, err.Error())
	}

	user, token, err := s.AuthService.Register(ctx.Context(), req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, auth.ErrInvalidUsername) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, auth.ErrRegistrationDisabled) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, auth.ErrInviteRequired) || errors.Is(err, auth.ErrInvalidInvite) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, auth.ErrInvalidEmail) {
			return utils.BadRequest(ctx, "a valid email address is required")
		}
		if errors.Is(err, auth.ErrEmailTaken) {
			return utils.Conflict(ctx, "that email address is already in use")
		}
		if errors.Is(err, auth.ErrPasswordTooShort) {
			minLen := s.SettingsService.GetInt(ctx.Context(), config.SettingMinPasswordLength)
			return utils.BadRequest(ctx, fmt.Sprintf("password must be at least %d characters", minLen))
		}
		if errors.Is(err, usersvc.ErrUsernameTaken) {
			return utils.Conflict(ctx, "username already taken")
		}
		return utils.InternalError(ctx, "failed to register")
	}

	ip, _ := ctx.Locals("client_ip").(string)
	go func() {
		_ = s.UserService.UpdateIP(context.Background(), user.ID, ip)
	}()

	s.setSessionCookie(ctx, token)
	s.setAppSessionToken(ctx, token)
	return ctx.Status(fiber.StatusCreated).JSON(user)
}

func (s *Service) login(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.LoginRequest](ctx)
	if !ok {
		return nil
	}
	if err := validateCredentials(&req); err != nil {
		return utils.BadRequest(ctx, err.Error())
	}

	user, token, err := s.AuthService.Login(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, usersvc.ErrInvalidCredentials) {
			return utils.Unauthorized(ctx, "invalid username or password")
		}
		if errors.Is(err, auth.ErrUserBanned) {
			return utils.Forbidden(ctx, "your account has been banned")
		}
		return utils.InternalError(ctx, "failed to login")
	}

	ip, _ := ctx.Locals("client_ip").(string)
	go func() {
		_ = s.UserService.UpdateIP(context.Background(), user.ID, ip)
	}()

	s.setSessionCookie(ctx, token)
	s.setAppSessionToken(ctx, token)
	return ctx.JSON(user)
}

func (s *Service) logout(ctx fiber.Ctx) error {
	token := ctx.Cookies(session.CookieName)
	if bearer := session.BearerToken(ctx.Get("Authorization")); bearer != "" {
		token = bearer
	}
	if err := s.AuthService.Logout(ctx.Context(), token); err != nil {
		return utils.InternalError(ctx, "failed to logout")
	}
	s.clearSessionCookie(ctx)
	return utils.OK(ctx)
}

func (s *Service) forgotPassword(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.ForgotPasswordRequest](ctx)
	if !ok {
		return nil
	}
	if req.Username == "" {
		return utils.BadRequest(ctx, "username is required")
	}

	err := s.AuthService.ForgotPassword(ctx.Context(), req.Username)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return utils.NotFound(ctx, "user not found")
		}
		if errors.Is(err, auth.ErrNoEmailAddress) {
			return utils.BadRequest(ctx, "user has no email set")
		}
		if errors.Is(err, auth.ErrEmailDisabled) {
			return utils.BadRequest(ctx, "password reset is not available")
		}
		return utils.InternalError(ctx, "failed to send reset email")
	}

	return utils.OK(ctx)
}

func (s *Service) resetPassword(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.ResetPasswordRequest](ctx)
	if !ok {
		return nil
	}
	if req.Token == "" || req.NewPassword == "" {
		return utils.BadRequest(ctx, "token and new password are required")
	}

	err := s.AuthService.ResetPassword(ctx.Context(), req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidResetToken) {
			return utils.BadRequest(ctx, "reset link is invalid or has expired")
		}
		if errors.Is(err, auth.ErrPasswordTooShort) {
			minLen := s.SettingsService.GetInt(ctx.Context(), config.SettingMinPasswordLength)
			return utils.BadRequest(ctx, fmt.Sprintf("password must be at least %d characters", minLen))
		}
		return utils.InternalError(ctx, "failed to reset password")
	}

	return utils.OK(ctx)
}

func (s *Service) setEmail(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.SetEmailRequest](ctx)
	if !ok {
		return nil
	}

	err := s.AuthService.SetEmail(ctx.Context(), userID, req.Email)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidEmail) {
			return utils.BadRequest(ctx, "a valid email address is required")
		}
		if errors.Is(err, auth.ErrEmailTaken) {
			return utils.Conflict(ctx, "that email address is already in use")
		}
		return utils.InternalError(ctx, "failed to set email")
	}

	return utils.OK(ctx)
}

func (s *Service) verifyEmail(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.VerifyEmailRequest](ctx)
	if !ok {
		return nil
	}
	if req.Token == "" {
		return utils.BadRequest(ctx, "token is required")
	}

	err := s.AuthService.VerifyEmail(ctx.Context(), req.Token)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidVerificationToken) {
			return utils.BadRequest(ctx, "verification link is invalid or has expired")
		}
		return utils.InternalError(ctx, "failed to verify email")
	}

	return utils.OK(ctx)
}

func (s *Service) resendVerification(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	err := s.AuthService.ResendVerification(ctx.Context(), userID)
	if err != nil {
		if errors.Is(err, auth.ErrNoEmailAddress) {
			return utils.BadRequest(ctx, "set an email address first")
		}
		if errors.Is(err, auth.ErrEmailAlreadyVerified) {
			return utils.BadRequest(ctx, "your email is already verified")
		}
		return utils.InternalError(ctx, "failed to resend verification email")
	}

	return utils.OK(ctx)
}

func (s *Service) setupSiteInfoRoute(r fiber.Router) {
	r.Get("/site-info", s.siteInfo)
}

func (s *Service) setupStaffRoute(r fiber.Router) {
	r.Get("/staff", s.staff)
}

func (s *Service) staff(ctx fiber.Ctx) error {
	staff, err := s.UserService.ListStaff(ctx.Context())
	if err != nil {
		return utils.InternalError(ctx, "failed to load staff")
	}
	return ctx.JSON(staff)
}

func (s *Service) siteInfo(ctx fiber.Ctx) error {
	return ctx.JSON(s.SiteInfoService.Get(ctx.Context()))
}

func (s *Service) setupGetRulesRoute(r fiber.Router) {
	r.Get("/rules/:page", s.getRules)
}

func (s *Service) getRules(ctx fiber.Ctx) error {
	page := ctx.Params("page")
	def, ok := rulesSettings[page]
	if !ok {
		return utils.NotFound(ctx, "unknown page")
	}

	return ctx.JSON(fiber.Map{
		"page":  page,
		"rules": s.SettingsService.Get(ctx.Context(), def),
	})
}
