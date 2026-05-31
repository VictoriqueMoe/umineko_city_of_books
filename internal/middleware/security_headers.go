package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

func SecurityHeaders() fiber.Handler {
	h := helmet.New(helmet.Config{
		XFrameOptions:             "DENY",
		ContentTypeNosniff:        "nosniff",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		ContentSecurityPolicy:     "frame-ancestors 'none'",
		PermissionPolicy:          "geolocation=(), camera=(), microphone=(self)",
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
		return err
	}
}
