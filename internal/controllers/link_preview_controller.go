package controllers

import (
	"errors"
	"time"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/linkpreview"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

const (
	linkPreviewsPerMinute = 120
)

func (s *Service) getAllLinkPreviewRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupLinkPreviewRoute,
	}
}

func (s *Service) setupLinkPreviewRoute(r fiber.Router) {
	r.Get("/link-preview", s.optionalAuth(), limiter.New(limiter.Config{
		Max:        linkPreviewsPerMinute,
		Expiration: time.Minute,
	}), s.linkPreview)
}

func (s *Service) linkPreview(ctx fiber.Ctx) error {
	rawURL := ctx.Query("url")

	preview, err := s.LinkPreviewService.Resolve(ctx.Context(), rawURL)
	if err != nil {
		if errors.Is(err, linkpreview.ErrInvalidURL) {
			return utils.BadRequest(ctx, "url must be an absolute http or https url")
		}

		return utils.InternalError(ctx, "failed to resolve link preview", err)
	}

	return ctx.JSON(preview)
}
