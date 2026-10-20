package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	minClientNonceLength = 16
	maxClientNonceLength = 64
)

var (
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
	if err := m.assertRoomMember(ctx, roomID, userID); err != nil {
		return nil, err
	}

	page := bounds.NewPage(limit, offset)

	rows, total, err := m.chatRepo.GetMessagesForViewer(ctx, spec.ChatMessagePage{RoomID: roomID, ViewerID: userID, Limit: page.Limit(), Offset: page.Offset()})
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	messages, err := m.hydrateMessageRows(ctx, userID, rows)
	if err != nil {
		return nil, err
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    total,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	}, nil
}

func (m *messagesService) GetMessagesBefore(ctx context.Context, userID, roomID uuid.UUID, before string, limit int) (*dto.ChatMessageListResponse, error) {
	if err := m.assertRoomMember(ctx, roomID, userID); err != nil {
		return nil, err
	}

	limit = bounds.NewPage(limit, 0).Limit()

	rows, err := m.chatRepo.GetMessagesBefore(ctx, spec.ChatMessageCursorPage{RoomID: roomID, ViewerID: userID, Before: before, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("get messages before: %w", err)
	}

	messages, err := m.hydrateMessageRows(ctx, userID, rows)
	if err != nil {
		return nil, err
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    -1,
		Limit:    limit,
	}, nil
}

func (m *messagesService) ListRoomAttachments(ctx context.Context, userID, roomID uuid.UUID, kind model.AttachmentKind, before string, limit int) (*dto.ChatMessageListResponse, error) {
	if err := m.assertRoomMember(ctx, roomID, userID); err != nil {
		return nil, err
	}

	limit = bounds.NewPage(limit, 0).Limit()

	rows, err := m.chatRepo.ListRoomAttachments(ctx, spec.ChatRoomAttachmentQuery{
		RoomID: roomID, ViewerID: userID, Before: before, Limit: limit,
		Kind: kind,
	})
	if err != nil {
		return nil, fmt.Errorf("list room attachments: %w", err)
	}

	messages, err := m.hydrateMessageRows(ctx, userID, rows)
	if err != nil {
		return nil, err
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    -1,
		Limit:    limit,
	}, nil
}

func (m *messagesService) SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest, files []upload.FileUpload) (*dto.ChatMessageResponse, error) {
	if req.Body == "" && len(files) == 0 {
		return nil, ErrMissingFields
	}
	if !validClientNonce(req.ClientNonce) {
		return nil, ErrInvalidClientNonce
	}
	if req.Body != "" {
		if err := m.contentFilter.Check(ctx, req.Body); err != nil {
			return nil, err
		}
	}

	if err := m.assertRoomMember(ctx, roomID, senderID); err != nil {
		return nil, err
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

	if err := m.assertBlocksAllowParticipation(ctx, roomRow, senderID, members); err != nil {
		return nil, err
	}

	if err := m.assertRecipientsAcceptNewThread(ctx, roomRow, senderID, members); err != nil {
		return nil, err
	}

	validated, err := upload.ValidateMedia(ctx, m.settingsSvc, files)
	if err != nil {
		return nil, err
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
		parent, err := m.chatRepo.GetMessageByID(ctx, *req.ReplyToID)
		if err != nil {
			return nil, fmt.Errorf("get reply parent: %w", err)
		}
		if parent != nil && parent.RoomID == roomID {
			replyToID = req.ReplyToID
			replyToAuthor = parent.SenderID
			replyToPreview = &dto.ChatMessageReplyPreview{
				ID:         parent.ID,
				SenderID:   parent.SenderID,
				SenderName: resolveSenderName(parent.SenderNickname, parent.SenderDisplayName, parent.SenderUsername),
				Body:       parent.Body,
			}
		}
	}

	created, err := m.chatRepo.InsertMessageAndMarkRead(ctx, spec.NewChatMessage{
		RoomID:      roomID,
		SenderID:    senderID,
		Body:        req.Body,
		ReplyToID:   replyToID,
		ClientNonce: req.ClientNonce,
	})
	if replay, ok := errors.AsType[*daoerr.ChatMessageReplayError](err); ok {
		return m.replayedMessage(ctx, roomID, sender, replay.Existing)
	}
	if err != nil {
		return nil, err
	}

	msgID := created.ID

	mediaResponses, err := validated.Save(ctx, m.uploader, m.uploadSvc, upload.MediaTarget{
		SubDir: "chat",
		Record: func(r upload.MediaRecord) (int64, error) {
			return m.chatRepo.AddMessageMedia(ctx, spec.NewChatMessageMedia{
				TargetID:     msgID,
				MediaURL:     r.MediaURL,
				MediaType:    r.MediaType,
				ThumbnailURL: r.ThumbnailURL,
				Filename:     r.Filename,
				SortOrder:    r.SortOrder,
				IsSpoiler:    r.File.IsSpoiler,
				Width:        r.Width,
				Height:       r.Height,
			})
		},
		UpdateURL:   m.chatRepo.UpdateMessageMediaURL,
		UpdateThumb: m.chatRepo.UpdateMessageMediaThumbnail,
	})
	if err != nil {
		paths, delErr := m.chatRepo.DeleteMessageWithMedia(context.WithoutCancel(ctx), msgID)
		if delErr != nil {
			logger.Ctx(ctx).Error().Err(delErr).Str("message_id", msgID.String()).Msg("failed to roll back message after media save failure")
		}

		m.uploadSvc.Delete(paths...)

		return nil, err
	}

	resp := m.sentMessageResponse(ctx, *created, sender, mediaResponses, replyToPreview)
	displayName := resp.Sender.DisplayName

	var mentionedIDs map[uuid.UUID]struct{}
	if !sender.IsBot {
		mentionedIDs = m.resolveMentions(ctx, req.Body, senderID, members)
	}

	msg := ws.Message{
		Type: "chat_message",
		Data: resp,
	}
	m.hub.SendToUsers(members, msg)

	recipients := make([]uuid.UUID, 0, len(members))
	for _, memberID := range members {
		if memberID == senderID {
			continue
		}

		recipients = append(recipients, memberID)
	}

	if !isEphemeralSystemRoom(roomRow) {
		m.sideEffectsWG.Add(1)
		go m.dispatchPostSendSideEffects(roomID, senderID, msgID, recipients, roomRow, mentionedIDs, replyToAuthor)

		if m.botObserver != nil && !sender.IsBot {
			botEvent := BotMessageEvent{
				RoomID:        roomID,
				RoomType:      string(roomRow.Type),
				SenderID:      senderID,
				SenderName:    displayName,
				SenderHandle:  sender.Username,
				MessageID:     msgID,
				Body:          req.Body,
				Members:       members,
				MentionedIDs:  mentionedIDs,
				ReplyToID:     replyToID,
				ReplyToAuthor: replyToAuthor,
			}

			m.sideEffectsWG.Go(func() {
				m.botObserver.ObserveMessage(botEvent)
			})
		}
	}

	if sender.IsBot {
		return resp, nil
	}

	for i := range chatTriggers {
		if req.Body == chatTriggers[i].text {
			audioMsg := ws.Message{
				Type: "chat_audio",
				Data: map[string]any{
					"room_id": roomID.String(),
					"url":     chatTriggers[i].audioURL,
					"volume":  chatTriggers[i].volume,
				},
			}
			m.hub.SendToUsers(members, audioMsg)
		}
	}

	return resp, nil
}

func (m *messagesService) replayedMessage(ctx context.Context, roomID uuid.UUID, sender *model.User, existing model.ChatMessageRow) (*dto.ChatMessageResponse, error) {
	if existing.RoomID != roomID {
		return nil, ErrClientNonceReused
	}

	mediaBatch, err := m.chatRepo.GetMessageMediaBatch(ctx, []uuid.UUID{existing.ID})
	if err != nil {
		return nil, fmt.Errorf("reload replayed message media: %w", err)
	}

	return m.sentMessageResponse(ctx, existing, sender, mediaBatch[existing.ID], replyPreviewFromRow(existing)), nil
}

func (m *messagesService) sentMessageResponse(
	ctx context.Context,
	row model.ChatMessageRow,
	sender *model.User,
	media []dto.PostMediaResponse,
	replyTo *dto.ChatMessageReplyPreview,
) *dto.ChatMessageResponse {
	displayName := sender.DisplayName
	avatarURL := sender.AvatarURL
	memberRows, err := m.chatRepo.GetRoomMembersDetailed(ctx, row.RoomID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", row.RoomID.String()).Msg("sent message: room alias lookup failed, using the profile name")
	}
	for _, mr := range memberRows {
		if mr.UserID == sender.ID {
			if mr.Nickname != "" {
				displayName = mr.Nickname
			}
			if mr.MemberAvatarURL != "" {
				avatarURL = mr.MemberAvatarURL
			}
			break
		}
	}

	senderVanity, err := m.vanityRoleRepo.GetRolesForUser(ctx, sender.ID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("user_id", sender.ID.String()).Msg("sent message: vanity roles lookup failed")
	}

	return &dto.ChatMessageResponse{
		ID:     row.ID,
		RoomID: row.RoomID,
		Sender: dto.UserResponse{
			ID:          sender.ID,
			Username:    sender.Username,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			Role:        role.Role(sender.Role),
			VanityRoles: m.toVanityRoleResponses(senderVanity),
		},
		Body:        row.Body,
		CreatedAt:   row.CreatedAt,
		Media:       media,
		ReplyTo:     replyTo,
		Reactions:   []dto.ReactionGroup{},
		ClientNonce: row.ClientNonce,
	}
}

func validClientNonce(nonce string) bool {
	if nonce == "" {
		return true
	}
	if len(nonce) < minClientNonceLength || len(nonce) > maxClientNonceLength {
		return false
	}

	for _, r := range nonce {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if !isLetter && !isDigit && r != '-' && r != '_' {
			return false
		}
	}

	return true
}

func (m *messagesService) assertRecipientsAcceptNewThread(ctx context.Context, roomRow *model.ChatRoomSendContext, senderID uuid.UUID, members []uuid.UUID) error {
	if !sendContextCapabilities(roomRow).requiresRecipientOptIn || roomRow.LastMessageAt.Valid {
		return nil
	}

	recipients := make([]uuid.UUID, 0, len(members))
	for _, memberID := range members {
		if memberID == senderID {
			continue
		}

		recipients = append(recipients, memberID)
	}
	if len(recipients) == 0 {
		return nil
	}

	users, err := m.userRepo.GetByIDs(ctx, recipients)
	if err != nil {
		return fmt.Errorf("get thread recipients: %w", err)
	}

	for _, user := range users {
		if !user.DmsEnabled {
			return ErrDmsDisabled
		}
	}

	return nil
}

func isEphemeralSystemRoom(roomRow *model.ChatRoomSendContext) bool {
	if roomRow == nil || !roomRow.IsSystem {
		return false
	}

	return roomRow.SystemKind == SystemKindLiveStream || roomRow.SystemKind == SystemKindWatchParty
}

func (m *messagesService) dispatchPostSendSideEffects(
	roomID, senderID, msgID uuid.UUID,
	recipients []uuid.UUID,
	roomRow *model.ChatRoomSendContext,
	mentionedIDs map[uuid.UUID]struct{},
	replyToAuthor uuid.UUID,
) {
	defer m.sideEffectsWG.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgRef := fmt.Sprintf("chat_message:%s", msgID)

	fallbackType, fallbackMessage := plainMessageNotification(roomRow)
	caps := sendContextCapabilities(roomRow)

	away := make([]uuid.UUID, 0, len(recipients))
	for _, memberID := range recipients {
		if m.hub.IsUserViewing(roomID, memberID) {
			continue
		}

		away = append(away, memberID)
	}
	if len(away) == 0 {
		return
	}

	muted := m.mutedRecipients(ctx, roomID, away)

	params := make([]dto.NotifyParams, 0, len(away))
	for _, memberID := range away {
		_, isMentioned := mentionedIDs[memberID]
		isReplyTarget := replyToAuthor != uuid.Nil && memberID == replyToAuthor
		_, isMuted := muted[memberID]

		switch {
		case isMentioned:
			params = append(params, dto.NotifyParams{
				RecipientID:   memberID,
				ActorID:       senderID,
				Type:          dto.NotifChatMention,
				ReferenceID:   roomID,
				ReferenceType: msgRef,
			})
		case isReplyTarget && !isMuted:
			params = append(params, dto.NotifyParams{
				RecipientID:   memberID,
				ActorID:       senderID,
				Type:          dto.NotifChatReply,
				ReferenceID:   roomID,
				ReferenceType: msgRef,
			})
		case !isMuted:
			params = append(params, dto.NotifyParams{
				RecipientID:   memberID,
				ActorID:       senderID,
				Type:          fallbackType,
				ReferenceID:   roomID,
				ReferenceType: msgRef,
				Message:       fallbackMessage,
			})
		}
	}

	if len(params) > 0 {
		m.notifSvc.NotifyMany(ctx, params)
	}

	if !caps.countsTowardsUnread {
		return
	}

	for _, memberID := range away {
		if _, isMuted := muted[memberID]; isMuted {
			continue
		}

		total, err := m.chatRepo.CountUnreadRoomsForUser(ctx, memberID)
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("user_id", memberID.String()).Msg("unread bump skipped, count failed")

			continue
		}

		m.hub.SendToUser(memberID, ws.Message{
			Type: "chat_unread_bumped",
			Data: map[string]any{
				"room_id": roomID,
				"total":   total,
			},
		})
	}
}

func (m *messagesService) mutedRecipients(ctx context.Context, roomID uuid.UUID, recipients []uuid.UUID) map[uuid.UUID]struct{} {
	muted := make(map[uuid.UUID]struct{}, len(recipients))

	ids, err := m.chatRepo.ListMutedMemberIDs(ctx, spec.ChatRoomMemberSet{RoomID: roomID, UserIDs: recipients})
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Int("recipients", len(recipients)).Msg("mute state unknown, treating every recipient as muted")
		ids = recipients
	}

	for _, id := range ids {
		muted[id] = struct{}{}
	}

	return muted
}

func (m *messagesService) resolveMentions(ctx context.Context, body string, senderID uuid.UUID, members []uuid.UUID) map[uuid.UUID]struct{} {
	matches := mention.Pattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	usernames := make([]string, 0, len(matches))
	for i := range matches {
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
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("sender_id", senderID.String()).Msg("mentions dropped, username lookup failed")

		return nil
	}
	if len(users) == 0 {
		return nil
	}

	memberSet := make(map[uuid.UUID]struct{}, len(members))
	for i := range members {
		memberSet[members[i]] = struct{}{}
	}

	mentioned := make(map[uuid.UUID]struct{}, len(users))
	for i := range users {
		uid := users[i].ID
		if uid == senderID {
			continue
		}
		if _, isMember := memberSet[uid]; !isMember {
			continue
		}
		blocked, err := m.blockSvc.IsBlockedEither(ctx, senderID, uid)
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("sender_id", senderID.String()).Str("user_id", uid.String()).Msg("mention dropped, block check failed")

			continue
		}
		if blocked {
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
	if err := m.assertRoomMember(ctx, roomID, userID); err != nil {
		return err
	}

	if err := m.chatRepo.MarkRoomRead(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: userID}); err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}

	readAt := time.Now().UTC().Format(time.RFC3339)

	if err := m.notifSvc.MarkChatRoomRead(ctx, userID, roomID); err != nil {
		return fmt.Errorf("mark chat room notifications read: %w", err)
	}

	sendCtx, err := m.chatRepo.GetRoomSendContext(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room send context: %w", err)
	}

	caps := sendContextCapabilities(sendCtx)

	read := map[string]any{"room_id": roomID}
	if caps.countsTowardsUnread {
		total, countErr := m.chatRepo.CountUnreadRoomsForUser(ctx, userID)
		if countErr != nil {
			return fmt.Errorf("count unread rooms: %w", countErr)
		}

		read["total"] = total
	}

	m.hub.SendToUser(userID, ws.Message{Type: "chat_read", Data: read})

	if caps.pairwiseReadReceipts {
		m.fanOutReadReceipt(ctx, roomID, userID, readAt)
	}

	return nil
}

func (m *messagesService) fanOutReadReceipt(ctx context.Context, roomID, readerID uuid.UUID, readAt string) {
	members, err := m.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("read receipt broadcast: member lookup failed")

		return
	}

	receipt := ws.Message{
		Type: "chat_read_receipt",
		Data: map[string]any{
			"room_id": roomID,
			"user_id": readerID,
			"read_at": readAt,
		},
	}
	for _, memberID := range members {
		if memberID == readerID {
			continue
		}

		m.hub.SendToUser(memberID, receipt)
	}
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

func (m *messagesService) IsRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	isMember, err := m.chatRepo.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}

	return isMember, nil
}

func (m *messagesService) EditMessage(ctx context.Context, messageID, actorID uuid.UUID, body string) (*dto.ChatMessageResponse, error) {
	if body == "" {
		return nil, ErrMissingFields
	}
	if err := m.contentFilter.Check(ctx, body); err != nil {
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

	if err := m.assertRoomMember(ctx, msg.RoomID, actorID); err != nil {
		return nil, err
	}

	if err := m.checkSenderTimeout(ctx, msg.RoomID, actorID); err != nil {
		return nil, err
	}

	if err := m.parent.enforceBannedWords(ctx, msg.RoomID, actorID, body); err != nil {
		return nil, err
	}

	if err := m.chatRepo.EditMessage(ctx, spec.ChatMessageUpdate{MessageID: messageID, Body: body}); err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}

	updated, err := m.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("reload message: %w", err)
	}
	if updated == nil {
		return nil, ErrRoomNotFound
	}

	mediaBatch, err := m.chatRepo.GetMessageMediaBatch(ctx, []uuid.UUID{messageID})
	if err != nil {
		return nil, fmt.Errorf("reload message media: %w", err)
	}

	reactionBatch, err := m.chatRepo.GetReactionsBatch(ctx, spec.ChatReactionsQuery{MessageIDs: []uuid.UUID{messageID}, ViewerID: actorID})
	if err != nil {
		return nil, fmt.Errorf("reload message reactions: %w", err)
	}

	vanityRows, err := m.vanityRoleRepo.GetRolesForUser(ctx, updated.SenderID)
	if err != nil {
		return nil, fmt.Errorf("reload sender roles: %w", err)
	}

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

	modKind := ""
	if msg.SenderID != actorID {
		kind, err := m.moderatorKind(ctx, msg.RoomID, actorID)
		if err != nil {
			return err
		}
		if kind == "" {
			return ErrMessageDeletePermission
		}

		modKind = kind
	}

	paths, err := m.chatRepo.DeleteMessageWithMedia(ctx, messageID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	m.uploadSvc.Delete(paths...)

	m.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_message_deleted",
		Data: map[string]any{
			"room_id":    msg.RoomID,
			"message_id": messageID,
		},
	})

	m.auditMessageDelete(ctx, *msg, actorID, modKind)

	return nil
}

func (m *messagesService) auditMessageDelete(ctx context.Context, msg model.ChatMessageRow, actorID uuid.UUID, modKind string) {
	room, err := m.chatRepo.GetRoomSendContext(ctx, msg.RoomID)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("room_id", msg.RoomID.String()).Msg("audit chat message delete: get room send context failed")
		return
	}
	if !isAuditableSendContext(room) {
		return
	}

	action := audit.ActionChatMessageDelete
	details := fmt.Sprintf("message=%s", msg.ID)
	if modKind != "" {
		action = audit.ActionChatMessageDeleteMod
		details = fmt.Sprintf("message=%s by=%s", msg.ID, modKind)
	}

	m.writeAudit(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: audit.TargetChatRoom,
		TargetID:   msg.RoomID.String(),
		Details:    details,
		SubjectID:  msg.SenderID,
	})
}
