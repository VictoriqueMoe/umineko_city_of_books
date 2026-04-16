package controllers

import (
	"errors"
	"time"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/giphy"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func giphyError(ctx fiber.Ctx, err error, fallback string) error {
	if errors.Is(err, giphy.ErrDisabled) {
		return utils.NotFound(ctx, "gif search is not configured")
	}
	var rl *giphy.RateLimitError
	if errors.As(err, &rl) {
		return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":    "rate limited",
			"reset_at": rl.ResetAt.UTC().Format(time.RFC3339),
		})
	}
	return utils.InternalError(ctx, fallback)
}

func (s *Service) getAllGiphyRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupGiphySearchRoute,
		s.setupGiphyTrendingRoute,
	}
}

func (s *Service) setupGiphySearchRoute(r fiber.Router) {
	r.Get("/giphy/search", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.giphySearch)
}

func (s *Service) setupGiphyTrendingRoute(r fiber.Router) {
	r.Get("/giphy/trending", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.giphyTrending)
}

func (s *Service) giphySearch(ctx fiber.Ctx) error {
	q := ctx.Query("q")
	if q == "" {
		return utils.BadRequest(ctx, "query is required")
	}
	offset := fiber.Query[int](ctx, "offset", 0)
	limit := fiber.Query[int](ctx, "limit", 24)

	resp, err := s.GiphyService.Search(ctx.Context(), q, offset, limit)
	if err != nil {
		return giphyError(ctx, err, "gif search failed")
	}
	return ctx.JSON(resp)
}

func (s *Service) giphyTrending(ctx fiber.Ctx) error {
	offset := fiber.Query[int](ctx, "offset", 0)
	limit := fiber.Query[int](ctx, "limit", 24)

	resp, err := s.GiphyService.Trending(ctx.Context(), offset, limit)
	if err != nil {
		return giphyError(ctx, err, "gif trending failed")
	}
	return ctx.JSON(resp)
}
