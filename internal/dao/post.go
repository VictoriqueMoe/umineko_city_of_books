package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"umineko_city_of_books/internal/dao/utils"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	postDAO struct {
		db *sql.DB
		*commentDAO[uuid.UUID]
		*likeDAO
		*mediaDAO
		*viewDAO
	}
)

var sharedContentTables = map[string]string{
	"post":    "posts",
	"art":     "art_pieces",
	"ship":    "ships",
	"mystery": "mysteries",
	"theory":  "theories",
	"fanfic":  "fanfics",
}

func excludeClauseQ(column string, ids []uuid.UUID) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := utils.QuestionArgs(ids)

	return " AND " + column + " NOT IN (" + strings.Join(placeholders, ",") + ")", args
}

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

func scanPostRow(row interface{ Scan(...any) error }, p *model.PostRow) error {
	var (
		createdAt time.Time
		updatedAt sql.NullTime
	)
	err := row.Scan(
		&p.ID, &p.UserID, &p.Corner, &p.Body, &createdAt, &updatedAt,
		&p.AuthorUsername, &p.AuthorDisplayName, &p.AuthorAvatarURL,
		&p.AuthorRole,
		&p.LikeCount, &p.CommentCount, &p.UserLiked, &p.ViewCount, &p.ResolvedStatus,
		&p.SharedContentID, &p.SharedContentType,
	)
	if err != nil {
		return err
	}
	p.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if updatedAt.Valid {
		p.UpdatedAt = new(updatedAt.Time.UTC().Format(time.RFC3339))
	}
	return nil
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
		jitter := `((ascii(substr(p.id::text, 1, 1)) * 7 + ascii(substr(p.id::text, 5, 1)) * 13 + ?) % 1000) / 2500.0`
		if hasFollowBoost {
			return `
				ORDER BY (
					(1.0
						+ LEAST((SELECT COUNT(*) FROM post_likes WHERE post_id = p.id), 50) * 0.15
						+ LEAST((SELECT COUNT(*) FROM post_comments WHERE post_id = p.id), 30) * 0.3
						+ CASE WHEN EXISTS(SELECT 1 FROM follows WHERE follower_id = ? AND following_id = p.user_id) THEN 3.0 ELSE 0 END
					) / (1.0 + EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 3600.0 * 0.3)
					+ ` + jitter + `
				) DESC`
		}
		return `
			ORDER BY (
				(1.0
					+ LEAST((SELECT COUNT(*) FROM post_likes WHERE post_id = p.id), 50) * 0.15
					+ LEAST((SELECT COUNT(*) FROM post_comments WHERE post_id = p.id), 30) * 0.3
				) / (1.0 + EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 3600.0 * 0.3)
				+ ` + jitter + `
			) DESC`
	}
}

func (r *postDAO) Create(ctx context.Context, spec repository.NewPost, tx ...*sql.Tx) (*model.PostRow, error) {
	var (
		created           model.PostRow
		sharedContentID   *string
		sharedContentType *string
	)

	if spec.SharedContent != nil {
		sharedContentID = &spec.SharedContent.ID
		sharedContentType = &spec.SharedContent.Type
	}

	err := scanPostRow(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH p AS (
		     INSERT INTO posts (user_id, corner, body, shared_content_id, shared_content_type)
		     VALUES ($1, $2, $3, $4, $5)
		     RETURNING id, user_id, corner, body, created_at, updated_at, view_count, shared_content_id, shared_content_type
		 )
		 SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
		        u.username, u.display_name, u.avatar_url,
		        COALESCE(r.role, ''),
		        0, 0, FALSE, p.view_count, ''::text,
		        p.shared_content_id, p.shared_content_type
		 FROM p
		 JOIN users u ON u.id = p.user_id
		 LEFT JOIN user_roles r ON r.user_id = p.user_id`,
		spec.UserID, spec.Corner, spec.Body, sharedContentID, sharedContentType,
	), &created)
	if err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}

	return &created, nil
}

func (r *postDAO) AddMedia(ctx context.Context, spec repository.NewPostMedia, tx ...*sql.Tx) (int64, error) {
	return r.mediaDAO.AddMedia(ctx, spec.PostID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *postDAO) AddCommentMedia(ctx context.Context, spec repository.NewPostCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.commentDAO.AddCommentMedia(ctx, spec.CommentID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *postDAO) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.updatePost(ctx, id, &userID, body, tx...)
}

func (r *postDAO) UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.updatePost(ctx, id, nil, body, tx...)
}

func (r *postDAO) updatePost(ctx context.Context, id uuid.UUID, userID *uuid.UUID, body string, tx ...*sql.Tx) error {
	var (
		res sql.Result
		err error
	)
	if userID != nil {
		res, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE posts SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
			body, id, *userID,
		)
	} else {
		res, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE posts SET body = $1, updated_at = NOW() WHERE id = $2`,
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

func (r *postDAO) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.PostRow, error) {
	var p model.PostRow
	err := scanPostRow(txOrDB(r.db, tx).QueryRowContext(ctx, utils.Rebind(postSelectBase+` WHERE p.id = ?`), viewerID, id), &p)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get post: %w", err)
	}
	return &p, nil
}

func (r *postDAO) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM posts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("post not found or not owned")
	}
	return nil
}

func (r *postDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete post: %w", err)
	}
	return nil
}

func (r *postDAO) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	var total int
	whereParts := []string{"p.corner = ?"}
	args := []any{corner}

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
	exclSQL, exclArgs := excludeClauseQ("p.user_id", excludeUserIDs)
	whereClause += exclSQL
	countArgs := append(args, exclArgs...)

	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		utils.Rebind(`SELECT COUNT(*) FROM posts p JOIN users u ON p.user_id = u.id`+whereClause), countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	orderClause := postOrderClause(sort, true)
	query := postSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []any{viewerID}
	queryArgs = append(queryArgs, countArgs...)
	if sort == "" || sort == "relevance" {
		queryArgs = append(queryArgs, viewerID, seed)
	}
	queryArgs = append(queryArgs, limit, offset)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, utils.Rebind(query), queryArgs...)
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

func (r *postDAO) ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	var total int
	exclSQL, exclArgs := excludeClauseQ("user_id", excludeUserIDs)
	countQuery := `SELECT COUNT(*) FROM posts WHERE corner = ? AND (user_id = ? OR user_id IN (SELECT following_id FROM follows WHERE follower_id = ?))` + exclSQL
	countArgs := []any{corner, userID, userID}
	countArgs = append(countArgs, exclArgs...)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, utils.Rebind(countQuery), countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count following posts: %w", err)
	}

	exclSQL2, exclArgs2 := excludeClauseQ("p.user_id", excludeUserIDs)
	whereClause := ` WHERE p.corner = ? AND (p.user_id = ? OR p.user_id IN (SELECT following_id FROM follows WHERE follower_id = ?))` + exclSQL2
	orderClause := postOrderClause(sort, false)
	query := postSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []any{userID, corner, userID, userID}
	queryArgs = append(queryArgs, exclArgs2...)
	if sort == "" || sort == "relevance" {
		queryArgs = append(queryArgs, seed)
	}
	queryArgs = append(queryArgs, limit, offset)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, utils.Rebind(query), queryArgs...)
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

func (r *postDAO) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM posts WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user posts: %w", err)
	}

	query := utils.Rebind(postSelectBase + ` WHERE p.user_id = ? ORDER BY p.created_at DESC LIMIT ? OFFSET ?`)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, viewerID, userID, limit, offset)
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

func (r *postDAO) GetPostAuthorID(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var userID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT user_id FROM posts WHERE id = $1`, postID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get post author: %w", err)
	}
	return userID, nil
}

func (r *postDAO) GetSharedContentAuthor(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) (uuid.UUID, error) {
	table, ok := sharedContentTables[contentType]
	if !ok {
		return uuid.Nil, fmt.Errorf("unknown shared content type: %s", contentType)
	}
	var userID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT user_id FROM `+table+` WHERE id = $1`, contentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get shared content author: %w", err)
	}
	return userID, nil
}

func (r *postDAO) ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO suggestion_resolved (post_id, resolved_by, status) VALUES ($1, $2, $3)
		 ON CONFLICT (post_id) DO UPDATE SET status = $4, resolved_by = $5, resolved_at = NOW()`,
		postID, resolvedBy, status, status, resolvedBy,
	)
	if err != nil {
		return fmt.Errorf("resolve suggestion: %w", err)
	}
	return nil
}

func (r *postDAO) UnresolveSuggestion(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM suggestion_resolved WHERE post_id = $1`, postID)
	if err != nil {
		return fmt.Errorf("unresolve suggestion: %w", err)
	}
	return nil
}

func (r *postDAO) CountUserPostsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user posts today: %w", err)
	}
	return count, nil
}

func (r *postDAO) GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT corner, COUNT(*) FROM posts GROUP BY corner`)
	if err != nil {
		return nil, fmt.Errorf("corner counts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var (
			corner string
			count  int
		)
		if err := rows.Scan(&corner, &count); err != nil {
			return nil, fmt.Errorf("scan corner count: %w", err)
		}
		result[corner] = count
	}
	return result, rows.Err()
}

func (r *postDAO) GetShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COALESCE(share_count, 0) FROM share_counts WHERE content_id = $1 AND content_type = $2`,
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

func (r *postDAO) GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string, tx ...*sql.Tx) (map[string]int, error) {
	if len(contentIDs) == 0 {
		return nil, nil
	}

	placeholders, args := buildPlaceholders(contentIDs)
	args = append(args, contentType)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT content_id, share_count FROM share_counts WHERE content_id IN (`+placeholders+`) AND content_type = ?`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get share counts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var (
			id    string
			count int
		)
		if err := rows.Scan(&id, &count); err != nil {
			return nil, fmt.Errorf("scan share count: %w", err)
		}
		result[id] = count
	}
	return result, rows.Err()
}

func (r *postDAO) IncrementShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO share_counts (content_id, content_type, share_count) VALUES ($1, $2, 1) ON CONFLICT (content_id, content_type) DO UPDATE SET share_count = share_counts.share_count + 1`,
		contentID, contentType,
	)
	if err != nil {
		return fmt.Errorf("increment share count: %w", err)
	}
	return nil
}

func (r *postDAO) DecrementShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE share_counts SET share_count = GREATEST(share_count - 1, 0) WHERE content_id = $1 AND content_type = $2`,
		contentID, contentType,
	)
	if err != nil {
		return fmt.Errorf("decrement share count: %w", err)
	}
	return nil
}

func (r *postDAO) GetSharedContentFields(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (*string, *string, error) {
	var contentID, contentType *string
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT shared_content_id, shared_content_type FROM posts WHERE id = $1`, postID,
	).Scan(&contentID, &contentType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get shared content fields: %w", err)
	}
	return contentID, contentType, nil
}

func (r *postDAO) GetSharedContentPreviews(refs []repository.SharedContentRef, tx ...*sql.Tx) map[string]*dto.SharedContentPreview {
	result := make(map[string]*dto.SharedContentPreview)
	if len(refs) == 0 {
		return result
	}

	ctx := context.Background()

	grouped := make(map[string][]string)
	for _, ref := range refs {
		grouped[ref.Type] = append(grouped[ref.Type], ref.ID)
	}

	for contentType, ids := range grouped {
		switch contentType {
		case "post":
			r.fetchPostPreviews(ctx, ids, result, tx...)
		case "art":
			r.fetchArtPreviews(ctx, ids, result, tx...)
		case "ship":
			r.fetchShipPreviews(ctx, ids, result, tx...)
		case "mystery":
			r.fetchMysteryPreviews(ctx, ids, result, tx...)
		case "theory":
			r.fetchTheoryPreviews(ctx, ids, result, tx...)
		case "fanfic":
			r.fetchFanficPreviews(ctx, ids, result, tx...)
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

func buildPlaceholders[T any](ids []T) (string, []any) {
	placeholders, args := utils.QuestionArgs(ids)

	return strings.Join(placeholders, ", "), args
}

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}

func (r *postDAO) fetchPostPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT p.id, p.body, p.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM post_likes WHERE post_id = p.id) as like_count,
			(SELECT COUNT(*) FROM post_comments WHERE post_id = p.id) as comment_count,
			p.corner
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = p.user_id
		WHERE p.id IN (`+placeholders+`)`), args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, body, userID, username, displayName, avatarURL, authorRole, corner string
			likeCount, commentCount                                                int
		)
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

	mediaRows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT post_id, media_url, media_type, thumbnail_url, sort_order
		FROM post_media WHERE post_id IN (`+placeholders+`) ORDER BY sort_order LIMIT 4`), args...,
	)
	if err != nil {
		return
	}
	defer mediaRows.Close()

	for mediaRows.Next() {
		var (
			postID, mediaURL, mediaType, thumbnailURL string
			sortOrder                                 int
		)
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

func (r *postDAO) fetchArtPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT a.id, a.title, a.description, a.image_url, a.thumbnail_url, a.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''), a.corner
		FROM art a
		JOIN users u ON a.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = a.user_id
		WHERE a.id IN (`+placeholders+`)`), args...,
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

func (r *postDAO) fetchShipPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT s.id, s.title, s.description, s.image_url, s.thumbnail_url, s.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0)
		FROM ships s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = s.user_id
		WHERE s.id IN (`+placeholders+`)`), args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, title, description, imageURL, thumbnailURL, userID, username, displayName, avatarURL, authorRole string
			voteScore                                                                                            int
		)
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

func (r *postDAO) fetchMysteryPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT m.id, m.title, m.body, m.difficulty, m.solved, m.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM mysteries m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = m.user_id
		WHERE m.id IN (`+placeholders+`)`), args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, title, body, difficulty, userID, username, displayName, avatarURL, authorRole string
			solved                                                                            bool
		)
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

func (r *postDAO) fetchTheoryPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT t.id, t.title, t.body, t.series, t.credibility_score, t.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM theories t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = t.user_id
		WHERE t.id IN (`+placeholders+`)`), args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, title, body, series, userID, username, displayName, avatarURL, authorRole string
			credibilityScore                                                              float64
		)
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

func (r *postDAO) fetchFanficPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT f.id, f.title, f.summary, f.series, f.rating, f.cover_image_url, f.cover_thumbnail_url, f.word_count,
			(SELECT COUNT(*) FROM fanfic_chapters WHERE fanfic_id = f.id),
			f.user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM fanfics f
		JOIN users u ON f.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = f.user_id
		WHERE f.id IN (`+placeholders+`)`), args...,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, title, summary, series, rating, coverImageURL, coverThumbnailURL, userID, username, displayName, avatarURL, authorRole string
			wordCount, chapterCount                                                                                                    int
		)
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

func (r *postDAO) AddEmbed(ctx context.Context, spec repository.NewEmbed, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO embeds (owner_id, owner_type, url, embed_type, title, description, image, site_name, video_id, sort_order) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		spec.OwnerID, spec.OwnerType, spec.URL, spec.EmbedType, spec.Title, spec.Description, spec.Image, spec.SiteName, spec.VideoID, spec.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("add embed: %w", err)
	}
	return nil
}

func (r *postDAO) DeleteEmbeds(ctx context.Context, ownerID string, ownerType string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM embeds WHERE owner_id = $1 AND owner_type = $2`, ownerID, ownerType)
	if err != nil {
		return fmt.Errorf("delete embeds: %w", err)
	}
	return nil
}

func (r *postDAO) UpdateEmbed(ctx context.Context, spec repository.EmbedUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE embeds SET title = $1, description = $2, image = $3, site_name = $4, fetched_at = NOW() WHERE id = $5`,
		spec.Title, spec.Description, spec.Image, spec.SiteName, spec.ID,
	)
	if err != nil {
		return fmt.Errorf("update embed: %w", err)
	}
	return nil
}

func (r *postDAO) GetStaleEmbeds(ctx context.Context, olderThan string, limit int, tx ...*sql.Tx) ([]model.EmbedRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE embed_type = 'link' AND fetched_at < NOW() + CAST($1 AS INTERVAL) LIMIT $2`,
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

func (r *postDAO) GetEmbeds(ctx context.Context, ownerID string, ownerType string, tx ...*sql.Tx) ([]model.EmbedRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE owner_id = $1 AND owner_type = $2 ORDER BY sort_order`,
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

func (r *postDAO) GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string, tx ...*sql.Tx) (map[string][]model.EmbedRow, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}

	placeholders, args := buildPlaceholders(ownerIDs)
	args = append(args, ownerType)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT id, owner_id, url, embed_type, title, description, image, site_name, video_id, sort_order FROM embeds WHERE owner_id IN (`+placeholders+`) AND owner_type = ? ORDER BY sort_order`),
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

func (r *postDAO) CreatePoll(ctx context.Context, postID uuid.UUID, durationSeconds int, expiresAt string, tx ...*sql.Tx) (*model.PollRow, error) {
	var (
		created model.PollRow
		expires time.Time
	)

	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO post_polls (post_id, duration_seconds, expires_at) VALUES ($1, $2, $3)
		 RETURNING id, post_id, duration_seconds, expires_at`,
		postID, durationSeconds, expiresAt,
	).Scan(&created.ID, &created.PostID, &created.DurationSeconds, &expires); err != nil {
		return nil, fmt.Errorf("create poll: %w", err)
	}

	created.ExpiresAt = expires.UTC().Format(time.RFC3339)

	return &created, nil
}

func (r *postDAO) AddPollOption(ctx context.Context, pollID string, label string, sortOrder int, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO post_poll_options (poll_id, label, sort_order) VALUES ($1, $2, $3)`,
		pollID, label, sortOrder,
	); err != nil {
		return fmt.Errorf("add poll option: %w", err)
	}

	return nil
}

func (r *postDAO) GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.PollRow, []model.PollOptionRow, *int, error) {
	var (
		poll      model.PollRow
		expiresAt time.Time
	)
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id, post_id, duration_seconds, expires_at FROM post_polls WHERE post_id = $1`, postID,
	).Scan(&poll.ID, &poll.PostID, &poll.DurationSeconds, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get poll: %w", err)
	}
	poll.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT o.id, o.poll_id, o.label, o.sort_order,
			(SELECT COUNT(*) FROM post_poll_votes WHERE option_id = o.id)
		FROM post_poll_options o
		WHERE o.poll_id = $1
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
		err := txOrDB(r.db, tx).QueryRowContext(ctx,
			`SELECT option_id FROM post_poll_votes WHERE poll_id = $1 AND user_id = $2`, poll.ID, viewerID,
		).Scan(&optID)
		if err == nil {
			votedOption = &optID
		}
	}

	return &poll, options, votedOption, nil
}

func (r *postDAO) GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error) {
	if len(postIDs) == 0 {
		return nil, nil, nil, nil
	}

	placeholders, args := buildPlaceholders(postIDs)

	pollRows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT id, post_id, duration_seconds, expires_at FROM post_polls WHERE post_id IN (`+placeholders+`)`), args...,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch get polls: %w", err)
	}
	defer pollRows.Close()

	polls := make(map[uuid.UUID]*model.PollRow)
	pollToPost := make(map[string]uuid.UUID)
	var pollIDs []string
	for pollRows.Next() {
		var (
			p         model.PollRow
			expiresAt time.Time
		)
		if err := pollRows.Scan(&p.ID, &p.PostID, &p.DurationSeconds, &expiresAt); err != nil {
			return nil, nil, nil, fmt.Errorf("scan poll: %w", err)
		}
		p.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
		postUUID, _ := uuid.Parse(p.PostID)
		polls[postUUID] = &p
		pollToPost[p.ID] = postUUID
		pollIDs = append(pollIDs, p.ID)
	}
	if err := pollRows.Err(); err != nil {
		return nil, nil, nil, err
	}
	if len(pollIDs) == 0 {
		return polls, nil, nil, nil
	}

	pPlaceholders, pArgs := buildPlaceholders(pollIDs)

	optRows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(`SELECT o.id, o.poll_id, o.label, o.sort_order,
			(SELECT COUNT(*) FROM post_poll_votes WHERE option_id = o.id)
		FROM post_poll_options o
		WHERE o.poll_id IN (`+pPlaceholders+`)
		ORDER BY o.sort_order`), pArgs...,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch get poll options: %w", err)
	}
	defer optRows.Close()

	optionsByPost := make(map[uuid.UUID][]model.PollOptionRow)
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
		vRows, err := txOrDB(r.db, tx).QueryContext(ctx,
			utils.Rebind(`SELECT v.poll_id, v.option_id FROM post_poll_votes v
			WHERE v.poll_id IN (`+pPlaceholders+`) AND v.user_id = ?`),
			append(pArgs, viewerID)...,
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("batch get poll votes: %w", err)
		}
		defer vRows.Close()
		for vRows.Next() {
			var (
				pollID string
				optID  int
			)
			if err := vRows.Scan(&pollID, &optID); err != nil {
				return nil, nil, nil, fmt.Errorf("scan poll vote: %w", err)
			}
			postUUID := pollToPost[pollID]
			votes[postUUID] = new(optID)
		}
	}

	return polls, optionsByPost, votes, nil
}

func (r *postDAO) VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO post_poll_votes (poll_id, user_id, option_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		pollID, userID, optionID,
	)
	if err != nil {
		return fmt.Errorf("vote poll: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("already voted")
	}
	return nil
}
