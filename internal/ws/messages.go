package ws

import "github.com/google/uuid"

const TypingMessageType = "typing"

func TypingMessage(roomID, userID uuid.UUID) Message {
	return Message{
		Type: TypingMessageType,
		Data: map[string]any{
			"room_id": roomID.String(),
			"user_id": userID.String(),
		},
	}
}
