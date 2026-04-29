package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	OCRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string) error
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string, asAdmin bool) error
		UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.OCRow, error)
		GetAuthorID(ctx context.Context, ocID uuid.UUID) (uuid.UUID, error)
		List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.OCRow, int, error)
		ListSummariesByUser(ctx context.Context, userID uuid.UUID) ([]model.OCSummaryRow, error)
		HasOC(ctx context.Context, userID uuid.UUID, name string) (bool, error)

		AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int) (int64, error)
		UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string) error
		UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int) error
		DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID) error
		GetGallery(ctx context.Context, ocID uuid.UUID) ([]model.OCImageRow, error)
		GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID) (map[uuid.UUID][]model.OCImageRow, error)

		Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error
		Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error
		Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error

		CreateComment(ctx context.Context, id uuid.UUID, ocID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCCommentRow, int, error)
		GetCommentOCID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.OCCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.OCCommentMediaRow, error)
	}

	ocRepository struct {
		db *sql.DB
	}
)

const ocSelectBase = `
	SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
		o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		COALESCE((SELECT SUM(value) FROM oc_votes WHERE oc_id = o.id), 0),
		COALESCE((SELECT value FROM oc_votes WHERE oc_id = o.id AND user_id = $1), 0),
		(SELECT COUNT(*) FROM oc_favourites WHERE oc_id = o.id),
		EXISTS(SELECT 1 FROM oc_favourites WHERE oc_id = o.id AND user_id = $1),
		(SELECT COUNT(*) FROM oc_comments WHERE oc_id = o.id)
	FROM ocs o
	JOIN users u ON o.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = o.user_id`

func scanOCRow(row interface{ Scan(...interface{}) error }, o *model.OCRow) error {
	var createdAt, updatedAt time.Time
	if err := row.Scan(
		&o.ID, &o.UserID, &o.Name, &o.Description, &o.Series, &o.CustomSeriesName,
		&o.ImageURL, &o.ThumbnailURL, &createdAt, &updatedAt,
		&o.AuthorUsername, &o.AuthorDisplayName, &o.AuthorAvatarURL, &o.AuthorRole,
		&o.VoteScore, &o.UserVote, &o.FavouriteCount, &o.UserFavourited, &o.CommentCount,
	); err != nil {
		return err
	}
	o.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	updated := updatedAt.UTC().Format(time.RFC3339)
	o.UpdatedAt = &updated
	return nil
}

func (r *ocRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ocs (id, user_id, name, description, series, custom_series_name) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, name, description, series, customSeriesName,
	)
	if err != nil {
		return fmt.Errorf("create oc: %w", err)
	}
	return nil
}

func (r *ocRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string, asAdmin bool) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var res sql.Result
		var err error
		if asAdmin {
			res, err = tx.ExecContext(ctx,
				`UPDATE ocs SET name = $1, description = $2, series = $3, custom_series_name = $4, updated_at = NOW() WHERE id = $5`,
				name, description, series, customSeriesName, id,
			)
		} else {
			res, err = tx.ExecContext(ctx,
				`UPDATE ocs SET name = $1, description = $2, series = $3, custom_series_name = $4, updated_at = NOW() WHERE id = $5 AND user_id = $6`,
				name, description, series, customSeriesName, id, userID,
			)
		}
		if err != nil {
			return fmt.Errorf("update oc: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("oc not found or not owned")
		}
		return nil
	})
}

func (r *ocRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ocs SET image_url = $1, thumbnail_url = $2 WHERE id = $3`,
		imageURL, thumbnailURL, id,
	)
	if err != nil {
		return fmt.Errorf("update oc image: %w", err)
	}
	return nil
}

func (r *ocRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ocs WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete oc: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("oc not found or not owned")
	}
	return nil
}

func (r *ocRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ocs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete oc: %w", err)
	}
	return nil
}

func (r *ocRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.OCRow, error) {
	var o model.OCRow
	err := scanOCRow(r.db.QueryRowContext(ctx, ocSelectBase+` WHERE o.id = $2`, viewerID, id), &o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get oc: %w", err)
	}
	return &o, nil
}

func (r *ocRepository) GetAuthorID(ctx context.Context, ocID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM ocs WHERE id = $1`, ocID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get oc author: %w", err)
	}
	return userID, nil
}

func (r *ocRepository) HasOC(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM ocs WHERE user_id = $1 AND lower(name) = lower($2))`,
		userID, name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check oc exists: %w", err)
	}
	return exists, nil
}

func (r *ocRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCRow, int, error) {
	buildWhere := func(startIdx int) (string, []interface{}, int) {
		idx := startIdx
		next := func() string {
			s := fmt.Sprintf("$%d", idx)
			idx++
			return s
		}
		parts := []string{"1=1"}
		var args []interface{}
		if series != "" {
			parts = append(parts, "o.series = "+next())
			args = append(args, series)
		}
		if customSeriesName != "" {
			parts = append(parts, "lower(o.custom_series_name) = lower("+next()+")")
			args = append(args, customSeriesName)
		}
		if ownerID != uuid.Nil {
			parts = append(parts, "o.user_id = "+next())
			args = append(args, ownerID)
		}
		if crackOCsOnly {
			parts = append(parts, fmt.Sprintf("COALESCE((SELECT SUM(value) FROM oc_votes WHERE oc_id = o.id), 0) <= %d", -3))
		}
		exclSQL, exclArgs := ExcludeClause("o.user_id", excludeUserIDs, idx)
		idx += len(exclArgs)
		args = append(args, exclArgs...)
		return " WHERE " + strings.Join(parts, " AND ") + exclSQL, args, idx
	}

	countWhere, countArgs, _ := buildWhere(1)
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ocs o`+countWhere, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ocs: %w", err)
	}

	listWhere, listArgs, nextIdx := buildWhere(2)
	limitPH := fmt.Sprintf("$%d", nextIdx)
	offsetPH := fmt.Sprintf("$%d", nextIdx+1)
	orderClause := ocOrderClause(sort)
	query := ocSelectBase + listWhere + orderClause + ` LIMIT ` + limitPH + ` OFFSET ` + offsetPH

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, listArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ocs: %w", err)
	}
	defer rows.Close()

	var ocs []model.OCRow
	for rows.Next() {
		var o model.OCRow
		if err := scanOCRow(rows, &o); err != nil {
			return nil, 0, fmt.Errorf("scan oc: %w", err)
		}
		ocs = append(ocs, o)
	}
	return ocs, total, rows.Err()
}

func ocOrderClause(sort string) string {
	voteScore := `COALESCE((SELECT SUM(value) FROM oc_votes WHERE oc_id = o.id), 0)`
	favouriteCount := `(SELECT COUNT(*) FROM oc_favourites WHERE oc_id = o.id)`
	switch sort {
	case "top":
		return ` ORDER BY ` + voteScore + ` DESC, o.created_at DESC`
	case "crack":
		return ` ORDER BY ` + voteScore + ` ASC, o.created_at DESC`
	case "favourites":
		return ` ORDER BY ` + favouriteCount + ` DESC, o.created_at DESC`
	case "comments":
		return ` ORDER BY (SELECT COUNT(*) FROM oc_comments WHERE oc_id = o.id) DESC, o.created_at DESC`
	case "name":
		return ` ORDER BY lower(o.name) ASC`
	case "old":
		return ` ORDER BY o.created_at ASC`
	default:
		return ` ORDER BY o.created_at DESC`
	}
}

func (r *ocRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.OCRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ocs WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user ocs: %w", err)
	}

	query := ocSelectBase + ` WHERE o.user_id = $2 ORDER BY o.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, viewerID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user ocs: %w", err)
	}
	defer rows.Close()

	var ocs []model.OCRow
	for rows.Next() {
		var o model.OCRow
		if err := scanOCRow(rows, &o); err != nil {
			return nil, 0, fmt.Errorf("scan oc: %w", err)
		}
		ocs = append(ocs, o)
	}
	return ocs, total, rows.Err()
}

func (r *ocRepository) ListSummariesByUser(ctx context.Context, userID uuid.UUID) ([]model.OCSummaryRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, series, custom_series_name, thumbnail_url FROM ocs WHERE user_id = $1 ORDER BY lower(name) ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list oc summaries: %w", err)
	}
	defer rows.Close()

	var summaries []model.OCSummaryRow
	for rows.Next() {
		var s model.OCSummaryRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Series, &s.CustomSeriesName, &s.ThumbnailURL); err != nil {
			return nil, fmt.Errorf("scan oc summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

func (r *ocRepository) AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO oc_images (oc_id, image_url, thumbnail_url, caption, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		ocID, imageURL, thumbnailURL, caption, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add oc gallery image: %w", err)
	}
	return id, nil
}

func (r *ocRepository) UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oc_images SET image_url = $1 WHERE id = $2`, imageURL, id)
	if err != nil {
		return fmt.Errorf("update oc gallery image url: %w", err)
	}
	return nil
}

func (r *ocRepository) UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oc_images SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update oc gallery image thumbnail: %w", err)
	}
	return nil
}

func (r *ocRepository) UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int) error {
	if caption == nil && sortOrder == nil {
		return nil
	}
	parts := make([]string, 0, 2)
	args := make([]interface{}, 0, 4)
	idx := 1
	if caption != nil {
		parts = append(parts, fmt.Sprintf("caption = $%d", idx))
		args = append(args, *caption)
		idx++
	}
	if sortOrder != nil {
		parts = append(parts, fmt.Sprintf("sort_order = $%d", idx))
		args = append(args, *sortOrder)
		idx++
	}
	args = append(args, id, ocID)
	query := fmt.Sprintf(`UPDATE oc_images SET %s WHERE id = $%d AND oc_id = $%d`, strings.Join(parts, ", "), idx, idx+1)
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update oc gallery image: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery image not found or not in oc")
	}
	return nil
}

func (r *ocRepository) DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM oc_images WHERE id = $1 AND oc_id = $2`, id, ocID)
	if err != nil {
		return fmt.Errorf("delete oc gallery image: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery image not found or not in oc")
	}
	return nil
}

func (r *ocRepository) GetGallery(ctx context.Context, ocID uuid.UUID) ([]model.OCImageRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, oc_id, image_url, thumbnail_url, caption, sort_order FROM oc_images WHERE oc_id = $1 ORDER BY sort_order ASC, id ASC`,
		ocID,
	)
	if err != nil {
		return nil, fmt.Errorf("get oc gallery: %w", err)
	}
	defer rows.Close()

	var images []model.OCImageRow
	for rows.Next() {
		var m model.OCImageRow
		if err := rows.Scan(&m.ID, &m.OCID, &m.ImageURL, &m.ThumbnailURL, &m.Caption, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan oc gallery image: %w", err)
		}
		images = append(images, m)
	}
	return images, rows.Err()
}

func (r *ocRepository) GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID) (map[uuid.UUID][]model.OCImageRow, error) {
	if len(ocIDs) == 0 {
		return nil, nil
	}
	placeholders := "$1"
	args := []interface{}{ocIDs[0]}
	for i, id := range ocIDs[1:] {
		placeholders += fmt.Sprintf(", $%d", i+2)
		args = append(args, id)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, oc_id, image_url, thumbnail_url, caption, sort_order FROM oc_images WHERE oc_id IN (`+placeholders+`) ORDER BY sort_order ASC, id ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get oc gallery: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.OCImageRow)
	for rows.Next() {
		var m model.OCImageRow
		if err := rows.Scan(&m.ID, &m.OCID, &m.ImageURL, &m.ThumbnailURL, &m.Caption, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan oc gallery image: %w", err)
		}
		result[m.OCID] = append(result[m.OCID], m)
	}
	return result, rows.Err()
}

func (r *ocRepository) Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error {
	if value == 0 {
		_, err := r.db.ExecContext(ctx,
			`DELETE FROM oc_votes WHERE user_id = $1 AND oc_id = $2`,
			userID, ocID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO oc_votes (user_id, oc_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, oc_id) DO UPDATE SET value = EXCLUDED.value`,
		userID, ocID, value,
	)
	if err != nil {
		return fmt.Errorf("vote oc: %w", err)
	}
	return nil
}

func (r *ocRepository) Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO oc_favourites (user_id, oc_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, ocID,
	)
	if err != nil {
		return fmt.Errorf("favourite oc: %w", err)
	}
	return nil
}

func (r *ocRepository) Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM oc_favourites WHERE user_id = $1 AND oc_id = $2`,
		userID, ocID,
	)
	if err != nil {
		return fmt.Errorf("unfavourite oc: %w", err)
	}
	return nil
}

func (r *ocRepository) CreateComment(ctx context.Context, id uuid.UUID, ocID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO oc_comments (id, oc_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, ocID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create oc comment: %w", err)
	}
	return nil
}

func (r *ocRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE oc_comments SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update oc comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *ocRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE oc_comments SET body = $1, updated_at = NOW() WHERE id = $2`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update oc comment: %w", err)
	}
	return nil
}

func (r *ocRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM oc_comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete oc comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *ocRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oc_comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete oc comment: %w", err)
	}
	return nil
}

func (r *ocRepository) GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCCommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs, 2)
	var total int
	countArgs := []interface{}{ocID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oc_comments WHERE oc_id = $1`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count oc comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	queryArgs := []interface{}{viewerID, ocID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.oc_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM oc_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM oc_comment_likes WHERE comment_id = c.id AND user_id = $1)
		FROM oc_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.oc_id = $2`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT `+limitPH+` OFFSET `+offsetPH,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get oc comments: %w", err)
	}
	defer rows.Close()

	var comments []model.OCCommentRow
	for rows.Next() {
		var c model.OCCommentRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.OCID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan oc comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		c.UpdatedAt = timePtrToString(updatedAt)
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *ocRepository) GetCommentOCID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var ocID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT oc_id FROM oc_comments WHERE id = $1`, commentID).Scan(&ocID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get oc comment oc id: %w", err)
	}
	return ocID, nil
}

func (r *ocRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM oc_comments WHERE id = $1`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get oc comment author: %w", err)
	}
	return userID, nil
}

func (r *ocRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO oc_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like oc comment: %w", err)
	}
	return nil
}

func (r *ocRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM oc_comment_likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike oc comment: %w", err)
	}
	return nil
}

func (r *ocRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO oc_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add oc comment media: %w", err)
	}
	return id, nil
}

func (r *ocRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oc_comment_media SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update oc comment media url: %w", err)
	}
	return nil
}

func (r *ocRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oc_comment_media SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update oc comment media thumbnail: %w", err)
	}
	return nil
}

func (r *ocRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.OCCommentMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM oc_comment_media WHERE comment_id = $1 ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get oc comment media: %w", err)
	}
	defer rows.Close()

	var media []model.OCCommentMediaRow
	for rows.Next() {
		var m model.OCCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan oc comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *ocRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.OCCommentMediaRow, error) {
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
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM oc_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get oc comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.OCCommentMediaRow)
	for rows.Next() {
		var m model.OCCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan oc comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}
