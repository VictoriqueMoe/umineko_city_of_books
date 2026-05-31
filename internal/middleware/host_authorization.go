package middleware

import (
	"context"
	"net/url"

	"umineko_city_of_books/internal/config"
	appLogger "umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/hostauthorization"
)

func HostAuthorization(settingsSvc settings.Service) fiber.Handler {
	return hostauthorization.New(hostauthorization.Config{
		Next: func(ctx fiber.Ctx) bool {
			return ctx.Path() == "/health" || ctx.Path() == "/api/v1/livekit/webhook"
		},
		AllowedHostsFunc: func(host string) bool {
			base := settingsSvc.Get(context.Background(), config.SettingBaseURL)

			parsed, err := url.Parse(base)
			if err != nil {
				return false
			}

			expected := parsed.Hostname()
			if expected == "" {
				return false
			}

			return host == expected
		},
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			appLogger.Log.Err(err).
				Str("client_ip", ctx.IP()).
				Str("host", ctx.Hostname()).
				Str("method", ctx.Method()).
				Str("path", ctx.Path()).
				Msg("hostauthorization: rejected request")

			return ctx.SendStatus(fiber.StatusForbidden)
		},
	})
}
