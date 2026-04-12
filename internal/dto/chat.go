package dto

import "github.com/google/uuid"

type (
	CreateGroupRoomRequest struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		IsPublic    bool        `json:"is_public"`
		IsRP        bool        `json:"is_rp"`
		Tags        []string    `json:"tags"`
		MemberIDs   []uuid.UUID `json:"member_ids"`
	}

	SendMessageRequest struct {
		Body      string     `json:"body"`
		ReplyToID *uuid.UUID `json:"reply_to_id,omitempty"`
	}

	ChatMessageReplyPreview struct {
		ID          uuid.UUID `json:"id"`
		SenderID    uuid.UUID `json:"sender_id"`
		SenderName  string    `json:"sender_name"`
		BodyPreview string    `json:"body_preview"`
	}

	ResolveDMResponse struct {
		Room      *ChatRoomResponse `json:"room"`
		Recipient UserResponse      `json:"recipient"`
	}

	SendDMResponse struct {
		Room    ChatRoomResponse    `json:"room"`
		Message ChatMessageResponse `json:"message"`
	}

	ChatRoomResponse struct {
		ID            uuid.UUID      `json:"id"`
		Name          string         `json:"name"`
		Description   string         `json:"description"`
		Type          string         `json:"type"`
		IsPublic      bool           `json:"is_public"`
		IsRP          bool           `json:"is_rp"`
		Tags          []string       `json:"tags"`
		ViewerRole    string         `json:"viewer_role,omitempty"`
		IsMember      bool           `json:"is_member"`
		MemberCount   int            `json:"member_count"`
		Members       []UserResponse `json:"members"`
		CreatedAt     string         `json:"created_at"`
		LastMessageAt string         `json:"last_message_at,omitempty"`
		Unread        bool           `json:"unread"`
	}

	ChatRoomMemberResponse struct {
		User     UserResponse `json:"user"`
		Role     string       `json:"role"`
		JoinedAt string       `json:"joined_at"`
	}

	ChatMessageResponse struct {
		ID        uuid.UUID                `json:"id"`
		RoomID    uuid.UUID                `json:"room_id"`
		Sender    UserResponse             `json:"sender"`
		Body      string                   `json:"body"`
		CreatedAt string                   `json:"created_at"`
		Media     []PostMediaResponse      `json:"media,omitempty"`
		ReplyTo   *ChatMessageReplyPreview `json:"reply_to,omitempty"`
	}

	ChatRoomListResponse struct {
		Rooms []ChatRoomResponse `json:"rooms"`
		Total int                `json:"total"`
	}

	ChatMessageListResponse struct {
		Messages []ChatMessageResponse `json:"messages"`
		Total    int                   `json:"total"`
		Limit    int                   `json:"limit"`
		Offset   int                   `json:"offset"`
	}
)
