package controllers

import (
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/session"
	usersvc "umineko_city_of_books/internal/user"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllAuthRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupRegisterRoute,
		s.setupLoginRoute,
		s.setupLogoutRoute,
		s.setupGetMeRoute,
		s.setupSiteInfoRoute,
	}
}

func (s *Service) setupRegisterRoute(r fiber.Router) {
	r.Post("/auth/register", s.register)
}

func (s *Service) setupLoginRoute(r fiber.Router) {
	r.Post("/auth/login", s.login)
}

func (s *Service) setupLogoutRoute(r fiber.Router) {
	r.Post("/auth/logout", s.logout)
}

func (s *Service) setupGetMeRoute(r fiber.Router) {
	r.Get("/auth/me", middleware.RequireAuth(s.AuthSession), s.getMe)
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
	var req dto.RegisterRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if err := validateCredentials(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	user, token, err := s.AuthService.Register(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, auth.ErrRegistrationDisabled) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, auth.ErrInviteRequired) || errors.Is(err, auth.ErrInvalidInvite) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, auth.ErrPasswordTooShort) {
			minLen := s.SettingsService.GetInt(ctx.Context(), config.SettingMinPasswordLength)
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("password must be at least %d characters", minLen),
			})
		}
		if errors.Is(err, usersvc.ErrUsernameTaken) {
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "username already taken",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to register",
		})
	}

	s.setSessionCookie(ctx, token)
	return ctx.Status(fiber.StatusCreated).JSON(user)
}

func (s *Service) login(ctx fiber.Ctx) error {
	var req dto.LoginRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if err := validateCredentials(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	user, token, err := s.AuthService.Login(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, usersvc.ErrInvalidCredentials) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid username or password",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to login",
		})
	}

	s.setSessionCookie(ctx, token)
	return ctx.JSON(user)
}

func (s *Service) logout(ctx fiber.Ctx) error {
	cookie := ctx.Cookies(session.CookieName)
	if err := s.AuthService.Logout(ctx.Context(), cookie); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to logout",
		})
	}
	s.clearSessionCookie(ctx)
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) getMe(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	user, err := s.AuthService.GetMe(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	return ctx.JSON(user)
}

func (s *Service) setupSiteInfoRoute(r fiber.Router) {
	r.Get("/site-info", s.siteInfo)
}

func (s *Service) siteInfo(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{
		"site_name":           s.SettingsService.Get(ctx.Context(), config.SettingSiteName),
		"site_description":    s.SettingsService.Get(ctx.Context(), config.SettingSiteDescription),
		"registration_type":   s.SettingsService.Get(ctx.Context(), config.SettingRegistrationType),
		"announcement_banner": s.SettingsService.Get(ctx.Context(), config.SettingAnnouncementBanner),
		"default_theme":       s.SettingsService.Get(ctx.Context(), config.SettingDefaultTheme),
		"maintenance_mode":    s.SettingsService.GetBool(ctx.Context(), config.SettingMaintenanceMode),
	})
}
