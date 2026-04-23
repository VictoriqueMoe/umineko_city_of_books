package controllers

import (
	"encoding/json"
	"errors"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllGameRoomRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupInviteGameRoute,
		s.setupListGameRoomsRoute,
		s.setupListLiveGameRoomsRoute,
		s.setupListFinishedGameRoomsRoute,
		s.setupGetGameRoomRoute,
		s.setupAcceptGameRoute,
		s.setupDeclineGameRoute,
		s.setupCancelGameRoute,
		s.setupGameActionRoute,
		s.setupResignGameRoute,
		s.setupGameScoreboardRoute,
		s.setupGetSpectatorChatRoute,
		s.setupPostSpectatorChatRoute,
	}
}

func (s *Service) setupInviteGameRoute(r fiber.Router) {
	r.Post("/game-rooms", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.inviteGame)
}

func (s *Service) setupListGameRoomsRoute(r fiber.Router) {
	r.Get("/game-rooms", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listGameRooms)
}

func (s *Service) setupListLiveGameRoomsRoute(r fiber.Router) {
	r.Get("/game-rooms/live", s.listLiveGameRooms)
}

func (s *Service) setupListFinishedGameRoomsRoute(r fiber.Router) {
	r.Get("/game-rooms/finished", s.listFinishedGameRooms)
}

func (s *Service) setupGetGameRoomRoute(r fiber.Router) {
	r.Get("/game-rooms/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getGameRoom)
}

func (s *Service) setupAcceptGameRoute(r fiber.Router) {
	r.Post("/game-rooms/:id/accept", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.acceptGame)
}

func (s *Service) setupDeclineGameRoute(r fiber.Router) {
	r.Post("/game-rooms/:id/decline", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.declineGame)
}

func (s *Service) setupCancelGameRoute(r fiber.Router) {
	r.Post("/game-rooms/:id/cancel", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.cancelGame)
}

func (s *Service) setupGameActionRoute(r fiber.Router) {
	r.Post("/game-rooms/:id/action", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.gameAction)
}

func (s *Service) setupResignGameRoute(r fiber.Router) {
	r.Post("/game-rooms/:id/resign", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.resignGame)
}

func (s *Service) setupGameScoreboardRoute(r fiber.Router) {
	r.Get("/games/:type/scoreboard", s.getGameScoreboard)
}

func (s *Service) setupGetSpectatorChatRoute(r fiber.Router) {
	r.Get("/game-rooms/:id/chat", s.getSpectatorChat)
}

func (s *Service) setupPostSpectatorChatRoute(r fiber.Router) {
	r.Post("/game-rooms/:id/chat", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.postSpectatorChat)
}

func gameRoomError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, gameroom.ErrNotFound):
		return utils.NotFound(ctx, "game room not found")
	case errors.Is(err, gameroom.ErrNotParticipant):
		return utils.Forbidden(ctx, "you are not a participant in this game")
	case errors.Is(err, gameroom.ErrNotInvitee):
		return utils.Forbidden(ctx, "only the invited player can do that")
	case errors.Is(err, gameroom.ErrNotInviter):
		return utils.Forbidden(ctx, "only the inviter can cancel this invite")
	case errors.Is(err, gameroom.ErrRoomNotPending):
		return utils.BadRequest(ctx, "this invite is no longer pending")
	case errors.Is(err, gameroom.ErrRoomNotActive):
		return utils.BadRequest(ctx, "this game is not active")
	case errors.Is(err, gameroom.ErrNotYourTurn):
		return utils.BadRequest(ctx, "it is not your turn")
	case errors.Is(err, gameroom.ErrSelfInvite):
		return utils.BadRequest(ctx, "you cannot invite yourself")
	case errors.Is(err, gameroom.ErrUnknownGameType):
		return utils.BadRequest(ctx, "unknown game type")
	case errors.Is(err, gameroom.ErrOpponentBlocked):
		return utils.Forbidden(ctx, "you cannot invite this user")
	case errors.Is(err, gameroom.ErrOpponentInactive):
		return utils.NotFound(ctx, "opponent not found")
	case errors.Is(err, gameroom.ErrEmptyChat):
		return utils.BadRequest(ctx, "message is empty")
	case errors.Is(err, gameroom.ErrPlayersCantChat):
		return utils.Forbidden(ctx, "players cannot use spectator chat")
	}
	return utils.InternalError(ctx, "game operation failed")
}

func (s *Service) inviteGame(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.GameInviteRequest](ctx)
	if !ok {
		return nil
	}
	room, err := s.GameRoomService.Invite(ctx.Context(), userID, req.OpponentID, req.GameType)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(room)
}

func (s *Service) listGameRooms(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	gameType := dto.GameType(ctx.Query("game_type"))
	statusStr := ctx.Query("status")
	var statuses []dto.GameStatus
	if statusStr != "" {
		statuses = append(statuses, dto.GameStatus(statusStr))
	}
	resp, err := s.GameRoomService.List(ctx.Context(), userID, gameroom.ListFilter{
		GameType: gameType,
		Statuses: statuses,
		Limit:    50,
	})
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) getGameRoom(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	room, err := s.GameRoomService.Get(ctx.Context(), roomID, userID)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(room)
}

func (s *Service) listLiveGameRooms(ctx fiber.Ctx) error {
	gameType := dto.GameType(ctx.Query("game_type"))
	resp, err := s.GameRoomService.ListLive(ctx.Context(), gameType, 50, 0)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) listFinishedGameRooms(ctx fiber.Ctx) error {
	gameType := dto.GameType(ctx.Query("game_type"))
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)
	resp, err := s.GameRoomService.ListFinished(ctx.Context(), gameType, limit, offset)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) getSpectatorChat(ctx fiber.Ctx) error {
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	resp, err := s.GameRoomService.GetSpectatorChat(ctx.Context(), roomID)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) postSpectatorChat(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.SpectatorChatRequest](ctx)
	if !ok {
		return nil
	}
	msg, err := s.GameRoomService.PostSpectatorChat(ctx.Context(), roomID, userID, req.Body)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		return gameRoomError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(msg)
}

func (s *Service) acceptGame(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	room, err := s.GameRoomService.Accept(ctx.Context(), roomID, userID)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(room)
}

func (s *Service) declineGame(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	if err := s.GameRoomService.Decline(ctx.Context(), roomID, userID); err != nil {
		return gameRoomError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) cancelGame(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	if err := s.GameRoomService.Cancel(ctx.Context(), roomID, userID); err != nil {
		return gameRoomError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) gameAction(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.GameActionRequest](ctx)
	if !ok {
		return nil
	}
	if len(req.Action) == 0 {
		return utils.BadRequest(ctx, "action is required")
	}
	room, err := s.GameRoomService.SubmitAction(ctx.Context(), roomID, userID, json.RawMessage(req.Action))
	if err != nil {
		if errors.Is(err, gameroom.ErrNotFound) || errors.Is(err, gameroom.ErrNotParticipant) || errors.Is(err, gameroom.ErrNotYourTurn) || errors.Is(err, gameroom.ErrRoomNotActive) || errors.Is(err, gameroom.ErrUnknownGameType) {
			return gameRoomError(ctx, err)
		}
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.JSON(room)
}

func (s *Service) getGameScoreboard(ctx fiber.Ctx) error {
	gameType := dto.GameType(ctx.Params("type"))
	resp, err := s.GameRoomService.Scoreboard(ctx.Context(), gameType)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) resignGame(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}
	room, err := s.GameRoomService.Resign(ctx.Context(), roomID, userID)
	if err != nil {
		return gameRoomError(ctx, err)
	}
	return ctx.JSON(room)
}
