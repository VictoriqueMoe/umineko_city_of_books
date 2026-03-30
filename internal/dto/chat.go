package dto

import "github.com/google/uuid"

type (
	CreateDMRequest struct {
		RecipientID uuid.UUID `json:"recipient_id"`
	}

	CreateGroupRoomRequest struct {
		Name      string      `json:"name"`
		MemberIDs []uuid.UUID `json:"member_ids"`
	}

	SendMessageRequest struct {
		Body string `json:"body"`
	}

	ChatRoomResponse struct {
		ID        uuid.UUID      `json:"id"`
		Name      string         `json:"name"`
		Type      string         `json:"type"`
		Members   []UserResponse `json:"members"`
		CreatedAt string         `json:"created_at"`
	}

	ChatMessageResponse struct {
		ID        uuid.UUID    `json:"id"`
		RoomID    uuid.UUID    `json:"room_id"`
		Sender    UserResponse `json:"sender"`
		Body      string       `json:"body"`
		CreatedAt string       `json:"created_at"`
	}

	ChatRoomListResponse struct {
		Rooms []ChatRoomResponse `json:"rooms"`
	}

	ChatMessageListResponse struct {
		Messages []ChatMessageResponse `json:"messages"`
		Total    int                   `json:"total"`
		Limit    int                   `json:"limit"`
		Offset   int                   `json:"offset"`
	}
)
