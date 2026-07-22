package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	commentDAO[K comparable] struct {
		db         *sql.DB
		table      string
		fk         string
		likesTable string
		mediaTable string
	}
)

func newCommentDAO[K comparable](db *sql.DB, table, fk, likesTable, mediaTable string) *commentDAO[K] {
	return &commentDAO[K]{db: db, table: table, fk: fk, likesTable: likesTable, mediaTable: mediaTable}
}

func (c *commentDAO[K]) CreateComment(ctx context.Context, id uuid.UUID, entityID K, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO `+c.table+` (id, `+c.fk+`, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, entityID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create comment in %s: %w", c.table, err)
	}

	return nil
}

func (c *commentDAO[K]) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return c.updateComment(ctx, id, &userID, body)
}

func (c *commentDAO[K]) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return c.updateComment(ctx, id, nil, body)
}

func (c *commentDAO[K]) updateComment(ctx context.Context, id uuid.UUID, userID *uuid.UUID, body string) error {
	var (
		res sql.Result
		err error
	)

	if userID != nil {
		res, err = c.db.ExecContext(ctx,
			`UPDATE `+c.table+` SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
			body, id, *userID,
		)
	} else {
		res, err = c.db.ExecContext(ctx,
			`UPDATE `+c.table+` SET body = $1, updated_at = NOW() WHERE id = $2`,
			body, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update comment in %s: %w", c.table, err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}

	return nil
}

func (c *commentDAO[K]) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := c.db.ExecContext(ctx, `DELETE FROM `+c.table+` WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete comment in %s: %w", c.table, err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}

	return nil
}

func (c *commentDAO[K]) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM `+c.table+` WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete comment in %s: %w", c.table, err)
	}

	return nil
}

func (c *commentDAO[K]) GetComments(ctx context.Context, entityID K, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]repository.CommentRow, int, error) {
	var total int

	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs, 2)
	countArgs := append([]interface{}{entityID}, exclArgs...)
	if err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+c.table+` WHERE `+c.fk+` = $1`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count comments in %s: %w", c.table, err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	rows, err := c.db.QueryContext(ctx,
		`SELECT c.id, c.`+c.fk+`::text, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''), (u.banned_at IS NOT NULL),
			(SELECT COUNT(*) FROM `+c.likesTable+` WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM `+c.likesTable+` WHERE comment_id = c.id AND user_id = $1)
		FROM `+c.table+` c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.`+c.fk+` = $2`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT `+limitPH+` OFFSET `+offsetPH,
		append([]interface{}{viewerID, entityID}, append(exclArgs2, limit, offset)...)...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get comments in %s: %w", c.table, err)
	}
	defer rows.Close()

	var comments []repository.CommentRow
	for rows.Next() {
		var (
			cm        repository.CommentRow
			createdAt time.Time
			updatedAt *time.Time
		)
		if err := rows.Scan(
			&cm.ID, &cm.EntityID, &cm.ParentID, &cm.UserID, &cm.Body, &createdAt, &updatedAt,
			&cm.AuthorUsername, &cm.AuthorDisplayName, &cm.AuthorAvatarURL, &cm.AuthorRole, &cm.AuthorBanned,
			&cm.LikeCount, &cm.UserLiked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan comment: %w", err)
		}

		cm.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		cm.UpdatedAt = timePtrToString(updatedAt)
		comments = append(comments, cm)
	}

	return comments, total, rows.Err()
}

func (c *commentDAO[K]) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (K, error) {
	var entityID K
	if err := c.db.QueryRowContext(ctx, `SELECT `+c.fk+` FROM `+c.table+` WHERE id = $1`, commentID).Scan(&entityID); err != nil {
		var zero K
		return zero, fmt.Errorf("get comment entity id from %s: %w", c.table, err)
	}

	return entityID, nil
}

func (c *commentDAO[K]) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	if err := c.db.QueryRowContext(ctx, `SELECT user_id FROM `+c.table+` WHERE id = $1`, commentID).Scan(&userID); err != nil {
		return uuid.Nil, fmt.Errorf("get comment author from %s: %w", c.table, err)
	}

	return userID, nil
}

func (c *commentDAO[K]) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO `+c.likesTable+` (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like comment in %s: %w", c.likesTable, err)
	}

	return nil
}

func (c *commentDAO[K]) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := c.db.ExecContext(ctx,
		`DELETE FROM `+c.likesTable+` WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike comment in %s: %w", c.likesTable, err)
	}

	return nil
}

func (c *commentDAO[K]) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := c.db.QueryRowContext(ctx,
		`INSERT INTO `+c.mediaTable+` (comment_id, media_url, media_type, thumbnail_url, sort_order)
		VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(sort_order) + 1 FROM `+c.mediaTable+` WHERE comment_id = $1), $5))
		RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add comment media in %s: %w", c.mediaTable, err)
	}

	return id, nil
}

func (c *commentDAO[K]) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM `+c.mediaTable+` WHERE comment_id = $1 ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get comment media in %s: %w", c.mediaTable, err)
	}
	defer rows.Close()

	var mediaList []model.PostMediaRow
	for rows.Next() {
		var m model.PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan comment media: %w", err)
		}

		mediaList = append(mediaList, m)
	}

	return mediaList, rows.Err()
}

func (c *commentDAO[K]) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "$1"
	args := []interface{}{commentIDs[0]}
	for i, id := range commentIDs[1:] {
		placeholders += fmt.Sprintf(", $%d", i+2)
		args = append(args, id)
	}

	rows, err := c.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM `+c.mediaTable+` WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get comment media in %s: %w", c.mediaTable, err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.PostMediaRow)
	for rows.Next() {
		var (
			m         model.PostMediaRow
			commentID uuid.UUID
		)
		if err := rows.Scan(&m.ID, &commentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan comment media: %w", err)
		}

		result[commentID] = append(result[commentID], m)
	}

	return result, rows.Err()
}

func (c *commentDAO[K]) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := c.db.ExecContext(ctx, `UPDATE `+c.mediaTable+` SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update comment media url in %s: %w", c.mediaTable, err)
	}

	return nil
}

func (c *commentDAO[K]) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := c.db.ExecContext(ctx, `UPDATE `+c.mediaTable+` SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update comment media thumbnail in %s: %w", c.mediaTable, err)
	}

	return nil
}
