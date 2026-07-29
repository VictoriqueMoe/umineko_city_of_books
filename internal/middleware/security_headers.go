package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

var (
	cspEnforcedDirectives = []string{
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"object-src 'none'",
	}

	cspReportOnlyDirectives = []string{
		"default-src 'self'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"object-src 'none'",
		"script-src 'self' blob: https://challenges.cloudflare.com",
		"worker-src 'self' blob:",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data: blob: https:",
		"media-src 'self' blob: https://quotes.auaurora.moe https://waifuvault.moe https://*.waifuvault.moe",
		"frame-src 'self' https://challenges.cloudflare.com https://www.youtube-nocookie.com https://www.youtube.com https://*.hyperbeam.com",
		"connect-src 'self' https: wss:",
	}

	contentSecurityPolicy           = strings.Join(cspEnforcedDirectives, "; ")
	contentSecurityPolicyReportOnly = strings.Join(cspReportOnlyDirectives, "; ")
)

func SecurityHeaders() fiber.Handler {
	h := helmet.New(helmet.Config{
		XFrameOptions:             "DENY",
		ContentTypeNosniff:        "nosniff",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		ContentSecurityPolicy:     contentSecurityPolicy,
		PermissionPolicy:          "geolocation=(), camera=(), microphone=(self), display-capture=(self)",
		CrossOriginEmbedderPolicy: "unsafe-none",
		CrossOriginOpenerPolicy:   "same-origin-allow-popups",
		CrossOriginResourcePolicy: "cross-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
		XSSProtection:             "0",
		HSTSMaxAge:                15552000,
		HSTSPreloadEnabled:        true,
	})

	return func(ctx fiber.Ctx) error {
		err := h(ctx)

		ctx.Response().Header.Del("Server")
		ctx.Set(fiber.HeaderContentSecurityPolicyReportOnly, contentSecurityPolicyReportOnly)

		return err
	}
}
