package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type mediaDAO struct {
	db    *sql.DB
	table string
	fk    string
}

func newMediaDAO(db *sql.DB, table string, fk string) *mediaDAO {
	return &mediaDAO{db: db, table: table, fk: fk}
}

func (m *mediaDAO) AddMedia(ctx context.Context, entityID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := m.db.QueryRowContext(ctx,
		`INSERT INTO `+m.table+` (`+m.fk+`, media_url, media_type, thumbnail_url, sort_order)
		VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(sort_order) + 1 FROM `+m.table+` WHERE `+m.fk+` = $1), $5))
		RETURNING id`,
		entityID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add media in %s: %w", m.table, err)
	}

	return id, nil
}

func (m *mediaDAO) DeleteMedia(ctx context.Context, id int64, entityID uuid.UUID) (string, error) {
	var mediaURL string
	err := m.db.QueryRowContext(ctx,
		`SELECT media_url FROM `+m.table+` WHERE id = $1 AND `+m.fk+` = $2`, id, entityID,
	).Scan(&mediaURL)
	if err != nil {
		return "", fmt.Errorf("media not found in %s: %w", m.table, err)
	}

	if _, err := m.db.ExecContext(ctx,
		`DELETE FROM `+m.table+` WHERE id = $1 AND `+m.fk+` = $2`, id, entityID,
	); err != nil {
		return "", fmt.Errorf("delete media in %s: %w", m.table, err)
	}

	return mediaURL, nil
}

func (m *mediaDAO) UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := m.db.ExecContext(ctx, `UPDATE `+m.table+` SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update media url in %s: %w", m.table, err)
	}

	return nil
}

func (m *mediaDAO) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := m.db.ExecContext(ctx, `UPDATE `+m.table+` SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update media thumbnail in %s: %w", m.table, err)
	}

	return nil
}

func (m *mediaDAO) GetMedia(ctx context.Context, entityID uuid.UUID) ([]model.PostMediaRow, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, `+m.fk+`, media_url, media_type, thumbnail_url, sort_order FROM `+m.table+` WHERE `+m.fk+` = $1 ORDER BY sort_order, id`,
		entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("get media in %s: %w", m.table, err)
	}
	defer rows.Close()

	var media []model.PostMediaRow
	for rows.Next() {
		var row model.PostMediaRow
		if err := rows.Scan(&row.ID, &row.PostID, &row.MediaURL, &row.MediaType, &row.ThumbnailURL, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan media in %s: %w", m.table, err)
		}
		media = append(media, row)
	}

	return media, rows.Err()
}

func (m *mediaDAO) GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	placeholders := "$1"
	args := []interface{}{entityIDs[0]}
	for i := 1; i < len(entityIDs); i++ {
		placeholders += fmt.Sprintf(", $%d", i+1)
		args = append(args, entityIDs[i])
	}

	rows, err := m.db.QueryContext(ctx,
		`SELECT id, `+m.fk+`, media_url, media_type, thumbnail_url, sort_order FROM `+m.table+` WHERE `+m.fk+` IN (`+placeholders+`) ORDER BY sort_order, id`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get media in %s: %w", m.table, err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.PostMediaRow)
	for rows.Next() {
		var row model.PostMediaRow
		if err := rows.Scan(&row.ID, &row.PostID, &row.MediaURL, &row.MediaType, &row.ThumbnailURL, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan media in %s: %w", m.table, err)
		}
		result[row.PostID] = append(result[row.PostID], row)
	}

	return result, rows.Err()
}
