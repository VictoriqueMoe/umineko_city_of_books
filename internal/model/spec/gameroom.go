package spec

import (
	"time"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	NewGameRoomInvite struct {
		GameType         string
		InitialStateJSON string
		InviterID        uuid.UUID
		OpponentID       uuid.UUID
	}

	GameRoomStart struct {
		RoomID     uuid.UUID
		UserID     uuid.UUID
		StateJSON  string
		TurnUserID *uuid.UUID
		Status     string
	}

	NewGameRoom struct {
		GameType         string
		InitialStateJSON string
		CreatedBy        uuid.UUID
	}

	NewGameRoomPlayer struct {
		RoomID uuid.UUID
		UserID uuid.UUID
		Slot   int
		Joined bool
	}

	GameRoomPlayerRef struct {
		RoomID uuid.UUID
		UserID uuid.UUID
	}

	GameRoomStatusUpdate struct {
		RoomID uuid.UUID
		Status string
	}

	GameRoomStateUpdate struct {
		RoomID     uuid.UUID
		StateJSON  string
		TurnUserID *uuid.UUID
	}

	GameRoomFinish struct {
		RoomID    uuid.UUID
		Status    string
		WinnerID  *uuid.UUID
		Result    string
		StateJSON string
	}

	NewGameRoomMove struct {
		RoomID     uuid.UUID
		Ply        int
		UserID     uuid.UUID
		ActionJSON string
	}

	GameRoomUserFilter struct {
		UserID   uuid.UUID
		GameType string
		Statuses []dto.GameStatus
		Limit    int
		Offset   int
	}

	GameRoomListFilter struct {
		GameType string
		Limit    int
		Offset   int
	}

	GameRoomIdleCancel struct {
		RoomID    uuid.UUID
		IdleSince time.Time
	}
)
