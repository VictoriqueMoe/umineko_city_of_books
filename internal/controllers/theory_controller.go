package controllers

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/theory/params"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	r.Get("/theories", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listTheories)
}

func (s *Service) setupCreateTheoryRoute(r fiber.Router) {
	r.Post("/theories", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createTheory)
}

func (s *Service) setupGetTheoryRoute(r fiber.Router) {
	r.Get("/theories/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getTheory)
}

func (s *Service) setupUpdateTheoryRoute(r fiber.Router) {
	r.Put("/theories/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateTheory)
}

func (s *Service) setupDeleteTheoryRoute(r fiber.Router) {
	r.Delete("/theories/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteTheory)
}

func (s *Service) setupVoteTheoryRoute(r fiber.Router) {
	r.Post("/theories/:id/vote", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voteTheory)
}

func (s *Service) setupCreateResponseRoute(r fiber.Router) {
	r.Post("/theories/:id/responses", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createResponse)
}

func (s *Service) setupDeleteResponseRoute(r fiber.Router) {
	r.Delete("/responses/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteResponse)
}

func (s *Service) setupVoteResponseRoute(r fiber.Router) {
	r.Post("/responses/:id/vote", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voteResponse)
}

func (s *Service) listTheories(ctx fiber.Ctx) error {
	sort := ctx.Query("sort", "new")
	episode := fiber.Query[int](ctx, "episode", 0)
	authorIDStr := ctx.Query("author")
	search := ctx.Query("search")
	series := ctx.Query("series", "umineko")
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)
	userID := utils.UserID(ctx)

	var authorID uuid.UUID
	if authorIDStr != "" {
		parsed, err := uuid.Parse(authorIDStr)
		if err != nil {
			return utils.BadRequest(ctx, "invalid author ID")
		}
		authorID = parsed
	}

	p := params.NewListParams(sort, episode, authorID, search, series, limit, offset)
	result, err := s.TheoryService.ListTheories(ctx.Context(), p, userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list theories")
	}

	return ctx.JSON(result)
}

func (s *Service) createTheory(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateTheoryRequest](ctx)
	if !ok {
		return nil
	}

	if req.Title == "" || req.Body == "" {
		return utils.BadRequest(ctx, "title and body are required")
	}

	id, err := s.TheoryService.CreateTheory(ctx.Context(), userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, theory.ErrRateLimited) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "daily theory limit reached",
			})
		}
		return utils.InternalError(ctx, "failed to create theory")
	}

	if req.Series != "" {
		s.Hub.BumpSidebarActivity("theories_" + req.Series)
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) getTheory(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	result, err := s.TheoryService.GetTheoryDetail(ctx.Context(), id, userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to get theory")
	}
	if result == nil {
		return utils.NotFound(ctx, "theory not found")
	}

	return ctx.JSON(result)
}

func (s *Service) updateTheory(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateTheoryRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.TheoryService.UpdateTheory(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		return utils.Forbidden(ctx, "cannot update this theory")
	}

	return utils.OK(ctx)
}

func (s *Service) deleteTheory(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.TheoryService.DeleteTheory(ctx.Context(), id, userID); err != nil {
		return utils.Forbidden(ctx, "cannot delete this theory")
	}

	return utils.OK(ctx)
}

func (s *Service) voteTheory(ctx fiber.Ctx) error {
	return s.vote(ctx, s.TheoryService.VoteTheory)
}

func (s *Service) vote(ctx fiber.Ctx, voteFunc func(context.Context, uuid.UUID, uuid.UUID, int) error) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.VoteRequest](ctx)
	if !ok {
		return nil
	}

	if req.Value != 1 && req.Value != -1 && req.Value != 0 {
		return utils.BadRequest(ctx, "value must be 1, -1, or 0")
	}

	if err := voteFunc(ctx.Context(), userID, id, req.Value); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to vote")
	}

	return utils.OK(ctx)
}

func (s *Service) createResponse(ctx fiber.Ctx) error {
	theoryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateResponseRequest](ctx)
	if !ok {
		return nil
	}

	if req.Side != "with_love" && req.Side != "without_love" {
		return utils.BadRequest(ctx, "side must be 'with_love' or 'without_love'")
	}

	if req.Body == "" {
		return utils.BadRequest(ctx, "body is required")
	}

	id, err := s.TheoryService.CreateResponse(ctx.Context(), theoryID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		if errors.Is(err, theory.ErrCannotRespondToOwnTheory) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, theory.ErrRateLimited) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "daily response limit reached",
			})
		}
		return utils.InternalError(ctx, "failed to create response")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) deleteResponse(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.TheoryService.DeleteResponse(ctx.Context(), id, userID); err != nil {
		return utils.Forbidden(ctx, "cannot delete this response")
	}

	return utils.OK(ctx)
}

func (s *Service) voteResponse(ctx fiber.Ctx) error {
	return s.vote(ctx, s.TheoryService.VoteResponse)
}
