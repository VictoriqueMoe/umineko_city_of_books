package middleware

import (
	"umineko_city_of_books/internal/config"
	appLogger "umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
)

func RejectForeignOrigins(settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if fiber.IsMethodSafe(ctx.Method()) {
			return ctx.Next()
		}

		origin := ctx.Get(fiber.HeaderOrigin)
		if origin == "" || ws.OriginAllowed(origin, settingsSvc.Get(ctx.Context(), config.SettingBaseURL)) {
			return ctx.Next()
		}

		appLogger.Ctx(ctx.Context()).Warn().Str("origin", origin).Str("path", ctx.Path()).Msg("write rejected: origin not allowed")

		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "origin not allowed",
		})
	}
}
