package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	chatBannedWordDAO struct {
		db *sql.DB
	}
)

func (r *chatBannedWordDAO) Create(ctx context.Context, spec repository.ChatBannedWordSpec, tx ...*sql.Tx) (*repository.ChatBannedWordRow, error) {
	var row repository.ChatBannedWordRow
	var createdByName sql.NullString

	err := getDb(r.db, tx).QueryRowContext(ctx,
		`WITH ins AS (
			INSERT INTO chat_banned_words (scope, room_id, pattern, match_mode, case_sensitive, action, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING *
		)
		SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username), w.created_at
		 FROM ins w
		 LEFT JOIN users u ON w.created_by = u.id`,
		spec.Scope, spec.RoomID, spec.Pattern, spec.MatchMode, spec.CaseSensitive, spec.Action, spec.CreatedBy,
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

func (r *chatBannedWordDAO) Update(ctx context.Context, id uuid.UUID, spec repository.ChatBannedWordUpdate, tx ...*sql.Tx) error {
	res, err := getDb(r.db, tx).ExecContext(ctx,
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

func (r *chatBannedWordDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := getDb(r.db, tx).ExecContext(ctx, `DELETE FROM chat_banned_words WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete banned word: %w", err)
	}
	return nil
}

func (r *chatBannedWordDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*repository.ChatBannedWordRow, error) {
	var row repository.ChatBannedWordRow
	var createdByName sql.NullString
	err := getDb(r.db, tx).QueryRowContext(ctx,
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

func (r *chatBannedWordDAO) ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]repository.ChatBannedWordRow, error) {
	return r.queryRows(ctx, tx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global'
		 ORDER BY w.created_at DESC`,
	)
}

func (r *chatBannedWordDAO) ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]repository.ChatBannedWordRow, error) {
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

func (r *chatBannedWordDAO) ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]repository.ChatBannedWordRow, error) {
	return r.queryRows(ctx, tx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global' OR (w.scope = 'room' AND w.room_id = $1)`,
		roomID,
	)
}

func (r *chatBannedWordDAO) queryRows(ctx context.Context, tx []*sql.Tx, query string, args ...any) ([]repository.ChatBannedWordRow, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query banned words: %w", err)
	}
	defer rows.Close()

	var result []repository.ChatBannedWordRow
	for rows.Next() {
		var row repository.ChatBannedWordRow
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
