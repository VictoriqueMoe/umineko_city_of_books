package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	AnnouncementRepository interface {
		Create(ctx context.Context, id uuid.UUID, authorID uuid.UUID, title string, body string) error
		Update(ctx context.Context, id uuid.UUID, title string, body string) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*AnnouncementRow, error)
		List(ctx context.Context, limit, offset int) ([]AnnouncementRow, int, error)
		GetLatest(ctx context.Context) (*AnnouncementRow, error)
		SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error
	}

	AnnouncementRow struct {
		ID                uuid.UUID
		Title             string
		Body              string
		AuthorID          uuid.UUID
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		Pinned            bool
		CreatedAt         string
		UpdatedAt         string
	}

	announcementRepository struct {
		db *sql.DB
	}
)

const announcementSelectBase = `SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
	u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
	FROM announcements a
	JOIN users u ON a.author_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id`

func scanAnnouncementRow(scanner interface {
	Scan(dest ...interface{}) error
}, row *AnnouncementRow) error {
	var pinned int
	err := scanner.Scan(
		&row.ID, &row.Title, &row.Body, &row.AuthorID, &pinned, &row.CreatedAt, &row.UpdatedAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
	)
	row.Pinned = pinned != 0
	return err
}

func (r *announcementRepository) Create(ctx context.Context, id uuid.UUID, authorID uuid.UUID, title string, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO announcements (id, author_id, title, body) VALUES (?, ?, ?, ?)`,
		id, authorID, title, body,
	)
	if err != nil {
		return fmt.Errorf("create announcement: %w", err)
	}
	return nil
}

func (r *announcementRepository) Update(ctx context.Context, id uuid.UUID, title string, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE announcements SET title = ?, body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, body, id,
	)
	if err != nil {
		return fmt.Errorf("update announcement: %w", err)
	}
	return nil
}

func (r *announcementRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM announcements WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

func (r *announcementRepository) GetByID(ctx context.Context, id uuid.UUID) (*AnnouncementRow, error) {
	var row AnnouncementRow
	err := scanAnnouncementRow(
		r.db.QueryRowContext(ctx, announcementSelectBase+` WHERE a.id = ?`, id),
		&row,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return &row, nil
}

func (r *announcementRepository) List(ctx context.Context, limit, offset int) ([]AnnouncementRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM announcements`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		announcementSelectBase+` ORDER BY a.pinned DESC, a.created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	var result []AnnouncementRow
	for rows.Next() {
		var row AnnouncementRow
		if err := scanAnnouncementRow(rows, &row); err != nil {
			return nil, 0, fmt.Errorf("scan announcement: %w", err)
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *announcementRepository) GetLatest(ctx context.Context) (*AnnouncementRow, error) {
	var row AnnouncementRow
	err := scanAnnouncementRow(
		r.db.QueryRowContext(ctx, announcementSelectBase+` ORDER BY a.pinned DESC, a.created_at DESC LIMIT 1`),
		&row,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest announcement: %w", err)
	}
	return &row, nil
}

func (r *announcementRepository) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	val := 0
	if pinned {
		val = 1
	}
	_, err := r.db.ExecContext(ctx, `UPDATE announcements SET pinned = ? WHERE id = ?`, val, id)
	if err != nil {
		return fmt.Errorf("set pinned: %w", err)
	}
	return nil
}
