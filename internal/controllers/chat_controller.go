package controllers

import (
	"errors"
	"io"
	"mime/multipart"
	"net/url"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func collectChatFileUploads(form *multipart.Form) []chat.FileUpload {
	if form == nil {
		return nil
	}
	headers := form.File["media"]
	if len(headers) == 0 {
		return nil
	}
	uploads := make([]chat.FileUpload, 0, len(headers))
	for i := 0; i < len(headers); i++ {
		h := headers[i]
		uploads = append(uploads, chat.FileUpload{
			ContentType: h.Header.Get("Content-Type"),
			Size:        h.Size,
			Open: func() (io.ReadCloser, error) {
				return h.Open()
			},
		})
	}
	return uploads
}

func parseReplyToID(form *multipart.Form) (*uuid.UUID, bool) {
	if form == nil {
		return nil, true
	}
	values := form.Value["reply_to_id"]
	if len(values) == 0 || values[0] == "" {
		return nil, true
	}
	id, err := uuid.Parse(values[0])
	if err != nil {
		return nil, false
	}
	return &id, true
}

func formValue(form *multipart.Form, key string) string {
	if form == nil {
		return ""
	}
	values := form.Value[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

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
		s.setupInviteMembersRoute,
		s.setupKickMemberRoute,
		s.setupSetMemberTimeoutRoute,
		s.setupClearMemberTimeoutRoute,
		s.setupSetRoomMuteRoute,
		s.setupGetMessagesRoute,
		s.setupSendMessageRoute,
		s.setupDeleteChatRoute,
		s.setupChatUnreadCountRoute,
		s.setupMarkRoomReadRoute,
		s.setupSetRoomNicknameRoute,
		s.setupSetRoomAvatarRoute,
		s.setupClearRoomAvatarRoute,
		s.setupSetMemberNicknameAsModRoute,
		s.setupUnlockMemberNicknameRoute,
		s.setupPinMessageRoute,
		s.setupUnpinMessageRoute,
		s.setupListPinnedMessagesRoute,
		s.setupAddReactionRoute,
		s.setupRemoveReactionRoute,
		s.setupDeleteMessageRoute,
		s.setupEditMessageRoute,
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

	form, _ := ctx.MultipartForm()
	body := formValue(form, "body")
	files := collectChatFileUploads(form)

	resp, err := s.ChatService.SendDMMessage(ctx.Context(), userID, recipientID, body, files)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, upload.ErrFileTooLarge) || errors.Is(err, upload.ErrInvalidFileType) || errors.Is(err, upload.ErrInvalidVideoType) {
			return utils.BadRequest(ctx, err.Error())
		}
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
		if utils.MapFilterError(ctx, err) {
			return nil
		}
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

	form, _ := ctx.MultipartForm()
	replyToID, ok := parseReplyToID(form)
	if !ok {
		return utils.BadRequest(ctx, "invalid reply_to_id")
	}
	req := dto.SendMessageRequest{
		Body:      formValue(form, "body"),
		ReplyToID: replyToID,
	}
	files := collectChatFileUploads(form)

	resp, err := s.ChatService.SendMessage(ctx.Context(), userID, roomID, req, files)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, chat.ErrUserBlocked) {
			return utils.Forbidden(ctx, "you cannot message this user")
		}
		if errors.Is(err, chat.ErrTimedOut) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, chat.ErrMissingFields) {
			return utils.BadRequest(ctx, "message body is required")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "you are not a member of this room")
		}
		if errors.Is(err, upload.ErrFileTooLarge) || errors.Is(err, upload.ErrInvalidFileType) || errors.Is(err, upload.ErrInvalidVideoType) {
			return utils.BadRequest(ctx, err.Error())
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

	var req dto.JoinRoomRequest
	if len(ctx.Body()) > 0 {
		if r, ok := utils.BindJSON[dto.JoinRoomRequest](ctx); ok {
			req = r
		} else {
			return nil
		}
	}

	resp, err := s.ChatService.JoinRoom(ctx.Context(), roomID, userID, req.Ghost)
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
		if errors.Is(err, chat.ErrGhostRequiresStaff) {
			return utils.Forbidden(ctx, "only site moderators or admins can join as a ghost")
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

func (s *Service) setupInviteMembersRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/members", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.inviteMembers)
}

func (s *Service) inviteMembers(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	req, ok := utils.BindJSON[dto.InviteMembersRequest](ctx)
	if !ok {
		return nil
	}
	if len(req.UserIDs) == 0 {
		return utils.BadRequest(ctx, "user_ids is required")
	}

	resp, err := s.ChatService.InviteMembers(ctx.Context(), userID, roomID, req.UserIDs)
	if err != nil {
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "room not found")
		}
		if errors.Is(err, chat.ErrNotGroupRoom) {
			return utils.BadRequest(ctx, "only group rooms support invites")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "this room is managed automatically")
		}
		if errors.Is(err, chat.ErrNotHost) {
			return utils.Forbidden(ctx, "only the host can invite members")
		}
		return utils.InternalError(ctx, "failed to invite members")
	}
	return ctx.JSON(resp)
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
		if errors.Is(err, chat.ErrTargetImmune) {
			return utils.Forbidden(ctx, "moderators and admins cannot be kicked")
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

func (s *Service) setupSetMemberTimeoutRoute(r fiber.Router) {
	r.Put("/chat/rooms/:roomID/members/:userID/timeout", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setMemberTimeout)
}

func (s *Service) setMemberTimeout(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.SetMemberTimeoutRequest](ctx)
	if !ok {
		return nil
	}

	member, err := s.ChatService.SetMemberTimeout(ctx.Context(), roomID, actorID, targetID, req)
	if err != nil {
		if errors.Is(err, chat.ErrNotHost) {
			return utils.Forbidden(ctx, "only room hosts and site moderators can set timeouts")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.NotFound(ctx, "user is not a member")
		}
		if errors.Is(err, chat.ErrCannotKickHost) {
			return utils.BadRequest(ctx, "cannot timeout the host")
		}
		if errors.Is(err, chat.ErrTargetImmune) {
			return utils.Forbidden(ctx, "moderators and admins cannot be timed out")
		}
		if errors.Is(err, chat.ErrInvalidTimeoutDuration) {
			return utils.BadRequest(ctx, "invalid timeout duration")
		}
		if errors.Is(err, chat.ErrTimeoutLockedByStaff) {
			return utils.Forbidden(ctx, "this timeout can only be changed by site moderators or admins")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "this room is managed automatically")
		}
		return utils.InternalError(ctx, "failed to set timeout")
	}
	return ctx.JSON(member)
}

func (s *Service) setupClearMemberTimeoutRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/members/:userID/timeout", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.clearMemberTimeout)
}

func (s *Service) clearMemberTimeout(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}

	member, err := s.ChatService.ClearMemberTimeout(ctx.Context(), roomID, actorID, targetID)
	if err != nil {
		if errors.Is(err, chat.ErrNotHost) {
			return utils.Forbidden(ctx, "only room hosts and site moderators can clear timeouts")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.NotFound(ctx, "user is not a member")
		}
		if errors.Is(err, chat.ErrTimeoutLockedByStaff) {
			return utils.Forbidden(ctx, "this timeout can only be removed by site moderators or admins")
		}
		if errors.Is(err, chat.ErrSystemRoom) {
			return utils.Forbidden(ctx, "this room is managed automatically")
		}
		return utils.InternalError(ctx, "failed to clear timeout")
	}
	return ctx.JSON(member)
}

func (s *Service) setupSetRoomNicknameRoute(r fiber.Router) {
	r.Put("/chat/rooms/:roomID/me", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setRoomNickname)
}

func (s *Service) setRoomNickname(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.UpdateMemberProfileRequest](ctx)
	if !ok {
		return nil
	}

	member, err := s.ChatService.SetRoomNickname(ctx.Context(), roomID, userID, req.Nickname)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		if errors.Is(err, chat.ErrNicknameLocked) {
			return utils.Forbidden(ctx, "nickname has been locked by a moderator")
		}
		return utils.InternalError(ctx, "failed to update nickname")
	}
	return ctx.JSON(member)
}

func (s *Service) setupSetRoomAvatarRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/me/avatar", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setRoomAvatar)
}

func (s *Service) setRoomAvatar(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	file, err := ctx.FormFile("avatar")
	if err != nil {
		return utils.BadRequest(ctx, "avatar file is required")
	}
	src, err := file.Open()
	if err != nil {
		return utils.BadRequest(ctx, "failed to read file")
	}
	defer src.Close()

	member, err := s.ChatService.SetRoomAvatar(ctx.Context(), roomID, userID, file.Header.Get("Content-Type"), file.Size, src)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		if errors.Is(err, chat.ErrNicknameLocked) {
			return utils.Forbidden(ctx, "nickname has been locked by a moderator")
		}
		if errors.Is(err, upload.ErrFileTooLarge) || errors.Is(err, upload.ErrInvalidFileType) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to upload avatar")
	}
	return ctx.JSON(member)
}

func (s *Service) setupClearRoomAvatarRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/me/avatar", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.clearRoomAvatar)
}

func (s *Service) clearRoomAvatar(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	member, err := s.ChatService.ClearRoomAvatar(ctx.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		if errors.Is(err, chat.ErrNicknameLocked) {
			return utils.Forbidden(ctx, "nickname has been locked by a moderator")
		}
		return utils.InternalError(ctx, "failed to clear avatar")
	}
	return ctx.JSON(member)
}

func (s *Service) setupSetMemberNicknameAsModRoute(r fiber.Router) {
	r.Put("/chat/rooms/:roomID/members/:userID/nickname", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setMemberNicknameAsMod)
}

func (s *Service) setMemberNicknameAsMod(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.UpdateMemberProfileRequest](ctx)
	if !ok {
		return nil
	}

	member, err := s.ChatService.SetMemberNicknameAsMod(ctx.Context(), roomID, actorID, targetID, req.Nickname)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member of this room")
		}
		if errors.Is(err, chat.ErrModRoleRequired) {
			return utils.Forbidden(ctx, "only site moderators or admins can change another member's nickname")
		}
		if errors.Is(err, chat.ErrTargetImmune) {
			return utils.Forbidden(ctx, "this member's nickname cannot be changed by moderators")
		}
		return utils.InternalError(ctx, "failed to set member nickname")
	}
	return ctx.JSON(member)
}

func (s *Service) setupUnlockMemberNicknameRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID/members/:userID/nickname", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlockMemberNickname)
}

func (s *Service) unlockMemberNickname(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}
	targetID, ok := utils.ParseIDParam(ctx, "userID")
	if !ok {
		return nil
	}

	member, err := s.ChatService.UnlockMemberNickname(ctx.Context(), roomID, actorID, targetID)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member of this room")
		}
		if errors.Is(err, chat.ErrModRoleRequired) {
			return utils.Forbidden(ctx, "only site moderators or admins can unlock a nickname")
		}
		if errors.Is(err, chat.ErrTargetImmune) {
			return utils.Forbidden(ctx, "this member is not affected by nickname locks")
		}
		return utils.InternalError(ctx, "failed to unlock nickname")
	}
	return ctx.JSON(member)
}

func (s *Service) setupPinMessageRoute(r fiber.Router) {
	r.Post("/chat/messages/:messageID/pin", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.pinMessage)
}

func (s *Service) pinMessage(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}

	if err := s.ChatService.PinMessage(ctx.Context(), messageID, userID); err != nil {
		if errors.Is(err, chat.ErrNotHost) {
			return utils.Forbidden(ctx, "only the host can pin messages")
		}
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "message not found")
		}
		return utils.InternalError(ctx, "failed to pin message")
	}
	return utils.OK(ctx)
}

func (s *Service) setupUnpinMessageRoute(r fiber.Router) {
	r.Delete("/chat/messages/:messageID/pin", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unpinMessage)
}

func (s *Service) unpinMessage(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}

	if err := s.ChatService.UnpinMessage(ctx.Context(), messageID, userID); err != nil {
		if errors.Is(err, chat.ErrNotHost) {
			return utils.Forbidden(ctx, "only the host can unpin messages")
		}
		if errors.Is(err, chat.ErrMessageNotPinned) {
			return utils.BadRequest(ctx, "message is not pinned")
		}
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "message not found")
		}
		return utils.InternalError(ctx, "failed to unpin message")
	}
	return utils.OK(ctx)
}

func (s *Service) setupListPinnedMessagesRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/pins", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listPinnedMessages)
}

func (s *Service) listPinnedMessages(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	roomID, ok := utils.ParseIDParam(ctx, "roomID")
	if !ok {
		return nil
	}

	resp, err := s.ChatService.ListPinnedMessages(ctx.Context(), roomID, userID)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		return utils.InternalError(ctx, "failed to list pinned messages")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupAddReactionRoute(r fiber.Router) {
	r.Post("/chat/messages/:messageID/reactions", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.addReaction)
}

func (s *Service) addReaction(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.AddReactionRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ChatService.AddReaction(ctx.Context(), messageID, userID, req.Emoji); err != nil {
		if errors.Is(err, chat.ErrInvalidEmoji) {
			return utils.BadRequest(ctx, "invalid emoji")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		if errors.Is(err, chat.ErrTimedOut) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "message not found")
		}
		return utils.InternalError(ctx, "failed to add reaction")
	}
	return utils.OK(ctx)
}

func (s *Service) setupDeleteMessageRoute(r fiber.Router) {
	r.Delete("/chat/messages/:messageID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteMessage)
}

func (s *Service) deleteMessage(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}

	if err := s.ChatService.DeleteMessage(ctx.Context(), messageID, userID); err != nil {
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "message not found")
		}
		if errors.Is(err, chat.ErrMessageDeletePermission) {
			return utils.Forbidden(ctx, "you do not have permission to delete this message")
		}
		return utils.InternalError(ctx, "failed to delete message")
	}
	return utils.OK(ctx)
}

func (s *Service) setupEditMessageRoute(r fiber.Router) {
	r.Patch("/chat/messages/:messageID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.editMessage)
}

func (s *Service) editMessage(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}

	req, ok := utils.BindJSON[dto.EditMessageRequest](ctx)
	if !ok {
		return nil
	}

	resp, err := s.ChatService.EditMessage(ctx.Context(), messageID, userID, req.Body)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "message not found")
		}
		if errors.Is(err, chat.ErrMessageEditPermission) {
			return utils.Forbidden(ctx, "you can only edit your own messages")
		}
		if errors.Is(err, chat.ErrCannotEditSystemMessage) {
			return utils.BadRequest(ctx, "system messages cannot be edited")
		}
		if errors.Is(err, chat.ErrMissingFields) {
			return utils.BadRequest(ctx, "message body is required")
		}
		if errors.Is(err, chat.ErrTimedOut) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to edit message")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupRemoveReactionRoute(r fiber.Router) {
	r.Delete("/chat/messages/:messageID/reactions/:emoji", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.removeReaction)
}

func (s *Service) removeReaction(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	messageID, ok := utils.ParseIDParam(ctx, "messageID")
	if !ok {
		return nil
	}
	emojiRaw := ctx.Params("emoji")
	emoji, err := url.PathUnescape(emojiRaw)
	if err != nil {
		return utils.BadRequest(ctx, "invalid emoji")
	}

	if err := s.ChatService.RemoveReaction(ctx.Context(), messageID, userID, emoji); err != nil {
		if errors.Is(err, chat.ErrInvalidEmoji) {
			return utils.BadRequest(ctx, "invalid emoji")
		}
		if errors.Is(err, chat.ErrNotMember) {
			return utils.Forbidden(ctx, "not a member")
		}
		if errors.Is(err, chat.ErrRoomNotFound) {
			return utils.NotFound(ctx, "message not found")
		}
		return utils.InternalError(ctx, "failed to remove reaction")
	}
	return utils.OK(ctx)
}
