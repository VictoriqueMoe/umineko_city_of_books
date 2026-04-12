package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

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
	userID := ctx.Locals("userID").(uuid.UUID)
	var req struct {
		Sort string `json:"sort"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := s.UserRepo.UpdateGameBoardSort(ctx.Context(), userID, req.Sort); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) updateAppearance(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)
	var req struct {
		Theme      string `json:"theme"`
		Font       string `json:"font"`
		WideLayout bool   `json:"wide_layout"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := s.UserRepo.UpdateAppearance(ctx.Context(), userID, req.Theme, req.Font, req.WideLayout); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
