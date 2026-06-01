package controllers

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
)

func (s *Service) setupListWatchPartiesRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/watch-parties", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listWatchParties)
}

func (s *Service) setupStartWatchPartyRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/watch-parties", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.startWatchParty)
}

func (s *Service) setupJoinWatchPartyRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/watch-parties/:sessionID/join", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.joinWatchParty)
}

func (s *Service) setupLeaveWatchPartyRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/watch-parties/:sessionID/participants/me", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.leaveWatchParty)
}

func (s *Service) setupGrantWatchPartyControlRoute(r fiber.Router) {
	r.Patch("/chat/rooms/:roomID/watch-parties/:sessionID/participants/:userID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.grantWatchPartyControl)
}

func (s *Service) setupKickWatchPartyParticipantRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/watch-parties/:sessionID/participants/:userID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.kickWatchPartyParticipant)
}

func (s *Service) setupEndWatchPartyRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/watch-parties/:sessionID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.endWatchParty)
}

func (s *Service) setupListWatchPartyMessagesRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/watch-parties/:sessionID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listWatchPartyMessages)
}

func (s *Service) setupSendWatchPartyMessageRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/watch-parties/:sessionID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.sendWatchPartyMessage)
}

func (s *Service) setupIdentifyWatchPartyRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/watch-parties/:sessionID/identify", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.identifyWatchPartyParticipant)
}

func (s *Service) setupWatchPartyVoiceTokenRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/watch-parties/:sessionID/voice/token", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.watchPartyVoiceToken)
}

func (s *Service) setupWatchPartyVoiceMuteRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/watch-parties/:sessionID/voice/participants/:userID/mute", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.watchPartyVoiceMute)
}

func mapWatchPartyError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, chat.ErrWatchPartyDisabled):
		{
			return utils.ServiceUnavailable(ctx, "watch parties are not configured")
		}
	case errors.Is(err, chat.ErrNotMember):
		{
			return utils.Forbidden(ctx, "you are not a member of this room")
		}
	case errors.Is(err, chat.ErrWatchPartyNotActive):
		{
			return utils.NotFound(ctx, "no such active watch party")
		}
	case errors.Is(err, chat.ErrWatchPartyNotController):
		{
			return utils.Forbidden(ctx, "only the watch party controller can do that")
		}
	case errors.Is(err, chat.ErrWatchPartyNotParticipant):
		{
			return utils.Forbidden(ctx, "you are not a participant of this watch party")
		}
	case errors.Is(err, chat.ErrWatchPartyWrongRoomType):
		{
			return utils.BadRequest(ctx, "watch parties are only available in group chat rooms")
		}
	case errors.Is(err, chat.ErrWatchPartyMessageEmpty):
		{
			return utils.BadRequest(ctx, "message body is required")
		}
	case errors.Is(err, chat.ErrWatchPartyMessageTooLong):
		{
			return utils.BadRequest(ctx, "message is too long")
		}
	case errors.Is(err, chat.ErrWatchPartyOutranked):
		{
			return utils.Forbidden(ctx, "you do not outrank the current controller")
		}
	case errors.Is(err, chat.ErrWatchPartyCannotKickSelf):
		{
			return utils.BadRequest(ctx, "use leave to remove yourself")
		}
	case errors.Is(err, chat.ErrWatchPartyCannotKick):
		{
			return utils.Forbidden(ctx, "you do not outrank this participant")
		}
	case errors.Is(err, chat.ErrRoomNotFound):
		{
			return utils.NotFound(ctx, "room not found")
		}
	case errors.Is(err, chat.ErrWatchPartyInvalidType):
		{
			return utils.BadRequest(ctx, "invalid watch party type")
		}
	case errors.Is(err, chat.ErrVoiceDisabled):
		{
			return utils.ServiceUnavailable(ctx, "voice chat is not configured")
		}
	case errors.Is(err, chat.ErrVoiceMuteForbidden):
		{
			return utils.Forbidden(ctx, "you cannot mute participants here")
		}
	}
	return utils.InternalError(ctx, "watch party request failed: "+err.Error(), err)
}

func (s *Service) listWatchParties(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	resp, err := s.ChatService.ListWatchParties(ctx.Context(), roomID, actorID)
	if err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) startWatchParty(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	var req dto.StartWatchPartyRequest
	if len(ctx.Body()) > 0 {
		if parsed, ok := utils.BindJSON[dto.StartWatchPartyRequest](ctx); ok {
			req = parsed
		} else {
			return nil
		}
	}
	resp, err := s.ChatService.StartWatchParty(ctx.Context(), roomID, actorID, req.StartURL, req.Region, req.Title, req.Type)
	if err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(resp)
}

func (s *Service) watchPartyVoiceToken(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	token, url, err := s.ChatService.MintSessionVoiceToken(ctx.Context(), roomID, sessionID, actorID)
	if err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return ctx.JSON(dto.VoiceTokenResponse{Token: token, URL: url})
}

func (s *Service) watchPartyVoiceMute(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.VoiceMuteRequest](ctx)
	if !ok {
		return nil
	}
	if err := s.ChatService.ForceMuteSessionVoice(ctx.Context(), roomID, sessionID, actorID, targetID, req.Muted); err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) joinWatchParty(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	resp, err := s.ChatService.JoinWatchParty(ctx.Context(), roomID, sessionID, actorID)
	if err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) leaveWatchParty(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	if err := s.ChatService.LeaveWatchParty(ctx.Context(), roomID, sessionID, actorID); err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) grantWatchPartyControl(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	if err := s.ChatService.GrantWatchPartyControl(ctx.Context(), roomID, sessionID, actorID, targetID); err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) kickWatchPartyParticipant(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	if err := s.ChatService.KickWatchPartyParticipant(ctx.Context(), roomID, sessionID, actorID, targetID); err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) endWatchParty(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	if err := s.ChatService.EndWatchParty(ctx.Context(), roomID, sessionID, actorID, "controller_ended"); err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) listWatchPartyMessages(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	resp, err := s.ChatService.GetWatchPartyMessages(ctx.Context(), roomID, sessionID, actorID)
	if err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) identifyWatchPartyParticipant(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.IdentifyWatchPartyParticipantRequest](ctx)
	if !ok {
		return nil
	}
	if err := s.ChatService.IdentifyWatchPartyParticipant(ctx.Context(), roomID, sessionID, actorID, req.Identifier); err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) sendWatchPartyMessage(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	sessionID, ok := utils.ParseIDParam(ctx, "sessionID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.SendWatchPartyMessageRequest](ctx)
	if !ok {
		return nil
	}
	msg, err := s.ChatService.SendWatchPartyMessage(ctx.Context(), roomID, sessionID, actorID, req.Body)
	if err != nil {
		return mapWatchPartyError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(msg)
}
