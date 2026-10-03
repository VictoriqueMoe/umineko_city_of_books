package controllers

import (
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage"

	"github.com/gofiber/fiber/v3"
)

type (
	OGImageHandler struct {
		storage  storage.Service
		settings settings.Service
		images   *og.ImageService
	}
)

func NewOGImageHandler(storageSvc storage.Service, settingsService settings.Service, images *og.ImageService) *OGImageHandler {
	return &OGImageHandler{storage: storageSvc, settings: settingsService, images: images}
}

func (s *Service) getAllOGImageRoutes() []FSetupRoute {
	return []FSetupRoute{
		NewOGImageHandler(s.StorageService, s.SettingsService, s.OGImageService).Register,
	}
}

func (h *OGImageHandler) Register(app fiber.Router) {
	app.Get("/og-image/*", h.serve)
}

func (h *OGImageHandler) serve(ctx fiber.Ctx) error {
	rel := ctx.Params("*")
	if !strings.HasSuffix(strings.ToLower(rel), ".jpg") {
		return fiber.ErrNotFound
	}

	key := rel[:len(rel)-len(".jpg")] + ".webp"

	info, err := h.storage.Stat(ctx.Context(), key)
	if err != nil {
		return utils.StoredObjectError(ctx, err)
	}

	maxPixels := h.settings.GetInt(ctx.Context(), config.SettingMaxImagePixels)

	data, err := h.images.JPEG(ctx.Context(), info, maxPixels)
	if err != nil {
		logger.Ctx(ctx.Context()).Warn().Err(err).Str("key", key).Msg("og image conversion failed, serving original webp")

		return utils.SendStoredObject(ctx, h.storage, info)
	}

	return ctx.Type("jpg").Send(data)
}
