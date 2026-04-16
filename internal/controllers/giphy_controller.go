package controllers

import (
	"errors"
	"strings"
	"time"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/giphy"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"
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
		s.setupGiphyFavouritesListRoute,
		s.setupGiphyFavouritesAddRoute,
		s.setupGiphyFavouritesRemoveRoute,
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

func (s *Service) setupGiphyFavouritesListRoute(r fiber.Router) {
	r.Get("/giphy/favourites", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.giphyFavouritesList)
}

func (s *Service) setupGiphyFavouritesAddRoute(r fiber.Router) {
	r.Post("/giphy/favourites", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.giphyFavouritesAdd)
}

func (s *Service) setupGiphyFavouritesRemoveRoute(r fiber.Router) {
	r.Delete("/giphy/favourites/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.giphyFavouritesRemove)
}

func (s *Service) giphyFavouritesList(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	offset := fiber.Query[int](ctx, "offset", 0)
	limit := fiber.Query[int](ctx, "limit", 50)

	favs, total, err := s.GiphyFavouriteService.List(ctx.Context(), userID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to load favourites")
	}
	return ctx.JSON(fiber.Map{
		"data":  favs,
		"total": total,
	})
}

func (s *Service) giphyFavouritesAdd(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req giphyfavourite.Favourite
	if err := ctx.Bind().Body(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	req.GiphyID = strings.TrimSpace(req.GiphyID)
	req.URL = strings.TrimSpace(req.URL)
	if req.GiphyID == "" || req.URL == "" {
		return utils.BadRequest(ctx, "giphy_id and url are required")
	}
	if err := s.GiphyFavouriteService.Add(ctx.Context(), userID, req); err != nil {
		return utils.InternalError(ctx, "failed to add favourite")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) giphyFavouritesRemove(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	giphyID := strings.TrimSpace(ctx.Params("id"))
	if giphyID == "" {
		return utils.BadRequest(ctx, "id is required")
	}
	if err := s.GiphyFavouriteService.Remove(ctx.Context(), userID, giphyID); err != nil {
		return utils.InternalError(ctx, "failed to remove favourite")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
