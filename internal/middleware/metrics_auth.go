package middleware

import (
	"crypto/subtle"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
)

func RequireMetricsToken(settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if settingsSvc == nil {
			return ctx.Next()
		}

		expected := settingsSvc.Get(ctx.Context(), config.SettingMetricsToken)
		if expected == "" {
			return ctx.Next()
		}

		presented := session.BearerToken(ctx.Get("Authorization"))
		if presented == "" {
			presented = ctx.Get("X-Metrics-Token")
		}

		if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}

		return ctx.Next()
	}
}
