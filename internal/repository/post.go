package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	PostRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, body string, sharedContentID *string, sharedContentType *string) error
		UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.PostRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string) ([]model.PostRow, int, error)
		ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.PostRow, int, error)

		AddMedia(ctx context.Context, postID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		DeleteMedia(ctx context.Context, id int64, postID uuid.UUID) (string, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMedia(ctx context.Context, postID uuid.UUID) ([]model.PostMediaRow, error)
		GetMediaBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		GetLikedBy(ctx context.Context, postID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, postID uuid.UUID, viewerHash string) (bool, error)
		GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)

		ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string) error
		UnresolveSuggestion(ctx context.Context, postID uuid.UUID) error

		CreateComment(ctx context.Context, id uuid.UUID, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostCommentRow, int, error)
		GetCommentPostID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		CountUserPostsToday(ctx context.Context, userID uuid.UUID) (int, error)
		GetCornerCounts(ctx context.Context) (map[string]int, error)

		GetShareCount(ctx context.Context, contentID string, contentType string) (int, error)
		GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string) (map[string]int, error)
		IncrementShareCount(ctx context.Context, contentID string, contentType string) error
		DecrementShareCount(ctx context.Context, contentID string, contentType string) error
		GetSharedContentFields(ctx context.Context, postID uuid.UUID) (*string, *string, error)

		CreatePollWithOptions(ctx context.Context, pollID uuid.UUID, postID uuid.UUID, durationSeconds int, expiresAt string, options []string) error
		GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID) (*model.PollRow, []model.PollOptionRow, *int, error)
		GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error)
		VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int) error

		AddEmbed(ctx context.Context, ownerID string, ownerType string, url string, embedType string, title string, description string, image string, siteName string, videoID string, sortOrder int) error
		DeleteEmbeds(ctx context.Context, ownerID string, ownerType string) error
		UpdateEmbed(ctx context.Context, id int, title string, description string, image string, siteName string) error
		GetEmbeds(ctx context.Context, ownerID string, ownerType string) ([]model.EmbedRow, error)
		GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string) (map[string][]model.EmbedRow, error)
		GetStaleEmbeds(ctx context.Context, olderThan string, limit int) ([]model.EmbedRow, error)
	}

	postRepository struct {
		db *sql.DB
	}

	SharedContentRef struct {
		ID   string
		Type string
	}
)

const postSelectBase = `
	SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
		u.username, u.display_name, u.avatar_url,
		COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM post_likes WHERE post_id = p.id),
		(SELECT COUNT(*) FROM post_comments WHERE post_id = p.id),
		EXISTS(SELECT 1 FROM post_likes WHERE post_id = p.id AND user_id = ?),
		p.view_count,
		COALESCE((SELECT status FROM suggestion_resolved WHERE post_id = p.id), ''),
		p.shared_content_id,
		p.shared_content_type
	FROM posts p
	JOIN users u ON p.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = p.user_id`

func scanPostRow(row interface{ Scan(...interface{}) error }, p *model.PostRow) error {
	var userLikedInt int
	err := row.Scan(
		&p.ID, &p.UserID, &p.Corner, &p.Body, &p.CreatedAt, &p.UpdatedAt,
		&p.AuthorUsername, &p.AuthorDisplayName, &p.AuthorAvatarURL,
		&p.AuthorRole,
		&p.LikeCount, &p.CommentCount, &userLikedInt, &p.ViewCount, &p.ResolvedStatus,
		&p.SharedContentID, &p.SharedContentType,
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

func (r *postRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, body string, sharedContentID *string, sharedContentType *string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO posts (id, user_id, corner, body, shared_content_id, shared_content_type) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, corner, body, sharedContentID, sharedContentType,
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

func (r *postRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.PostRow, error) {
	var p model.PostRow
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

func (r *postRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string) ([]model.PostRow, int, error) {
	var total int
	whereParts := []string{"p.corner = ?"}
	args := []interface{}{corner}

	if search != "" {
		whereParts = append(whereParts, "(p.body LIKE ? OR u.display_name LIKE ? OR u.username LIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	switch resolvedFilter {
	case "open":
		whereParts = append(whereParts, "NOT EXISTS(SELECT 1 FROM suggestion_resolved WHERE post_id = p.id)")
	case "done":
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM suggestion_resolved WHERE post_id = p.id AND status = 'done')")
	case "archived":
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM suggestion_resolved WHERE post_id = p.id AND status = 'archived')")
	}

	whereClause := " WHERE " + strings.Join(whereParts, " AND ")
	exclSQL, exclArgs := ExcludeClause("p.user_id", excludeUserIDs)
	whereClause += exclSQL
	countArgs := append(args, exclArgs...)

	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts p JOIN users u ON p.user_id = u.id`+whereClause, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	orderClause := postOrderClause(sort, true)
	query := postSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, countArgs...)
	if sort == "" || sort == "relevance" {
		queryArgs = append(queryArgs, seed, viewerID)
	}
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	var posts []model.PostRow
	for rows.Next() {
		var p model.PostRow
		if err := scanPostRow(rows, &p); err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *postRepository) ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostRow, int, error) {
	var total int
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs)
	countQuery := `SELECT COUNT(*) FROM posts WHERE corner = ? AND (user_id = ? OR user_id IN (SELECT following_id FROM follows WHERE follower_id = ?))` + exclSQL
	countArgs := []interface{}{corner, userID, userID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count following posts: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("p.user_id", excludeUserIDs)
	whereClause := ` WHERE p.corner = ? AND (p.user_id = ? OR p.user_id IN (SELECT following_id FROM follows WHERE follower_id = ?))` + exclSQL2
	orderClause := postOrderClause(sort, false)
	query := postSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []interface{}{userID, corner, userID, userID}
	queryArgs = append(queryArgs, exclArgs2...)
	if sort == "" || sort == "relevance" {
		queryArgs = append(queryArgs, seed)
	}
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list following posts: %w", err)
	}
	defer rows.Close()

	var posts []model.PostRow
	for rows.Next() {
		var p model.PostRow
		if err := scanPostRow(rows, &p); err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *postRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.PostRow, int, error) {
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

	var posts []model.PostRow
	for rows.Next() {
		var p model.PostRow
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

func (r *postRepository) GetMedia(ctx context.Context, postID uuid.UUID) ([]model.PostMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, post_id, media_url, media_type, thumbnail_url, sort_order FROM post_media WHERE post_id = ? ORDER BY sort_order`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("get post media: %w", err)
	}
	defer rows.Close()

	var media []model.PostMediaRow
	for rows.Next() {
		var m model.PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan post media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *postRepository) GetMediaBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
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

	result := make(map[uuid.UUID][]model.PostMediaRow)
	for rows.Next() {
		var m model.PostMediaRow
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

func (r *postRepository) GetLikedBy(ctx context.Context, postID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error) {
	exclSQL, exclArgs := ExcludeClause("pl.user_id", excludeUserIDs)
	queryArgs := []interface{}{postID}
	queryArgs = append(queryArgs, exclArgs...)
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM post_likes pl
		JOIN users u ON pl.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE pl.post_id = ?`+exclSQL+`
		ORDER BY pl.created_at DESC`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("get liked by: %w", err)
	}
	defer rows.Close()

	var users []model.PostLikeUser
	for rows.Next() {
		var u model.PostLikeUser
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

func (r *postRepository) ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO suggestion_resolved (post_id, resolved_by, status) VALUES (?, ?, ?)
		 ON CONFLICT(post_id) DO UPDATE SET status = ?, resolved_by = ?, resolved_at = CURRENT_TIMESTAMP`,
		postID, resolvedBy, status, status, resolvedBy,
	)
	if err != nil {
		return fmt.Errorf("resolve suggestion: %w", err)
	}
	return nil
}

func (r *postRepository) UnresolveSuggestion(ctx context.Context, postID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM suggestion_resolved WHERE post_id = ?`, postID)
	if err != nil {
		return fmt.Errorf("unresolve suggestion: %w", err)
	}
	return nil
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

func (r *postRepository) GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostCommentRow, int, error) {
	var total int
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs)
	countArgs := []interface{}{postID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_comments WHERE post_id = ?`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.post_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url,
			COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM post_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM post_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM post_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.post_id = ?`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?`,
		append([]interface{}{viewerID, postID}, append(exclArgs2, limit, offset)...)...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get comments: %w", err)
	}
	defer rows.Close()

	var comments []model.PostCommentRow
	for rows.Next() {
		var c model.PostCommentRow
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

func (r *postRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM post_comment_media WHERE comment_id = ? ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get comment media: %w", err)
	}
	defer rows.Close()

	var media []model.PostMediaRow
	for rows.Next() {
		var m model.PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *postRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
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

	result := make(map[uuid.UUID][]model.PostMediaRow)
	for rows.Next() {
		var m model.PostMediaRow
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

func (r *postRepository) GetShareCount(ctx context.Context, contentID string, contentType string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(share_count, 0) FROM share_counts WHERE content_id = ? AND content_type = ?`,
		contentID, contentType,
	).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get share count: %w", err)
	}
	return count, nil
}

func (r *postRepository) GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string) (map[string]int, error) {
	if len(contentIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{contentIDs[0]}
	for _, id := range contentIDs[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}
	args = append(args, contentType)

	rows, err := r.db.QueryContext(ctx,
		`SELECT content_id, share_count FROM share_counts WHERE content_id IN (`+placeholders+`) AND content_type = ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get share counts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, fmt.Errorf("scan share count: %w", err)
		}
		result[id] = count
	}
	return result, rows.Err()
}

func (r *postRepository) IncrementShareCount(ctx context.Context, contentID string, contentType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO share_counts (content_id, content_type, share_count) VALUES (?, ?, 1) ON CONFLICT(content_id, content_type) DO UPDATE SET share_count = share_count + 1`,
		contentID, contentType,
	)
	if err != nil {
		return fmt.Errorf("increment share count: %w", err)
	}
	return nil
}

func (r *postRepository) DecrementShareCount(ctx context.Context, contentID string, contentType string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE share_counts SET share_count = MAX(share_count - 1, 0) WHERE content_id = ? AND content_type = ?`,
		contentID, contentType,
	)
	if err != nil {
		return fmt.Errorf("decrement share count: %w", err)
	}
	return nil
}

func (r *postRepository) GetSharedContentFields(ctx context.Context, postID uuid.UUID) (*string, *string, error) {
	var contentID, contentType *string
	err := r.db.QueryRowContext(ctx,
		`SELECT shared_content_id, shared_content_type FROM posts WHERE id = ?`, postID,
	).Scan(&contentID, &contentType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get shared content fields: %w", err)
	}
	return contentID, contentType, nil
}

func GetSharedContentPreviews(db *sql.DB, refs []SharedContentRef) map[string]*dto.SharedContentPreview {
	result := make(map[string]*dto.SharedContentPreview)
	if len(refs) == 0 {
		return result
	}

	grouped := make(map[string][]string)
	for _, ref := range refs {
		grouped[ref.Type] = append(grouped[ref.Type], ref.ID)
	}

	for contentType, ids := range grouped {
		switch contentType {
		case "post":
			fetchPostPreviews(db, ids, result)
		case "art":
			fetchArtPreviews(db, ids, result)
		case "ship":
			fetchShipPreviews(db, ids, result)
		case "mystery":
			fetchMysteryPreviews(db, ids, result)
		case "theory":
			fetchTheoryPreviews(db, ids, result)
		case "fanfic":
			fetchFanficPreviews(db, ids, result)
		}
	}

	for _, ref := range refs {
		key := ref.Type + ":" + ref.ID
		if _, ok := result[key]; !ok {
			result[key] = &dto.SharedContentPreview{
				ID:          ref.ID,
				ContentType: ref.Type,
				Deleted:     true,
				URL:         contentURL(ref.Type, ref.ID),
			}
		}
	}

	return result
}

func contentURL(contentType, id string) string {
	switch contentType {
	case "post":
		return "/game-board/" + id
	case "art":
		return "/gallery/art/" + id
	case "ship":
		return "/ships/" + id
	case "mystery":
		return "/mystery/" + id
	case "theory":
		return "/theory/" + id
	case "fanfic":
		return "/fanfiction/" + id
	default:
		return "/"
	}
}

func buildPlaceholders(ids []string) (string, []interface{}) {
	placeholders := "?"
	args := []interface{}{ids[0]}
	for _, id := range ids[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}
	return placeholders, args
}

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}

func fetchPostPreviews(db *sql.DB, ids []string, result map[string]*dto.SharedContentPreview) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := db.Query(
		`SELECT p.id, p.body, p.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM post_likes WHERE post_id = p.id) as like_count,
			(SELECT COUNT(*) FROM post_comments WHERE post_id = p.id) as comment_count,
			p.corner
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = p.user_id
		WHERE p.id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, body, userID, username, displayName, avatarURL, authorRole, corner string
		var likeCount, commentCount int
		if err := rows.Scan(&id, &body, &userID, &username, &displayName, &avatarURL, &authorRole, &likeCount, &commentCount, &corner); err != nil {
			continue
		}
		uid, _ := uuid.Parse(userID)
		result["post:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "post",
			Body:        truncateBody(body, 200),
			Author: &dto.UserResponse{
				ID:          uid,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Role:        role.Role(authorRole),
			},
			URL:          "/game-board/" + id,
			Corner:       corner,
			LikeCount:    likeCount,
			CommentCount: commentCount,
		}
	}

	mediaRows, err := db.Query(
		`SELECT post_id, media_url, media_type, thumbnail_url, sort_order
		FROM post_media WHERE post_id IN (`+placeholders+`) ORDER BY sort_order LIMIT 4`, args...,
	)
	if err != nil {
		return
	}
	defer mediaRows.Close()

	for mediaRows.Next() {
		var postID, mediaURL, mediaType, thumbnailURL string
		var sortOrder int
		if err := mediaRows.Scan(&postID, &mediaURL, &mediaType, &thumbnailURL, &sortOrder); err != nil {
			continue
		}
		key := "post:" + postID
		if preview, ok := result[key]; ok {
			if len(preview.Media) < 4 {
				preview.Media = append(preview.Media, dto.PostMediaResponse{
					MediaURL:     mediaURL,
					MediaType:    mediaType,
					ThumbnailURL: thumbnailURL,
					SortOrder:    sortOrder,
				})
			}
		}
	}
}

func fetchArtPreviews(db *sql.DB, ids []string, result map[string]*dto.SharedContentPreview) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := db.Query(
		`SELECT a.id, a.title, a.description, a.image_url, a.thumbnail_url, a.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''), a.corner
		FROM art a
		JOIN users u ON a.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = a.user_id
		WHERE a.id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, description, imageURL, thumbnailURL, userID, username, displayName, avatarURL, authorRole, corner string
		if err := rows.Scan(&id, &title, &description, &imageURL, &thumbnailURL, &userID, &username, &displayName, &avatarURL, &authorRole, &corner); err != nil {
			continue
		}
		img := thumbnailURL
		if img == "" {
			img = imageURL
		}
		uid, _ := uuid.Parse(userID)
		result["art:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "art",
			Title:       title,
			Body:        truncateBody(description, 200),
			ImageURL:    img,
			Author: &dto.UserResponse{
				ID:          uid,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Role:        role.Role(authorRole),
			},
			URL:    "/gallery/art/" + id,
			Corner: corner,
		}
	}
}

func fetchShipPreviews(db *sql.DB, ids []string, result map[string]*dto.SharedContentPreview) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := db.Query(
		`SELECT s.id, s.title, s.description, s.image_url, s.thumbnail_url, s.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0)
		FROM ships s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = s.user_id
		WHERE s.id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, description, imageURL, thumbnailURL, userID, username, displayName, avatarURL, authorRole string
		var voteScore int
		if err := rows.Scan(&id, &title, &description, &imageURL, &thumbnailURL, &userID, &username, &displayName, &avatarURL, &authorRole, &voteScore); err != nil {
			continue
		}
		img := thumbnailURL
		if img == "" {
			img = imageURL
		}
		uid, _ := uuid.Parse(userID)
		result["ship:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "ship",
			Title:       title,
			Body:        truncateBody(description, 200),
			ImageURL:    img,
			Author: &dto.UserResponse{
				ID:          uid,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Role:        role.Role(authorRole),
			},
			URL:       "/ships/" + id,
			VoteScore: voteScore,
		}
	}
}

func fetchMysteryPreviews(db *sql.DB, ids []string, result map[string]*dto.SharedContentPreview) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := db.Query(
		`SELECT m.id, m.title, m.body, m.difficulty, m.solved, m.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM mysteries m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = m.user_id
		WHERE m.id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, body, difficulty, userID, username, displayName, avatarURL, authorRole string
		var solved bool
		if err := rows.Scan(&id, &title, &body, &difficulty, &solved, &userID, &username, &displayName, &avatarURL, &authorRole); err != nil {
			continue
		}
		uid, _ := uuid.Parse(userID)
		result["mystery:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "mystery",
			Title:       title,
			Body:        truncateBody(body, 200),
			Difficulty:  difficulty,
			Solved:      solved,
			Author: &dto.UserResponse{
				ID:          uid,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Role:        role.Role(authorRole),
			},
			URL: "/mystery/" + id,
		}
	}
}

func fetchTheoryPreviews(db *sql.DB, ids []string, result map[string]*dto.SharedContentPreview) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := db.Query(
		`SELECT t.id, t.title, t.body, t.series, t.credibility_score, t.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM theories t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = t.user_id
		WHERE t.id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, body, series, userID, username, displayName, avatarURL, authorRole string
		var credibilityScore float64
		if err := rows.Scan(&id, &title, &body, &series, &credibilityScore, &userID, &username, &displayName, &avatarURL, &authorRole); err != nil {
			continue
		}
		uid, _ := uuid.Parse(userID)
		result["theory:"+id] = &dto.SharedContentPreview{
			ID:               id,
			ContentType:      "theory",
			Title:            title,
			Body:             truncateBody(body, 200),
			Series:           series,
			CredibilityScore: credibilityScore,
			Author: &dto.UserResponse{
				ID:          uid,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Role:        role.Role(authorRole),
			},
			URL: "/theory/" + id,
		}
	}
}

func fetchFanficPreviews(db *sql.DB, ids []string, result map[string]*dto.SharedContentPreview) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := db.Query(
		`SELECT f.id, f.title, f.summary, f.series, f.rating, f.cover_image_url, f.cover_thumbnail_url, f.word_count,
			(SELECT COUNT(*) FROM fanfic_chapters WHERE fanfic_id = f.id),
			f.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM fanfics f
		JOIN users u ON f.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = f.user_id
		WHERE f.id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, summary, series, rating, coverImageURL, coverThumbnailURL, userID, username, displayName, avatarURL, authorRole string
		var wordCount, chapterCount int
		if err := rows.Scan(&id, &title, &summary, &series, &rating, &coverImageURL, &coverThumbnailURL, &wordCount, &chapterCount, &userID, &username, &displayName, &avatarURL, &authorRole); err != nil {
			continue
		}
		img := coverThumbnailURL
		if img == "" {
			img = coverImageURL
		}
		uid, _ := uuid.Parse(userID)
		result["fanfic:"+id] = &dto.SharedContentPreview{
			ID:           id,
			ContentType:  "fanfic",
			Title:        title,
			Body:         truncateBody(summary, 200),
			ImageURL:     img,
			Series:       series,
			Rating:       rating,
			WordCount:    wordCount,
			ChapterCount: chapterCount,
			Author: &dto.UserResponse{
				ID:          uid,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Role:        role.Role(authorRole),
			},
			URL: "/fanfiction/" + id,
		}
	}
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

func (r *postRepository) GetStaleEmbeds(ctx context.Context, olderThan string, limit int) ([]model.EmbedRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE embed_type = 'link' AND fetched_at < datetime('now', ?) LIMIT ?`,
		olderThan, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get stale embeds: %w", err)
	}
	defer rows.Close()

	var embeds []model.EmbedRow
	for rows.Next() {
		var e model.EmbedRow
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.URL, &e.EmbedType, &e.Title, &e.Desc, &e.Image, &e.SiteName, &e.VideoID, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan stale embed: %w", err)
		}
		embeds = append(embeds, e)
	}
	return embeds, rows.Err()
}

func (r *postRepository) GetEmbeds(ctx context.Context, ownerID string, ownerType string) ([]model.EmbedRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE owner_id = ? AND owner_type = ? ORDER BY sort_order`,
		ownerID, ownerType,
	)
	if err != nil {
		return nil, fmt.Errorf("get embeds: %w", err)
	}
	defer rows.Close()

	var embeds []model.EmbedRow
	for rows.Next() {
		var e model.EmbedRow
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.URL, &e.EmbedType, &e.Title, &e.Desc, &e.Image, &e.SiteName, &e.VideoID, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan embed: %w", err)
		}
		embeds = append(embeds, e)
	}
	return embeds, rows.Err()
}

func (r *postRepository) GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string) (map[string][]model.EmbedRow, error) {
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

	result := make(map[string][]model.EmbedRow)
	for rows.Next() {
		var e model.EmbedRow
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.URL, &e.EmbedType, &e.Title, &e.Desc, &e.Image, &e.SiteName, &e.VideoID, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan embed: %w", err)
		}
		result[e.OwnerID] = append(result[e.OwnerID], e)
	}
	return result, rows.Err()
}

func (r *postRepository) CreatePollWithOptions(ctx context.Context, pollID uuid.UUID, postID uuid.UUID, durationSeconds int, expiresAt string, options []string) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO post_polls (id, post_id, duration_seconds, expires_at) VALUES (?, ?, ?, ?)`,
			pollID, postID, durationSeconds, expiresAt,
		); err != nil {
			return fmt.Errorf("create poll: %w", err)
		}
		for i, label := range options {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO post_poll_options (poll_id, label, sort_order) VALUES (?, ?, ?)`,
				pollID, label, i,
			); err != nil {
				return fmt.Errorf("add poll option: %w", err)
			}
		}
		return nil
	})
}

func (r *postRepository) GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID) (*model.PollRow, []model.PollOptionRow, *int, error) {
	var poll model.PollRow
	err := r.db.QueryRowContext(ctx,
		`SELECT id, post_id, duration_seconds, expires_at FROM post_polls WHERE post_id = ?`, postID,
	).Scan(&poll.ID, &poll.PostID, &poll.DurationSeconds, &poll.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get poll: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT o.id, o.poll_id, o.label, o.sort_order,
			(SELECT COUNT(*) FROM post_poll_votes WHERE option_id = o.id)
		FROM post_poll_options o
		WHERE o.poll_id = ?
		ORDER BY o.sort_order`, poll.ID,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get poll options: %w", err)
	}
	defer rows.Close()

	var options []model.PollOptionRow
	for rows.Next() {
		var o model.PollOptionRow
		if err := rows.Scan(&o.ID, &o.PollID, &o.Label, &o.SortOrder, &o.VoteCount); err != nil {
			return nil, nil, nil, fmt.Errorf("scan poll option: %w", err)
		}
		options = append(options, o)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	var votedOption *int
	if viewerID != uuid.Nil {
		var optID int
		err := r.db.QueryRowContext(ctx,
			`SELECT option_id FROM post_poll_votes WHERE poll_id = ? AND user_id = ?`, poll.ID, viewerID,
		).Scan(&optID)
		if err == nil {
			votedOption = &optID
		}
	}

	return &poll, options, votedOption, nil
}

func (r *postRepository) GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error) {
	if len(postIDs) == 0 {
		return nil, nil, nil, nil
	}

	placeholders := "?"
	args := []interface{}{postIDs[0]}
	for _, id := range postIDs[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}

	pollRows, err := r.db.QueryContext(ctx,
		`SELECT id, post_id, duration_seconds, expires_at FROM post_polls WHERE post_id IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch get polls: %w", err)
	}
	defer pollRows.Close()

	polls := make(map[uuid.UUID]*model.PollRow)
	var pollIDs []string
	for pollRows.Next() {
		var p model.PollRow
		if err := pollRows.Scan(&p.ID, &p.PostID, &p.DurationSeconds, &p.ExpiresAt); err != nil {
			return nil, nil, nil, fmt.Errorf("scan poll: %w", err)
		}
		postUUID, _ := uuid.Parse(p.PostID)
		polls[postUUID] = &p
		pollIDs = append(pollIDs, p.ID)
	}
	if err := pollRows.Err(); err != nil {
		return nil, nil, nil, err
	}
	if len(pollIDs) == 0 {
		return polls, nil, nil, nil
	}

	pPlaceholders := "?"
	pArgs := []interface{}{pollIDs[0]}
	for _, pid := range pollIDs[1:] {
		pPlaceholders += ", ?"
		pArgs = append(pArgs, pid)
	}

	optRows, err := r.db.QueryContext(ctx,
		`SELECT o.id, o.poll_id, o.label, o.sort_order,
			(SELECT COUNT(*) FROM post_poll_votes WHERE option_id = o.id)
		FROM post_poll_options o
		WHERE o.poll_id IN (`+pPlaceholders+`)
		ORDER BY o.sort_order`, pArgs...,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch get poll options: %w", err)
	}
	defer optRows.Close()

	optionsByPost := make(map[uuid.UUID][]model.PollOptionRow)
	pollToPost := make(map[string]uuid.UUID)
	for postUUID, p := range polls {
		pollToPost[p.ID] = postUUID
	}
	for optRows.Next() {
		var o model.PollOptionRow
		if err := optRows.Scan(&o.ID, &o.PollID, &o.Label, &o.SortOrder, &o.VoteCount); err != nil {
			return nil, nil, nil, fmt.Errorf("scan poll option: %w", err)
		}
		postUUID := pollToPost[o.PollID]
		optionsByPost[postUUID] = append(optionsByPost[postUUID], o)
	}
	if err := optRows.Err(); err != nil {
		return nil, nil, nil, err
	}

	votes := make(map[uuid.UUID]*int)
	if viewerID != uuid.Nil {
		vRows, err := r.db.QueryContext(ctx,
			`SELECT v.poll_id, v.option_id FROM post_poll_votes v
			WHERE v.poll_id IN (`+pPlaceholders+`) AND v.user_id = ?`,
			append(pArgs, viewerID)...,
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("batch get poll votes: %w", err)
		}
		defer vRows.Close()
		for vRows.Next() {
			var pollID string
			var optID int
			if err := vRows.Scan(&pollID, &optID); err != nil {
				return nil, nil, nil, fmt.Errorf("scan poll vote: %w", err)
			}
			postUUID := pollToPost[pollID]
			v := optID
			votes[postUUID] = &v
		}
	}

	return polls, optionsByPost, votes, nil
}

func (r *postRepository) VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO post_poll_votes (poll_id, user_id, option_id) VALUES (?, ?, ?)`,
		pollID, userID, optionID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "PRIMARY") {
			return fmt.Errorf("already voted")
		}
		return fmt.Errorf("vote poll: %w", err)
	}
	return nil
}
