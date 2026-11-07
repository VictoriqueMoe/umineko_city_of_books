package middleware

import (
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	RestrictedUploadHoursLocal = "restrictedUploadHours"
)

func RequireEstablishedAccount(authzSvc authz.Service, settingsSvc settings.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if authzSvc == nil || settingsSvc == nil {
			logger.Ctx(ctx.Context()).Error().Msg("new account upload gate is not wired, uploads are ungated")

			return ctx.Next()
		}

		if !utils.IsMultipart(ctx) {
			return ctx.Next()
		}

		userID, ok := ctx.Locals("userID").(uuid.UUID)
		if !ok || userID == uuid.Nil {
			return ctx.Next()
		}

		if !authzSvc.IsRestrictedNewAccount(ctx.Context(), userID) {
			return ctx.Next()
		}

		ctx.Locals(RestrictedUploadHoursLocal, settingsSvc.GetInt(ctx.Context(), config.SettingNewAccountHours))

		return ctx.Next()
	}
}
