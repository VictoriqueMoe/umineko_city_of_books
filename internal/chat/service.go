package chat

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
var tagAllowedRegex = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		t = strings.ReplaceAll(t, " ", "-")
		t = tagAllowedRegex.ReplaceAllString(t, "")
		t = strings.Trim(t, "-")
		if t == "" {
			continue
		}
		if len(t) > 30 {
			t = t[:30]
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= 10 {
			break
		}
	}
	return out
}

type (
	Service interface {
		ResolveDMRoom(ctx context.Context, senderID, recipientID uuid.UUID) (*dto.ResolveDMResponse, error)
		SendDMMessage(ctx context.Context, senderID, recipientID uuid.UUID, body string) (*dto.SendDMResponse, error)
		CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error)
		ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error)
		ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, limit, offset int) (*dto.ChatRoomListResponse, error)
		ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, limit, offset int) (*dto.ChatRoomListResponse, error)
		GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error)

		SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest) (*dto.ChatMessageResponse, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error
		JoinRoom(ctx context.Context, roomID, userID uuid.UUID) (*dto.ChatRoomResponse, error)
		LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error
		KickMember(ctx context.Context, hostID, roomID, targetID uuid.UUID) error
		GetMembers(ctx context.Context, viewerID, roomID uuid.UUID) ([]dto.ChatRoomMemberResponse, error)
		GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
		MarkRead(ctx context.Context, roomID, userID uuid.UUID) error
		UploadMessageMedia(ctx context.Context, messageID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
	}

	service struct {
		chatRepo    repository.ChatRepository
		userRepo    repository.UserRepository
		notifSvc    notification.Service
		blockSvc    block.Service
		settingsSvc settings.Service
		uploader    *media.Uploader
		hub         *ws.Hub
	}
)

func NewService(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	notifSvc notification.Service,
	blockSvc block.Service,
	uploadSvc upload.Service,
	settingsSvc settings.Service,
	mediaProc *media.Processor,
	hub *ws.Hub,
) Service {
	return &service{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
		notifSvc:    notifSvc,
		blockSvc:    blockSvc,
		settingsSvc: settingsSvc,
		uploader:    media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		hub:         hub,
	}
}

func (s *service) checkDMPreconditions(ctx context.Context, senderID, recipientID uuid.UUID) (*model.User, error) {
	if senderID == recipientID {
		return nil, ErrCannotDMSelf
	}
	recipient, err := s.userRepo.GetByID(ctx, recipientID)
	if err != nil {
		return nil, fmt.Errorf("get recipient: %w", err)
	}
	if recipient == nil {
		return nil, ErrUserNotFound
	}
	if !recipient.DmsEnabled {
		return nil, ErrDmsDisabled
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, senderID, recipientID); blocked {
		return nil, ErrUserBlocked
	}
	return recipient, nil
}

func (s *service) ResolveDMRoom(ctx context.Context, senderID, recipientID uuid.UUID) (*dto.ResolveDMResponse, error) {
	recipient, err := s.checkDMPreconditions(ctx, senderID, recipientID)
	if err != nil {
		return nil, err
	}

	resp := &dto.ResolveDMResponse{
		Recipient: *recipient.ToResponse(),
	}

	existingID, err := s.chatRepo.FindDMRoom(ctx, senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("find dm room: %w", err)
	}
	if existingID == uuid.Nil {
		return resp, nil
	}

	room, err := s.buildRoomResponse(ctx, existingID, senderID)
	if err != nil {
		return nil, err
	}
	resp.Room = room
	return resp, nil
}

func (s *service) SendDMMessage(ctx context.Context, senderID, recipientID uuid.UUID, body string) (*dto.SendDMResponse, error) {
	if body == "" {
		return nil, ErrMissingFields
	}
	if _, err := s.checkDMPreconditions(ctx, senderID, recipientID); err != nil {
		return nil, err
	}

	roomID, err := s.chatRepo.CreateDMRoomAtomic(ctx, uuid.New(), senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("create dm room: %w", err)
	}

	msgResp, err := s.SendMessage(ctx, senderID, roomID, dto.SendMessageRequest{Body: body})
	if err != nil {
		return nil, err
	}

	roomResp, err := s.buildRoomResponse(ctx, roomID, senderID)
	if err != nil {
		return nil, err
	}

	return &dto.SendDMResponse{
		Room:    *roomResp,
		Message: *msgResp,
	}, nil
}

func (s *service) CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrMissingFields
	}
	if len(name) > 80 {
		name = name[:80]
	}
	description := strings.TrimSpace(req.Description)
	if len(description) > 500 {
		description = description[:500]
	}
	tags := sanitizeTags(req.Tags)

	roomID := uuid.New()
	if err := s.chatRepo.CreateRoom(ctx, roomID, name, description, "group", req.IsPublic, req.IsRP, creatorID); err != nil {
		return nil, fmt.Errorf("create group room: %w", err)
	}
	if len(tags) > 0 {
		if err := s.chatRepo.AddRoomTags(ctx, roomID, tags); err != nil {
			return nil, fmt.Errorf("add room tags: %w", err)
		}
	}
	if err := s.chatRepo.AddMemberWithRole(ctx, roomID, creatorID, "host"); err != nil {
		return nil, fmt.Errorf("add creator to group: %w", err)
	}

	invitedIDs := make([]uuid.UUID, 0, len(req.MemberIDs))
	for _, memberID := range req.MemberIDs {
		if memberID == creatorID {
			continue
		}
		if blocked, _ := s.blockSvc.IsBlockedEither(ctx, creatorID, memberID); blocked {
			continue
		}
		if err := s.chatRepo.AddMemberWithRole(ctx, roomID, memberID, "member"); err != nil {
			return nil, fmt.Errorf("add member to group: %w", err)
		}
		invitedIDs = append(invitedIDs, memberID)
	}

	if len(invitedIDs) > 0 {
		go s.notifyInvited(creatorID, roomID, name, invitedIDs)
	}

	return s.buildRoomResponse(ctx, roomID, creatorID)
}

func (s *service) notifyInvited(inviterID, roomID uuid.UUID, roomName string, invitedIDs []uuid.UUID) {
	bgCtx := context.Background()
	baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
	linkURL := fmt.Sprintf("%s/rooms/%s", baseURL, roomID)

	actorName := "Someone"
	if inviter, err := s.userRepo.GetByID(bgCtx, inviterID); err == nil && inviter != nil {
		actorName = inviter.DisplayName
	}
	subject, body := notification.NotifEmail(actorName, "added you to a chat room", roomName, linkURL)

	for _, invitedID := range invitedIDs {
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   invitedID,
			ActorID:       inviterID,
			Type:          dto.NotifChatRoomInvite,
			ReferenceID:   roomID,
			ReferenceType: "chat_room",
			EmailSubject:  subject,
			EmailBody:     body,
		})
		s.hub.SendToUser(invitedID, ws.Message{
			Type: "chat_room_invited",
			Data: map[string]interface{}{
				"room_id": roomID,
			},
		})
	}
}

func (s *service) ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, limit, offset int) (*dto.ChatRoomListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	tag = strings.ToLower(strings.TrimSpace(tag))
	rows, total, err := s.chatRepo.ListPublicRooms(ctx, search, isRPOnly, tag, viewerID, blockedIDs, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := range rows {
		room := s.rowToResponse(rows[i])
		rooms = append(rooms, room)
	}
	return &dto.ChatRoomListResponse{Rooms: rooms, Total: total}, nil
}

func (s *service) ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, limit, offset int) (*dto.ChatRoomListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if role != "host" && role != "member" {
		role = ""
	}
	tag = strings.ToLower(strings.TrimSpace(tag))
	rows, total, err := s.chatRepo.ListUserGroupRooms(ctx, userID, search, isRPOnly, tag, role, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user group rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := range rows {
		rooms = append(rooms, s.rowToResponse(rows[i]))
	}
	return &dto.ChatRoomListResponse{Rooms: rooms, Total: total}, nil
}

func (s *service) JoinRoom(ctx context.Context, roomID, userID uuid.UUID) (*dto.ChatRoomResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	if row.Type != "group" {
		return nil, ErrNotGroupRoom
	}
	if !row.IsPublic {
		return nil, ErrNotPublic
	}
	if row.IsMember {
		return s.buildRoomResponse(ctx, roomID, userID)
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, row.CreatedBy); blocked {
		return nil, ErrUserBlocked
	}
	cap := s.settingsSvc.GetInt(ctx, config.SettingMaxChatRoomMembers)
	if cap > 0 && row.MemberCount >= cap {
		return nil, ErrRoomFull
	}

	if err := s.chatRepo.AddMemberWithRole(ctx, roomID, userID, "member"); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	resp, err := s.buildRoomResponse(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)
	joiner, _ := s.userRepo.GetByID(ctx, userID)
	if joiner != nil {
		event := ws.Message{
			Type: "chat_member_joined",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user":    joiner.ToResponse(),
			},
		}
		for _, mid := range members {
			s.hub.SendToUser(mid, event)
		}
	}
	return resp, nil
}

func (s *service) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil || !row.IsMember {
		return ErrNotMember
	}
	if row.ViewerRole == "host" {
		return ErrCannotLeaveAsHost
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)

	if err := s.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	leaver, _ := s.userRepo.GetByID(ctx, userID)
	if leaver != nil {
		event := ws.Message{
			Type: "chat_member_left",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user_id": userID,
			},
		}
		for _, mid := range members {
			s.hub.SendToUser(mid, event)
		}
	}
	return nil
}

func (s *service) KickMember(ctx context.Context, hostID, roomID, targetID uuid.UUID) error {
	hostRole, err := s.chatRepo.GetMemberRole(ctx, roomID, hostID)
	if err != nil {
		return fmt.Errorf("get host role: %w", err)
	}
	if hostRole != "host" {
		return ErrNotHost
	}
	targetRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return ErrNotMember
	}
	if targetRole == "host" {
		return ErrCannotKickHost
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)

	if err := s.chatRepo.RemoveMember(ctx, roomID, targetID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	leftEvent := ws.Message{
		Type: "chat_member_left",
		Data: map[string]interface{}{
			"room_id": roomID,
			"user_id": targetID,
		},
	}
	for _, mid := range members {
		if mid == targetID {
			continue
		}
		s.hub.SendToUser(mid, leftEvent)
	}
	s.hub.SendToUser(targetID, ws.Message{
		Type: "chat_kicked",
		Data: map[string]interface{}{
			"room_id": roomID,
		},
	})
	return nil
}

func (s *service) GetMembers(ctx context.Context, viewerID, roomID uuid.UUID) ([]dto.ChatRoomMemberResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, err := s.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	members := make([]dto.ChatRoomMemberResponse, 0, len(rows))
	for i := range rows {
		m := rows[i]
		members = append(members, dto.ChatRoomMemberResponse{
			User: dto.UserResponse{
				ID:          m.UserID,
				Username:    m.Username,
				DisplayName: m.DisplayName,
				AvatarURL:   m.AvatarURL,
				Role:        role.Role(m.AuthorRole),
			},
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		})
	}
	return members, nil
}

func (s *service) ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error) {
	rows, err := s.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		members, err := s.getRoomMemberResponses(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		resp := s.rowToResponse(row)
		resp.Members = members
		rooms = append(rooms, resp)
	}

	return &dto.ChatRoomListResponse{Rooms: rooms}, nil
}

func (s *service) rowToResponse(row repository.ChatRoomRow) dto.ChatRoomResponse {
	return dto.ChatRoomResponse{
		ID:            row.ID,
		Name:          row.Name,
		Description:   row.Description,
		Type:          row.Type,
		IsPublic:      row.IsPublic,
		IsRP:          row.IsRP,
		Tags:          row.Tags,
		ViewerRole:    row.ViewerRole,
		IsMember:      row.IsMember,
		MemberCount:   row.MemberCount,
		CreatedAt:     row.CreatedAt,
		LastMessageAt: nullStr(row.LastMessageAt),
		Unread:        isUnread(row.LastMessageAt, row.LastReadAt),
	}
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func messageRowToResponse(row repository.ChatMessageRow, media []dto.PostMediaResponse) dto.ChatMessageResponse {
	resp := dto.ChatMessageResponse{
		ID:     row.ID,
		RoomID: row.RoomID,
		Sender: dto.UserResponse{
			ID:          row.SenderID,
			Username:    row.SenderUsername,
			DisplayName: row.SenderDisplayName,
			AvatarURL:   row.SenderAvatarURL,
		},
		Body:      row.Body,
		CreatedAt: row.CreatedAt,
		Media:     media,
	}
	if row.ReplyToID != nil && row.ReplyToSenderID != nil && row.ReplyToSenderName != nil && row.ReplyToBody != nil {
		preview := *row.ReplyToBody
		if len(preview) > 140 {
			preview = preview[:140] + "..."
		}
		resp.ReplyTo = &dto.ChatMessageReplyPreview{
			ID:          *row.ReplyToID,
			SenderID:    *row.ReplyToSenderID,
			SenderName:  *row.ReplyToSenderName,
			BodyPreview: preview,
		}
	}
	return resp
}

func isUnread(lastMessageAt, lastReadAt sql.NullString) bool {
	if !lastMessageAt.Valid {
		return false
	}
	if !lastReadAt.Valid {
		return true
	}
	return lastMessageAt.String > lastReadAt.String
}

func (s *service) GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, total, err := s.chatRepo.GetMessages(ctx, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	messageIDs := make([]uuid.UUID, len(rows))
	for i := 0; i < len(rows); i++ {
		messageIDs[i] = rows[i].ID
	}
	mediaBatch, _ := s.chatRepo.GetMessageMediaBatch(ctx, messageIDs)

	messages := make([]dto.ChatMessageResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		messages = append(messages, messageRowToResponse(row, mediaBatch[row.ID]))
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (s *service) SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest) (*dto.ChatMessageResponse, error) {
	if req.Body == "" {
		return nil, ErrMissingFields
	}

	isMember, err := s.chatRepo.IsMember(ctx, roomID, senderID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}
	for _, memberID := range members {
		if memberID != senderID {
			if blocked, _ := s.blockSvc.IsBlockedEither(ctx, senderID, memberID); blocked {
				return nil, ErrUserBlocked
			}
		}
	}

	sender, err := s.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrUserNotFound
	}

	var replyToID *uuid.UUID
	var replyToPreview *dto.ChatMessageReplyPreview
	var replyToAuthor uuid.UUID
	if req.ReplyToID != nil {
		parent, perr := s.chatRepo.GetMessageByID(ctx, *req.ReplyToID)
		if perr == nil && parent != nil && parent.RoomID == roomID {
			replyToID = req.ReplyToID
			replyToAuthor = parent.SenderID
			preview := parent.Body
			if len(preview) > 140 {
				preview = preview[:140] + "..."
			}
			replyToPreview = &dto.ChatMessageReplyPreview{
				ID:          parent.ID,
				SenderID:    parent.SenderID,
				SenderName:  parent.SenderDisplayName,
				BodyPreview: preview,
			}
		}
	}

	msgID := uuid.New()
	if err := s.chatRepo.InsertMessage(ctx, msgID, roomID, senderID, req.Body, replyToID); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	if err := s.chatRepo.MarkRoomRead(ctx, roomID, senderID); err != nil {
		return nil, fmt.Errorf("mark sender read: %w", err)
	}

	resp := &dto.ChatMessageResponse{
		ID:     msgID,
		RoomID: roomID,
		Sender: dto.UserResponse{
			ID:          sender.ID,
			Username:    sender.Username,
			DisplayName: sender.DisplayName,
			AvatarURL:   sender.AvatarURL,
		},
		Body:      req.Body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ReplyTo:   replyToPreview,
	}

	roomRow, _ := s.chatRepo.GetRoomByID(ctx, roomID, senderID)
	isGroup := roomRow != nil && roomRow.Type == "group"

	var mentionedIDs map[uuid.UUID]struct{}
	if isGroup {
		mentionedIDs = s.resolveMentions(ctx, req.Body, roomID, senderID)
	}

	members, err = s.chatRepo.GetRoomMembers(ctx, roomID)
	if err == nil {
		msg := ws.Message{
			Type: "chat_message",
			Data: resp,
		}
		for i := 0; i < len(members); i++ {
			memberID := members[i]
			if memberID == senderID {
				continue
			}
			s.hub.SendToUser(memberID, msg)

			if isGroup {
				if _, isMentioned := mentionedIDs[memberID]; isMentioned {
					_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
						RecipientID:   memberID,
						ActorID:       senderID,
						Type:          dto.NotifChatMention,
						ReferenceID:   roomID,
						ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
					})
				} else if replyToAuthor != uuid.Nil && memberID == replyToAuthor {
					_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
						RecipientID:   memberID,
						ActorID:       senderID,
						Type:          dto.NotifChatReply,
						ReferenceID:   roomID,
						ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
					})
				}
			} else {
				_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
					RecipientID:   memberID,
					ActorID:       senderID,
					Type:          dto.NotifChatMessage,
					ReferenceID:   roomID,
					ReferenceType: "chat",
				})
			}

			if !isGroup {
				total, countErr := s.chatRepo.CountUnreadRoomsForUser(ctx, memberID)
				if countErr == nil {
					s.hub.SendToUser(memberID, ws.Message{
						Type: "chat_unread_bumped",
						Data: map[string]interface{}{
							"room_id": roomID,
							"total":   total,
						},
					})
				}
			}
		}
	}

	return resp, nil
}

func (s *service) resolveMentions(ctx context.Context, body string, roomID, senderID uuid.UUID) map[uuid.UUID]struct{} {
	matches := mentionRegex.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	mentioned := make(map[uuid.UUID]struct{})
	for _, m := range matches {
		username := m[1]
		if _, dup := seen[username]; dup {
			continue
		}
		seen[username] = struct{}{}
		u, err := s.userRepo.GetByUsername(ctx, username)
		if err != nil || u == nil {
			continue
		}
		if u.ID == senderID {
			continue
		}
		isMember, err := s.chatRepo.IsMember(ctx, roomID, u.ID)
		if err != nil || !isMember {
			continue
		}
		if blocked, _ := s.blockSvc.IsBlockedEither(ctx, senderID, u.ID); blocked {
			continue
		}
		mentioned[u.ID] = struct{}{}
	}
	return mentioned
}

func (s *service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.chatRepo.CountUnreadRoomsForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

func (s *service) MarkRead(ctx context.Context, roomID, userID uuid.UUID) error {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := s.chatRepo.MarkRoomRead(ctx, roomID, userID); err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}

	readAt := time.Now().UTC().Format(time.RFC3339)

	total, _ := s.chatRepo.CountUnreadRoomsForUser(ctx, userID)
	s.hub.SendToUser(userID, ws.Message{
		Type: "chat_read",
		Data: map[string]interface{}{
			"room_id": roomID,
			"total":   total,
		},
	})

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err == nil {
		receipt := ws.Message{
			Type: "chat_read_receipt",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user_id": userID,
				"read_at": readAt,
			},
		}
		for i := 0; i < len(members); i++ {
			memberID := members[i]
			if memberID == userID {
				continue
			}
			s.hub.SendToUser(memberID, receipt)
		}
	}

	return nil
}

func (s *service) GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}

	var roomIDs []uuid.UUID
	for _, row := range rows {
		roomIDs = append(roomIDs, row.ID)
	}
	return roomIDs, nil
}

func (s *service) DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil || !row.IsMember {
		return ErrNotMember
	}

	if row.Type == "group" && row.ViewerRole == "host" {
		members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)
		if err := s.chatRepo.DeleteMessages(ctx, roomID); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := s.chatRepo.DeleteRoom(ctx, roomID); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}
		event := ws.Message{
			Type: "chat_room_deleted",
			Data: map[string]interface{}{
				"room_id": roomID,
			},
		}
		for _, mid := range members {
			s.hub.SendToUser(mid, event)
		}
		return nil
	}

	if err := s.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	remaining, err := s.chatRepo.CountRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("count remaining members: %w", err)
	}
	if remaining == 0 {
		if err := s.chatRepo.DeleteMessages(ctx, roomID); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := s.chatRepo.DeleteRoom(ctx, roomID); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}
	}

	return nil
}

func (s *service) UploadMessageMedia(ctx context.Context, messageID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	senderID, err := s.chatRepo.GetMessageSenderID(ctx, messageID)
	if err != nil {
		return nil, ErrRoomNotFound
	}
	if senderID != userID {
		return nil, fmt.Errorf("not the message sender")
	}

	return s.uploader.SaveAndRecord(ctx, "chat", contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.chatRepo.AddMessageMedia(ctx, messageID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.chatRepo.UpdateMessageMediaURL,
		s.chatRepo.UpdateMessageMediaThumbnail,
	)
}

func (s *service) buildRoomResponse(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatRoomResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrNotMember
	}

	members, err := s.getRoomMemberResponses(ctx, roomID)
	if err != nil {
		return nil, err
	}

	resp := s.rowToResponse(*row)
	resp.Members = members
	return &resp, nil
}

func (s *service) getRoomMemberResponses(ctx context.Context, roomID uuid.UUID) ([]dto.UserResponse, error) {
	memberIDs, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}

	var members []dto.UserResponse
	for _, memberID := range memberIDs {
		user, err := s.userRepo.GetByID(ctx, memberID)
		if err != nil || user == nil {
			continue
		}
		members = append(members, *user.ToResponse())
	}
	return members, nil
}
