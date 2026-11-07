package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	PendingUploadDAO interface {
		Insert(ctx context.Context, s spec.NewPendingUpload, tx ...*sql.Tx) error
		MarkReady(ctx context.Context, s spec.ProcessedUpload, tx ...*sql.Tx) (bool, error)
		MarkFailed(ctx context.Context, s spec.UserUpload, tx ...*sql.Tx) (bool, error)
		FailProcessing(ctx context.Context, tx ...*sql.Tx) (int64, error)
		List(ctx context.Context, s spec.UserUploads, tx ...*sql.Tx) ([]model.PendingUploadRow, error)
		Claim(ctx context.Context, s spec.UploadClaim, tx ...*sql.Tx) ([]model.PendingUploadRow, error)
		Delete(ctx context.Context, s spec.UserUpload, tx ...*sql.Tx) ([]model.PendingUploadRow, error)
		DeleteExpired(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]model.PendingUploadRow, error)
	}

	pendingUploadDAO struct {
		db *sql.DB
	}
)

func toPendingUploadRow(row sqlcgen.PendingUpload) model.PendingUploadRow {
	return model.PendingUploadRow{
		ID:           row.ID,
		UserID:       row.UserID,
		MediaType:    row.MediaType,
		Filename:     row.Filename,
		FileSize:     row.FileSize,
		Status:       model.PendingUploadStatus(row.Status),
		MediaURL:     row.MediaUrl,
		ThumbnailURL: row.ThumbnailUrl,
		Width:        int(row.Width),
		Height:       int(row.Height),
		CreatedAt:    row.CreatedAt,
	}
}

func toPendingUploadRows(rows []sqlcgen.PendingUpload) []model.PendingUploadRow {
	out := make([]model.PendingUploadRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPendingUploadRow(row))
	}

	return out
}

func (r *pendingUploadDAO) Insert(ctx context.Context, s spec.NewPendingUpload, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).InsertPendingUpload(ctx, sqlcgen.InsertPendingUploadParams{
		ID:        s.ID,
		UserID:    s.UserID,
		MediaType: s.MediaType,
		Filename:  s.Filename,
		FileSize:  s.FileSize,
	})
	if err != nil {
		return fmt.Errorf("insert pending upload %s: %w", s.ID, err)
	}

	return nil
}

func (r *pendingUploadDAO) MarkReady(ctx context.Context, s spec.ProcessedUpload, tx ...*sql.Tx) (bool, error) {
	n, err := genQueries(r.db, tx).MarkPendingUploadReady(ctx, sqlcgen.MarkPendingUploadReadyParams{
		ID:           s.ID,
		MediaUrl:     s.MediaURL,
		ThumbnailUrl: s.ThumbnailURL,
		Width:        int32(s.Width),
		Height:       int32(s.Height),
	})
	if err != nil {
		return false, fmt.Errorf("mark pending upload %s ready: %w", s.ID, err)
	}

	return n > 0, nil
}

func (r *pendingUploadDAO) MarkFailed(ctx context.Context, s spec.UserUpload, tx ...*sql.Tx) (bool, error) {
	n, err := genQueries(r.db, tx).MarkPendingUploadFailed(ctx, s.ID)
	if err != nil {
		return false, fmt.Errorf("mark pending upload %s failed: %w", s.ID, err)
	}

	return n > 0, nil
}

func (r *pendingUploadDAO) FailProcessing(ctx context.Context, tx ...*sql.Tx) (int64, error) {
	n, err := genQueries(r.db, tx).FailProcessingPendingUploads(ctx)
	if err != nil {
		return 0, fmt.Errorf("fail processing pending uploads: %w", err)
	}

	return n, nil
}

func (r *pendingUploadDAO) List(ctx context.Context, s spec.UserUploads, tx ...*sql.Tx) ([]model.PendingUploadRow, error) {
	rows, err := genQueries(r.db, tx).ListPendingUploads(ctx, sqlcgen.ListPendingUploadsParams{
		UserID:  s.UserID,
		Column2: joinUUIDs(s.IDs),
	})
	if err != nil {
		return nil, fmt.Errorf("list pending uploads: %w", err)
	}

	return toPendingUploadRows(rows), nil
}

func (r *pendingUploadDAO) Claim(ctx context.Context, s spec.UploadClaim, tx ...*sql.Tx) ([]model.PendingUploadRow, error) {
	rows, err := genQueries(r.db, tx).ClaimPendingUploads(ctx, sqlcgen.ClaimPendingUploadsParams{
		UserID:    s.UserID,
		Column2:   joinUUIDs(s.IDs),
		CreatedAt: s.CreatedAfter,
		Column4:   strings.Join(s.MediaTypes, ","),
	})
	if err != nil {
		return nil, fmt.Errorf("claim pending uploads: %w", err)
	}

	return toPendingUploadRows(rows), nil
}

func (r *pendingUploadDAO) Delete(ctx context.Context, s spec.UserUpload, tx ...*sql.Tx) ([]model.PendingUploadRow, error) {
	rows, err := genQueries(r.db, tx).DeletePendingUpload(ctx, sqlcgen.DeletePendingUploadParams{ID: s.ID, UserID: s.UserID})
	if err != nil {
		return nil, fmt.Errorf("delete pending upload %s: %w", s.ID, err)
	}

	return toPendingUploadRows(rows), nil
}

func (r *pendingUploadDAO) DeleteExpired(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]model.PendingUploadRow, error) {
	rows, err := genQueries(r.db, tx).DeleteExpiredPendingUploads(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("delete expired pending uploads: %w", err)
	}

	return toPendingUploadRows(rows), nil
}
