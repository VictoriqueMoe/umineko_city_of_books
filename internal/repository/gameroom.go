package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
		ListIdleActive(ctx context.Context, idleSince time.Time) ([]GameRoomRow, error)
	}

	ScoreboardRow struct {
		UserID uuid.UUID
		Wins   int
		Losses int
		Draws  int
	}

	gameRoomRepository struct {
		db *sql.DB
	}
)

func (r *gameRoomRepository) CreateRoom(ctx context.Context, id uuid.UUID, gameType, initialStateJSON string, createdBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO game_rooms (id, game_type, status, state_json, created_by) VALUES ($1, $2, 'pending', $3, $4)`,
		id, gameType, initialStateJSON, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create game room: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) AddPlayer(ctx context.Context, roomID, userID uuid.UUID, slot int, joined bool) error {
	var err error
	if joined {
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO game_room_players (room_id, user_id, slot, joined, joined_at) VALUES ($1, $2, $3, TRUE, NOW())`,
			roomID, userID, slot,
		)
	} else {
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO game_room_players (room_id, user_id, slot, joined) VALUES ($1, $2, $3, FALSE)`,
			roomID, userID, slot,
		)
	}
	if err != nil {
		return fmt.Errorf("add player: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) GetRoom(ctx context.Context, id uuid.UUID) (*GameRoomRow, error) {
	var row GameRoomRow
	var result sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, game_type, status, state_json, turn_user_id, winner_user_id, result, created_by, created_at, updated_at, finished_at
         FROM game_rooms WHERE id = $1`, id,
	).Scan(&row.ID, &row.GameType, &row.Status, &row.StateJSON, &row.TurnUserID, &row.WinnerID, &result, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt, &row.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get game room: %w", err)
	}
	if result.Valid {
		row.Result = result.String
	}
	return &row, nil
}

func (r *gameRoomRepository) GetPlayers(ctx context.Context, roomID uuid.UUID) ([]GameRoomPlayerRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, slot, joined, joined_at, last_seen_at FROM game_room_players WHERE room_id = $1 ORDER BY slot`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get players: %w", err)
	}
	defer rows.Close()

	var players []GameRoomPlayerRow
	for rows.Next() {
		var p GameRoomPlayerRow
		if err := rows.Scan(&p.UserID, &p.Slot, &p.Joined, &p.JoinedAt, &p.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan player: %w", err)
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func (r *gameRoomRepository) IsParticipant(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM game_room_players WHERE room_id = $1 AND user_id = $2`, roomID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("is participant: %w", err)
	}
	return count > 0, nil
}

func (r *gameRoomRepository) GetPlayerSlot(ctx context.Context, roomID, userID uuid.UUID) (int, error) {
	var slot int
	err := r.db.QueryRowContext(ctx,
		`SELECT slot FROM game_room_players WHERE room_id = $1 AND user_id = $2`, roomID, userID,
	).Scan(&slot)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("player not in room")
	}
	if err != nil {
		return 0, fmt.Errorf("get player slot: %w", err)
	}
	return slot, nil
}

func (r *gameRoomRepository) SetPlayerJoined(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_room_players SET joined = TRUE, joined_at = COALESCE(joined_at, NOW()), last_seen_at = NOW() WHERE room_id = $1 AND user_id = $2`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set player joined: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) TouchPlayerSeen(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_room_players SET last_seen_at = NOW() WHERE room_id = $1 AND user_id = $2`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("touch player seen: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) SetStatus(ctx context.Context, roomID uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_rooms SET status = $1, updated_at = NOW() WHERE id = $2`, status, roomID,
	)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) SetState(ctx context.Context, roomID uuid.UUID, stateJSON string, turnUserID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_rooms SET state_json = $1, turn_user_id = $2, updated_at = NOW() WHERE id = $3`,
		stateJSON, turnUserID, roomID,
	)
	if err != nil {
		return fmt.Errorf("set state: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) FinishRoom(ctx context.Context, roomID uuid.UUID, status string, winner *uuid.UUID, result, stateJSON string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_rooms SET status = $1, winner_user_id = $2, result = $3, state_json = $4, finished_at = NOW(), updated_at = NOW() WHERE id = $5`,
		status, winner, result, stateJSON, roomID,
	)
	if err != nil {
		return fmt.Errorf("finish room: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) AppendMove(ctx context.Context, roomID uuid.UUID, ply int, userID uuid.UUID, actionJSON string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO game_room_moves (room_id, ply, user_id, action_json) VALUES ($1, $2, $3, $4)`,
		roomID, ply, userID, actionJSON,
	)
	if err != nil {
		return fmt.Errorf("append move: %w", err)
	}
	return nil
}

func (r *gameRoomRepository) ListMoves(ctx context.Context, roomID uuid.UUID) ([]GameRoomMoveRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ply, user_id, action_json, created_at FROM game_room_moves WHERE room_id = $1 ORDER BY ply`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list moves: %w", err)
	}
	defer rows.Close()

	var moves []GameRoomMoveRow
	for rows.Next() {
		var m GameRoomMoveRow
		if err := rows.Scan(&m.Ply, &m.UserID, &m.ActionRaw, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan move: %w", err)
		}
		moves = append(moves, m)
	}
	return moves, rows.Err()
}

func (r *gameRoomRepository) NextPly(ctx context.Context, roomID uuid.UUID) (int, error) {
	var ply sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(ply) FROM game_room_moves WHERE room_id = $1`, roomID,
	).Scan(&ply)
	if err != nil {
		return 0, fmt.Errorf("next ply: %w", err)
	}
	if !ply.Valid {
		return 0, nil
	}
	return int(ply.Int64) + 1, nil
}

func (r *gameRoomRepository) ListLive(ctx context.Context, gameType string, limit, offset int) ([]GameRoomRow, int, error) {
	var clauses []string
	var args []any
	clauses = append(clauses, `status = 'active'`)
	if gameType != "" {
		args = append(args, gameType)
		clauses = append(clauses, fmt.Sprintf(`game_type = $%d`, len(args)))
	}
	where := strings.Join(clauses, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM game_rooms WHERE %s`, where), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count live rooms: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, game_type, status, state_json, turn_user_id, winner_user_id, result, created_by, created_at, updated_at, finished_at
                     FROM game_rooms WHERE %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx), args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list live rooms: %w", err)
	}
	defer rows.Close()

	var out []GameRoomRow
	for rows.Next() {
		var row GameRoomRow
		var result sql.NullString
		if err := rows.Scan(&row.ID, &row.GameType, &row.Status, &row.StateJSON, &row.TurnUserID, &row.WinnerID, &result, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt, &row.FinishedAt); err != nil {
			return nil, 0, fmt.Errorf("scan live room: %w", err)
		}
		if result.Valid {
			row.Result = result.String
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (r *gameRoomRepository) ListFinished(ctx context.Context, gameType string, limit, offset int) ([]GameRoomRow, int, error) {
	var clauses []string
	var args []any
	clauses = append(clauses, `status IN ('finished', 'abandoned')`)
	if gameType != "" {
		args = append(args, gameType)
		clauses = append(clauses, fmt.Sprintf(`game_type = $%d`, len(args)))
	}
	where := strings.Join(clauses, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM game_rooms WHERE %s`, where), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count finished rooms: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, game_type, status, state_json, turn_user_id, winner_user_id, result, created_by, created_at, updated_at, finished_at
                     FROM game_rooms WHERE %s ORDER BY finished_at DESC LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx), args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list finished rooms: %w", err)
	}
	defer rows.Close()

	var out []GameRoomRow
	for rows.Next() {
		var row GameRoomRow
		var result sql.NullString
		if err := rows.Scan(&row.ID, &row.GameType, &row.Status, &row.StateJSON, &row.TurnUserID, &row.WinnerID, &result, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt, &row.FinishedAt); err != nil {
			return nil, 0, fmt.Errorf("scan finished room: %w", err)
		}
		if result.Valid {
			row.Result = result.String
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (r *gameRoomRepository) CountLive(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM game_rooms WHERE status = 'active'`,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("count live rooms: %w", err)
	}
	return n, nil
}

func (r *gameRoomRepository) Scoreboard(ctx context.Context, gameType string) ([]ScoreboardRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.user_id,
                SUM(CASE WHEN r.winner_user_id = p.user_id THEN 1 ELSE 0 END) AS wins,
                SUM(CASE WHEN r.winner_user_id IS NOT NULL AND r.winner_user_id != p.user_id THEN 1 ELSE 0 END) AS losses,
                SUM(CASE WHEN r.winner_user_id IS NULL THEN 1 ELSE 0 END) AS draws
         FROM game_room_players p
         JOIN game_rooms r ON r.id = p.room_id
         WHERE r.game_type = $1 AND r.status IN ('finished', 'abandoned') AND p.joined = TRUE
         GROUP BY p.user_id
         ORDER BY wins DESC, (wins - losses) DESC`,
		gameType,
	)
	if err != nil {
		return nil, fmt.Errorf("scoreboard: %w", err)
	}
	defer rows.Close()

	var out []ScoreboardRow
	for rows.Next() {
		var sr ScoreboardRow
		if err := rows.Scan(&sr.UserID, &sr.Wins, &sr.Losses, &sr.Draws); err != nil {
			return nil, fmt.Errorf("scan scoreboard: %w", err)
		}
		out = append(out, sr)
	}
	return out, rows.Err()
}

func (r *gameRoomRepository) ListIdleActive(ctx context.Context, idleSince time.Time) ([]GameRoomRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, game_type, status, state_json, turn_user_id, winner_user_id, result, created_by, created_at, updated_at, finished_at
         FROM game_rooms WHERE status = 'active' AND updated_at < $1`,
		idleSince.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("list idle active rooms: %w", err)
	}
	defer rows.Close()

	var out []GameRoomRow
	for rows.Next() {
		var row GameRoomRow
		var result sql.NullString
		if err := rows.Scan(&row.ID, &row.GameType, &row.Status, &row.StateJSON, &row.TurnUserID, &row.WinnerID, &result, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt, &row.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan idle active room: %w", err)
		}
		if result.Valid {
			row.Result = result.String
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *gameRoomRepository) ListForUser(ctx context.Context, userID uuid.UUID, gameType string, statuses []dto.GameStatus, limit, offset int) ([]GameRoomRow, int, error) {
	var clauses []string
	args := []any{userID}
	clauses = append(clauses, `EXISTS (SELECT 1 FROM game_room_players p WHERE p.room_id = r.id AND p.user_id = $1)`)
	if gameType != "" {
		args = append(args, gameType)
		clauses = append(clauses, fmt.Sprintf(`r.game_type = $%d`, len(args)))
	}
	if len(statuses) > 0 {
		placeholders := make([]string, len(statuses))
		for i, s := range statuses {
			args = append(args, string(s))
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		clauses = append(clauses, fmt.Sprintf(`r.status IN (%s)`, strings.Join(placeholders, ",")))
	}
	where := strings.Join(clauses, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM game_rooms r WHERE %s`, where), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rooms: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT r.id, r.game_type, r.status, r.state_json, r.turn_user_id, r.winner_user_id, r.result, r.created_by, r.created_at, r.updated_at, r.finished_at
                     FROM game_rooms r WHERE %s ORDER BY r.updated_at DESC LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx), args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	var out []GameRoomRow
	for rows.Next() {
		var row GameRoomRow
		var result sql.NullString
		if err := rows.Scan(&row.ID, &row.GameType, &row.Status, &row.StateJSON, &row.TurnUserID, &row.WinnerID, &result, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt, &row.FinishedAt); err != nil {
			return nil, 0, fmt.Errorf("scan room: %w", err)
		}
		if result.Valid {
			row.Result = result.String
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}
