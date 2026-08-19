package middleware

import (
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func carriesUpload(ctx fiber.Ctx) bool {
	form, err := ctx.MultipartForm()
	if err != nil || form == nil {
		return false
	}

	return len(form.File) > 0
}

func RequireEstablishedAccount(authzSvc authz.Service, settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if authzSvc == nil || settingsSvc == nil {
			logger.Log.Error().Msg("new account upload gate is not wired, uploads are ungated")

			return ctx.Next()
		}

		if !carriesUpload(ctx) {
			return ctx.Next()
		}

		userID, ok := ctx.Locals("userID").(uuid.UUID)
		if !ok || userID == uuid.Nil {
			return ctx.Next()
		}

		if !authzSvc.IsRestrictedNewAccount(ctx.Context(), userID) {
			return ctx.Next()
		}

		hours := settingsSvc.GetInt(ctx.Context(), config.SettingNewAccountHours)

		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fmt.Sprintf("new accounts cannot post attachments for their first %d hours", hours),
		})
	}
}
