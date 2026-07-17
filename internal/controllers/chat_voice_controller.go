package controllers

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
)

func (s *Service) setupVoiceTokenRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/voice/token", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voiceToken)
}

func (s *Service) setupVoiceMuteRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/voice/participants/:userID/mute", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voiceMute)
}

func (s *Service) setupLiveKitWebhookRoute(r fiber.Router) {
	r.Post("/livekit/webhook", s.livekitWebhook)
}

func (s *Service) voiceToken(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	token, url, err := s.ChatService.MintVoiceToken(ctx.Context(), roomID, userID)
	if err != nil {
		return mapVoiceError(ctx, err)
	}

	return ctx.JSON(dto.VoiceTokenResponse{Token: token, URL: url})
}

func (s *Service) voiceMute(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)

	roomID, ok := utils.ParseIDParam(ctx, "roomID")
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

	if err := s.ChatService.ForceMuteVoice(ctx.Context(), roomID, actorID, targetID, req.Muted); err != nil {
		return mapVoiceError(ctx, err)
	}

	return utils.OK(ctx)
}

func (s *Service) decorateVoiceCounts(list *dto.ChatRoomListResponse) {
	if list == nil {
		return
	}

	for i := range list.Rooms {
		list.Rooms[i].VoiceCount = s.ChatService.VoiceCount(list.Rooms[i].ID)
		list.Rooms[i].VoiceParticipants = s.ChatService.VoiceParticipants(list.Rooms[i].ID)
	}
}

func (s *Service) livekitWebhook(ctx fiber.Ctx) error {
	authHeader := ctx.Get("Authorization")
	body := ctx.Body()

	handled, err := s.StreamService.HandleWebhook(ctx.Context(), authHeader, body)
	if err != nil {
		return utils.BadRequest(ctx, "invalid webhook")
	}
	if handled {
		return utils.OK(ctx)
	}

	if err := s.ChatService.HandleVoiceWebhook(ctx.Context(), authHeader, body); err != nil {
		return utils.BadRequest(ctx, "invalid webhook")
	}

	return utils.OK(ctx)
}

func mapVoiceError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, chat.ErrVoiceDisabled):
		{
			return utils.ServiceUnavailable(ctx, "voice chat is not configured")
		}
	case errors.Is(err, chat.ErrNotMember):
		{
			return utils.Forbidden(ctx, "you are not a member of this room")
		}
	case errors.Is(err, chat.ErrRoomNotFound):
		{
			return utils.NotFound(ctx, "room not found")
		}
	case errors.Is(err, chat.ErrUserBlocked):
		{
			return utils.Forbidden(ctx, "you cannot call this user")
		}
	case errors.Is(err, chat.ErrVoiceMuteForbidden):
		{
			return utils.Forbidden(ctx, "you cannot mute participants here")
		}
	}
	return utils.InternalError(ctx, "voice request failed: "+err.Error(), err)
}
