package middleware

import (
	"context"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	appLogger "umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/rs/zerolog"
)

func httpStatusToSentry(status int) sentry.SpanStatus {
	switch {
	case status >= 200 && status < 300:
		return sentry.SpanStatusOK
	case status == 400:
		return sentry.SpanStatusInvalidArgument
	case status == 401:
		return sentry.SpanStatusUnauthenticated
	case status == 403:
		return sentry.SpanStatusPermissionDenied
	case status == 404:
		return sentry.SpanStatusNotFound
	case status == 409:
		return sentry.SpanStatusAlreadyExists
	case status == 429:
		return sentry.SpanStatusResourceExhausted
	case status == 499:
		return sentry.SpanStatusCanceled
	case status >= 500 && status < 600:
		return sentry.SpanStatusInternalError
	}
	return sentry.SpanStatusUnknown
}

func Setup(app *fiber.App, settingsSvc settings.Service, sessionMgr *session.Manager, authzSvc authz.Service) {
	app.Server().MaxRequestBodySize = settingsSvc.GetInt(context.Background(), config.SettingMaxBodySize)

	app.Use(Tracing())
	app.Use(SecurityHeaders())
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

	app.Use(func(ctx fiber.Ctx) error {
		ip := ctx.IP()
		if ip == "" {
			if addr := ctx.RequestCtx().RemoteAddr(); addr != nil {
				ip = addr.String()
			}
		}
		ctx.Locals("client_ip", ip)
		return ctx.Next()
	})

	app.Use(func(ctx fiber.Ctx) error {
		start := time.Now()

		tx := sentry.StartTransaction(
			ctx.Context(),
			ctx.Method()+" "+ctx.Path(),
			sentry.WithOpName("http.server"),
		)
		defer tx.Finish()

		err := ctx.Next()

		tx.Status = httpStatusToSentry(ctx.Response().StatusCode())
		tx.SetData("http.method", ctx.Method())
		tx.SetData("http.route", ctx.Path())
		tx.SetData("http.response.status_code", ctx.Response().StatusCode())

		if shouldSkipRequestLog(ctx) {
			return err
		}
		latency := time.Since(start)
		status := ctx.Response().StatusCode()
		ip, _ := ctx.Locals("client_ip").(string)

		event := appLogger.Log.Info()
		if status >= 500 {
			event = appLogger.Log.Error()
		} else if status >= 400 {
			event = appLogger.Log.Warn()
		}

		if traceID, _ := ctx.Locals("trace_id").(string); traceID != "" {
			event = event.Str("trace_id", traceID)
		}

		query := string(ctx.RequestCtx().QueryArgs().QueryString())
		pathWithQuery := ctx.Path()
		if query != "" {
			pathWithQuery += "?" + query
		}

		event.Msgf("| %d | %14s | %s | %s %s", status, latency, ip, ctx.Method(), pathWithQuery)
		return err
	})

	app.Use(maintenanceMiddleware(settingsSvc, sessionMgr, authzSvc))
}

func shouldSkipRequestLog(ctx fiber.Ctx) bool {
	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		return false
	}
	path := ctx.Path()
	if strings.HasPrefix(path, "/uploads/") ||
		strings.HasPrefix(path, "/static/assets/") ||
		strings.HasPrefix(path, "/assets/") ||
		strings.HasPrefix(path, "/favicon") {
		return true
	}
	return ctx.Method() == "GET" && ctx.Response().StatusCode() < 400
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

		if path == "/api/v1/site-info" || path == "/api/v1/auth/login" || path == "/api/v1/auth/session" || path == "/api/v1/ws" {
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
