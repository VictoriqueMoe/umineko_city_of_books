package middleware

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
)

var (
	turnstileClient = &http.Client{Timeout: 5 * time.Second}
)

func RequireTurnstile(settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if !settingsSvc.GetBool(ctx.Context(), config.SettingTurnstileEnabled) {
			return ctx.Next()
		}

		secretKey := settingsSvc.Get(ctx.Context(), config.SettingTurnstileSecretKey)
		if secretKey == "" {
			logger.Ctx(ctx.Context()).Error().Msg("turnstile is enabled but no secret key is set, refusing rather than letting requests through unverified")

			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "verification is temporarily unavailable, please try again later",
			})
		}

		var partial struct {
			TurnstileToken string `json:"turnstile_token"`
		}
		if err := json.Unmarshal(ctx.Body(), &partial); err != nil || partial.TurnstileToken == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification required",
			})
		}

		resp, err := turnstileClient.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
			url.Values{
				"secret":   {secretKey},
				"response": {partial.TurnstileToken},
			},
		)
		if err != nil {
			logger.Ctx(ctx.Context()).Error().Err(err).Msg("turnstile verification request failed")
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification failed",
			})
		}
		defer resp.Body.Close()

		var result struct {
			Success    bool     `json:"success"`
			ErrorCodes []string `json:"error-codes"`
			Hostname   string   `json:"hostname"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			logger.Ctx(ctx.Context()).Error().Err(err).Int("status", resp.StatusCode).Msg("turnstile response decode failed")
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification failed",
			})
		}

		if !result.Success {
			logger.Ctx(ctx.Context()).Warn().
				Strs("error_codes", result.ErrorCodes).
				Str("hostname", result.Hostname).
				Str("path", ctx.Path()).
				Msg("turnstile verification rejected")

			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "verification failed, please try again",
			})
		}

		return ctx.Next()
	}
}
