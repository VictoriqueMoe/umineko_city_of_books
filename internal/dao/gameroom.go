package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	GameRoomDAO interface {
		CreateRoom(ctx context.Context, s spec.NewGameRoom, tx ...*sql.Tx) (*model.GameRoomRow, error)
		AddPlayer(ctx context.Context, s spec.NewGameRoomPlayer, tx ...*sql.Tx) error
		GetRoom(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GameRoomRow, error)
		GetPlayers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.GameRoomPlayerRow, error)
		IsParticipant(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) (bool, error)
		GetPlayerSlot(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) (int, error)
		SetPlayerJoined(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) error
		TouchPlayerSeen(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) error
		SetStatus(ctx context.Context, s spec.GameRoomStatusUpdate, tx ...*sql.Tx) error
		SetState(ctx context.Context, s spec.GameRoomStateUpdate, tx ...*sql.Tx) error
		FinishRoom(ctx context.Context, s spec.GameRoomFinish, tx ...*sql.Tx) error
		AppendMove(ctx context.Context, s spec.NewGameRoomMove, tx ...*sql.Tx) error
		ListMoves(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.GameRoomMoveRow, error)
		NextPly(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error)
		ListForUser(ctx context.Context, q spec.GameRoomUserFilter, tx ...*sql.Tx) ([]model.GameRoomRow, int, error)
		ListLive(ctx context.Context, q spec.GameRoomListFilter, tx ...*sql.Tx) ([]model.GameRoomRow, int, error)
		ListFinished(ctx context.Context, q spec.GameRoomListFilter, tx ...*sql.Tx) ([]model.GameRoomRow, int, error)
		CountLive(ctx context.Context, tx ...*sql.Tx) (int, error)
		Scoreboard(ctx context.Context, gameType string, tx ...*sql.Tx) ([]model.ScoreboardRow, error)
		GetTopWinnerIDs(ctx context.Context, gameType string, tx ...*sql.Tx) ([]string, error)
		ListIdleActive(ctx context.Context, idleSince time.Time, tx ...*sql.Tx) ([]model.GameRoomRow, error)
		CancelIdleRoom(ctx context.Context, s spec.GameRoomIdleCancel, tx ...*sql.Tx) (bool, error)
	}

	gameRoomDAO struct {
		db *sql.DB
	}

	gameRoomJoinRow = sqlcgen.GetGameRoomRow
)

var (
	ErrRoomNotActive = errors.New("game room is not active")
)

func toGameRoomRow(row gameRoomJoinRow) model.GameRoomRow {
	return model.GameRoomRow{
		ID:         row.ID,
		GameType:   row.GameType,
		Status:     row.Status,
		StateJSON:  string(row.StateJson),
		TurnUserID: row.TurnUserID,
		WinnerID:   row.WinnerUserID,
		Result:     row.Result,
		CreatedBy:  row.CreatedBy,
		CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  row.UpdatedAt.UTC().Format(time.RFC3339),
		FinishedAt: nullTimeToStringPtr(row.FinishedAt),
	}
}

func (r *gameRoomDAO) CreateRoom(ctx context.Context, s spec.NewGameRoom, tx ...*sql.Tx) (*model.GameRoomRow, error) {
	created, err := genQueries(r.db, tx).CreateGameRoom(ctx, sqlcgen.CreateGameRoomParams{
		GameType:  s.GameType,
		StateJson: json.RawMessage(s.InitialStateJSON),
		CreatedBy: s.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create game room: %w", err)
	}

	return new(toGameRoomRow(gameRoomJoinRow(created))), nil
}

func (r *gameRoomDAO) AddPlayer(ctx context.Context, s spec.NewGameRoomPlayer, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	var err error
	if s.Joined {
		err = queries.AddGameRoomPlayerJoined(ctx, sqlcgen.AddGameRoomPlayerJoinedParams{
			RoomID: s.RoomID,
			UserID: s.UserID,
			Slot:   int32(s.Slot),
		})
	} else {
		err = queries.AddGameRoomPlayerPending(ctx, sqlcgen.AddGameRoomPlayerPendingParams{
			RoomID: s.RoomID,
			UserID: s.UserID,
			Slot:   int32(s.Slot),
		})
	}
	if err != nil {
		return fmt.Errorf("add player: %w", err)
	}

	return nil
}

func (r *gameRoomDAO) GetRoom(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GameRoomRow, error) {
	row, err := genQueries(r.db, tx).GetGameRoom(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get game room: %w", err)
	}

	return new(toGameRoomRow(row)), nil
}

func (r *gameRoomDAO) GetPlayers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.GameRoomPlayerRow, error) {
	rows, err := genQueries(r.db, tx).GetGameRoomPlayers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get players: %w", err)
	}

	var players []model.GameRoomPlayerRow
	for _, row := range rows {
		players = append(players, model.GameRoomPlayerRow{
			UserID:     row.UserID,
			Slot:       int(row.Slot),
			Joined:     row.Joined,
			JoinedAt:   nullTimeToStringPtr(row.JoinedAt),
			LastSeenAt: row.LastSeenAt.UTC().Format(time.RFC3339),
		})
	}

	return players, nil
}

func (r *gameRoomDAO) IsParticipant(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountGameRoomParticipant(ctx, sqlcgen.CountGameRoomParticipantParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return false, fmt.Errorf("is participant: %w", err)
	}

	return count > 0, nil
}

func (r *gameRoomDAO) GetPlayerSlot(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) (int, error) {
	slot, err := genQueries(r.db, tx).GetGameRoomPlayerSlot(ctx, sqlcgen.GetGameRoomPlayerSlotParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("player not in room")
	}
	if err != nil {
		return 0, fmt.Errorf("get player slot: %w", err)
	}

	return int(slot), nil
}

func (r *gameRoomDAO) SetPlayerJoined(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetGameRoomPlayerJoined(ctx, sqlcgen.SetGameRoomPlayerJoinedParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set player joined: %w", err)
	}

	return nil
}

func (r *gameRoomDAO) TouchPlayerSeen(ctx context.Context, s spec.GameRoomPlayerRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).TouchGameRoomPlayerSeen(ctx, sqlcgen.TouchGameRoomPlayerSeenParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("touch player seen: %w", err)
	}

	return nil
}

func (r *gameRoomDAO) SetStatus(ctx context.Context, s spec.GameRoomStatusUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetGameRoomStatus(ctx, sqlcgen.SetGameRoomStatusParams{
		Status: s.Status,
		ID:     s.RoomID,
	})
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}

	return nil
}

func (r *gameRoomDAO) SetState(ctx context.Context, s spec.GameRoomStateUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetGameRoomState(ctx, sqlcgen.SetGameRoomStateParams{
		StateJson:  json.RawMessage(s.StateJSON),
		TurnUserID: s.TurnUserID,
		ID:         s.RoomID,
	})
	if err != nil {
		return fmt.Errorf("set state: %w", err)
	}

	return nil
}

func (r *gameRoomDAO) FinishRoom(ctx context.Context, s spec.GameRoomFinish, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).FinishGameRoom(ctx, sqlcgen.FinishGameRoomParams{
		Status:       s.Status,
		WinnerUserID: s.WinnerID,
		Result:       sql.NullString{String: s.Result, Valid: true},
		StateJson:    json.RawMessage(s.StateJSON),
		ID:           s.RoomID,
	})
	if err != nil {
		return fmt.Errorf("finish room: %w", err)
	}

	if n == 0 {
		return ErrRoomNotActive
	}

	return nil
}

func (r *gameRoomDAO) AppendMove(ctx context.Context, s spec.NewGameRoomMove, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AppendGameRoomMove(ctx, sqlcgen.AppendGameRoomMoveParams{
		RoomID:     s.RoomID,
		Ply:        int32(s.Ply),
		UserID:     s.UserID,
		ActionJson: json.RawMessage(s.ActionJSON),
	})
	if err != nil {
		return fmt.Errorf("append move: %w", err)
	}

	return nil
}

func (r *gameRoomDAO) ListMoves(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.GameRoomMoveRow, error) {
	rows, err := genQueries(r.db, tx).ListGameRoomMoves(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list moves: %w", err)
	}

	var moves []model.GameRoomMoveRow
	for _, row := range rows {
		moves = append(moves, model.GameRoomMoveRow{
			Ply:       int(row.Ply),
			UserID:    row.UserID,
			ActionRaw: string(row.ActionJson),
			CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return moves, nil
}

func (r *gameRoomDAO) NextPly(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error) {
	ply, err := genQueries(r.db, tx).GetGameRoomNextPly(ctx, roomID)
	if err != nil {
		return 0, fmt.Errorf("next ply: %w", err)
	}

	return int(ply), nil
}

func (r *gameRoomDAO) ListLive(ctx context.Context, q spec.GameRoomListFilter, tx ...*sql.Tx) ([]model.GameRoomRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountLiveGameRooms(ctx, q.GameType)
	if err != nil {
		return nil, 0, fmt.Errorf("count live rooms: %w", err)
	}

	rows, err := queries.ListLiveGameRooms(ctx, sqlcgen.ListLiveGameRoomsParams{
		Column1: q.GameType,
		Limit:   int32(q.Limit),
		Offset:  int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list live rooms: %w", err)
	}

	var out []model.GameRoomRow
	for _, row := range rows {
		out = append(out, toGameRoomRow(gameRoomJoinRow(row)))
	}

	return out, int(total), nil
}

func (r *gameRoomDAO) ListFinished(ctx context.Context, q spec.GameRoomListFilter, tx ...*sql.Tx) ([]model.GameRoomRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountFinishedGameRooms(ctx, q.GameType)
	if err != nil {
		return nil, 0, fmt.Errorf("count finished rooms: %w", err)
	}

	rows, err := queries.ListFinishedGameRooms(ctx, sqlcgen.ListFinishedGameRoomsParams{
		Column1: q.GameType,
		Limit:   int32(q.Limit),
		Offset:  int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list finished rooms: %w", err)
	}

	var out []model.GameRoomRow
	for _, row := range rows {
		out = append(out, toGameRoomRow(gameRoomJoinRow(row)))
	}

	return out, int(total), nil
}

func (r *gameRoomDAO) CountLive(ctx context.Context, tx ...*sql.Tx) (int, error) {
	n, err := genQueries(r.db, tx).CountActiveGameRooms(ctx)
	if err != nil {
		return 0, fmt.Errorf("count live rooms: %w", err)
	}

	return int(n), nil
}

func (r *gameRoomDAO) Scoreboard(ctx context.Context, gameType string, tx ...*sql.Tx) ([]model.ScoreboardRow, error) {
	rows, err := genQueries(r.db, tx).GetGameRoomScoreboard(ctx, gameType)
	if err != nil {
		return nil, fmt.Errorf("scoreboard: %w", err)
	}

	var out []model.ScoreboardRow
	for _, row := range rows {
		out = append(out, model.ScoreboardRow{
			UserID: row.UserID,
			Wins:   int(row.Wins),
			Losses: int(row.Losses),
			Draws:  int(row.Draws),
		})
	}

	return out, nil
}

func (r *gameRoomDAO) GetTopWinnerIDs(ctx context.Context, gameType string, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).GetGameRoomTopWinnerIDs(ctx, gameType)
	if err != nil {
		return nil, fmt.Errorf("get top winner ids: %w", err)
	}

	var out []string
	for _, id := range rows {
		out = append(out, id.String())
	}

	return out, nil
}

func (r *gameRoomDAO) CancelIdleRoom(ctx context.Context, s spec.GameRoomIdleCancel, tx ...*sql.Tx) (bool, error) {
	n, err := genQueries(r.db, tx).CancelIdleGameRoom(ctx, sqlcgen.CancelIdleGameRoomParams{
		ID:        s.RoomID,
		UpdatedAt: s.IdleSince.UTC(),
	})
	if err != nil {
		return false, fmt.Errorf("cancel idle room: %w", err)
	}

	return n > 0, nil
}

func (r *gameRoomDAO) ListIdleActive(ctx context.Context, idleSince time.Time, tx ...*sql.Tx) ([]model.GameRoomRow, error) {
	rows, err := genQueries(r.db, tx).ListIdleActiveGameRooms(ctx, idleSince.UTC())
	if err != nil {
		return nil, fmt.Errorf("list idle active rooms: %w", err)
	}

	var out []model.GameRoomRow
	for _, row := range rows {
		out = append(out, toGameRoomRow(gameRoomJoinRow(row)))
	}

	return out, nil
}

func (r *gameRoomDAO) ListForUser(ctx context.Context, q spec.GameRoomUserFilter, tx ...*sql.Tx) ([]model.GameRoomRow, int, error) {
	statuses := make([]string, 0, len(q.Statuses))
	for _, status := range q.Statuses {
		statuses = append(statuses, string(status))
	}
	statusList := strings.Join(statuses, ",")

	queries := genQueries(r.db, tx)

	total, err := queries.CountGameRoomsForUser(ctx, sqlcgen.CountGameRoomsForUserParams{
		UserID:  q.UserID,
		Column2: q.GameType,
		Column3: statusList,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count rooms: %w", err)
	}

	rows, err := queries.ListGameRoomsForUser(ctx, sqlcgen.ListGameRoomsForUserParams{
		UserID:  q.UserID,
		Column2: q.GameType,
		Column3: statusList,
		Limit:   int32(q.Limit),
		Offset:  int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list rooms: %w", err)
	}

	var out []model.GameRoomRow
	for _, row := range rows {
		out = append(out, toGameRoomRow(gameRoomJoinRow(row)))
	}

	return out, int(total), nil
}
