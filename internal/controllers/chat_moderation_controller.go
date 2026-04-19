package controllers

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
)

func (s *Service) setupListRoomBansRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/bans", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listRoomBans)
}

func (s *Service) setupBanMemberRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/bans/:userID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.banMember)
}

func (s *Service) setupUnbanMemberRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/bans/:userID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unbanMember)
}

func (s *Service) setupListRoomBannedWordsRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/banned-words", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listRoomBannedWords)
}

func (s *Service) setupCreateRoomBannedWordRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/banned-words", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createRoomBannedWord)
}

func (s *Service) setupUpdateRoomBannedWordRoute(r fiber.Router) {
	r.Put("/chat/rooms/:roomID/banned-words/:ruleID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateRoomBannedWord)
}

func (s *Service) setupDeleteRoomBannedWordRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/banned-words/:ruleID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteRoomBannedWord)
}

func mapBanError(ctx fiber.Ctx, err error) error {
	if errors.Is(err, chat.ErrRoomNotFound) {
		return utils.NotFound(ctx, "room not found")
	}
	if errors.Is(err, chat.ErrNotHost) {
		return utils.Forbidden(ctx, "only the host or a moderator can do this")
	}
	if errors.Is(err, chat.ErrCannotBanStaff) {
		return utils.Forbidden(ctx, "the host and site staff cannot be banned")
	}
	if errors.Is(err, chat.ErrSystemRoom) {
		return utils.Forbidden(ctx, "this room is managed automatically")
	}
	return utils.InternalError(ctx, "failed")
}

func (s *Service) listRoomBans(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	bans, err := s.ChatService.ListRoomBans(ctx.Context(), actorID, roomID)
	if err != nil {
		return mapBanError(ctx, err)
	}
	return ctx.JSON(dto.ChatRoomBanListResponse{Bans: bans})
}

func (s *Service) banMember(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	var req dto.BanMemberRequest
	if len(ctx.Body()) > 0 {
		if r, ok := utils.BindJSON[dto.BanMemberRequest](ctx); ok {
			req = r
		} else {
			return nil
		}
	}
	if err := s.ChatService.BanMember(ctx.Context(), actorID, roomID, targetID, req.Reason); err != nil {
		return mapBanError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) unbanMember(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	if err := s.ChatService.UnbanMember(ctx.Context(), actorID, roomID, targetID); err != nil {
		return mapBanError(ctx, err)
	}
	return utils.OK(ctx)
}

func mapBannedWordError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, chat.ErrRoomNotFound):
		return utils.NotFound(ctx, "not found")
	case errors.Is(err, chat.ErrNotHost):
		return utils.Forbidden(ctx, "only the host or a moderator can do this")
	case errors.Is(err, chat.ErrModRoleRequired):
		return utils.Forbidden(ctx, "admin permission required")
	case errors.Is(err, chat.ErrMissingFields):
		return utils.BadRequest(ctx, "pattern is required")
	case errors.Is(err, chat.ErrInvalidBannedWordMode):
		return utils.BadRequest(ctx, "invalid match mode")
	case errors.Is(err, chat.ErrInvalidBannedWordAction):
		return utils.BadRequest(ctx, "invalid action")
	case errors.Is(err, chat.ErrInvalidBannedWordRegex):
		return utils.BadRequest(ctx, "invalid regex pattern")
	case errors.Is(err, chat.ErrBannedWordRuleMismatch):
		return utils.NotFound(ctx, "rule not found for this scope")
	}
	return utils.InternalError(ctx, "failed")
}

func (s *Service) listRoomBannedWords(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	rules, err := s.ChatService.ListRoomBannedWords(ctx.Context(), actorID, roomID)
	if err != nil {
		return mapBannedWordError(ctx, err)
	}
	return ctx.JSON(dto.BannedWordRuleListResponse{Rules: rules})
}

func (s *Service) createRoomBannedWord(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.CreateBannedWordRequest](ctx)
	if !ok {
		return nil
	}
	rule, err := s.ChatService.CreateRoomBannedWord(ctx.Context(), actorID, roomID, req)
	if err != nil {
		return mapBannedWordError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(rule)
}

func (s *Service) updateRoomBannedWord(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	ruleID, ok := utils.ParseIDParam(ctx, "ruleID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.UpdateBannedWordRequest](ctx)
	if !ok {
		return nil
	}
	rule, err := s.ChatService.UpdateRoomBannedWord(ctx.Context(), actorID, roomID, ruleID, req)
	if err != nil {
		return mapBannedWordError(ctx, err)
	}
	return ctx.JSON(rule)
}

func (s *Service) deleteRoomBannedWord(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	ruleID, ok := utils.ParseIDParam(ctx, "ruleID")
	if !ok {
		return nil
	}
	if err := s.ChatService.DeleteRoomBannedWord(ctx.Context(), actorID, roomID, ruleID); err != nil {
		return mapBannedWordError(ctx, err)
	}
	return utils.OK(ctx)
}
