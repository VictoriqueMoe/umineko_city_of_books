package middleware

import (
	"context"
	"strconv"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rs/zerolog"
)

func Setup(app *fiber.App, settingsSvc settings.Service, sessionMgr *session.Manager, authzSvc authz.Service) {
	app.Server().MaxRequestBodySize = settingsSvc.GetInt(context.Background(), config.SettingMaxBodySize)

	app.Use(etag.New())

	app.Use(func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if err := ctx.Next(); err != nil {
			return err
		}
		switch {
		case strings.HasPrefix(path, "/static/assets/") || strings.HasPrefix(path, "/assets/"):
			ctx.Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(path, "/uploads/"):
			ctx.Set("Cache-Control", "public, max-age=2592000")
		case strings.HasPrefix(path, "/api"):
			ctx.Set("Cache-Control", "no-cache, must-revalidate")
		default:
			ctx.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			ctx.Set("Pragma", "no-cache")
			ctx.Set("Expires", "0")
		}
		return nil
	})

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOriginsFunc: func(origin string) bool {
			allowed := settingsSvc.Get(context.Background(), config.SettingBaseURL)
			return origin == allowed
		},
	}))

	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path} ${queryParams}\n",
		TimeFormat: "2006-01-02 15:04:05",
		Next: func(ctx fiber.Ctx) bool {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				return false
			}
			path := ctx.Path()
			return strings.HasPrefix(path, "/uploads/") ||
				strings.HasPrefix(path, "/static/assets/") ||
				strings.HasPrefix(path, "/assets/") ||
				strings.HasPrefix(path, "/favicon")
		},
	}))

	app.Use(maintenanceMiddleware(settingsSvc, sessionMgr, authzSvc))
}

func maintenanceMiddleware(settingsSvc settings.Service, sessionMgr *session.Manager, authzSvc authz.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if !settingsSvc.GetBool(ctx.Context(), config.SettingMaintenanceMode) {
			return ctx.Next()
		}

		path := ctx.Path()

		if !strings.HasPrefix(path, "/api") {
			return ctx.Next()
		}

		if path == "/api/v1/site-info" || path == "/api/v1/auth/login" || path == "/api/v1/auth/me" || path == "/api/v1/ws" {
			return ctx.Next()
		}

		cookie := ctx.Cookies(session.CookieName)
		if cookie != "" {
			if userID, err := sessionMgr.Validate(ctx.Context(), cookie); err == nil {
				if authzSvc.Can(ctx.Context(), userID, authz.PermManageSettings) {
					return ctx.Next()
				}
			}
		}

		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "site is under maintenance",
		})
	}
}

type BodyLimitListener struct {
	app *fiber.App
}

func NewBodyLimitListener(app *fiber.App) *BodyLimitListener {
	return &BodyLimitListener{app: app}
}

func (l *BodyLimitListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingMaxBodySize.Key {
		return
	}

	size, err := strconv.Atoi(value)
	if err != nil || size <= 0 {
		return
	}

	l.app.Server().MaxRequestBodySize = size
}
