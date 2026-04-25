package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type (
	ChatBannedWordRepository interface {
		Create(ctx context.Context, spec ChatBannedWordSpec) (uuid.UUID, error)
		Update(ctx context.Context, id uuid.UUID, spec ChatBannedWordUpdate) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*ChatBannedWordRow, error)
		ListGlobal(ctx context.Context) ([]ChatBannedWordRow, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error)
		ListApplicable(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error)
	}

	chatBannedWordRepository struct {
		db *sql.DB
	}

	ChatBannedWordSpec struct {
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
	}

	ChatBannedWordUpdate struct {
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
	}

	ChatBannedWordRow struct {
		ID            uuid.UUID
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
		CreatedByName string
		CreatedAt     string
	}
)

func (r *chatBannedWordRepository) Create(ctx context.Context, spec ChatBannedWordSpec) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_banned_words (id, scope, room_id, pattern, match_mode, case_sensitive, action, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, spec.Scope, spec.RoomID, spec.Pattern, spec.MatchMode, spec.CaseSensitive, spec.Action, spec.CreatedBy,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create banned word: %w", err)
	}
	return id, nil
}

func (r *chatBannedWordRepository) Update(ctx context.Context, id uuid.UUID, spec ChatBannedWordUpdate) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE chat_banned_words SET pattern = $1, match_mode = $2, case_sensitive = $3, action = $4 WHERE id = $5`,
		spec.Pattern, spec.MatchMode, spec.CaseSensitive, spec.Action, id,
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

func (r *chatBannedWordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_banned_words WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete banned word: %w", err)
	}
	return nil
}

func (r *chatBannedWordRepository) GetByID(ctx context.Context, id uuid.UUID) (*ChatBannedWordRow, error) {
	var row ChatBannedWordRow
	var createdByName sql.NullString
	err := r.db.QueryRowContext(ctx,
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

func (r *chatBannedWordRepository) ListGlobal(ctx context.Context) ([]ChatBannedWordRow, error) {
	return r.queryRows(ctx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global'
		 ORDER BY w.created_at DESC`,
	)
}

func (r *chatBannedWordRepository) ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error) {
	return r.queryRows(ctx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'room' AND w.room_id = $1
		 ORDER BY w.created_at DESC`,
		roomID,
	)
}

func (r *chatBannedWordRepository) ListApplicable(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error) {
	return r.queryRows(ctx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global' OR (w.scope = 'room' AND w.room_id = $1)`,
		roomID,
	)
}

func (r *chatBannedWordRepository) queryRows(ctx context.Context, query string, args ...interface{}) ([]ChatBannedWordRow, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query banned words: %w", err)
	}
	defer rows.Close()

	var result []ChatBannedWordRow
	for rows.Next() {
		var row ChatBannedWordRow
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
