package middleware

import "github.com/gofiber/fiber/v3"

func SecurityHeaders() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if err := ctx.Next(); err != nil {
			return err
		}
		ctx.Set("X-Frame-Options", "DENY")
		ctx.Set("X-Content-Type-Options", "nosniff")
		ctx.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		ctx.Set("Content-Security-Policy", "frame-ancestors 'none'")
		ctx.Response().Header.Del("Server")
		return nil
	}
}
