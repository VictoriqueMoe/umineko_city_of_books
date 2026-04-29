package controllers

import (
	"errors"

	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/sidebar"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllHomeRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupGetHomeActivity,
		s.setupGetSidebarActivity,
		s.setupGetSidebarLastVisited,
		s.setupMarkSidebarVisited,
	}
}

func (s *Service) setupGetHomeActivity(r fiber.Router) {
	r.Get("/home/activity", s.getHomeActivity)
}

func (s *Service) setupGetSidebarActivity(r fiber.Router) {
	r.Get("/sidebar/activity", s.getSidebarActivity)
}

func (s *Service) setupGetSidebarLastVisited(r fiber.Router) {
	r.Get("/sidebar/last-visited", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getSidebarLastVisited)
}

func (s *Service) setupMarkSidebarVisited(r fiber.Router) {
	r.Post("/sidebar/last-visited", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.markSidebarVisited)
}

func (s *Service) getSidebarLastVisited(ctx fiber.Ctx) error {
	userID := ctrlutils.UserID(ctx)
	resp, err := s.SidebarService.ListVisited(ctx.Context(), userID)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load sidebar last visited", err)
	}
	return ctx.JSON(resp)
}

func (s *Service) markSidebarVisited(ctx fiber.Ctx) error {
	userID := ctrlutils.UserID(ctx)
	var body dto.MarkSidebarVisitedRequest
	if err := ctx.Bind().JSON(&body); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}

	if err := s.SidebarService.MarkVisited(ctx.Context(), userID, body.Key); err != nil {
		switch {
		case errors.Is(err, sidebar.ErrEmptyKey):
			return ctrlutils.BadRequest(ctx, "key is required")
		case errors.Is(err, sidebar.ErrKeyTooLong):
			return ctrlutils.BadRequest(ctx, "key too long")
		}
		return ctrlutils.InternalError(ctx, "failed to mark sidebar visited", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) getSidebarActivity(ctx fiber.Ctx) error {
	resp, err := s.HomeFeedService.SidebarActivity(ctx.Context())
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load sidebar activity", err)
	}
	return ctx.JSON(resp)
}

func (s *Service) getHomeActivity(ctx fiber.Ctx) error {
	resp, err := s.HomeFeedService.HomeActivity(ctx.Context())
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load home activity", err)
	}
	return ctx.JSON(resp)
}
