package controllers

import (
	"errors"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllBlockRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupBlockUser,
		s.setupUnblockUser,
		s.setupGetBlockStatus,
		s.setupListBlockedUsers,
	}
}

func (s *Service) setupBlockUser(r fiber.Router) {
	r.Post("/users/:id/block", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.blockUser)
}

func (s *Service) setupUnblockUser(r fiber.Router) {
	r.Delete("/users/:id/block", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unblockUser)
}

func (s *Service) setupGetBlockStatus(r fiber.Router) {
	r.Get("/users/:id/block-status", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getBlockStatus)
}

func (s *Service) setupListBlockedUsers(r fiber.Router) {
	r.Get("/blocked-users", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listBlockedUsers)
}

func (s *Service) listBlockedUsers(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)
	users, err := s.BlockService.GetBlockedUsers(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list blocked users"})
	}

	type blockedUserResp struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		DisplayName string    `json:"display_name"`
		AvatarURL   string    `json:"avatar_url"`
		BlockedAt   string    `json:"blocked_at"`
	}

	result := make([]blockedUserResp, len(users))
	for i, u := range users {
		result[i] = blockedUserResp{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			BlockedAt:   u.BlockedAt,
		}
	}

	if result == nil {
		result = []blockedUserResp{}
	}

	return ctx.JSON(fiber.Map{"users": result})
}

func (s *Service) blockUser(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.BlockService.Block(ctx.Context(), userID, targetID); err != nil {
		if errors.Is(err, block.ErrCannotBlockSelf) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, block.ErrCannotBlockStaff) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to block user"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unblockUser(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.BlockService.Unblock(ctx.Context(), userID, targetID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unblock user"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) getBlockStatus(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	blocked, err := s.BlockService.IsBlocked(ctx.Context(), userID, targetID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to check block status"})
	}

	blockedBy, err := s.BlockService.IsBlocked(ctx.Context(), targetID, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to check block status"})
	}

	return ctx.JSON(fiber.Map{
		"blocking":   blocked,
		"blocked_by": blockedBy,
	})
}
