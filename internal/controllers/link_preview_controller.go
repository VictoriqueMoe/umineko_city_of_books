package controllers

import (
	"errors"
	"strings"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/linkpreview"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllLinkPreviewRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupLinkPreviewRoute,
	}
}

func (s *Service) setupLinkPreviewRoute(r fiber.Router) {
	r.Get("/link-preview", s.optionalAuth(), s.RateLimiter.Limit(cache.RateLimitLinkPreview, config.SettingRateLimitLinkPreview, middleware.ByClientIP), s.linkPreview)
}

func (s *Service) linkPreview(ctx fiber.Ctx) error {
	rawURL := strings.Clone(ctx.Query("url"))

	preview, err := s.LinkPreviewService.Resolve(ctx.Context(), rawURL)
	if err != nil {
		if errors.Is(err, linkpreview.ErrInvalidURL) {
			return utils.BadRequest(ctx, "url must be an absolute http or https url")
		}

		if errors.Is(err, linkpreview.ErrBusy) {
			return utils.ServiceUnavailable(ctx, "too many link previews are being fetched, please try again shortly")
		}

		return utils.InternalError(ctx, "failed to resolve link preview", err)
	}

	return ctx.JSON(preview)
}
