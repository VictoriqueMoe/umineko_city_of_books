package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

var (
	mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
	chatTriggers = []chatTrigger{
		{
			text:     "<VERY GOOOOOD>",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/46/64501229",
			volume:   0.5,
		},
		{
			text:     "nipah",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/28/92100174",
			volume:   0.5,
		},
		{
			text:     "<OH YEAAAAHHH>",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/46/64501228",
			volume:   0.5,
		},
		{
			text:     "<One more>",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/46/64501230",
			volume:   0.5,
		},
		{
			text:     "<Die the DEATH>",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/47/54600058",
			volume:   0.5,
		},
		{
			text:     "<Sentence to DEATH>",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/47/54600059",
			volume:   0.5,
		},
		{
			text:     "<Great equalizer is the DEATH>",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/47/54600060",
			volume:   0.5,
		},
		{
			text:     "meep",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/28/92100173",
			volume:   0.5,
		},
		{
			text:     "*cackle*cackle*",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/05/11000261",
			volume:   0.5,
		},
		{
			text:     "*giggle*giggle*, *cackle*cackle*cackle*ahahahahahahahahahahahahahahahahahahahahahahahahahahahahahahahahahahahaha!!!",
			audioURL: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/27/20700135",
			volume:   0.5,
		},
	}
)

type (
	messagesService struct {
		*core
		parent *service
	}

	chatTrigger struct {
		text     string
		audioURL string
		volume   float64
	}
)

func (m *messagesService) GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error) {
	isMember, err := m.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, total, err := m.chatRepo.GetMessages(ctx, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	return &dto.ChatMessageListResponse{
		Messages: m.hydrateMessageRows(ctx, userID, rows),
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (m *messagesService) GetMessagesBefore(ctx context.Context, userID, roomID uuid.UUID, before string, limit int) (*dto.ChatMessageListResponse, error) {
	isMember, err := m.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := m.chatRepo.GetMessagesBefore(ctx, roomID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("get messages before: %w", err)
	}

	return &dto.ChatMessageListResponse{
		Messages: m.hydrateMessageRows(ctx, userID, rows),
		Total:    -1,
		Limit:    limit,
	}, nil
}

func (m *messagesService) SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest, files []FileUpload) (*dto.ChatMessageResponse, error) {
	if req.Body == "" && len(files) == 0 {
		return nil, ErrMissingFields
	}
	if req.Body != "" {
		if err := m.filterTexts(ctx, req.Body); err != nil {
			return nil, err
		}
	}

	isMember, err := m.chatRepo.IsMember(ctx, roomID, senderID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	if err := m.ensureLockAllowsRoom(ctx, senderID, roomID); err != nil {
		return nil, err
	}

	if err := m.checkSenderTimeout(ctx, roomID, senderID); err != nil {
		return nil, err
	}

	if err := m.parent.enforceBannedWords(ctx, roomID, senderID, req.Body); err != nil {
		return nil, err
	}

	members, err := m.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}

	roomRow, err := m.chatRepo.GetRoomSendContext(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room send context: %w", err)
	}
	if roomRow == nil {
		return nil, ErrRoomNotFound
	}

	if err := m.assertBlocksAllowSend(ctx, roomRow, senderID, members); err != nil {
		return nil, err
	}

	for i := 0; i < len(files); i++ {
		if err := m.validateMediaFile(ctx, files[i]); err != nil {
			return nil, err
		}
	}

	sender, err := m.userRepo.GetByID(ctx, senderID)
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
		parent, perr := m.chatRepo.GetMessageByID(ctx, *req.ReplyToID)
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
				SenderName:  resolveSenderName(parent.SenderNickname, parent.SenderDisplayName, parent.SenderUsername),
				BodyPreview: preview,
			}
		}
	}

	msgID := uuid.New()
	if err := m.chatRepo.InsertMessage(ctx, msgID, roomID, senderID, req.Body, replyToID); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	mediaResponses, err := m.saveMessageMedia(ctx, msgID, files)
	if err != nil {
		if delErr := m.chatRepo.DeleteMessage(ctx, msgID); delErr != nil {
			logger.Log.Error().Err(delErr).Str("message_id", msgID.String()).Msg("failed to roll back message after media save failure")
		}
		return nil, err
	}

	if err := m.chatRepo.MarkRoomRead(ctx, roomID, senderID); err != nil {
		return nil, fmt.Errorf("mark sender read: %w", err)
	}

	displayName := sender.DisplayName
	avatarURL := sender.AvatarURL
	memberRows, _ := m.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	for _, mr := range memberRows {
		if mr.UserID == senderID {
			if mr.Nickname != "" {
				displayName = mr.Nickname
			}
			if mr.MemberAvatarURL != "" {
				avatarURL = mr.MemberAvatarURL
			}
			break
		}
	}

	senderVanity, _ := m.vanityRoleRepo.GetRolesForUser(ctx, senderID)

	resp := &dto.ChatMessageResponse{
		ID:     msgID,
		RoomID: roomID,
		Sender: dto.UserResponse{
			ID:          sender.ID,
			Username:    sender.Username,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			Role:        role.Role(sender.Role),
			VanityRoles: m.toVanityRoleResponses(senderVanity),
		},
		Body:      req.Body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Media:     mediaResponses,
		ReplyTo:   replyToPreview,
		Reactions: []dto.ReactionGroup{},
	}

	isGroup := roomRow.Type == "group"

	var mentionedIDs map[uuid.UUID]struct{}
	if isGroup {
		mentionedIDs = m.resolveMentions(ctx, req.Body, senderID, members)
	}

	msg := ws.Message{
		Type: "chat_message",
		Data: resp,
	}
	recipients := make([]uuid.UUID, 0, len(members))
	for i := 0; i < len(members); i++ {
		memberID := members[i]
		if memberID == senderID {
			continue
		}
		m.hub.SendToUser(memberID, msg)
		recipients = append(recipients, memberID)
	}

	if !isLiveStreamRoom(roomRow) {
		m.sideEffectsWG.Add(1)
		go m.dispatchPostSendSideEffects(roomID, senderID, msgID, recipients, roomRow, mentionedIDs, replyToAuthor, isGroup)
	}

	for i := 0; i < len(chatTriggers); i++ {
		if req.Body == chatTriggers[i].text {
			audioMsg := ws.Message{
				Type: "chat_audio",
				Data: map[string]any{
					"room_id": roomID.String(),
					"url":     chatTriggers[i].audioURL,
					"volume":  chatTriggers[i].volume,
				},
			}
			for j := 0; j < len(members); j++ {
				m.hub.SendToUser(members[j], audioMsg)
			}
		}
	}

	return resp, nil
}

func isLiveStreamRoom(roomRow *repository.ChatRoomSendContext) bool {
	return roomRow != nil && roomRow.IsSystem && roomRow.SystemKind == SystemKindLiveStream
}

func hostBlockApplies(roomRow *repository.ChatRoomSendContext) bool {
	if !roomRow.IsSystem {
		return true
	}

	return roomRow.SystemKind == SystemKindLiveStream
}

func (m *messagesService) assertBlocksAllowSend(ctx context.Context, roomRow *repository.ChatRoomSendContext, senderID uuid.UUID, members []uuid.UUID) error {
	if roomRow.Type == "dm" {
		for i := 0; i < len(members); i++ {
			if members[i] == senderID {
				continue
			}

			if blocked, _ := m.blockSvc.IsBlockedEither(ctx, senderID, members[i]); blocked {
				return ErrUserBlocked
			}
		}

		return nil
	}

	if !hostBlockApplies(roomRow) || roomRow.CreatedBy == senderID {
		return nil
	}

	if blocked, _ := m.blockSvc.IsBlocked(ctx, roomRow.CreatedBy, senderID); blocked {
		return ErrBlockedByRoomHost
	}

	return nil
}

func (m *messagesService) dispatchPostSendSideEffects(
	roomID, senderID, msgID uuid.UUID,
	recipients []uuid.UUID,
	roomRow *repository.ChatRoomSendContext,
	mentionedIDs map[uuid.UUID]struct{},
	replyToAuthor uuid.UUID,
	isGroup bool,
) {
	defer m.sideEffectsWG.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < len(recipients); i++ {
		memberID := recipients[i]
		inRoom := m.hub.IsUserViewing(roomID, memberID)
		if inRoom {
			continue
		}

		if isGroup {
			_, isMentioned := mentionedIDs[memberID]
			isReplyTarget := replyToAuthor != uuid.Nil && memberID == replyToAuthor

			if isMentioned {
				_ = m.notifSvc.Notify(ctx, dto.NotifyParams{
					RecipientID:   memberID,
					ActorID:       senderID,
					Type:          dto.NotifChatMention,
					ReferenceID:   roomID,
					ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
				})
			} else if isReplyTarget {
				_ = m.notifSvc.Notify(ctx, dto.NotifyParams{
					RecipientID:   memberID,
					ActorID:       senderID,
					Type:          dto.NotifChatReply,
					ReferenceID:   roomID,
					ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
				})
			} else {
				muted, _ := m.chatRepo.IsMuted(ctx, roomID, memberID)
				if !muted {
					roomName := ""
					if roomRow != nil {
						roomName = roomRow.Name
					}
					_ = m.notifSvc.Notify(ctx, dto.NotifyParams{
						RecipientID:   memberID,
						ActorID:       senderID,
						Type:          dto.NotifChatRoomMessage,
						ReferenceID:   roomID,
						ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
						Message:       fmt.Sprintf("sent a message in %s", roomName),
					})
				}
			}
		} else {
			_ = m.notifSvc.Notify(ctx, dto.NotifyParams{
				RecipientID:   memberID,
				ActorID:       senderID,
				Type:          dto.NotifChatMessage,
				ReferenceID:   roomID,
				ReferenceType: "chat",
			})
		}

		total, countErr := m.chatRepo.CountUnreadRoomsForUser(ctx, memberID)
		if countErr == nil {
			m.hub.SendToUser(memberID, ws.Message{
				Type: "chat_unread_bumped",
				Data: map[string]interface{}{
					"room_id": roomID,
					"total":   total,
				},
			})
		}
	}
}

func (m *messagesService) resolveMentions(ctx context.Context, body string, senderID uuid.UUID, members []uuid.UUID) map[uuid.UUID]struct{} {
	matches := mentionRegex.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	usernames := make([]string, 0, len(matches))
	for i := 0; i < len(matches); i++ {
		username := matches[i][1]
		if _, dup := seen[username]; dup {
			continue
		}
		seen[username] = struct{}{}
		usernames = append(usernames, username)
	}
	if len(usernames) == 0 {
		return nil
	}

	users, err := m.userRepo.GetByUsernames(ctx, usernames)
	if err != nil || len(users) == 0 {
		return nil
	}

	memberSet := make(map[uuid.UUID]struct{}, len(members))
	for i := 0; i < len(members); i++ {
		memberSet[members[i]] = struct{}{}
	}

	mentioned := make(map[uuid.UUID]struct{}, len(users))
	for i := 0; i < len(users); i++ {
		uid := users[i].ID
		if uid == senderID {
			continue
		}
		if _, isMember := memberSet[uid]; !isMember {
			continue
		}
		if blocked, _ := m.blockSvc.IsBlockedEither(ctx, senderID, uid); blocked {
			continue
		}
		mentioned[uid] = struct{}{}
	}
	return mentioned
}

func (m *messagesService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := m.chatRepo.CountUnreadRoomsForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

func (m *messagesService) MarkRead(ctx context.Context, roomID, userID uuid.UUID) error {
	isMember, err := m.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := m.chatRepo.MarkRoomRead(ctx, roomID, userID); err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}

	readAt := time.Now().UTC().Format(time.RFC3339)

	total, _ := m.chatRepo.CountUnreadRoomsForUser(ctx, userID)
	m.hub.SendToUser(userID, ws.Message{
		Type: "chat_read",
		Data: map[string]interface{}{
			"room_id": roomID,
			"total":   total,
		},
	})

	members, err := m.chatRepo.GetRoomMembers(ctx, roomID)
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
			m.hub.SendToUser(memberID, receipt)
		}
	}

	return nil
}

func (m *messagesService) GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := m.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}

	var roomIDs []uuid.UUID
	for _, row := range rows {
		roomIDs = append(roomIDs, row.ID)
	}
	return roomIDs, nil
}

func (m *messagesService) validateMediaFile(ctx context.Context, f FileUpload) error {
	isVideo := strings.HasPrefix(f.ContentType, "video/")

	var maxSize int64
	var allowed map[string]string
	var typeErr error
	if isVideo {
		maxSize = int64(m.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		allowed = upload.AllowedVideoTypes
		typeErr = upload.ErrInvalidVideoType
	} else {
		maxSize = int64(m.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		allowed = upload.AllowedImageTypes
		typeErr = upload.ErrInvalidFileType
	}

	if f.Size > maxSize {
		return fmt.Errorf("file size %dMB exceeds maximum %dMB", f.Size/(1024*1024), maxSize/(1024*1024))
	}

	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open media: %w", err)
	}
	defer r.Close()

	sniffed, _, err := upload.DetectContentType(r)
	if err != nil {
		return err
	}
	if _, ok := allowed[sniffed]; !ok {
		return typeErr
	}
	return nil
}

func (m *messagesService) saveMessageMedia(ctx context.Context, messageID uuid.UUID, files []FileUpload) ([]dto.PostMediaResponse, error) {
	if len(files) == 0 {
		return nil, nil
	}

	results := make([]dto.PostMediaResponse, 0, len(files))
	for i := 0; i < len(files); i++ {
		f := files[i]
		r, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open media: %w", err)
		}
		saved, saveErr := m.uploader.SaveAndRecord(ctx, "chat", f.ContentType, f.Size, r,
			func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
				return m.chatRepo.AddMessageMedia(ctx, messageID, mediaURL, mediaType, thumbURL, sortOrder)
			},
			m.chatRepo.UpdateMessageMediaURL,
			m.chatRepo.UpdateMessageMediaThumbnail,
		)
		r.Close()
		if saveErr != nil {
			return nil, saveErr
		}
		results = append(results, *saved)
	}
	return results, nil
}

func (m *messagesService) EditMessage(ctx context.Context, messageID, actorID uuid.UUID, body string) (*dto.ChatMessageResponse, error) {
	if body == "" {
		return nil, ErrMissingFields
	}
	if err := m.filterTexts(ctx, body); err != nil {
		return nil, err
	}

	msg, err := m.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return nil, ErrRoomNotFound
	}
	if msg.IsSystem {
		return nil, ErrCannotEditSystemMessage
	}
	if msg.SenderID != actorID {
		return nil, ErrMessageEditPermission
	}

	if err := m.checkSenderTimeout(ctx, msg.RoomID, actorID); err != nil {
		return nil, err
	}

	if err := m.chatRepo.EditMessage(ctx, messageID, body); err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}

	updated, err := m.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil || updated == nil {
		return nil, fmt.Errorf("reload message: %w", err)
	}

	mediaBatch, _ := m.chatRepo.GetMessageMediaBatch(ctx, []uuid.UUID{messageID})
	reactionBatch, _ := m.chatRepo.GetReactionsBatch(ctx, []uuid.UUID{messageID}, actorID)
	vanityRows, _ := m.vanityRoleRepo.GetRolesForUser(ctx, updated.SenderID)
	resp := m.messageRowToResponse(*updated, mediaBatch[messageID], reactionBatch[messageID], m.toVanityRoleResponses(vanityRows))

	m.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_message_edited",
		Data: resp,
	})

	return &resp, nil
}

func (m *messagesService) DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error {
	msg, err := m.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	if msg.SenderRoleTyped.IsSiteStaff() && msg.SenderID != actorID {
		return ErrCannotDeleteStaffMessage
	}

	if msg.SenderID != actorID {
		canMod, err := m.canModerateRoom(ctx, msg.RoomID, actorID)
		if err != nil {
			return err
		}
		if !canMod {
			return ErrMessageDeletePermission
		}
	}

	if err := m.chatRepo.DeleteMessage(ctx, messageID); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	m.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_message_deleted",
		Data: map[string]interface{}{
			"room_id":    msg.RoomID,
			"message_id": messageID,
		},
	})
	return nil
}
