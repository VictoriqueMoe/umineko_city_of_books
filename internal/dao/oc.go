package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	ocDAO struct {
		db *sql.DB
		*ownedDAO
		*voteDAO
		*commentDAO[uuid.UUID]
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

func scanOCRow(row interface{ Scan(...any) error }, o *model.OCRow) error {
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
	o.UpdatedAt = new(updatedAt.UTC().Format(time.RFC3339))
	return nil
}

func (r *ocDAO) Create(ctx context.Context, spec repository.NewOC, tx ...*sql.Tx) (*model.OCRow, error) {
	var created model.OCRow
	err := scanOCRow(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH o AS (
		     INSERT INTO ocs (user_id, name, description, series, custom_series_name)
		     VALUES ($1, $2, $3, $4, $5)
		     RETURNING *
		 )
		 SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
		        o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
		        u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		        0, 0, 0, FALSE, 0
		 FROM o
		 JOIN users u ON o.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = o.user_id`,
		spec.UserID, spec.Name, spec.Description, spec.Series, spec.CustomSeriesName,
	), &created)
	if err != nil {
		return nil, fmt.Errorf("create oc: %w", err)
	}

	return &created, nil
}

func (r *ocDAO) Update(ctx context.Context, spec repository.OCUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var res sql.Result
		var err error
		if spec.AsAdmin {
			res, err = tx.ExecContext(ctx,
				`UPDATE ocs SET name = $1, description = $2, series = $3, custom_series_name = $4, updated_at = NOW() WHERE id = $5`,
				spec.Name, spec.Description, spec.Series, spec.CustomSeriesName, spec.ID,
			)
		} else {
			res, err = tx.ExecContext(ctx,
				`UPDATE ocs SET name = $1, description = $2, series = $3, custom_series_name = $4, updated_at = NOW() WHERE id = $5 AND user_id = $6`,
				spec.Name, spec.Description, spec.Series, spec.CustomSeriesName, spec.ID, spec.UserID,
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

func (r *ocDAO) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE ocs SET image_url = $1, thumbnail_url = $2 WHERE id = $3`,
		imageURL, thumbnailURL, id,
	)
	if err != nil {
		return fmt.Errorf("update oc image: %w", err)
	}
	return nil
}

func appendOCPaths(paths []string, values ...string) []string {
	for i := range values {
		if values[i] == "" {
			continue
		}

		paths = append(paths, values[i])
	}

	return paths
}

func (r *ocDAO) GetImagePaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var imageURL, thumbnailURL string

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT image_url, thumbnail_url FROM ocs WHERE id = $1`, ocID,
	).Scan(&imageURL, &thumbnailURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get oc image paths: %w", err)
	}

	return appendOCPaths(nil, imageURL, thumbnailURL), nil
}

func (r *ocDAO) GetGalleryPaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT image_url, thumbnail_url FROM oc_images WHERE oc_id = $1 ORDER BY sort_order, id`,
		ocID,
	)
	if err != nil {
		return nil, fmt.Errorf("get oc gallery paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var imageURL, thumbnailURL string
		if err := rows.Scan(&imageURL, &thumbnailURL); err != nil {
			return nil, fmt.Errorf("scan oc gallery path: %w", err)
		}

		paths = appendOCPaths(paths, imageURL, thumbnailURL)
	}

	return paths, rows.Err()
}

func (r *ocDAO) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.OCRow, error) {
	var o model.OCRow
	err := scanOCRow(txOrDB(r.db, tx).QueryRowContext(ctx, ocSelectBase+` WHERE o.id = $2`, viewerID, id), &o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get oc: %w", err)
	}
	return &o, nil
}

func (r *ocDAO) HasOC(ctx context.Context, userID uuid.UUID, name string, tx ...*sql.Tx) (bool, error) {
	var exists bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM ocs WHERE user_id = $1 AND lower(name) = lower($2))`,
		userID, name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check oc exists: %w", err)
	}
	return exists, nil
}

func (r *ocDAO) List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.OCRow, int, error) {
	buildWhere := func(startIdx int) (string, []any, int) {
		idx := startIdx
		next := func() string {
			s := fmt.Sprintf("$%d", idx)
			idx++
			return s
		}
		parts := []string{"1=1"}
		var args []any
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
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ocs o`+countWhere, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ocs: %w", err)
	}

	listWhere, listArgs, nextIdx := buildWhere(2)
	limitPH := fmt.Sprintf("$%d", nextIdx)
	offsetPH := fmt.Sprintf("$%d", nextIdx+1)
	orderClause := ocOrderClause(sort)
	query := ocSelectBase + listWhere + orderClause + ` LIMIT ` + limitPH + ` OFFSET ` + offsetPH

	queryArgs := []any{viewerID}
	queryArgs = append(queryArgs, listArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, queryArgs...)
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

func (r *ocDAO) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.OCRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM ocs WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user ocs: %w", err)
	}

	query := ocSelectBase + ` WHERE o.user_id = $2 ORDER BY o.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, viewerID, userID, limit, offset)
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

func (r *ocDAO) ListSummariesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.OCSummaryRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

func (r *ocDAO) AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int, tx ...*sql.Tx) (int64, error) {
	var id int64
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO oc_images (oc_id, image_url, thumbnail_url, caption, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		ocID, imageURL, thumbnailURL, caption, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add oc gallery image: %w", err)
	}
	return id, nil
}

func (r *ocDAO) UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `UPDATE oc_images SET image_url = $1 WHERE id = $2`, imageURL, id)
	if err != nil {
		return fmt.Errorf("update oc gallery image url: %w", err)
	}
	return nil
}

func (r *ocDAO) UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `UPDATE oc_images SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update oc gallery image thumbnail: %w", err)
	}
	return nil
}

func (r *ocDAO) UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int, tx ...*sql.Tx) error {
	if caption == nil && sortOrder == nil {
		return nil
	}
	parts := make([]string, 0, 2)
	args := make([]any, 0, 4)
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
	res, err := txOrDB(r.db, tx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update oc gallery image: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery image not found or not in oc")
	}
	return nil
}

func (r *ocDAO) DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM oc_images WHERE id = $1 AND oc_id = $2`, id, ocID)
	if err != nil {
		return fmt.Errorf("delete oc gallery image: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery image not found or not in oc")
	}
	return nil
}

func (r *ocDAO) GetGallery(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]model.OCImageRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

func (r *ocDAO) GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.OCImageRow, error) {
	if len(ocIDs) == 0 {
		return nil, nil
	}
	placeholders, args := utils.PlaceholderArgs(ocIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, oc_id, image_url, thumbnail_url, caption, sort_order FROM oc_images WHERE oc_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY sort_order ASC, id ASC`,
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

func (r *ocDAO) Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO oc_favourites (user_id, oc_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, ocID,
	)
	if err != nil {
		return fmt.Errorf("favourite oc: %w", err)
	}
	return nil
}

func (r *ocDAO) Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM oc_favourites WHERE user_id = $1 AND oc_id = $2`,
		userID, ocID,
	)
	if err != nil {
		return fmt.Errorf("unfavourite oc: %w", err)
	}
	return nil
}
