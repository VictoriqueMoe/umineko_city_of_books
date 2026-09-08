package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	mediaDAO struct {
		sqlcSource
		q mediaQuerier
	}
)

func newMediaDAO(db *sql.DB, q mediaQuerier) *mediaDAO {
	return &mediaDAO{sqlcSource: newSQLCSource(db), q: q}
}

func (m *mediaDAO) AddMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error) {
	id, err := m.q.Add(ctx, m.queries(tx), s)
	if err != nil {
		return 0, fmt.Errorf("add media: %w", err)
	}

	return id, nil
}

func (m *mediaDAO) DeleteMedia(ctx context.Context, s spec.MediaDeletion, tx ...*sql.Tx) (string, error) {
	queries := m.queries(tx)

	mediaURL, err := m.q.URL(ctx, queries, s)
	if err != nil {
		return "", fmt.Errorf("media not found: %w", err)
	}

	if err := m.q.DeleteRow(ctx, queries, s); err != nil {
		return "", fmt.Errorf("delete media: %w", err)
	}

	return mediaURL, nil
}

func (m *mediaDAO) UpdateMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	if err := m.q.SetURL(ctx, m.queries(tx), s); err != nil {
		return fmt.Errorf("update media url: %w", err)
	}

	return nil
}

func (m *mediaDAO) UpdateMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	if err := m.q.SetThumbnail(ctx, m.queries(tx), s); err != nil {
		return fmt.Errorf("update media thumbnail: %w", err)
	}

	return nil
}

func (m *mediaDAO) GetMedia(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	rows, err := m.q.Media(ctx, m.queries(tx), entityID)
	if err != nil {
		return nil, fmt.Errorf("get media: %w", err)
	}

	return rows, nil
}

func (m *mediaDAO) GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	rows, err := m.q.MediaBatch(ctx, m.queries(tx), entityIDs)
	if err != nil {
		return nil, fmt.Errorf("batch get media: %w", err)
	}

	return rows, nil
}
