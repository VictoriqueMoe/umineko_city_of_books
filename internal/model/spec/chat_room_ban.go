package spec

import (
	"github.com/google/uuid"
)

type (
	NewChatRoomBan struct {
		RoomID   uuid.UUID
		UserID   uuid.UUID
		BannedBy *uuid.UUID
		Reason   string
	}

	ChatRoomUnban struct {
		ChatMemberRef

		ActorID uuid.UUID
	}
)
