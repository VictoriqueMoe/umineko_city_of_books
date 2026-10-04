package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	StoredFileDAO interface {
		Get(ctx context.Context, s spec.StoredFileKey, tx ...*sql.Tx) (*model.StoredFileRow, error)
		Upsert(ctx context.Context, s spec.NewStoredFile, tx ...*sql.Tx) error
		Backfill(ctx context.Context, s spec.StoredFileBackfill, tx ...*sql.Tx) (int64, error)
		Delete(ctx context.Context, s spec.StoredFileKeys, tx ...*sql.Tx) error
		ListByKeys(ctx context.Context, s spec.StoredFileKeys, tx ...*sql.Tx) ([]model.StoredFileRow, error)
		ListByPrefix(ctx context.Context, s spec.StoredFilePrefix, tx ...*sql.Tx) ([]model.StoredFileRow, error)
		CountByBackend(ctx context.Context, s spec.StoredFileBackend, tx ...*sql.Tx) (int64, error)
	}

	storedFileDAO struct {
		db *sql.DB
	}
)

func toStoredFileRow(row sqlcgen.StoredFile) model.StoredFileRow {
	return model.StoredFileRow{
		Key:       row.Key,
		Backend:   config.StorageBackend(row.Backend),
		Location:  row.Location,
		Size:      row.Size,
		CreatedAt: row.CreatedAt,
	}
}

func (r *storedFileDAO) Get(ctx context.Context, s spec.StoredFileKey, tx ...*sql.Tx) (*model.StoredFileRow, error) {
	row, err := genQueries(r.db, tx).GetStoredFile(ctx, s.Key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get stored file %s: %w", s.Key, err)
	}

	return new(toStoredFileRow(row)), nil
}

func (r *storedFileDAO) Upsert(ctx context.Context, s spec.NewStoredFile, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertStoredFile(ctx, sqlcgen.UpsertStoredFileParams{
		Key:      s.Key,
		Backend:  sqlcgen.StorageBackend(s.Backend),
		Location: s.Location,
		Size:     s.Size,
	})
	if err != nil {
		return fmt.Errorf("upsert stored file %s: %w", s.Key, err)
	}

	return nil
}

func (r *storedFileDAO) Backfill(ctx context.Context, s spec.StoredFileBackfill, tx ...*sql.Tx) (int64, error) {
	if len(s.Keys) != len(s.Sizes) || len(s.Keys) != len(s.Locations) {
		return 0, fmt.Errorf("backfill stored files: %d keys, %d locations and %d sizes", len(s.Keys), len(s.Locations), len(s.Sizes))
	}

	if len(s.Keys) == 0 {
		return 0, nil
	}

	inserted, err := genQueries(r.db, tx).BackfillStoredFiles(ctx, sqlcgen.BackfillStoredFilesParams{
		Keys:      s.Keys,
		Backend:   sqlcgen.StorageBackend(s.Backend),
		Locations: s.Locations,
		Sizes:     s.Sizes,
	})
	if err != nil {
		return 0, fmt.Errorf("backfill stored files: %w", err)
	}

	return inserted, nil
}

func (r *storedFileDAO) Delete(ctx context.Context, s spec.StoredFileKeys, tx ...*sql.Tx) error {
	if len(s.Keys) == 0 {
		return nil
	}

	if err := genQueries(r.db, tx).DeleteStoredFiles(ctx, s.Keys); err != nil {
		return fmt.Errorf("delete stored files: %w", err)
	}

	return nil
}

func (r *storedFileDAO) ListByKeys(ctx context.Context, s spec.StoredFileKeys, tx ...*sql.Tx) ([]model.StoredFileRow, error) {
	if len(s.Keys) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListStoredFilesByKeys(ctx, s.Keys)
	if err != nil {
		return nil, fmt.Errorf("list stored files by keys: %w", err)
	}

	result := make([]model.StoredFileRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, toStoredFileRow(row))
	}

	return result, nil
}

func (r *storedFileDAO) ListByPrefix(ctx context.Context, s spec.StoredFilePrefix, tx ...*sql.Tx) ([]model.StoredFileRow, error) {
	rows, err := genQueries(r.db, tx).ListStoredFilesByPrefix(ctx, s.Prefix)
	if err != nil {
		return nil, fmt.Errorf("list stored files by prefix %q: %w", s.Prefix, err)
	}

	result := make([]model.StoredFileRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, toStoredFileRow(row))
	}

	return result, nil
}

func (r *storedFileDAO) CountByBackend(ctx context.Context, s spec.StoredFileBackend, tx ...*sql.Tx) (int64, error) {
	count, err := genQueries(r.db, tx).CountStoredFilesByBackend(ctx, sqlcgen.StorageBackend(s.Backend))
	if err != nil {
		return 0, fmt.Errorf("count stored files on %s: %w", s.Backend, err)
	}

	return count, nil
}
