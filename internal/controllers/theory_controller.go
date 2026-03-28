package controllers

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/theory/params"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllTheoryRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListTheoriesRoute,
		s.setupCreateTheoryRoute,
		s.setupGetTheoryRoute,
		s.setupUpdateTheoryRoute,
		s.setupDeleteTheoryRoute,
		s.setupVoteTheoryRoute,
		s.setupCreateResponseRoute,
		s.setupDeleteResponseRoute,
		s.setupVoteResponseRoute,
	}
}

func (s *Service) setupListTheoriesRoute(r fiber.Router) {
	r.Get("/theories", middleware.OptionalAuth(s.AuthSession), s.listTheories)
}

func (s *Service) setupCreateTheoryRoute(r fiber.Router) {
	r.Post("/theories", middleware.RequireAuth(s.AuthSession), s.createTheory)
}

func (s *Service) setupGetTheoryRoute(r fiber.Router) {
	r.Get("/theories/:id<int>", middleware.OptionalAuth(s.AuthSession), s.getTheory)
}

func (s *Service) setupUpdateTheoryRoute(r fiber.Router) {
	r.Put("/theories/:id<int>", middleware.RequireAuth(s.AuthSession), s.updateTheory)
}

func (s *Service) setupDeleteTheoryRoute(r fiber.Router) {
	r.Delete("/theories/:id<int>", middleware.RequireAuth(s.AuthSession), s.deleteTheory)
}

func (s *Service) setupVoteTheoryRoute(r fiber.Router) {
	r.Post("/theories/:id<int>/vote", middleware.RequireAuth(s.AuthSession), s.voteTheory)
}

func (s *Service) setupCreateResponseRoute(r fiber.Router) {
	r.Post("/theories/:id<int>/responses", middleware.RequireAuth(s.AuthSession), s.createResponse)
}

func (s *Service) setupDeleteResponseRoute(r fiber.Router) {
	r.Delete("/responses/:id<int>", middleware.RequireAuth(s.AuthSession), s.deleteResponse)
}

func (s *Service) setupVoteResponseRoute(r fiber.Router) {
	r.Post("/responses/:id<int>/vote", middleware.RequireAuth(s.AuthSession), s.voteResponse)
}

func (s *Service) listTheories(ctx fiber.Ctx) error {
	sort := ctx.Query("sort", "new")
	episode := fiber.Query[int](ctx, "episode", 0)
	authorID := fiber.Query[int](ctx, "author", 0)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)
	userID, _ := ctx.Locals("userID").(int)

	p := params.NewListParams(sort, episode, authorID, limit, offset)
	result, err := s.TheoryService.ListTheories(ctx.Context(), p, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list theories",
		})
	}

	return ctx.JSON(result)
}

func (s *Service) createTheory(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(int)

	var req dto.CreateTheoryRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Title == "" || req.Body == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "title and body are required",
		})
	}

	id, err := s.TheoryService.CreateTheory(ctx.Context(), userID, req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create theory",
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) getTheory(ctx fiber.Ctx) error {
	id := fiber.Params[int](ctx, "id")
	userID, _ := ctx.Locals("userID").(int)

	result, err := s.TheoryService.GetTheoryDetail(ctx.Context(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get theory",
		})
	}
	if result == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "theory not found",
		})
	}

	return ctx.JSON(result)
}

func (s *Service) updateTheory(ctx fiber.Ctx) error {
	id := fiber.Params[int](ctx, "id")
	userID := ctx.Locals("userID").(int)

	var req dto.CreateTheoryRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := s.TheoryService.UpdateTheory(ctx.Context(), id, userID, req.Title, req.Body, req.Episode); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "cannot update this theory",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) deleteTheory(ctx fiber.Ctx) error {
	id := fiber.Params[int](ctx, "id")
	userID := ctx.Locals("userID").(int)

	if err := s.TheoryService.DeleteTheory(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "cannot delete this theory",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) voteTheory(ctx fiber.Ctx) error {
	return s.vote(ctx, s.TheoryService.VoteTheory)
}

func (s *Service) vote(ctx fiber.Ctx, voteFunc func(context.Context, int, int, int) error) error {
	id := fiber.Params[int](ctx, "id")
	userID := ctx.Locals("userID").(int)

	var req dto.VoteRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Value != 1 && req.Value != -1 && req.Value != 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "value must be 1, -1, or 0",
		})
	}

	if err := voteFunc(ctx.Context(), userID, id, req.Value); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to vote",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) createResponse(ctx fiber.Ctx) error {
	theoryID := fiber.Params[int](ctx, "id")
	userID := ctx.Locals("userID").(int)

	var req dto.CreateResponseRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Side != "with_love" && req.Side != "without_love" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "side must be 'with_love' or 'without_love'",
		})
	}

	if req.Body == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "body is required",
		})
	}

	id, err := s.TheoryService.CreateResponse(ctx.Context(), theoryID, userID, req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create response",
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) deleteResponse(ctx fiber.Ctx) error {
	id := fiber.Params[int](ctx, "id")
	userID := ctx.Locals("userID").(int)

	if err := s.TheoryService.DeleteResponse(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "cannot delete this response",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) voteResponse(ctx fiber.Ctx) error {
	return s.vote(ctx, s.TheoryService.VoteResponse)
}
