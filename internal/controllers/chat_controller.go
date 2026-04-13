package controllers

import (
	"errors"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllChatRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupResolveDMRoute,
		s.setupSendFirstDMRoute,
		s.setupCreateGroupRoomRoute,
		s.setupListRoomsRoute,
		s.setupListMyGroupRoomsRoute,
		s.setupListPublicRoomsRoute,
		s.setupJoinRoomRoute,
		s.setupLeaveRoomRoute,
		s.setupGetRoomMembersRoute,
		s.setupKickMemberRoute,
		s.setupSetRoomMuteRoute,
		s.setupGetMessagesRoute,
		s.setupSendMessageRoute,
		s.setupDeleteChatRoute,
		s.setupChatUnreadCountRoute,
		s.setupMarkRoomReadRoute,
		s.setupUploadChatMessageMediaRoute,
	}
}

func (s *Service) setupResolveDMRoute(r fiber.Router) {
	r.Get("/chat/dm/:userID/resolve", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.resolveDM)
}

func (s *Service) setupSendFirstDMRoute(r fiber.Router) {
	r.Post("/chat/dm/:userID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.sendFirstDM)
}

func (s *Service) setupCreateGroupRoomRoute(r fiber.Router) {
	r.Post("/chat/rooms", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createGroupRoom)
}

func (s *Service) setupListRoomsRoute(r fiber.Router) {
	r.Get("/chat/rooms", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listRooms)
}

func (s *Service) setupGetMessagesRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getMessages)
}

func dmRouteError(ctx fiber.Ctx, err error) error {
	if errors.Is(err, chat.ErrUserBlocked) {
		return utils.Forbidden(ctx, "you cannot message this user")
	}
	if errors.Is(err, chat.ErrDmsDisabled) {
		return utils.Forbidden(ctx, "recipient has DMs disabled")
	}
	if errors.Is(err, chat.ErrUserNotFound) {
		return utils.NotFound(ctx, "user not found")
	}
	if errors.Is(err, chat.ErrCannotDMSelf) {
		return utils.BadRequest(ctx, "cannot DM yourself")
	}
	if errors.Is(err, chat.ErrMissingFields) {
		return utils.BadRequest(ctx, "message body is required")
	}
	return utils.InternalError(ctx, "chat operation failed")
}

func (s *Service) resolveDM(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	recipientID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}

	resp, err := s.ChatService.ResolveDMRoom(ctx.Context(), userID, recipientID)
	if err != nil {
		return dmRouteError(ctx, err)
	}
	return ctx.JSON(resp)
}

func (s *Service) sendFirstDM(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	recipientID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}

	req, ok := utils.BindJSON[dto.SendMessageRequest](ctx)
	if !ok {
		return nil
	}

	resp, err := s.ChatService.SendDMMessage(ctx.Context(), userID, recipientID, req.Body)
	if err != nil {
		return dmRouteError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(resp)
}

func (s *Service) createGroupRoom(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateGroupRoomRequest](ctx)
	if !ok {
		return nil
	}

	room, err := s.ChatService.CreateGroupRoom(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, chat.ErrMissingFields) {
			return utils.BadRequest(ctx, "room name is required")
		}
		return utils.InternalError(ctx, "failed to create group room")
	}

	return ctx.Status(fiber.StatusCreated).JSON(room)
}

func (s *Service) listRooms(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	resp, err := s.ChatService.ListRooms(ctx.Context(), userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list rooms")
	}

	return ctx.JSON(resp)
}

func (s *Service) setupSendMessageRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.sendMessage)
}

func (s *Service) sendMessage(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	req, ok := utils.BindJSON[dto.SendMessageRequest](ctx)
	if !ok {
		return nil
	}

	resp, err := s.ChatService.SendMessage(ctx.Context(), userID, roomID, req)
	if err != nil {
		if errors.Is(err, chat.ErrUserBlocked) {
			return utils.Forbidden(ctx, "you cannot message this user")
		}
		if errors.Is(err, chat.ErrMissingFields) {
			return utils.BadRequest(ctx, "message body is required")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "you are not a member of this room")
		}
		return utils.InternalError(ctx, "failed to send message")
	}

	return ctx.Status(fiber.StatusCreated).JSON(resp)
}

func (s *Service) getMessages(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	limit := fiber.Query[int](ctx, "limit", 50)
	before := ctx.Query("before")

	var resp *dto.ChatMessageListResponse
	var err error
	if before != "" {
		resp, err = s.ChatService.GetMessagesBefore(ctx.Context(), userID, roomID, before, limit)
	} else {
		offset := fiber.Query[int](ctx, "offset", 0)
		resp, err = s.ChatService.GetMessages(ctx.Context(), userID, roomID, limit, offset)
	}
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "you are not a member of this room")
		}
		return utils.InternalError(ctx, "failed to get messages")
	}

	return ctx.JSON(resp)
}

func (s *Service) setupDeleteChatRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteChat)
}

func (s *Service) deleteChat(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	if err := s.ChatService.DeleteChat(ctx.Context(), roomID, userID); err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "you are not a member of this chat")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "system rooms cannot be deleted")
		}
		return utils.InternalError(ctx, "failed to delete chat")
	}

	return utils.OK(ctx)
}

func (s *Service) setupChatUnreadCountRoute(r fiber.Router) {
	r.Get("/chat/unread-count", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.chatUnreadCount)
}

func (s *Service) chatUnreadCount(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	count, err := s.ChatService.GetUnreadCount(ctx.Context(), userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to get unread count")
	}
	return ctx.JSON(fiber.Map{"count": count})
}

func (s *Service) setupMarkRoomReadRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/read", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.markRoomRead)
}

func (s *Service) markRoomRead(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	if err := s.ChatService.MarkRead(ctx.Context(), roomID, userID); err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "you are not a member of this chat")
		}
		return utils.InternalError(ctx, "failed to mark room read")
	}
	return utils.OK(ctx)
}

func (s *Service) setupUploadChatMessageMediaRoute(r fiber.Router) {
	r.Post("/chat/messages/:messageID/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadChatMessageMedia)
}

func (s *Service) uploadChatMessageMedia(ctx fiber.Ctx) error {
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	file, err := ctx.FormFile("media")
	if err != nil {
		return utils.BadRequest(ctx, "no media file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	result, err := s.ChatService.UploadMessageMedia(ctx.Context(), messageID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) setupListMyGroupRoomsRoute(r fiber.Router) {
	r.Get("/chat/rooms/mine", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listMyGroupRooms)
}

func (s *Service) listMyGroupRooms(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	search := ctx.Query("search")
	tag := ctx.Query("tag")
	role := ctx.Query("role")
	isRPOnly := ctx.Query("rp") == "true"
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	resp, err := s.ChatService.ListUserGroupRooms(ctx.Context(), userID, search, isRPOnly, tag, role, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list rooms")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupListPublicRoomsRoute(r fiber.Router) {
	r.Get("/chat/rooms/public", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listPublicRooms)
}

func (s *Service) listPublicRooms(ctx fiber.Ctx) error {
	viewerID := utils.UserID(ctx)
	search := ctx.Query("search")
	tag := ctx.Query("tag")
	isRPOnly := ctx.Query("rp") == "true"
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	resp, err := s.ChatService.ListPublicRooms(ctx.Context(), search, isRPOnly, tag, viewerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list public rooms")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupJoinRoomRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/join", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.joinRoom)
}

func (s *Service) joinRoom(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	resp, err := s.ChatService.JoinRoom(ctx.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "room not found")
		}
		if errors.Is(err, chat.ErrNotGroupRoom) {
			return utils.BadRequest(ctx, "not a group room")
		}
		if errors.Is(err, chat.ErrNotPublic) {
			return utils.Forbidden(ctx, "room is not public")
		}
		if errors.Is(err, chat.ErrRoomFull) {
			return utils.Conflict(ctx, "room is full")
		}
		if errors.Is(err, chat.ErrUserBlocked) {
			return utils.Forbidden(ctx, "you cannot join this room")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "this room is managed automatically")
		}
		return utils.InternalError(ctx, "failed to join room")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupLeaveRoomRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/leave", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.leaveRoom)
}

func (s *Service) leaveRoom(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	if err := s.ChatService.LeaveRoom(ctx.Context(), roomID, userID); err != nil {
		if errors.Is(err, chat.ErrCannotLeaveAsHost) {
			return utils.Forbidden(ctx, "host cannot leave their own room")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "this room is managed automatically")
		}
		return utils.InternalError(ctx, "failed to leave room")
	}
	return utils.OK(ctx)
}

func (s *Service) setupSetRoomMuteRoute(r fiber.Router) {
	r.Put("/chat/rooms/:roomID/mute", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setRoomMute)
}

func (s *Service) setRoomMute(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	var req struct {
		Muted bool `json:"muted"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	if err := s.ChatService.SetRoomMuted(ctx.Context(), roomID, userID, req.Muted); err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		return utils.InternalError(ctx, "failed to set mute")
	}
	return ctx.JSON(fiber.Map{"muted": req.Muted})
}

func (s *Service) setupGetRoomMembersRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/members", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getRoomMembers)
}

func (s *Service) getRoomMembers(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	members, err := s.ChatService.GetMembers(ctx.Context(), userID, roomID)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		return utils.InternalError(ctx, "failed to get members")
	}
	return ctx.JSON(fiber.Map{"members": members})
}

func (s *Service) setupKickMemberRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/members/:userID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.kickMember)
}

func (s *Service) kickMember(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}

	if err := s.ChatService.KickMember(ctx.Context(), userID, roomID, targetID); err != nil {
		if errors.Is(err, chat.ErrNotHost) {
			return utils.Forbidden(ctx, "only the host can kick members")
		}
		if errors.Is(err, chat.ErrCannotKickHost) {
			return utils.BadRequest(ctx, "cannot kick the host")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.NotFound(ctx, "user is not a member")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "this room is managed automatically")
		}
		return utils.InternalError(ctx, "failed to kick member")
	}
	return utils.OK(ctx)
}
