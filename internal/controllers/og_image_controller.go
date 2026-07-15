package controllers

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
)

type (
	OGImageHandler struct {
		uploadDir string
		settings  settings.Service
	}
)

func NewOGImageHandler(uploadDir string, settingsService settings.Service) *OGImageHandler {
	return &OGImageHandler{uploadDir: uploadDir, settings: settingsService}
}

func (s *Service) getAllOGImageRoutes() []FSetupRoute {
	return []FSetupRoute{
		NewOGImageHandler(s.UploadService.GetUploadDir(), s.SettingsService).Register,
	}
}

func (h *OGImageHandler) Register(app fiber.Router) {
	app.Get("/og-image/*", h.serve)
}

func (h *OGImageHandler) serve(ctx fiber.Ctx) error {
	rel := ctx.Params("*")
	if !strings.HasSuffix(strings.ToLower(rel), ".jpg") {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	webpRel := rel[:len(rel)-len(".jpg")] + ".webp"
	clean := path.Clean("/" + webpRel)
	fullPath := filepath.Join(h.uploadDir, filepath.FromSlash(clean))

	if _, err := os.Stat(fullPath); err != nil {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	maxPixels := h.settings.GetInt(ctx.Context(), config.SettingMaxImagePixels)

	data, err := media.WebPToJPEG(ctx.Context(), fullPath, maxPixels)
	if err != nil {
		logger.Log.Warn().Err(err).Str("path", fullPath).Msg("og image conversion failed, serving original webp")
		return ctx.SendFile(fullPath)
	}

	return ctx.Type("jpg").Send(data)
}
