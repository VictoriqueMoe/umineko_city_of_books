package controllers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/middleware"
	usersvc "umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/usersecret"
)

func (s *Service) getAllUserPreferencesRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupUpdateGameBoardSort,
		s.setupUpdateAppearance,
		s.setupUnlockSecret,
		s.setupUpdateChatbotOptIn,
	}
}

func (s *Service) setupUpdateGameBoardSort(r fiber.Router) {
	r.Put("/preferences/game-board-sort", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateGameBoardSort)
}

func (s *Service) setupUpdateAppearance(r fiber.Router) {
	r.Put("/preferences/appearance", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateAppearance)
}

func (s *Service) setupUnlockSecret(r fiber.Router) {
	r.Put("/preferences/secret-unlock", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlockSecret)
}

func (s *Service) setupUpdateChatbotOptIn(r fiber.Router) {
	r.Put("/preferences/chatbot-opt-in", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateChatbotOptIn)
}

func (s *Service) updateGameBoardSort(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req struct {
		Sort string `json:"sort"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	if err := s.UserService.UpdateGameBoardSort(ctx.Context(), userID, req.Sort); err != nil {
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
	if err := s.UserService.UpdateAppearance(ctx.Context(), userID, req.Theme, req.Font, req.WideLayout); err != nil {
		return utils.InternalError(ctx, "failed to save")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) updateChatbotOptIn(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req struct {
		OptedIn bool `json:"opted_in"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}

	if err := s.UserService.SetChatbotOptIn(ctx.Context(), userID, req.OptedIn); err != nil {
		if errors.Is(err, usersvc.ErrChatbotOptInUnavailable) {
			return utils.Conflict(ctx, "character opt-in is not available right now")
		}

		return utils.InternalError(ctx, "failed to save", err)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlockSecret(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req struct {
		Secret string `json:"secret"`
		Phrase string `json:"phrase"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}

	result, err := s.UserSecretService.Unlock(ctx.Context(), userID, req.Secret, req.Phrase)
	if err != nil {
		switch {
		case errors.Is(err, usersecret.ErrInvalidRequest):
			return utils.BadRequest(ctx, "invalid request")
		case errors.Is(err, usersecret.ErrHuntAlreadySolved):
			return utils.BadRequest(ctx, "hunt already solved")
		}
		return utils.InternalError(ctx, "failed to save", err)
	}

	if result.HasParent && s.SecretService != nil {
		parentID := string(result.Parent.ID)
		if result.IsParent {
			s.SecretService.BroadcastSolved(ctx.Context(), parentID, userID, time.Now().UTC().Format(time.RFC3339))
		} else {
			s.SecretService.BroadcastProgress(ctx.Context(), parentID, userID)
		}
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
