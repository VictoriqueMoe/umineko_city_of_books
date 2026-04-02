package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type (
	PostRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, body string) error
		UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*PostRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int) ([]PostRow, int, error)
		ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int) ([]PostRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]PostRow, int, error)

		AddMedia(ctx context.Context, postID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		DeleteMedia(ctx context.Context, id int64, postID uuid.UUID) (string, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMedia(ctx context.Context, postID uuid.UUID) ([]PostMediaRow, error)
		GetMediaBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostMediaRow, error)

		Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		GetLikedBy(ctx context.Context, postID uuid.UUID) ([]PostLikeUser, error)
		RecordView(ctx context.Context, postID uuid.UUID, viewerHash string) (bool, error)
		GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)

		CreateComment(ctx context.Context, id uuid.UUID, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]PostCommentRow, int, error)
		GetCommentPostID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]PostMediaRow, error)

		CountUserPostsToday(ctx context.Context, userID uuid.UUID) (int, error)
		GetCornerCounts(ctx context.Context) (map[string]int, error)

		AddEmbed(ctx context.Context, ownerID string, ownerType string, url string, embedType string, title string, description string, image string, siteName string, videoID string, sortOrder int) error
		DeleteEmbeds(ctx context.Context, ownerID string, ownerType string) error
		UpdateEmbed(ctx context.Context, id int, title string, description string, image string, siteName string) error
		GetEmbeds(ctx context.Context, ownerID string, ownerType string) ([]EmbedRow, error)
		GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string) (map[string][]EmbedRow, error)
		GetStaleEmbeds(ctx context.Context, olderThan string, limit int) ([]EmbedRow, error)
	}

	postRepository struct {
		db *sql.DB
	}
)

const postSelectBase = `
	SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
		u.username, u.display_name, u.avatar_url,
		COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM post_likes WHERE post_id = p.id),
		(SELECT COUNT(*) FROM post_comments WHERE post_id = p.id),
		EXISTS(SELECT 1 FROM post_likes WHERE post_id = p.id AND user_id = ?),
		p.view_count
	FROM posts p
	JOIN users u ON p.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = p.user_id`

func scanPostRow(row interface{ Scan(...interface{}) error }, p *PostRow) error {
	var userLikedInt int
	err := row.Scan(
		&p.ID, &p.UserID, &p.Corner, &p.Body, &p.CreatedAt, &p.UpdatedAt,
		&p.AuthorUsername, &p.AuthorDisplayName, &p.AuthorAvatarURL,
		&p.AuthorRole,
		&p.LikeCount, &p.CommentCount, &userLikedInt, &p.ViewCount,
	)
	p.UserLiked = userLikedInt == 1
	return err
}

func postOrderClause(sort string, hasFollowBoost bool) string {
	switch sort {
	case "new":
		return ` ORDER BY p.created_at DESC`
	case "likes":
		return ` ORDER BY (SELECT COUNT(*) FROM post_likes WHERE post_id = p.id) DESC, p.created_at DESC`
	case "comments":
		return ` ORDER BY (SELECT COUNT(*) FROM post_comments WHERE post_id = p.id) DESC, p.created_at DESC`
	case "views":
		return ` ORDER BY p.view_count DESC, p.created_at DESC`
	default:
		jitter := `((unicode(substr(p.id, 1, 1)) * 7 + unicode(substr(p.id, 5, 1)) * 13 + ?) % 1000) / 2500.0`
		if hasFollowBoost {
			return `
				ORDER BY (
					(1.0
						+ MIN((SELECT COUNT(*) FROM post_likes WHERE post_id = p.id), 50) * 0.15
						+ MIN((SELECT COUNT(*) FROM post_comments WHERE post_id = p.id), 30) * 0.3
						+ CASE WHEN EXISTS(SELECT 1 FROM follows WHERE follower_id = ? AND following_id = p.user_id) THEN 3.0 ELSE 0 END
					) / (1.0 + (julianday('now') - julianday(p.created_at)) * 24.0 * 0.3)
					+ ` + jitter + `
				) DESC`
		}
		return `
			ORDER BY (
				(1.0
					+ MIN((SELECT COUNT(*) FROM post_likes WHERE post_id = p.id), 50) * 0.15
					+ MIN((SELECT COUNT(*) FROM post_comments WHERE post_id = p.id), 30) * 0.3
				) / (1.0 + (julianday('now') - julianday(p.created_at)) * 24.0 * 0.3)
				+ ` + jitter + `
			) DESC`
	}
}

func (r *postRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO posts (id, user_id, corner, body) VALUES (?, ?, ?, ?)`,
		id, userID, corner, body,
	)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	return nil
}

func (r *postRepository) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.updatePost(ctx, id, &userID, body)
}

func (r *postRepository) UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.updatePost(ctx, id, nil, body)
}

func (r *postRepository) updatePost(ctx context.Context, id uuid.UUID, userID *uuid.UUID, body string) error {
	var res sql.Result
	var err error
	if userID != nil {
		res, err = r.db.ExecContext(ctx,
			`UPDATE posts SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
			body, id, *userID,
		)
	} else {
		res, err = r.db.ExecContext(ctx,
			`UPDATE posts SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			body, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("post not found or not owned")
	}
	return nil
}

func (r *postRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*PostRow, error) {
	var p PostRow
	err := scanPostRow(r.db.QueryRowContext(ctx, postSelectBase+` WHERE p.id = ?`, viewerID, id), &p)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get post: %w", err)
	}
	return &p, nil
}

func (r *postRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM posts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("post not found or not owned")
	}
	return nil
}

func (r *postRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM posts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete post: %w", err)
	}
	return nil
}

func (r *postRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int) ([]PostRow, int, error) {
	var total int
	whereParts := []string{"p.corner = ?"}
	args := []interface{}{corner}

	if search != "" {
		whereParts = append(whereParts, "(p.body LIKE ? OR u.display_name LIKE ? OR u.username LIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	whereClause := " WHERE " + strings.Join(whereParts, " AND ")

	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts p JOIN users u ON p.user_id = u.id`+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	orderClause := postOrderClause(sort, true)
	query := postSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, args...)
	if sort == "" || sort == "relevance" {
		queryArgs = append(queryArgs, seed, viewerID)
	}
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	var posts []PostRow
	for rows.Next() {
		var p PostRow
		if err := scanPostRow(rows, &p); err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *postRepository) ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int) ([]PostRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts WHERE corner = ? AND (user_id = ? OR user_id IN (SELECT following_id FROM follows WHERE follower_id = ?))`,
		corner, userID, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count following posts: %w", err)
	}

	whereClause := ` WHERE p.corner = ? AND (p.user_id = ? OR p.user_id IN (SELECT following_id FROM follows WHERE follower_id = ?))`
	orderClause := postOrderClause(sort, false)
	query := postSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []interface{}{userID, corner, userID, userID}
	if sort == "" || sort == "relevance" {
		queryArgs = append(queryArgs, seed)
	}
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list following posts: %w", err)
	}
	defer rows.Close()

	var posts []PostRow
	for rows.Next() {
		var p PostRow
		if err := scanPostRow(rows, &p); err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *postRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]PostRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM posts WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user posts: %w", err)
	}

	query := postSelectBase + ` WHERE p.user_id = ? ORDER BY p.created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, viewerID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user posts: %w", err)
	}
	defer rows.Close()

	var posts []PostRow
	for rows.Next() {
		var p PostRow
		if err := scanPostRow(rows, &p); err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *postRepository) AddMedia(ctx context.Context, postID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO post_media (post_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		postID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add post media: %w", err)
	}
	return res.LastInsertId()
}

func (r *postRepository) DeleteMedia(ctx context.Context, id int64, postID uuid.UUID) (string, error) {
	var mediaURL string
	err := r.db.QueryRowContext(ctx, `SELECT media_url FROM post_media WHERE id = ? AND post_id = ?`, id, postID).Scan(&mediaURL)
	if err != nil {
		return "", fmt.Errorf("media not found: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM post_media WHERE id = ? AND post_id = ?`, id, postID)
	if err != nil {
		return "", fmt.Errorf("delete media: %w", err)
	}
	return mediaURL, nil
}

func (r *postRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE post_media SET media_url = ? WHERE id = ?`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update media url: %w", err)
	}
	return nil
}

func (r *postRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE post_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update media thumbnail: %w", err)
	}
	return nil
}

func (r *postRepository) GetMedia(ctx context.Context, postID uuid.UUID) ([]PostMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, post_id, media_url, media_type, thumbnail_url, sort_order FROM post_media WHERE post_id = ? ORDER BY sort_order`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("get post media: %w", err)
	}
	defer rows.Close()

	var media []PostMediaRow
	for rows.Next() {
		var m PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan post media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *postRepository) GetMediaBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostMediaRow, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{postIDs[0]}
	for _, id := range postIDs[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, post_id, media_url, media_type, thumbnail_url, sort_order FROM post_media WHERE post_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get post media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]PostMediaRow)
	for rows.Next() {
		var m PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan post media: %w", err)
		}
		result[m.PostID] = append(result[m.PostID], m)
	}
	return result, rows.Err()
}

func (r *postRepository) Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO post_likes (user_id, post_id) VALUES (?, ?)`,
		userID, postID,
	)
	if err != nil {
		return fmt.Errorf("like post: %w", err)
	}
	return nil
}

func (r *postRepository) Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM post_likes WHERE user_id = ? AND post_id = ?`,
		userID, postID,
	)
	if err != nil {
		return fmt.Errorf("unlike post: %w", err)
	}
	return nil
}

func (r *postRepository) GetLikedBy(ctx context.Context, postID uuid.UUID) ([]PostLikeUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM post_likes pl
		JOIN users u ON pl.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE pl.post_id = ?
		ORDER BY pl.created_at DESC`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("get liked by: %w", err)
	}
	defer rows.Close()

	var users []PostLikeUser
	for rows.Next() {
		var u PostLikeUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role); err != nil {
			return nil, fmt.Errorf("scan like user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *postRepository) RecordView(ctx context.Context, postID uuid.UUID, viewerHash string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO post_views (post_id, viewer_hash) VALUES (?, ?)`,
		postID, viewerHash,
	)
	if err != nil {
		return false, fmt.Errorf("record view: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		_, err = r.db.ExecContext(ctx, `UPDATE posts SET view_count = view_count + 1 WHERE id = ?`, postID)
		if err != nil {
			return false, fmt.Errorf("increment view count: %w", err)
		}
	}
	return n > 0, nil
}

func (r *postRepository) GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get post author: %w", err)
	}
	return userID, nil
}

func (r *postRepository) CreateComment(ctx context.Context, id uuid.UUID, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO post_comments (id, post_id, parent_id, user_id, body) VALUES (?, ?, ?, ?, ?)`,
		id, postID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *postRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.updateComment(ctx, id, &userID, body)
}

func (r *postRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.updateComment(ctx, id, nil, body)
}

func (r *postRepository) updateComment(ctx context.Context, id uuid.UUID, userID *uuid.UUID, body string) error {
	var res sql.Result
	var err error
	if userID != nil {
		res, err = r.db.ExecContext(ctx,
			`UPDATE post_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
			body, id, *userID,
		)
	} else {
		res, err = r.db.ExecContext(ctx,
			`UPDATE post_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			body, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *postRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM post_comments WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *postRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM post_comments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete comment: %w", err)
	}
	return nil
}

func (r *postRepository) GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]PostCommentRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_comments WHERE post_id = ?`, postID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.post_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url,
			COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM post_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM post_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM post_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?`,
		viewerID, postID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get comments: %w", err)
	}
	defer rows.Close()

	var comments []PostCommentRow
	for rows.Next() {
		var c PostCommentRow
		var userLikedInt int
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.ParentID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL,
			&c.AuthorRole, &c.LikeCount, &userLikedInt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan comment: %w", err)
		}
		c.UserLiked = userLikedInt == 1
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *postRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO post_comment_likes (user_id, comment_id) VALUES (?, ?)`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like comment: %w", err)
	}
	return nil
}

func (r *postRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM post_comment_likes WHERE user_id = ? AND comment_id = ?`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike comment: %w", err)
	}
	return nil
}

func (r *postRepository) GetCommentPostID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var postID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT post_id FROM post_comments WHERE id = ?`, commentID).Scan(&postID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get comment post id: %w", err)
	}
	return postID, nil
}

func (r *postRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM post_comments WHERE id = ?`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get comment author: %w", err)
	}
	return userID, nil
}

func (r *postRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO post_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add comment media: %w", err)
	}
	return res.LastInsertId()
}

func (r *postRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE post_comment_media SET media_url = ? WHERE id = ?`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update comment media url: %w", err)
	}
	return nil
}

func (r *postRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE post_comment_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update comment media thumbnail: %w", err)
	}
	return nil
}

func (r *postRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]PostMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM post_comment_media WHERE comment_id = ? ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get comment media: %w", err)
	}
	defer rows.Close()

	var media []PostMediaRow
	for rows.Next() {
		var m PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *postRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]PostMediaRow, error) {
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
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM post_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]PostMediaRow)
	for rows.Next() {
		var m PostMediaRow
		var commentID uuid.UUID
		if err := rows.Scan(&m.ID, &commentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan comment media: %w", err)
		}
		result[commentID] = append(result[commentID], m)
	}
	return result, rows.Err()
}

func (r *postRepository) CountUserPostsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts WHERE user_id = ? AND created_at > datetime('now', '-1 day')`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user posts today: %w", err)
	}
	return count, nil
}

func (r *postRepository) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT corner, COUNT(*) FROM posts GROUP BY corner`)
	if err != nil {
		return nil, fmt.Errorf("corner counts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var corner string
		var count int
		if err := rows.Scan(&corner, &count); err != nil {
			return nil, fmt.Errorf("scan corner count: %w", err)
		}
		result[corner] = count
	}
	return result, rows.Err()
}

func (r *postRepository) AddEmbed(ctx context.Context, ownerID string, ownerType string, url string, embedType string, title string, description string, image string, siteName string, videoID string, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO embeds (owner_id, owner_type, url, embed_type, title, description, image, site_name, video_id, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ownerID, ownerType, url, embedType, title, description, image, siteName, videoID, sortOrder,
	)
	if err != nil {
		return fmt.Errorf("add embed: %w", err)
	}
	return nil
}

func (r *postRepository) DeleteEmbeds(ctx context.Context, ownerID string, ownerType string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM embeds WHERE owner_id = ? AND owner_type = ?`, ownerID, ownerType)
	if err != nil {
		return fmt.Errorf("delete embeds: %w", err)
	}
	return nil
}

func (r *postRepository) UpdateEmbed(ctx context.Context, id int, title string, description string, image string, siteName string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE embeds SET title = ?, description = ?, image = ?, site_name = ?, fetched_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, description, image, siteName, id,
	)
	if err != nil {
		return fmt.Errorf("update embed: %w", err)
	}
	return nil
}

func (r *postRepository) GetStaleEmbeds(ctx context.Context, olderThan string, limit int) ([]EmbedRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE embed_type = 'link' AND fetched_at < datetime('now', ?) LIMIT ?`,
		olderThan, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get stale embeds: %w", err)
	}
	defer rows.Close()

	var embeds []EmbedRow
	for rows.Next() {
		var e EmbedRow
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.URL, &e.EmbedType, &e.Title, &e.Desc, &e.Image, &e.SiteName, &e.VideoID, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan stale embed: %w", err)
		}
		embeds = append(embeds, e)
	}
	return embeds, rows.Err()
}

func (r *postRepository) GetEmbeds(ctx context.Context, ownerID string, ownerType string) ([]EmbedRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE owner_id = ? AND owner_type = ? ORDER BY sort_order`,
		ownerID, ownerType,
	)
	if err != nil {
		return nil, fmt.Errorf("get embeds: %w", err)
	}
	defer rows.Close()

	var embeds []EmbedRow
	for rows.Next() {
		var e EmbedRow
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.URL, &e.EmbedType, &e.Title, &e.Desc, &e.Image, &e.SiteName, &e.VideoID, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan embed: %w", err)
		}
		embeds = append(embeds, e)
	}
	return embeds, rows.Err()
}

func (r *postRepository) GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string) (map[string][]EmbedRow, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{ownerIDs[0]}
	for _, id := range ownerIDs[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}
	args = append(args, ownerType)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE owner_id IN (`+placeholders+`) AND owner_type = ? ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get embeds: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]EmbedRow)
	for rows.Next() {
		var e EmbedRow
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.URL, &e.EmbedType, &e.Title, &e.Desc, &e.Image, &e.SiteName, &e.VideoID, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan embed: %w", err)
		}
		result[e.OwnerID] = append(result[e.OwnerID], e)
	}
	return result, rows.Err()
}
