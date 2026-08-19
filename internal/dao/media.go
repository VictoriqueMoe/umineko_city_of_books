package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dao/utils"
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

func (m *mediaDAO) AddMedia(ctx context.Context, entityID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, filename string, sortOrder int, tx ...*sql.Tx) (int64, error) {
	var id int64
	err := txOrDB(m.db, tx).QueryRowContext(ctx,
		`INSERT INTO `+m.table+` (`+m.fk+`, media_url, media_type, thumbnail_url, filename, sort_order)
		VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(sort_order) + 1 FROM `+m.table+` WHERE `+m.fk+` = $1), $6))
		RETURNING id`,
		entityID, mediaURL, mediaType, thumbnailURL, filename, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add media in %s: %w", m.table, err)
	}

	return id, nil
}

func (m *mediaDAO) DeleteMedia(ctx context.Context, id int64, entityID uuid.UUID, tx ...*sql.Tx) (string, error) {
	var mediaURL string
	err := txOrDB(m.db, tx).QueryRowContext(ctx,
		`SELECT media_url FROM `+m.table+` WHERE id = $1 AND `+m.fk+` = $2`, id, entityID,
	).Scan(&mediaURL)
	if err != nil {
		return "", fmt.Errorf("media not found in %s: %w", m.table, err)
	}

	if _, err := txOrDB(m.db, tx).ExecContext(ctx,
		`DELETE FROM `+m.table+` WHERE id = $1 AND `+m.fk+` = $2`, id, entityID,
	); err != nil {
		return "", fmt.Errorf("delete media in %s: %w", m.table, err)
	}

	return mediaURL, nil
}

func (m *mediaDAO) UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	_, err := txOrDB(m.db, tx).ExecContext(ctx, `UPDATE `+m.table+` SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update media url in %s: %w", m.table, err)
	}

	return nil
}

func (m *mediaDAO) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	_, err := txOrDB(m.db, tx).ExecContext(ctx, `UPDATE `+m.table+` SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update media thumbnail in %s: %w", m.table, err)
	}

	return nil
}

func (m *mediaDAO) GetMedia(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	rows, err := txOrDB(m.db, tx).QueryContext(ctx,
		`SELECT id, `+m.fk+`, media_url, media_type, thumbnail_url, COALESCE(filename, ''), sort_order FROM `+m.table+` WHERE `+m.fk+` = $1 ORDER BY sort_order, id`,
		entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("get media in %s: %w", m.table, err)
	}
	defer rows.Close()

	var media []model.PostMediaRow
	for rows.Next() {
		var row model.PostMediaRow
		if err := rows.Scan(&row.ID, &row.PostID, &row.MediaURL, &row.MediaType, &row.ThumbnailURL, &row.Filename, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan media in %s: %w", m.table, err)
		}
		media = append(media, row)
	}

	return media, rows.Err()
}

func (m *mediaDAO) GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	placeholders, args := utils.PlaceholderArgs(entityIDs, 1)

	rows, err := txOrDB(m.db, tx).QueryContext(ctx,
		`SELECT id, `+m.fk+`, media_url, media_type, thumbnail_url, COALESCE(filename, ''), sort_order FROM `+m.table+` WHERE `+m.fk+` IN (`+strings.Join(placeholders, ", ")+`) ORDER BY sort_order, id`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get media in %s: %w", m.table, err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.PostMediaRow)
	for rows.Next() {
		var row model.PostMediaRow
		if err := rows.Scan(&row.ID, &row.PostID, &row.MediaURL, &row.MediaType, &row.ThumbnailURL, &row.Filename, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan media in %s: %w", m.table, err)
		}
		result[row.PostID] = append(result[row.PostID], row)
	}

	return result, rows.Err()
}
