package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)


type (
	ArtRepository interface {
		CreateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, artType string, title string, description string, imageURL string, thumbnailURL string, tags []string, isSpoiler bool) error
		UpdateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, tags []string, isSpoiler bool, asAdmin bool) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ArtRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error)
		GetArtAuthorID(ctx context.Context, artID uuid.UUID) (uuid.UUID, error)
		GetImageURL(ctx context.Context, artID uuid.UUID) (string, error)

		Like(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		Unlike(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		GetLikedBy(ctx context.Context, artID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, artID uuid.UUID, viewerHash string) (bool, error)

		GetTags(ctx context.Context, artID uuid.UUID) ([]string, error)
		GetTagsBatch(ctx context.Context, artIDs []uuid.UUID) (map[uuid.UUID][]string, error)
		GetPopularTags(ctx context.Context, corner string, limit int) ([]model.TagCount, error)

		GetCornerCounts(ctx context.Context) (map[string]int, error)
		CountUserArtToday(ctx context.Context, userID uuid.UUID) (int, error)

		CreateComment(ctx context.Context, id uuid.UUID, artID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtCommentRow, int, error)
		GetCommentArtID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error

		SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error

		CreateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error
		UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error
		SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error
		DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		GetGalleryByID(ctx context.Context, id uuid.UUID) (*model.GalleryRow, error)
		ListGalleriesByUser(ctx context.Context, userID uuid.UUID) ([]model.GalleryRow, error)
		ListAllGalleries(ctx context.Context, corner string) ([]model.GalleryRow, error)
		GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int) ([]PreviewImage, error)
		ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error)
	}

	artRepository struct {
		db *sql.DB
	}
)

const artSelectBase = `
	SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
		a.gallery_id, a.created_at, a.updated_at,
		u.username, u.display_name, u.avatar_url,
		COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM art_likes WHERE art_id = a.id),
		(SELECT COUNT(*) FROM art_comments WHERE art_id = a.id),
		a.view_count,
		EXISTS(SELECT 1 FROM art_likes WHERE art_id = a.id AND user_id = $1),
		a.is_spoiler
	FROM art a
	JOIN users u ON a.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = a.user_id`

func scanArtRow(row interface{ Scan(...interface{}) error }, a *model.ArtRow) error {
	var createdAt time.Time
	var updatedAt *time.Time
	err := row.Scan(
		&a.ID, &a.UserID, &a.Corner, &a.ArtType, &a.Title, &a.Description, &a.ImageURL, &a.ThumbnailURL,
		&a.GalleryID, &createdAt, &updatedAt,
		&a.AuthorUsername, &a.AuthorDisplayName, &a.AuthorAvatarURL,
		&a.AuthorRole,
		&a.LikeCount, &a.CommentCount, &a.ViewCount, &a.UserLiked, &a.IsSpoiler,
	)
	if err != nil {
		return err
	}
	a.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	a.UpdatedAt = timePtrToString(updatedAt)
	return nil
}

func (r *artRepository) CreateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, artType string, title string, description string, imageURL string, thumbnailURL string, tags []string, isSpoiler bool) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO art (id, user_id, corner, art_type, title, description, image_url, thumbnail_url, is_spoiler) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			id, userID, corner, artType, title, description, imageURL, thumbnailURL, isSpoiler,
		); err != nil {
			return fmt.Errorf("create art: %w", err)
		}
		return insertArtTagsTx(ctx, tx, id, tags)
	})
}

func (r *artRepository) UpdateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, tags []string, isSpoiler bool, asAdmin bool) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var res sql.Result
		var err error
		if asAdmin {
			res, err = tx.ExecContext(ctx,
				`UPDATE art SET title = $1, description = $2, is_spoiler = $3, updated_at = NOW() WHERE id = $4`,
				title, description, isSpoiler, id,
			)
		} else {
			res, err = tx.ExecContext(ctx,
				`UPDATE art SET title = $1, description = $2, is_spoiler = $3, updated_at = NOW() WHERE id = $4 AND user_id = $5`,
				title, description, isSpoiler, id, userID,
			)
		}
		if err != nil {
			return fmt.Errorf("update art: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("art not found or not owned")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM art_tags WHERE art_id = $1`, id); err != nil {
			return fmt.Errorf("delete art tags: %w", err)
		}
		return insertArtTagsTx(ctx, tx, id, tags)
	})
}

func insertArtTagsTx(ctx context.Context, tx *sql.Tx, artID uuid.UUID, tags []string) error {
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO art_tags (art_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			artID, tag,
		); err != nil {
			return fmt.Errorf("add art tag: %w", err)
		}
	}
	return nil
}

func (r *artRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ArtRow, error) {
	var a model.ArtRow
	err := scanArtRow(r.db.QueryRowContext(ctx, artSelectBase+` WHERE a.id = $2`, viewerID, id), &a)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get art: %w", err)
	}
	return &a, nil
}

func (r *artRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM art WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete art: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("art not found or not owned")
	}
	return nil
}

func (r *artRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM art WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete art: %w", err)
	}
	return nil
}

func artOrderClause(sort string) string {
	switch sort {
	case "popular":
		return ` ORDER BY (SELECT COUNT(*) FROM art_likes WHERE art_id = a.id) DESC, a.created_at DESC`
	case "views":
		return ` ORDER BY a.view_count DESC, a.created_at DESC`
	default:
		return ` ORDER BY a.created_at DESC`
	}
}

func (r *artRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtRow, int, error) {
	var total int
	buildWhere := func(startIdx int) (string, []interface{}, int) {
		idx := startIdx
		next := func() string {
			s := fmt.Sprintf("$%d", idx)
			idx++
			return s
		}
		parts := []string{"a.corner = " + next()}
		args := []interface{}{corner}
		if artType != "" {
			parts = append(parts, "a.art_type = "+next())
			args = append(args, artType)
		}
		if search != "" {
			parts = append(parts, "(a.title LIKE "+next()+" OR a.description LIKE "+next()+" OR u.display_name LIKE "+next()+" OR u.username LIKE "+next()+")")
			like := "%" + search + "%"
			args = append(args, like, like, like, like)
		}
		if tag != "" {
			parts = append(parts, "EXISTS(SELECT 1 FROM art_tags WHERE art_id = a.id AND tag = "+next()+")")
			args = append(args, tag)
		}
		exclSQL, exclArgs := ExcludeClause("a.user_id", excludeUserIDs, idx)
		idx += len(exclArgs)
		args = append(args, exclArgs...)
		return " WHERE " + strings.Join(parts, " AND ") + exclSQL, args, idx
	}

	countWhere, countArgs, _ := buildWhere(1)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM art a JOIN users u ON a.user_id = u.id`+countWhere, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count art: %w", err)
	}

	listWhere, listArgs, nextIdx := buildWhere(2)
	limitPH := fmt.Sprintf("$%d", nextIdx)
	offsetPH := fmt.Sprintf("$%d", nextIdx+1)

	orderClause := artOrderClause(sort)
	query := artSelectBase + listWhere + orderClause + ` LIMIT ` + limitPH + ` OFFSET ` + offsetPH

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, listArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list art: %w", err)
	}
	defer rows.Close()

	var arts []model.ArtRow
	for rows.Next() {
		var a model.ArtRow
		if err := scanArtRow(rows, &a); err != nil {
			return nil, 0, fmt.Errorf("scan art: %w", err)
		}
		arts = append(arts, a)
	}
	return arts, total, rows.Err()
}

func (r *artRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM art WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user art: %w", err)
	}

	query := artSelectBase + ` WHERE a.user_id = $2 ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, viewerID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user art: %w", err)
	}
	defer rows.Close()

	var arts []model.ArtRow
	for rows.Next() {
		var a model.ArtRow
		if err := scanArtRow(rows, &a); err != nil {
			return nil, 0, fmt.Errorf("scan art: %w", err)
		}
		arts = append(arts, a)
	}
	return arts, total, rows.Err()
}

func (r *artRepository) GetArtAuthorID(ctx context.Context, artID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM art WHERE id = $1`, artID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get art author: %w", err)
	}
	return userID, nil
}

func (r *artRepository) GetImageURL(ctx context.Context, artID uuid.UUID) (string, error) {
	var url string
	err := r.db.QueryRowContext(ctx, `SELECT image_url FROM art WHERE id = $1`, artID).Scan(&url)
	if err != nil {
		return "", fmt.Errorf("get art image url: %w", err)
	}
	return url, nil
}

func (r *artRepository) Like(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO art_likes (user_id, art_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, artID,
	)
	if err != nil {
		return fmt.Errorf("like art: %w", err)
	}
	return nil
}

func (r *artRepository) Unlike(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM art_likes WHERE user_id = $1 AND art_id = $2`,
		userID, artID,
	)
	if err != nil {
		return fmt.Errorf("unlike art: %w", err)
	}
	return nil
}

func (r *artRepository) GetLikedBy(ctx context.Context, artID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error) {
	exclSQL, exclArgs := ExcludeClause("al.user_id", excludeUserIDs, 2)
	queryArgs := []interface{}{artID}
	queryArgs = append(queryArgs, exclArgs...)
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM art_likes al
		JOIN users u ON al.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE al.art_id = $1`+exclSQL+`
		ORDER BY al.created_at DESC`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("get art liked by: %w", err)
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

func (r *artRepository) RecordView(ctx context.Context, artID uuid.UUID, viewerHash string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO art_views (art_id, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		artID, viewerHash,
	)
	if err != nil {
		return false, fmt.Errorf("record art view: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		_, err = r.db.ExecContext(ctx, `UPDATE art SET view_count = view_count + 1 WHERE id = $1`, artID)
		if err != nil {
			return false, fmt.Errorf("increment art view count: %w", err)
		}
	}
	return n > 0, nil
}

func (r *artRepository) GetTags(ctx context.Context, artID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tag FROM art_tags WHERE art_id = $1 ORDER BY tag`, artID)
	if err != nil {
		return nil, fmt.Errorf("get art tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan art tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *artRepository) GetTagsBatch(ctx context.Context, artIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(artIDs) == 0 {
		return nil, nil
	}

	placeholders := "$1"
	args := []interface{}{artIDs[0]}
	for i, id := range artIDs[1:] {
		placeholders += fmt.Sprintf(", $%d", i+2)
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT art_id, tag FROM art_tags WHERE art_id IN (`+placeholders+`) ORDER BY tag`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get art tags: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]string)
	for rows.Next() {
		var artID uuid.UUID
		var tag string
		if err := rows.Scan(&artID, &tag); err != nil {
			return nil, fmt.Errorf("scan art tag: %w", err)
		}
		result[artID] = append(result[artID], tag)
	}
	return result, rows.Err()
}

func (r *artRepository) GetPopularTags(ctx context.Context, corner string, limit int) ([]model.TagCount, error) {
	query := `SELECT t.tag, COUNT(*) as cnt FROM art_tags t JOIN art a ON t.art_id = a.id`
	var args []interface{}

	if corner != "" {
		query += ` WHERE a.corner = $1`
		args = append(args, corner)
		query += ` GROUP BY t.tag ORDER BY cnt DESC LIMIT $2`
	} else {
		query += ` GROUP BY t.tag ORDER BY cnt DESC LIMIT $1`
	}
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get popular tags: %w", err)
	}
	defer rows.Close()

	var tags []model.TagCount
	for rows.Next() {
		var t model.TagCount
		if err := rows.Scan(&t.Tag, &t.Count); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *artRepository) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT corner, COUNT(*) FROM art GROUP BY corner`)
	if err != nil {
		return nil, fmt.Errorf("art corner counts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var corner string
		var count int
		if err := rows.Scan(&corner, &count); err != nil {
			return nil, fmt.Errorf("scan art corner count: %w", err)
		}
		result[corner] = count
	}
	return result, rows.Err()
}

func (r *artRepository) CountUserArtToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM art WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user art today: %w", err)
	}
	return count, nil
}

func (r *artRepository) CreateComment(ctx context.Context, id uuid.UUID, artID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO art_comments (id, art_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, artID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create art comment: %w", err)
	}
	return nil
}

func (r *artRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.updateArtComment(ctx, id, &userID, body)
}

func (r *artRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.updateArtComment(ctx, id, nil, body)
}

func (r *artRepository) updateArtComment(ctx context.Context, id uuid.UUID, userID *uuid.UUID, body string) error {
	var res sql.Result
	var err error
	if userID != nil {
		res, err = r.db.ExecContext(ctx,
			`UPDATE art_comments SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
			body, id, *userID,
		)
	} else {
		res, err = r.db.ExecContext(ctx,
			`UPDATE art_comments SET body = $1, updated_at = NOW() WHERE id = $2`,
			body, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update art comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("art comment not found or not owned")
	}
	return nil
}

func (r *artRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM art_comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete art comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("art comment not found or not owned")
	}
	return nil
}

func (r *artRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM art_comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete art comment: %w", err)
	}
	return nil
}

func (r *artRepository) GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtCommentRow, int, error) {
	var total int
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs, 2)
	countArgs := []interface{}{artID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM art_comments WHERE art_id = $1`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count art comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.art_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url,
			COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM art_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM art_comment_likes WHERE comment_id = c.id AND user_id = $1)
		FROM art_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.art_id = $2`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT `+limitPH+` OFFSET `+offsetPH,
		append([]interface{}{viewerID, artID}, append(exclArgs2, limit, offset)...)...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get art comments: %w", err)
	}
	defer rows.Close()

	var comments []model.ArtCommentRow
	for rows.Next() {
		var c model.ArtCommentRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.ArtID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL,
			&c.AuthorRole, &c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan art comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		c.UpdatedAt = timePtrToString(updatedAt)
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *artRepository) GetCommentArtID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var artID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT art_id FROM art_comments WHERE id = $1`, commentID).Scan(&artID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get comment art id: %w", err)
	}
	return artID, nil
}

func (r *artRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM art_comments WHERE id = $1`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get art comment author: %w", err)
	}
	return userID, nil
}

func (r *artRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO art_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like art comment: %w", err)
	}
	return nil
}

func (r *artRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM art_comment_likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike art comment: %w", err)
	}
	return nil
}

func (r *artRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO art_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add art comment media: %w", err)
	}
	return id, nil
}

func (r *artRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM art_comment_media WHERE comment_id = $1 ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get art comment media: %w", err)
	}
	defer rows.Close()

	var mediaList []model.PostMediaRow
	for rows.Next() {
		var m model.PostMediaRow
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan art comment media: %w", err)
		}
		mediaList = append(mediaList, m)
	}
	return mediaList, rows.Err()
}

func (r *artRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "$1"
	args := []interface{}{commentIDs[0]}
	for i, id := range commentIDs[1:] {
		placeholders += fmt.Sprintf(", $%d", i+2)
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM art_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get art comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.PostMediaRow)
	for rows.Next() {
		var m model.PostMediaRow
		var commentID uuid.UUID
		if err := rows.Scan(&m.ID, &commentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan art comment media: %w", err)
		}
		result[commentID] = append(result[commentID], m)
	}
	return result, rows.Err()
}

func (r *artRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE art_comment_media SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update art comment media url: %w", err)
	}
	return nil
}

func (r *artRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE art_comment_media SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update art comment media thumbnail: %w", err)
	}
	return nil
}

func (r *artRepository) SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE art SET gallery_id = $1 WHERE id = $2 AND user_id = $3`,
		galleryID, artID, userID,
	)
	if err != nil {
		return fmt.Errorf("set art gallery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("art not found or not owned")
	}
	return nil
}

func (r *artRepository) CreateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO galleries (id, user_id, name, description) VALUES ($1, $2, $3, $4)`,
		id, userID, name, description,
	)
	if err != nil {
		return fmt.Errorf("create gallery: %w", err)
	}
	return nil
}

func (r *artRepository) UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE galleries SET name = $1, description = $2, updated_at = NOW() WHERE id = $3 AND user_id = $4`,
		name, description, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update gallery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}
	return nil
}

func (r *artRepository) SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE galleries SET cover_art_id = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		coverArtID, galleryID, userID,
	)
	if err != nil {
		return fmt.Errorf("set gallery cover: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}
	return nil
}

func (r *artRepository) DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM art WHERE gallery_id = $1 AND user_id = $2`,
			id, userID,
		); err != nil {
			return fmt.Errorf("delete art in gallery: %w", err)
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM galleries WHERE id = $1 AND user_id = $2`, id, userID)
		if err != nil {
			return fmt.Errorf("delete gallery: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("gallery not found or not owned")
		}
		return nil
	})
}

func (r *artRepository) GetGalleryByID(ctx context.Context, id uuid.UUID) (*model.GalleryRow, error) {
	var g model.GalleryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
			COALESCE(a.image_url, ''), COALESCE(a.thumbnail_url, ''),
			(SELECT COUNT(*) FROM art WHERE gallery_id = g.id),
			g.created_at, g.updated_at,
			u.username, u.display_name, u.avatar_url
		FROM galleries g
		JOIN users u ON g.user_id = u.id
		LEFT JOIN art a ON g.cover_art_id = a.id
		WHERE g.id = $1`,
		id,
	).Scan(
		&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
		&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
		&createdAt, &updatedAt,
		&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get gallery: %w", err)
	}
	g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	g.UpdatedAt = timePtrToString(updatedAt)
	return &g, nil
}

func (r *artRepository) ListGalleriesByUser(ctx context.Context, userID uuid.UUID) ([]model.GalleryRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
			COALESCE(a.image_url, ''), COALESCE(a.thumbnail_url, ''),
			(SELECT COUNT(*) FROM art WHERE gallery_id = g.id),
			g.created_at, g.updated_at,
			u.username, u.display_name, u.avatar_url
		FROM galleries g
		JOIN users u ON g.user_id = u.id
		LEFT JOIN art a ON g.cover_art_id = a.id
		WHERE g.user_id = $1
		ORDER BY g.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list galleries: %w", err)
	}
	defer rows.Close()

	var galleries []model.GalleryRow
	for rows.Next() {
		var g model.GalleryRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
			&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
			&createdAt, &updatedAt,
			&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
		); err != nil {
			return nil, fmt.Errorf("scan gallery: %w", err)
		}
		g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		g.UpdatedAt = timePtrToString(updatedAt)
		galleries = append(galleries, g)
	}
	return galleries, rows.Err()
}

func (r *artRepository) ListAllGalleries(ctx context.Context, corner string) ([]model.GalleryRow, error) {
	query := `SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
			COALESCE(a.image_url, ''), COALESCE(a.thumbnail_url, ''),
			(SELECT COUNT(*) FROM art WHERE gallery_id = g.id),
			g.created_at, g.updated_at,
			u.username, u.display_name, u.avatar_url
		FROM galleries g
		JOIN users u ON g.user_id = u.id
		LEFT JOIN art a ON g.cover_art_id = a.id`
	args := []interface{}{}

	if corner != "" {
		query += ` WHERE EXISTS(SELECT 1 FROM art WHERE gallery_id = g.id AND corner = $1)`
		args = append(args, corner)
	}

	query += ` ORDER BY g.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all galleries: %w", err)
	}
	defer rows.Close()

	var galleries []model.GalleryRow
	for rows.Next() {
		var g model.GalleryRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
			&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
			&createdAt, &updatedAt,
			&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
		); err != nil {
			return nil, fmt.Errorf("scan gallery: %w", err)
		}
		g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		g.UpdatedAt = timePtrToString(updatedAt)
		galleries = append(galleries, g)
	}
	return galleries, rows.Err()
}

type PreviewImage struct {
	ThumbnailURL string
	ImageURL     string
}

func (r *artRepository) GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int) ([]PreviewImage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT thumbnail_url, image_url FROM art WHERE gallery_id = $1 ORDER BY created_at DESC LIMIT $2`,
		galleryID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get gallery preview images: %w", err)
	}
	defer rows.Close()

	var imgs []PreviewImage
	for rows.Next() {
		var p PreviewImage
		if err := rows.Scan(&p.ThumbnailURL, &p.ImageURL); err != nil {
			return nil, fmt.Errorf("scan preview image: %w", err)
		}
		imgs = append(imgs, p)
	}
	return imgs, rows.Err()
}

func (r *artRepository) ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM art WHERE gallery_id = $1`, galleryID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count gallery art: %w", err)
	}

	query := artSelectBase + ` WHERE a.gallery_id = $2 ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, viewerID, galleryID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list gallery art: %w", err)
	}
	defer rows.Close()

	var arts []model.ArtRow
	for rows.Next() {
		var a model.ArtRow
		if err := scanArtRow(rows, &a); err != nil {
			return nil, 0, fmt.Errorf("scan gallery art: %w", err)
		}
		arts = append(arts, a)
	}
	return arts, total, rows.Err()
}
