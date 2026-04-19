package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/secrets"
)

func (s *Service) getAllUserPreferencesRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupUpdateGameBoardSort,
		s.setupUpdateAppearance,
		s.setupUnlockSecret,
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

func (s *Service) unlockSecret(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	var req struct {
		Secret string `json:"secret"`
		Phrase string `json:"phrase"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	spec, ok := secrets.Lookup(req.Secret)
	if !ok {
		return utils.BadRequest(ctx, "invalid request")
	}
	sum := sha256.Sum256([]byte(req.Phrase))
	if hex.EncodeToString(sum[:]) != spec.ExpectedHash {
		return utils.BadRequest(ctx, "invalid request")
	}
	if err := s.UserSecretRepo.Unlock(ctx.Context(), userID, string(spec.ID)); err != nil {
		return utils.InternalError(ctx, "failed to save")
	}

	parent, ok := secrets.ParentOf(spec.ID)
	if ok && parent.Title != "" && s.SecretService != nil {
		parentID := string(parent.ID)
		if spec.ID == parent.ID {
			s.SecretService.BroadcastSolved(ctx.Context(), parentID, userID, time.Now().UTC().Format(time.RFC3339))
		} else {
			s.SecretService.BroadcastProgress(ctx.Context(), parentID, userID)
		}
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
