package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatBannedWordDAO interface {
		Create(ctx context.Context, s spec.ChatBannedWordSpec, tx ...*sql.Tx) (*model.ChatBannedWordRow, error)
		Update(ctx context.Context, s spec.ChatBannedWordUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatBannedWordRow, error)
		ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error)
		ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error)
	}

	chatBannedWordDAO struct {
		db *sql.DB
	}
)

func (r *chatBannedWordDAO) Create(ctx context.Context, s spec.ChatBannedWordSpec, tx ...*sql.Tx) (*model.ChatBannedWordRow, error) {
	var row model.ChatBannedWordRow
	var createdByName sql.NullString

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH ins AS (
			INSERT INTO chat_banned_words (scope, room_id, pattern, match_mode, case_sensitive, action, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING *
		)
		SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username), w.created_at
		 FROM ins w
		 LEFT JOIN users u ON w.created_by = u.id`,
		s.Scope, s.RoomID, s.Pattern, s.MatchMode, s.CaseSensitive, s.Action, s.CreatedBy,
	).Scan(&row.ID, &row.Scope, &row.RoomID, &row.Pattern, &row.MatchMode, &row.CaseSensitive, &row.Action,
		&row.CreatedBy, &createdByName, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create banned word: %w", err)
	}

	if createdByName.Valid {
		row.CreatedByName = createdByName.String
	}

	return &row, nil
}

func (r *chatBannedWordDAO) Update(ctx context.Context, s spec.ChatBannedWordUpdate, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_banned_words SET pattern = $1, match_mode = $2, case_sensitive = $3, action = $4 WHERE id = $5`,
		s.Pattern, s.MatchMode, s.CaseSensitive, s.Action, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update banned word: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *chatBannedWordDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chat_banned_words WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete banned word: %w", err)
	}

	return nil
}

func (r *chatBannedWordDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatBannedWordRow, error) {
	var row model.ChatBannedWordRow
	var createdByName sql.NullString

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.id = $1`,
		id,
	).Scan(&row.ID, &row.Scope, &row.RoomID, &row.Pattern, &row.MatchMode, &row.CaseSensitive, &row.Action,
		&row.CreatedBy, &createdByName, &row.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get banned word: %w", err)
	}

	if createdByName.Valid {
		row.CreatedByName = createdByName.String
	}

	return &row, nil
}

func (r *chatBannedWordDAO) ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	return r.queryRows(ctx, tx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global'
		 ORDER BY w.created_at DESC`,
	)
}

func (r *chatBannedWordDAO) ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	return r.queryRows(ctx, tx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'room' AND w.room_id = $1
		 ORDER BY w.created_at DESC`,
		roomID,
	)
}

func (r *chatBannedWordDAO) ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	return r.queryRows(ctx, tx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global' OR (w.scope = 'room' AND w.room_id = $1)`,
		roomID,
	)
}

func (r *chatBannedWordDAO) queryRows(ctx context.Context, tx []*sql.Tx, query string, args ...any) ([]model.ChatBannedWordRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query banned words: %w", err)
	}
	defer rows.Close()

	var result []model.ChatBannedWordRow

	for rows.Next() {
		var row model.ChatBannedWordRow
		var createdByName sql.NullString

		if err := rows.Scan(&row.ID, &row.Scope, &row.RoomID, &row.Pattern, &row.MatchMode, &row.CaseSensitive,
			&row.Action, &row.CreatedBy, &createdByName, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan banned word: %w", err)
		}

		if createdByName.Valid {
			row.CreatedByName = createdByName.String
		}

		result = append(result, row)
	}

	return result, rows.Err()
}
