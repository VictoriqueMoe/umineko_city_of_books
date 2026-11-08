package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

var (
	leaderboardGameTypes = []dto.GameType{
		dto.GameTypeChess,
		dto.GameTypeCheckers,
		dto.GameTypeOthello,
		dto.GameTypeMinesweeper,
		dto.GameTypeSnakesLadders,
	}
)

type (
	GameRoomRepository interface {
		dao.GameRoomDAO

		CreateInvite(ctx context.Context, invite spec.NewGameRoomInvite, tx ...*sql.Tx) (*model.GameRoomRow, error)
		Start(ctx context.Context, start spec.GameRoomStart, tx ...*sql.Tx) error
		InvalidateTopWinners(ctx context.Context, gameTypes ...dto.GameType)
	}

	gameRoomRepository struct {
		db *sql.DB
		dao.GameRoomDAO
		cache *cache.Manager
	}
)

const (
	gameRoomInviterSlot  = 0
	gameRoomOpponentSlot = 1
)

func NewGameRoomRepo(database *sql.DB, roomDAO dao.GameRoomDAO, c *cache.Manager) GameRoomRepository {
	return &gameRoomRepository{db: database, GameRoomDAO: roomDAO, cache: c}
}

func (r *gameRoomRepository) CreateInvite(ctx context.Context, invite spec.NewGameRoomInvite, tx ...*sql.Tx) (*model.GameRoomRow, error) {
	var created *model.GameRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.GameRoomDAO.CreateRoom(ctx, spec.NewGameRoom{
			GameType:         invite.GameType,
			InitialStateJSON: invite.InitialStateJSON,
			CreatedBy:        invite.InviterID,
		}, tx)
		if err != nil {
			return err
		}

		if err := r.GameRoomDAO.AddPlayer(ctx, spec.NewGameRoomPlayer{
			RoomID: created.ID,
			UserID: invite.InviterID,
			Slot:   gameRoomInviterSlot,
			Joined: true,
		}, tx); err != nil {
			return err
		}

		return r.GameRoomDAO.AddPlayer(ctx, spec.NewGameRoomPlayer{
			RoomID: created.ID,
			UserID: invite.OpponentID,
			Slot:   gameRoomOpponentSlot,
			Joined: false,
		}, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *gameRoomRepository) Start(ctx context.Context, start spec.GameRoomStart, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.GameRoomDAO.SetPlayerJoined(ctx, spec.GameRoomPlayerRef{
			RoomID: start.RoomID,
			UserID: start.UserID,
		}, tx); err != nil {
			return err
		}

		if err := r.GameRoomDAO.SetStatus(ctx, spec.GameRoomStatusUpdate{
			RoomID: start.RoomID,
			Status: start.Status,
		}, tx); err != nil {
			return err
		}

		return r.GameRoomDAO.SetState(ctx, spec.GameRoomStateUpdate{
			RoomID:     start.RoomID,
			StateJSON:  start.StateJSON,
			TurnUserID: start.TurnUserID,
		}, tx)
	})
}

func (r *gameRoomRepository) FinishRoom(ctx context.Context, s spec.GameRoomFinish, tx ...*sql.Tx) error {
	room, err := r.GameRoomDAO.GetRoom(ctx, s.RoomID, tx...)
	if err != nil {
		return err
	}

	if err := r.GameRoomDAO.FinishRoom(ctx, s, tx...); err != nil {
		return err
	}

	if room != nil {
		r.InvalidateTopWinners(ctx, dto.GameType(room.GameType))
	}

	return nil
}

func (r *gameRoomRepository) InvalidateTopWinners(ctx context.Context, gameTypes ...dto.GameType) {
	keys := make([]cache.Key, 0, len(gameTypes))
	for _, gameType := range gameTypes {
		keys = append(keys, cache.GameTopWinners.Key(string(gameType)))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate game top winner caches after write")
	}
}

func (r *gameRoomRepository) GetTopWinnerIDs(ctx context.Context, gameType string, tx ...*sql.Tx) ([]string, error) {
	load := func(ctx context.Context) ([]string, error) {
		return r.GameRoomDAO.GetTopWinnerIDs(ctx, gameType, tx...)
	}

	return r.cache.Load(ctx, cache.GameTopWinners.Key(gameType), load)
}
