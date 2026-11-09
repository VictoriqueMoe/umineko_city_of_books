package middleware

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

const (
	cspScriptSrc = "script-src 'self' blob: https://challenges.cloudflare.com"
)

var (
	cspDirectives = []string{
		"default-src 'self'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"object-src 'none'",
		"worker-src 'self' blob:",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data: blob: https:",
		"media-src 'self' blob: https:",
		"frame-src 'self' https://challenges.cloudflare.com https://www.youtube-nocookie.com https://www.youtube.com https://*.hyperbeam.com",
		"connect-src 'self' https: wss:",
	}

	contentSecurityPolicy = strings.Join(cspDirectives, "; ")
)

func SecurityHeaders() fiber.Handler {
	h := helmet.New(helmet.Config{
		XFrameOptions:             "DENY",
		ContentTypeNosniff:        "nosniff",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
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
		ctx.Set(fiber.HeaderContentSecurityPolicy, fmt.Sprintf("%s; %s 'nonce-%s'", contentSecurityPolicy, cspScriptSrc, rand.Text()))

		return err
	}
}
