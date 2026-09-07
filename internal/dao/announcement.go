package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	AnnouncementDAO interface {
		Create(ctx context.Context, s spec.NewAnnouncement, tx ...*sql.Tx) (*model.AnnouncementRow, error)
		Update(ctx context.Context, s spec.AnnouncementUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.AnnouncementRow, error)
		List(ctx context.Context, q spec.AnnouncementListQuery, tx ...*sql.Tx) ([]model.AnnouncementRow, int, error)
		GetLatest(ctx context.Context, tx ...*sql.Tx) (*model.AnnouncementRow, error)
		SetPinned(ctx context.Context, s spec.AnnouncementPinUpdate, tx ...*sql.Tx) error

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	announcementDAO struct {
		db *sql.DB
		*commentDAO[uuid.UUID]
	}
)

const (
	announcementSelectBase = `SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
	COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(r.role, '')
	FROM announcements a
	LEFT JOIN users u ON a.author_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id`
)

func scanAnnouncementRow(scanner interface {
	Scan(dest ...any) error
}, row *model.AnnouncementRow) error {
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

func (r *announcementDAO) Create(ctx context.Context, s spec.NewAnnouncement, tx ...*sql.Tx) (*model.AnnouncementRow, error) {
	var created model.AnnouncementRow

	err := scanAnnouncementRow(
		txOrDB(r.db, tx).QueryRowContext(ctx,
			`WITH a AS (
			     INSERT INTO announcements (author_id, title, body) VALUES ($1, $2, $3)
			     RETURNING id, title, body, author_id, pinned, created_at, updated_at
			 )
			 SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
			        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(r.role, '')
			 FROM a
			 LEFT JOIN users u ON u.id = a.author_id
			 LEFT JOIN user_roles r ON r.user_id = u.id`,
			s.AuthorID, s.Title, s.Body,
		),
		&created,
	)
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}

	return &created, nil
}

func (r *announcementDAO) Update(ctx context.Context, s spec.AnnouncementUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE announcements SET title = $1, body = $2, updated_at = NOW() WHERE id = $3`,
		s.Title, s.Body, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update announcement: %w", err)
	}

	return nil
}

func (r *announcementDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}

	return nil
}

func (r *announcementDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.AnnouncementRow, error) {
	var row model.AnnouncementRow

	err := scanAnnouncementRow(
		txOrDB(r.db, tx).QueryRowContext(ctx, announcementSelectBase+` WHERE a.id = $1`, id),
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

func (r *announcementDAO) List(ctx context.Context, q spec.AnnouncementListQuery, tx ...*sql.Tx) ([]model.AnnouncementRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM announcements`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		announcementSelectBase+` ORDER BY a.pinned DESC, a.created_at DESC LIMIT $1 OFFSET $2`,
		q.Limit, q.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	var result []model.AnnouncementRow
	for rows.Next() {
		var row model.AnnouncementRow
		if err := scanAnnouncementRow(rows, &row); err != nil {
			return nil, 0, fmt.Errorf("scan announcement: %w", err)
		}

		result = append(result, row)
	}

	return result, total, rows.Err()
}

func (r *announcementDAO) GetLatest(ctx context.Context, tx ...*sql.Tx) (*model.AnnouncementRow, error) {
	var row model.AnnouncementRow

	err := scanAnnouncementRow(
		txOrDB(r.db, tx).QueryRowContext(ctx, announcementSelectBase+` ORDER BY a.pinned DESC, a.created_at DESC LIMIT 1`),
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

func (r *announcementDAO) SetPinned(ctx context.Context, s spec.AnnouncementPinUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `UPDATE announcements SET pinned = $1 WHERE id = $2`, s.Pinned, s.ID)
	if err != nil {
		return fmt.Errorf("set pinned: %w", err)
	}

	return nil
}
