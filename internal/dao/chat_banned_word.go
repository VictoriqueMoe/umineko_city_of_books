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

func (r *chatBannedWordDAO) Create(ctx context.Context, spec repository.ChatBannedWordSpec) (uuid.UUID, error) {
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

func (r *chatBannedWordDAO) Update(ctx context.Context, id uuid.UUID, spec repository.ChatBannedWordUpdate) error {
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

func (r *chatBannedWordDAO) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_banned_words WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete banned word: %w", err)
	}
	return nil
}

func (r *chatBannedWordDAO) GetByID(ctx context.Context, id uuid.UUID) (*repository.ChatBannedWordRow, error) {
	var row repository.ChatBannedWordRow
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

func (r *chatBannedWordDAO) ListGlobal(ctx context.Context) ([]repository.ChatBannedWordRow, error) {
	return r.queryRows(ctx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global'
		 ORDER BY w.created_at DESC`,
	)
}

func (r *chatBannedWordDAO) ListForRoom(ctx context.Context, roomID uuid.UUID) ([]repository.ChatBannedWordRow, error) {
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

func (r *chatBannedWordDAO) ListApplicable(ctx context.Context, roomID uuid.UUID) ([]repository.ChatBannedWordRow, error) {
	return r.queryRows(ctx,
		`SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
		        w.created_by, COALESCE(u.display_name, u.username, ''), w.created_at
		 FROM chat_banned_words w
		 LEFT JOIN users u ON w.created_by = u.id
		 WHERE w.scope = 'global' OR (w.scope = 'room' AND w.room_id = $1)`,
		roomID,
	)
}

func (r *chatBannedWordDAO) queryRows(ctx context.Context, query string, args ...interface{}) ([]repository.ChatBannedWordRow, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
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
