package middleware

import (
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
)

func CacheHeaders(settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if err := ctx.Next(); err != nil {
			return err
		}

		if ctx.Response().StatusCode() >= 400 {
			ctx.Set("Cache-Control", "private, no-store")

			return nil
		}

		switch {
		case strings.HasPrefix(path, "/static/assets/") || strings.HasPrefix(path, "/assets/"):
			ctx.Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/og-image/"):
			if settingsSvc.GetBool(ctx.Context(), config.SettingPrivateMode) {
				ctx.Set("Cache-Control", "private, max-age=2592000")
			} else {
				ctx.Set("Cache-Control", "public, max-age=2592000")
			}
		case strings.HasPrefix(path, "/hls/"):
			if strings.HasSuffix(path, ".m3u8") {
				ctx.Set("Cache-Control", "no-cache")
				ctx.Set("Content-Type", "application/vnd.apple.mpegurl")
			} else {
				ctx.Set("Cache-Control", "public, max-age=31536000, immutable")
				if strings.HasSuffix(path, ".ts") {
					ctx.Set("Content-Type", "video/mp2t")
				}
			}
		case strings.HasPrefix(path, "/characters/") || strings.HasPrefix(path, "/sounds/") || strings.HasPrefix(path, "/favicon/"):
			ctx.Set("Cache-Control", "public, max-age=2592000")
		case strings.HasPrefix(path, "/api"):
			ctx.Set("Cache-Control", "no-cache, must-revalidate")
		default:
			ctx.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			ctx.Set("Pragma", "no-cache")
			ctx.Set("Expires", "0")
		}

		return nil
	}
}
