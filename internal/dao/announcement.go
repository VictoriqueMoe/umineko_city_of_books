package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	announcementDAO struct {
		db *sql.DB
		*commentDAO[uuid.UUID]
	}
)

const announcementSelectBase = `SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
	u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
	FROM announcements a
	JOIN users u ON a.author_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id`

func scanAnnouncementRow(scanner interface {
	Scan(dest ...interface{}) error
}, row *repository.AnnouncementRow) error {
	var (
		createdAt time.Time
		updatedAt time.Time
	)
	err := scanner.Scan(
		&row.ID, &row.Title, &row.Body, &row.AuthorID, &row.Pinned, &createdAt, &updatedAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
	)
	if err != nil {
		return err
	}
	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	row.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return nil
}

func (r *announcementDAO) Create(ctx context.Context, id uuid.UUID, authorID uuid.UUID, title string, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO announcements (id, author_id, title, body) VALUES ($1, $2, $3, $4)`,
		id, authorID, title, body,
	)
	if err != nil {
		return fmt.Errorf("create announcement: %w", err)
	}
	return nil
}

func (r *announcementDAO) Update(ctx context.Context, id uuid.UUID, title string, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE announcements SET title = $1, body = $2, updated_at = NOW() WHERE id = $3`,
		title, body, id,
	)
	if err != nil {
		return fmt.Errorf("update announcement: %w", err)
	}
	return nil
}

func (r *announcementDAO) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

func (r *announcementDAO) GetByID(ctx context.Context, id uuid.UUID) (*repository.AnnouncementRow, error) {
	var row repository.AnnouncementRow
	err := scanAnnouncementRow(
		r.db.QueryRowContext(ctx, announcementSelectBase+` WHERE a.id = $1`, id),
		&row,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return &row, nil
}

func (r *announcementDAO) List(ctx context.Context, limit, offset int) ([]repository.AnnouncementRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM announcements`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		announcementSelectBase+` ORDER BY a.pinned DESC, a.created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	var result []repository.AnnouncementRow
	for rows.Next() {
		var row repository.AnnouncementRow
		if err := scanAnnouncementRow(rows, &row); err != nil {
			return nil, 0, fmt.Errorf("scan announcement: %w", err)
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *announcementDAO) GetLatest(ctx context.Context) (*repository.AnnouncementRow, error) {
	var row repository.AnnouncementRow
	err := scanAnnouncementRow(
		r.db.QueryRowContext(ctx, announcementSelectBase+` ORDER BY a.pinned DESC, a.created_at DESC LIMIT 1`),
		&row,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest announcement: %w", err)
	}
	return &row, nil
}

func (r *announcementDAO) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE announcements SET pinned = $1 WHERE id = $2`, pinned, id)
	if err != nil {
		return fmt.Errorf("set pinned: %w", err)
	}
	return nil
}
