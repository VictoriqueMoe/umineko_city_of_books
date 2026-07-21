package repository

import (
	"context"
	"time"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	GameRoomRow struct {
		ID         uuid.UUID
		GameType   string
		Status     string
		StateJSON  string
		TurnUserID *uuid.UUID
		WinnerID   *uuid.UUID
		Result     string
		CreatedBy  uuid.UUID
		CreatedAt  string
		UpdatedAt  string
		FinishedAt *string
	}

	GameRoomPlayerRow struct {
		UserID     uuid.UUID
		Slot       int
		Joined     bool
		JoinedAt   *string
		LastSeenAt string
	}

	GameRoomMoveRow struct {
		Ply       int
		UserID    uuid.UUID
		ActionRaw string
		CreatedAt string
	}

	GameRoomRepository interface {
		CreateRoom(ctx context.Context, id uuid.UUID, gameType, initialStateJSON string, createdBy uuid.UUID) error
		AddPlayer(ctx context.Context, roomID, userID uuid.UUID, slot int, joined bool) error
		GetRoom(ctx context.Context, id uuid.UUID) (*GameRoomRow, error)
		GetPlayers(ctx context.Context, roomID uuid.UUID) ([]GameRoomPlayerRow, error)
		IsParticipant(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		GetPlayerSlot(ctx context.Context, roomID, userID uuid.UUID) (int, error)
		SetPlayerJoined(ctx context.Context, roomID, userID uuid.UUID) error
		TouchPlayerSeen(ctx context.Context, roomID, userID uuid.UUID) error
		SetStatus(ctx context.Context, roomID uuid.UUID, status string) error
		SetState(ctx context.Context, roomID uuid.UUID, stateJSON string, turnUserID *uuid.UUID) error
		FinishRoom(ctx context.Context, roomID uuid.UUID, status string, winner *uuid.UUID, result, stateJSON string) error
		AppendMove(ctx context.Context, roomID uuid.UUID, ply int, userID uuid.UUID, actionJSON string) error
		ListMoves(ctx context.Context, roomID uuid.UUID) ([]GameRoomMoveRow, error)
		NextPly(ctx context.Context, roomID uuid.UUID) (int, error)
		ListForUser(ctx context.Context, userID uuid.UUID, gameType string, statuses []dto.GameStatus, limit, offset int) ([]GameRoomRow, int, error)
		ListLive(ctx context.Context, gameType string, limit, offset int) ([]GameRoomRow, int, error)
		ListFinished(ctx context.Context, gameType string, limit, offset int) ([]GameRoomRow, int, error)
		CountLive(ctx context.Context) (int, error)
		Scoreboard(ctx context.Context, gameType string) ([]ScoreboardRow, error)
		GetTopWinnerIDs(ctx context.Context, gameType string) ([]string, error)
		ListIdleActive(ctx context.Context, idleSince time.Time) ([]GameRoomRow, error)
	}

	ScoreboardRow struct {
		UserID uuid.UUID
		Wins   int
		Losses int
		Draws  int
	}
)
