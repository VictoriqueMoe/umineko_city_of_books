package middleware

import (
	"encoding/json"
	"net/http"
	"net/url"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
)

func RequireTurnstile(settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if !settingsSvc.GetBool(ctx.Context(), config.SettingTurnstileEnabled) {
			return ctx.Next()
		}

		secretKey := settingsSvc.Get(ctx.Context(), config.SettingTurnstileSecretKey)
		if secretKey == "" {
			return ctx.Next()
		}

		var partial struct {
			TurnstileToken string `json:"turnstile_token"`
		}
		if err := json.Unmarshal(ctx.Body(), &partial); err != nil || partial.TurnstileToken == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification required",
			})
		}

		resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
			url.Values{
				"secret":   {secretKey},
				"response": {partial.TurnstileToken},
			},
		)
		if err != nil {
			logger.Log.Error().Err(err).Msg("turnstile verification request failed")
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification failed",
			})
		}
		defer resp.Body.Close()

		var result struct {
			Success bool `json:"success"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			logger.Log.Error().Err(err).Msg("turnstile response decode failed")
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification failed",
			})
		}

		if !result.Success {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification failed, please try again",
			})
		}

		return ctx.Next()
	}
}
