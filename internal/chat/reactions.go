package chat

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type reactionsService struct {
	*core
}

func (r *reactionsService) PinMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	msg, err := r.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	canMod, err := r.canModerateRoom(ctx, msg.RoomID, userID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}

	if err := r.chatRepo.PinMessage(ctx, messageID, userID); err != nil {
		return fmt.Errorf("pin message: %w", err)
	}

	r.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_message_pinned",
		Data: map[string]interface{}{
			"room_id":    msg.RoomID,
			"message_id": messageID,
			"pinned_at":  time.Now().UTC().Format(time.RFC3339),
			"pinned_by":  userID,
		},
	})
	return nil
}

func (r *reactionsService) UnpinMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	msg, err := r.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}
	if msg.PinnedAt == nil {
		return ErrMessageNotPinned
	}

	canMod, err := r.canModerateRoom(ctx, msg.RoomID, userID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}

	if err := r.chatRepo.UnpinMessage(ctx, messageID); err != nil {
		return fmt.Errorf("unpin message: %w", err)
	}

	r.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_message_unpinned",
		Data: map[string]interface{}{
			"room_id":    msg.RoomID,
			"message_id": messageID,
		},
	})
	return nil
}

func (r *reactionsService) ListPinnedMessages(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatMessageListResponse, error) {
	isMember, err := r.chatRepo.IsMember(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, err := r.chatRepo.ListPinnedMessages(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list pinned messages: %w", err)
	}

	messages := r.hydrateMessageRows(ctx, viewerID, rows)
	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    len(messages),
	}, nil
}

func (r *reactionsService) resolveMemberDisplayName(ctx context.Context, roomID, userID uuid.UUID) string {
	user, err := r.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return ""
	}
	name := user.DisplayName
	if name == "" {
		name = user.Username
	}

	rows, _ := r.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	for _, mr := range rows {
		if mr.UserID == userID {
			if mr.Nickname != "" {
				name = mr.Nickname
			}
			break
		}
	}
	return name
}

func (r *reactionsService) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if err := validateEmoji(emoji); err != nil {
		return err
	}

	msg, err := r.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	isMember, err := r.chatRepo.IsMember(ctx, msg.RoomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := r.checkSenderTimeout(ctx, msg.RoomID, userID); err != nil {
		return err
	}

	inserted, err := r.chatRepo.AddReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}
	if !inserted {
		return nil
	}

	displayName := r.resolveMemberDisplayName(ctx, msg.RoomID, userID)
	count, _ := r.chatRepo.CountReactions(ctx, messageID, emoji)

	r.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_reaction_added",
		Data: map[string]interface{}{
			"room_id":      msg.RoomID,
			"message_id":   messageID,
			"emoji":        emoji,
			"user_id":      userID,
			"display_name": displayName,
			"count":        count,
		},
	})
	return nil
}

func (r *reactionsService) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if err := validateEmoji(emoji); err != nil {
		return err
	}

	msg, err := r.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	isMember, err := r.chatRepo.IsMember(ctx, msg.RoomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	deleted, err := r.chatRepo.RemoveReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}
	if !deleted {
		return nil
	}

	displayName := r.resolveMemberDisplayName(ctx, msg.RoomID, userID)
	count, _ := r.chatRepo.CountReactions(ctx, messageID, emoji)

	r.broadcastToRoomMembers(ctx, msg.RoomID, ws.Message{
		Type: "chat_reaction_removed",
		Data: map[string]interface{}{
			"room_id":      msg.RoomID,
			"message_id":   messageID,
			"emoji":        emoji,
			"user_id":      userID,
			"display_name": displayName,
			"count":        count,
		},
	})
	return nil
}
