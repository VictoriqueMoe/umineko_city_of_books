package controllers

import (
	"strings"

	"umineko_city_of_books/internal/bounds"
	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/search"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllSearchRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupSearch,
		s.setupQuickSearch,
	}
}

func (s *Service) setupSearch(r fiber.Router) {
	r.Get("/search", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.search)
}

func (s *Service) setupQuickSearch(r fiber.Router) {
	r.Get("/search/quick", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.quickSearch)
}

func (s *Service) search(ctx fiber.Ctx) error {
	query := strings.TrimSpace(ctx.Query("q"))
	if query == "" {
		return ctx.JSON(fiber.Map{
			"results": []fiber.Map{},
			"total":   0,
		})
	}

	page := bounds.NewPage(fiber.Query[int](ctx, "limit", 20), fiber.Query[int](ctx, "offset", 0))
	types := s.SearchService.ParseTypes(ctx.Query("types"))
	viewerID, _ := ctrlutils.OptionalUserID(ctx)
	roomID, _ := uuid.Parse(ctx.Query("room"))

	results, total, err := s.SearchService.Search(ctx.Context(), query, types, page, viewerID, roomID)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to search", err)
	}

	return ctx.JSON(fiber.Map{
		"results": searchResultsToResponse(results),
		"total":   total,
	})
}

func (s *Service) quickSearch(ctx fiber.Ctx) error {
	query := strings.TrimSpace(ctx.Query("q"))
	if query == "" {
		return ctx.JSON(fiber.Map{"results": []fiber.Map{}})
	}

	perType := fiber.Query[int](ctx, "perType", 3)
	viewerID, _ := ctrlutils.OptionalUserID(ctx)

	results, err := s.SearchService.QuickSearch(ctx.Context(), query, perType, viewerID)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to quick search", err)
	}

	return ctx.JSON(fiber.Map{"results": searchResultsToResponse(results)})
}

func searchResultsToResponse(results []search.Result) []fiber.Map {
	out := make([]fiber.Map, len(results))
	for i, r := range results {
		out[i] = fiber.Map{
			"type":         string(r.EntityType),
			"id":           r.ID,
			"parent_id":    r.ParentID,
			"parent_title": r.ParentTitle,
			"title":        r.Title,
			"snippet":      r.Snippet,
			"url":          r.URL,
			"author": fiber.Map{
				"id":           r.AuthorID,
				"username":     r.AuthorUsername,
				"display_name": r.AuthorDisplayName,
				"avatar_url":   r.AuthorAvatarURL,
			},
			"created_at": r.CreatedAt,
		}
	}
	return out
}
