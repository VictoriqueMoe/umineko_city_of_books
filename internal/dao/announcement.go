package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	announcementDAO struct {
		db *sql.DB
	}

	announcementRepository struct {
		repository.AnnouncementRepository
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

func (r *announcementDAO) CreateComment(ctx context.Context, id uuid.UUID, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO announcement_comments (id, announcement_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, announcementID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create announcement comment: %w", err)
	}
	return nil
}

func (r *announcementDAO) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE announcement_comments SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update announcement comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *announcementDAO) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE announcement_comments SET body = $1, updated_at = NOW() WHERE id = $2`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update announcement comment: %w", err)
	}
	return nil
}

func (r *announcementDAO) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM announcement_comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete announcement comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *announcementDAO) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM announcement_comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete announcement comment: %w", err)
	}
	return nil
}

func (r *announcementDAO) GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]repository.AnnouncementCommentRow, int, error) {
	exclSQL, exclArgs := excludeClauseQ("user_id", excludeUserIDs)
	var total int
	countArgs := []interface{}{announcementID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, rebind(`SELECT COUNT(*) FROM announcement_comments WHERE announcement_id = ?`+exclSQL), countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count announcement comments: %w", err)
	}

	exclSQL2, exclArgs2 := excludeClauseQ("c.user_id", excludeUserIDs)
	queryArgs := []interface{}{viewerID, announcementID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		rebind(`SELECT c.id, c.announcement_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM announcement_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM announcement_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM announcement_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.announcement_id = ?`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?`),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get announcement comments: %w", err)
	}
	defer rows.Close()

	var comments []repository.AnnouncementCommentRow
	for rows.Next() {
		var (
			c         repository.AnnouncementCommentRow
			createdAt time.Time
			updatedAt sql.NullTime
		)
		if err := rows.Scan(
			&c.ID, &c.AnnouncementID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan announcement comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if updatedAt.Valid {
			c.UpdatedAt = new(updatedAt.Time.UTC().Format(time.RFC3339))
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *announcementDAO) GetCommentAnnouncementID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT announcement_id FROM announcement_comments WHERE id = $1`, commentID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get announcement comment announcement id: %w", err)
	}
	return id, nil
}

func (r *announcementDAO) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM announcement_comments WHERE id = $1`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get announcement comment author: %w", err)
	}
	return userID, nil
}

func (r *announcementDAO) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO announcement_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like announcement comment: %w", err)
	}
	return nil
}

func (r *announcementDAO) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM announcement_comment_likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike announcement comment: %w", err)
	}
	return nil
}

func (r *announcementDAO) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO announcement_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add announcement comment media: %w", err)
	}
	return id, nil
}

func (r *announcementDAO) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE announcement_comment_media SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update announcement comment media url: %w", err)
	}
	return nil
}

func (r *announcementDAO) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE announcement_comment_media SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update announcement comment media thumbnail: %w", err)
	}
	return nil
}

func (r *announcementDAO) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]repository.AnnouncementCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	var placeholders strings.Builder
	placeholders.WriteString("?")
	args := []any{commentIDs[0]}
	for i := 1; i < len(commentIDs); i++ {
		placeholders.WriteString(", ?")
		args = append(args, commentIDs[i])
	}

	rows, err := r.db.QueryContext(ctx,
		rebind(`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM announcement_comment_media WHERE comment_id IN (`+placeholders.String()+`) ORDER BY sort_order`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get announcement comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]repository.AnnouncementCommentMediaRow)
	for rows.Next() {
		var m repository.AnnouncementCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan announcement comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}
