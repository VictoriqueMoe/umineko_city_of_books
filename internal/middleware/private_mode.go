package middleware

import (
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
)

const PrivateGatedLocal = "private_gated"

var (
	privateModeExempt = map[string]bool{
		"/livez":  true,
		"/health": true,
	}

	privateModeSignIn = map[string]bool{
		"/api/v1/site-info":            true,
		"/api/v1/auth/session":         true,
		"/api/v1/auth/login":           true,
		"/api/v1/auth/logout":          true,
		"/api/v1/auth/register":        true,
		"/api/v1/auth/forgot-password": true,
		"/api/v1/auth/reset-password":  true,
		"/api/v1/auth/verify-email":    true,
	}
)

func RequireLogin(settingsSvc settings.Service, sessionMgr *session.Manager) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if privateModeExempt[path] {
			return ctx.Next()
		}

		if !settingsSvc.GetBool(ctx.Context(), config.SettingPrivateMode) {
			return ctx.Next()
		}

		if hasSession(ctx, sessionMgr) {
			return ctx.Next()
		}

		ctx.Locals(PrivateGatedLocal, true)

		if privateModeSignIn[path] || servesTheSignInPage(path) {
			return ctx.Next()
		}

		if strings.HasPrefix(path, "/api") {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "this site is private, please sign in",
			})
		}

		return ctx.SendStatus(fiber.StatusUnauthorized)
	}
}

func servesTheSignInPage(path string) bool {
	if strings.HasPrefix(path, "/uploads") || strings.HasPrefix(path, "/hls") || strings.HasPrefix(path, "/og-image") {
		return false
	}

	if strings.HasPrefix(path, "/sitemap") {
		return false
	}

	return !strings.HasPrefix(path, "/api")
}

func PrivateGated(ctx fiber.Ctx) bool {
	gated, _ := ctx.Locals(PrivateGatedLocal).(bool)

	return gated
}
