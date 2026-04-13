package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository/model"

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

		CreateComment(ctx context.Context, id uuid.UUID, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]AnnouncementCommentRow, int, error)
		GetCommentAnnouncementID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]AnnouncementCommentMediaRow, error)
	}

	AnnouncementCommentRow struct {
		ID                uuid.UUID
		AnnouncementID    uuid.UUID
		ParentID          *uuid.UUID
		UserID            uuid.UUID
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		UserLiked         bool
	}

	AnnouncementCommentMediaRow = model.CommentMediaRow

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

func (r *announcementRepository) CreateComment(ctx context.Context, id uuid.UUID, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO announcement_comments (id, announcement_id, parent_id, user_id, body) VALUES (?, ?, ?, ?, ?)`,
		id, announcementID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create announcement comment: %w", err)
	}
	return nil
}

func (r *announcementRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE announcement_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
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

func (r *announcementRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE announcement_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update announcement comment: %w", err)
	}
	return nil
}

func (r *announcementRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM announcement_comments WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete announcement comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *announcementRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM announcement_comments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete announcement comment: %w", err)
	}
	return nil
}

func (r *announcementRepository) GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]AnnouncementCommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs)
	var total int
	countArgs := []interface{}{announcementID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM announcement_comments WHERE announcement_id = ?`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count announcement comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs)
	queryArgs := []interface{}{viewerID, announcementID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.announcement_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM announcement_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM announcement_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM announcement_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.announcement_id = ?`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get announcement comments: %w", err)
	}
	defer rows.Close()

	var comments []AnnouncementCommentRow
	for rows.Next() {
		var c AnnouncementCommentRow
		var userLikedInt int
		if err := rows.Scan(
			&c.ID, &c.AnnouncementID, &c.ParentID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &userLikedInt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan announcement comment: %w", err)
		}
		c.UserLiked = userLikedInt == 1
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *announcementRepository) GetCommentAnnouncementID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT announcement_id FROM announcement_comments WHERE id = ?`, commentID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get announcement comment announcement id: %w", err)
	}
	return id, nil
}

func (r *announcementRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM announcement_comments WHERE id = ?`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get announcement comment author: %w", err)
	}
	return userID, nil
}

func (r *announcementRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO announcement_comment_likes (user_id, comment_id) VALUES (?, ?)`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like announcement comment: %w", err)
	}
	return nil
}

func (r *announcementRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM announcement_comment_likes WHERE user_id = ? AND comment_id = ?`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike announcement comment: %w", err)
	}
	return nil
}

func (r *announcementRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO announcement_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add announcement comment media: %w", err)
	}
	return res.LastInsertId()
}

func (r *announcementRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE announcement_comment_media SET media_url = ? WHERE id = ?`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update announcement comment media url: %w", err)
	}
	return nil
}

func (r *announcementRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE announcement_comment_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update announcement comment media thumbnail: %w", err)
	}
	return nil
}

func (r *announcementRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]AnnouncementCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{commentIDs[0]}
	for _, id := range commentIDs[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM announcement_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get announcement comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]AnnouncementCommentMediaRow)
	for rows.Next() {
		var m AnnouncementCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan announcement comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}
