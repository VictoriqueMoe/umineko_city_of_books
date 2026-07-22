package repository

import (
	"context"
	"time"

	"umineko_city_of_books/internal/cache"
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

type gameRoomRepository struct {
	dao   GameRoomRepository
	cache *cache.Manager
}

func NewGameRoomRepo(dao GameRoomRepository, c *cache.Manager) GameRoomRepository {
	return &gameRoomRepository{dao: dao, cache: c}
}

func (r *gameRoomRepository) CreateRoom(ctx context.Context, id uuid.UUID, gameType, initialStateJSON string, createdBy uuid.UUID) error {
	return r.dao.CreateRoom(ctx, id, gameType, initialStateJSON, createdBy)
}

func (r *gameRoomRepository) AddPlayer(ctx context.Context, roomID, userID uuid.UUID, slot int, joined bool) error {
	return r.dao.AddPlayer(ctx, roomID, userID, slot, joined)
}

func (r *gameRoomRepository) GetRoom(ctx context.Context, id uuid.UUID) (*GameRoomRow, error) {
	return r.dao.GetRoom(ctx, id)
}

func (r *gameRoomRepository) GetPlayers(ctx context.Context, roomID uuid.UUID) ([]GameRoomPlayerRow, error) {
	return r.dao.GetPlayers(ctx, roomID)
}

func (r *gameRoomRepository) IsParticipant(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.IsParticipant(ctx, roomID, userID)
}

func (r *gameRoomRepository) GetPlayerSlot(ctx context.Context, roomID, userID uuid.UUID) (int, error) {
	return r.dao.GetPlayerSlot(ctx, roomID, userID)
}

func (r *gameRoomRepository) SetPlayerJoined(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.SetPlayerJoined(ctx, roomID, userID)
}

func (r *gameRoomRepository) TouchPlayerSeen(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.TouchPlayerSeen(ctx, roomID, userID)
}

func (r *gameRoomRepository) SetStatus(ctx context.Context, roomID uuid.UUID, status string) error {
	return r.dao.SetStatus(ctx, roomID, status)
}

func (r *gameRoomRepository) SetState(ctx context.Context, roomID uuid.UUID, stateJSON string, turnUserID *uuid.UUID) error {
	return r.dao.SetState(ctx, roomID, stateJSON, turnUserID)
}

func (r *gameRoomRepository) FinishRoom(ctx context.Context, roomID uuid.UUID, status string, winner *uuid.UUID, result, stateJSON string) error {
	if err := r.dao.FinishRoom(ctx, roomID, status, winner, result, stateJSON); err != nil {
		return err
	}

	if room, err := r.dao.GetRoom(ctx, roomID); err == nil && room != nil {
		_ = r.cache.Del(ctx, cache.GameTopWinners.Key(room.GameType))
	}

	return nil
}

func (r *gameRoomRepository) AppendMove(ctx context.Context, roomID uuid.UUID, ply int, userID uuid.UUID, actionJSON string) error {
	return r.dao.AppendMove(ctx, roomID, ply, userID, actionJSON)
}

func (r *gameRoomRepository) ListMoves(ctx context.Context, roomID uuid.UUID) ([]GameRoomMoveRow, error) {
	return r.dao.ListMoves(ctx, roomID)
}

func (r *gameRoomRepository) NextPly(ctx context.Context, roomID uuid.UUID) (int, error) {
	return r.dao.NextPly(ctx, roomID)
}

func (r *gameRoomRepository) ListForUser(ctx context.Context, userID uuid.UUID, gameType string, statuses []dto.GameStatus, limit, offset int) ([]GameRoomRow, int, error) {
	return r.dao.ListForUser(ctx, userID, gameType, statuses, limit, offset)
}

func (r *gameRoomRepository) ListLive(ctx context.Context, gameType string, limit, offset int) ([]GameRoomRow, int, error) {
	return r.dao.ListLive(ctx, gameType, limit, offset)
}

func (r *gameRoomRepository) ListFinished(ctx context.Context, gameType string, limit, offset int) ([]GameRoomRow, int, error) {
	return r.dao.ListFinished(ctx, gameType, limit, offset)
}

func (r *gameRoomRepository) CountLive(ctx context.Context) (int, error) {
	return r.dao.CountLive(ctx)
}

func (r *gameRoomRepository) Scoreboard(ctx context.Context, gameType string) ([]ScoreboardRow, error) {
	return r.dao.Scoreboard(ctx, gameType)
}

func (r *gameRoomRepository) GetTopWinnerIDs(ctx context.Context, gameType string) ([]string, error) {
	key := cache.GameTopWinners.Key(gameType)

	if v, err := cache.Get[[]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetTopWinnerIDs(ctx, gameType)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.GameTopWinners.TTL)
	return v, nil
}

func (r *gameRoomRepository) ListIdleActive(ctx context.Context, idleSince time.Time) ([]GameRoomRow, error) {
	return r.dao.ListIdleActive(ctx, idleSince)
}
