package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/db"
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

	GameRoomDAO interface {
		CreateRoom(ctx context.Context, gameType, initialStateJSON string, createdBy uuid.UUID, tx ...*sql.Tx) (*GameRoomRow, error)
		AddPlayer(ctx context.Context, roomID, userID uuid.UUID, slot int, joined bool, tx ...*sql.Tx) error
		GetRoom(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*GameRoomRow, error)
		GetPlayers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]GameRoomPlayerRow, error)
		IsParticipant(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetPlayerSlot(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		SetPlayerJoined(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		TouchPlayerSeen(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		SetStatus(ctx context.Context, roomID uuid.UUID, status string, tx ...*sql.Tx) error
		SetState(ctx context.Context, roomID uuid.UUID, stateJSON string, turnUserID *uuid.UUID, tx ...*sql.Tx) error
		FinishRoom(ctx context.Context, roomID uuid.UUID, status string, winner *uuid.UUID, result, stateJSON string, tx ...*sql.Tx) error
		AppendMove(ctx context.Context, roomID uuid.UUID, ply int, userID uuid.UUID, actionJSON string, tx ...*sql.Tx) error
		ListMoves(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]GameRoomMoveRow, error)
		NextPly(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error)
		ListForUser(ctx context.Context, userID uuid.UUID, gameType string, statuses []dto.GameStatus, limit, offset int, tx ...*sql.Tx) ([]GameRoomRow, int, error)
		ListLive(ctx context.Context, gameType string, limit, offset int, tx ...*sql.Tx) ([]GameRoomRow, int, error)
		ListFinished(ctx context.Context, gameType string, limit, offset int, tx ...*sql.Tx) ([]GameRoomRow, int, error)
		CountLive(ctx context.Context, tx ...*sql.Tx) (int, error)
		Scoreboard(ctx context.Context, gameType string, tx ...*sql.Tx) ([]ScoreboardRow, error)
		GetTopWinnerIDs(ctx context.Context, gameType string, tx ...*sql.Tx) ([]string, error)
		ListIdleActive(ctx context.Context, idleSince time.Time, tx ...*sql.Tx) ([]GameRoomRow, error)
		CancelIdleRoom(ctx context.Context, roomID uuid.UUID, idleSince time.Time, tx ...*sql.Tx) (bool, error)
	}

	GameRoomRepository interface {
		GameRoomDAO

		CreateInvite(ctx context.Context, spec NewGameRoomInvite, tx ...*sql.Tx) (*GameRoomRow, error)
		Start(ctx context.Context, spec GameRoomStart, tx ...*sql.Tx) error
	}

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

	ScoreboardRow struct {
		UserID uuid.UUID
		Wins   int
		Losses int
		Draws  int
	}
)

const (
	gameRoomInviterSlot  = 0
	gameRoomOpponentSlot = 1
)

var (
	ErrRoomNotActive = errors.New("game room is not active")
)

type gameRoomRepository struct {
	db    *sql.DB
	dao   GameRoomDAO
	cache *cache.Manager
}

func NewGameRoomRepo(database *sql.DB, dao GameRoomDAO, c *cache.Manager) GameRoomRepository {
	return &gameRoomRepository{db: database, dao: dao, cache: c}
}

func (r *gameRoomRepository) CreateInvite(ctx context.Context, spec NewGameRoomInvite, tx ...*sql.Tx) (*GameRoomRow, error) {
	var created *GameRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateRoom(ctx, spec.GameType, spec.InitialStateJSON, spec.InviterID, tx)
		if err != nil {
			return err
		}

		if err := r.dao.AddPlayer(ctx, created.ID, spec.InviterID, gameRoomInviterSlot, true, tx); err != nil {
			return err
		}

		return r.dao.AddPlayer(ctx, created.ID, spec.OpponentID, gameRoomOpponentSlot, false, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *gameRoomRepository) Start(ctx context.Context, spec GameRoomStart, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.SetPlayerJoined(ctx, spec.RoomID, spec.UserID, tx); err != nil {
			return err
		}

		if err := r.dao.SetState(ctx, spec.RoomID, spec.StateJSON, spec.TurnUserID, tx); err != nil {
			return err
		}

		return r.dao.SetStatus(ctx, spec.RoomID, spec.Status, tx)
	})
}

func (r *gameRoomRepository) CreateRoom(ctx context.Context, gameType, initialStateJSON string, createdBy uuid.UUID, tx ...*sql.Tx) (*GameRoomRow, error) {
	return r.dao.CreateRoom(ctx, gameType, initialStateJSON, createdBy, tx...)
}

func (r *gameRoomRepository) AddPlayer(ctx context.Context, roomID, userID uuid.UUID, slot int, joined bool, tx ...*sql.Tx) error {
	return r.dao.AddPlayer(ctx, roomID, userID, slot, joined, tx...)
}

func (r *gameRoomRepository) GetRoom(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*GameRoomRow, error) {
	return r.dao.GetRoom(ctx, id, tx...)
}

func (r *gameRoomRepository) GetPlayers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]GameRoomPlayerRow, error) {
	return r.dao.GetPlayers(ctx, roomID, tx...)
}

func (r *gameRoomRepository) IsParticipant(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsParticipant(ctx, roomID, userID, tx...)
}

func (r *gameRoomRepository) GetPlayerSlot(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetPlayerSlot(ctx, roomID, userID, tx...)
}

func (r *gameRoomRepository) SetPlayerJoined(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetPlayerJoined(ctx, roomID, userID, tx...)
}

func (r *gameRoomRepository) TouchPlayerSeen(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.TouchPlayerSeen(ctx, roomID, userID, tx...)
}

func (r *gameRoomRepository) SetStatus(ctx context.Context, roomID uuid.UUID, status string, tx ...*sql.Tx) error {
	return r.dao.SetStatus(ctx, roomID, status, tx...)
}

func (r *gameRoomRepository) SetState(ctx context.Context, roomID uuid.UUID, stateJSON string, turnUserID *uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetState(ctx, roomID, stateJSON, turnUserID, tx...)
}

func (r *gameRoomRepository) FinishRoom(ctx context.Context, roomID uuid.UUID, status string, winner *uuid.UUID, result, stateJSON string, tx ...*sql.Tx) error {
	if err := r.dao.FinishRoom(ctx, roomID, status, winner, result, stateJSON, tx...); err != nil {
		return err
	}

	if room, err := r.dao.GetRoom(ctx, roomID, tx...); err == nil && room != nil {
		_ = r.cache.Del(ctx, cache.GameTopWinners.Key(room.GameType))
	}

	return nil
}

func (r *gameRoomRepository) AppendMove(ctx context.Context, roomID uuid.UUID, ply int, userID uuid.UUID, actionJSON string, tx ...*sql.Tx) error {
	return r.dao.AppendMove(ctx, roomID, ply, userID, actionJSON, tx...)
}

func (r *gameRoomRepository) ListMoves(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]GameRoomMoveRow, error) {
	return r.dao.ListMoves(ctx, roomID, tx...)
}

func (r *gameRoomRepository) NextPly(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.NextPly(ctx, roomID, tx...)
}

func (r *gameRoomRepository) ListForUser(ctx context.Context, userID uuid.UUID, gameType string, statuses []dto.GameStatus, limit, offset int, tx ...*sql.Tx) ([]GameRoomRow, int, error) {
	return r.dao.ListForUser(ctx, userID, gameType, statuses, limit, offset, tx...)
}

func (r *gameRoomRepository) ListLive(ctx context.Context, gameType string, limit, offset int, tx ...*sql.Tx) ([]GameRoomRow, int, error) {
	return r.dao.ListLive(ctx, gameType, limit, offset, tx...)
}

func (r *gameRoomRepository) ListFinished(ctx context.Context, gameType string, limit, offset int, tx ...*sql.Tx) ([]GameRoomRow, int, error) {
	return r.dao.ListFinished(ctx, gameType, limit, offset, tx...)
}

func (r *gameRoomRepository) CountLive(ctx context.Context, tx ...*sql.Tx) (int, error) {
	return r.dao.CountLive(ctx, tx...)
}

func (r *gameRoomRepository) Scoreboard(ctx context.Context, gameType string, tx ...*sql.Tx) ([]ScoreboardRow, error) {
	return r.dao.Scoreboard(ctx, gameType, tx...)
}

func (r *gameRoomRepository) GetTopWinnerIDs(ctx context.Context, gameType string, tx ...*sql.Tx) ([]string, error) {
	key := cache.GameTopWinners.Key(gameType)

	if v, err := cache.Get[[]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetTopWinnerIDs(ctx, gameType, tx...)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.GameTopWinners.TTL)
	return v, nil
}

func (r *gameRoomRepository) CancelIdleRoom(ctx context.Context, roomID uuid.UUID, idleSince time.Time, tx ...*sql.Tx) (bool, error) {
	return r.dao.CancelIdleRoom(ctx, roomID, idleSince, tx...)
}

func (r *gameRoomRepository) ListIdleActive(ctx context.Context, idleSince time.Time, tx ...*sql.Tx) ([]GameRoomRow, error) {
	return r.dao.ListIdleActive(ctx, idleSince, tx...)
}
