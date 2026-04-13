package controllers

import (
	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/middleware"
)

func (s *Service) getAllUserPreferencesRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupUpdateGameBoardSort,
		s.setupUpdateAppearance,
	}
}

func (s *Service) setupUpdateGameBoardSort(r fiber.Router) {
	r.Put("/preferences/game-board-sort", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateGameBoardSort)
}

func (s *Service) setupUpdateAppearance(r fiber.Router) {
	r.Put("/preferences/appearance", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateAppearance)
}

func (s *Service) updateGameBoardSort(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req struct {
		Sort string `json:"sort"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	if err := s.UserRepo.UpdateGameBoardSort(ctx.Context(), userID, req.Sort); err != nil {
		return utils.InternalError(ctx, "failed to save")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) updateAppearance(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req struct {
		Theme      string `json:"theme"`
		Font       string `json:"font"`
		WideLayout bool   `json:"wide_layout"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	if err := s.UserRepo.UpdateAppearance(ctx.Context(), userID, req.Theme, req.Font, req.WideLayout); err != nil {
		return utils.InternalError(ctx, "failed to save")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
