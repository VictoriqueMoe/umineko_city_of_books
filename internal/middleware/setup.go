package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dronebl"
	appLogger "umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog"
)

var (
	redactedQueryKeys = map[string]bool{
		"token":        true,
		"access_token": true,
		"session":      true,
		"key":          true,
		"secret":       true,
		"password":     true,
	}
)

func Setup(app *fiber.App, settingsSvc settings.Service, sessionMgr *session.Manager, authzSvc authz.Service, droneblChecker *dronebl.Checker) {
	app.Server().MaxRequestBodySize = settingsSvc.GetInt(context.Background(), config.SettingMaxBodySize)

	app.Use(fiberRecover.New(fiberRecover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(ctx fiber.Ctx, e any) {
			appLogger.Ctx(ctx.Context()).Error().
				Err(fmt.Errorf("panic: %v", e)).
				Str("method", ctx.Method()).
				Str("path", ctx.Path()).
				Str("stack", string(debug.Stack())).
				Msg("recovered from panic in request handler")
		},
	}))

	app.Use(Tracing())
	app.Use(HostAuthorization(settingsSvc))
	app.Use(SecurityHeaders())
	app.Use(etag.New())

	app.Use(CacheHeaders(settingsSvc))

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Client-Platform"},
		ExposeHeaders:    []string{"X-Session-Token"},
		AllowOriginsFunc: func(origin string) bool {
			allowed := settingsSvc.Get(context.Background(), config.SettingBaseURL)
			return origin == allowed || config.IsAppOrigin(origin)
		},
	}))

	app.Use(RejectPlainTextBodies())

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

	app.Use(RequireCleanIP(droneblChecker, sessionMgr))

	app.Use(func(ctx fiber.Ctx) error {
		start := time.Now()

		err := ctx.Next()

		if shouldSkipRequestLog(ctx) {
			return err
		}

		latency := time.Since(start)
		status := ctx.Response().StatusCode()
		ip, _ := ctx.Locals("client_ip").(string)

		reqLogger := appLogger.Ctx(ctx.Context())

		event := reqLogger.Info()
		if status >= 500 {
			event = reqLogger.Error()
		} else if status >= 400 {
			event = reqLogger.Warn()
		}

		if traceID, _ := ctx.Locals("trace_id").(string); traceID != "" {
			event = event.Str("trace_id", traceID)
		}

		query := redactedQueryString(ctx)
		pathWithQuery := ctx.Path()
		if query != "" {
			pathWithQuery += "?" + query
		}

		event.Msgf("| %d | %14s | %s | %s %s", status, latency, ip, ctx.Method(), pathWithQuery)
		return err
	})

	app.Use(RequireLogin(settingsSvc, sessionMgr))
	app.Use(maintenanceMiddleware(settingsSvc, sessionMgr, authzSvc))
}

func redactedQueryString(ctx fiber.Ctx) string {
	var b strings.Builder
	for key, value := range ctx.RequestCtx().QueryArgs().All() {
		if b.Len() > 0 {
			b.WriteByte('&')
		}

		b.Write(key)
		b.WriteByte('=')
		if redactedQueryKeys[strings.ToLower(string(key))] {
			b.WriteString("<redacted>")
			continue
		}

		b.Write(value)
	}

	return b.String()
}

func shouldSkipRequestLog(ctx fiber.Ctx) bool {
	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		return false
	}
	path := ctx.Path()
	if strings.HasPrefix(path, "/uploads/") ||
		strings.HasPrefix(path, "/hls/") ||
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

		path := strings.ToLower(ctx.Path())

		if !strings.HasPrefix(path, "/api") {
			return ctx.Next()
		}

		if path == "/api/v1/site-info" || path == "/api/v1/auth/login" || path == "/api/v1/auth/session" || path == "/api/v1/ws" {
			return ctx.Next()
		}

		token := SessionToken(ctx)
		if token != "" {
			if userID, err := sessionMgr.Validate(ctx.Context(), token); err == nil {
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
