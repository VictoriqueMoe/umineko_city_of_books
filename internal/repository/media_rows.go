package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

func addMediaRows(ctx context.Context, add func(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error), targetID uuid.UUID, rows []spec.NewMedia, tx *sql.Tx) ([]int64, error) {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		row.TargetID = targetID

		id, err := add(ctx, row, tx)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func claimUploads(ctx context.Context, pending dao.PendingUploadDAO, s spec.ClaimedUploads, tx *sql.Tx) ([]model.PendingUploadRow, error) {
	if len(s.Uploads) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(s.Uploads))
	seen := make(map[uuid.UUID]struct{}, len(s.Uploads))
	for _, ref := range s.Uploads {
		if _, dup := seen[ref.ID]; dup {
			return nil, fmt.Errorf("%w: upload %s is attached twice", daoerr.ErrUploadUnavailable, ref.ID)
		}

		seen[ref.ID] = struct{}{}
		ids = append(ids, ref.ID)
	}

	rows, err := pending.Claim(ctx, spec.UploadClaim{UserID: s.UserID, IDs: ids, MediaTypes: s.MediaTypes, CreatedAfter: time.Now().Add(-s.MaxAge)}, tx)
	if err != nil {
		return nil, err
	}

	if len(rows) != len(ids) {
		return nil, fmt.Errorf("%w: %d of %d uploads are ready", daoerr.ErrUploadUnavailable, len(rows), len(ids))
	}

	byID := make(map[uuid.UUID]model.PendingUploadRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	ordered := make([]model.PendingUploadRow, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, byID[id])
	}

	return ordered, nil
}

func claimMediaRows(ctx context.Context, pending dao.PendingUploadDAO, s spec.ClaimedUploads, tx *sql.Tx) ([]spec.NewMedia, error) {
	claimed, err := claimUploads(ctx, pending, s, tx)
	if err != nil {
		return nil, err
	}

	var rows []spec.NewMedia
	for i, upload := range claimed {
		rows = append(rows, spec.NewMedia{
			MediaURL:     upload.MediaURL,
			MediaType:    upload.MediaType,
			ThumbnailURL: upload.ThumbnailURL,
			Filename:     upload.Filename,
			SortOrder:    i,
			IsSpoiler:    s.Uploads[i].IsSpoiler,
		})
	}

	return rows, nil
}

func attachMedia(ctx context.Context, pending dao.PendingUploadDAO, add func(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error), s spec.MediaAttachment, tx *sql.Tx) (*model.PostMediaRow, error) {
	rows, err := claimMediaRows(ctx, pending, s.Uploads, tx)
	if err != nil {
		return nil, err
	}

	if len(rows) != 1 {
		return nil, fmt.Errorf("%w: expected one upload, got %d", daoerr.ErrUploadUnavailable, len(rows))
	}

	rows[0].SortOrder = s.SortOrder

	ids, err := addMediaRows(ctx, add, s.TargetID, rows, tx)
	if err != nil {
		return nil, err
	}

	row := rows[0]

	return &model.PostMediaRow{
		ID:           int(ids[0]),
		PostID:       s.TargetID,
		MediaURL:     row.MediaURL,
		MediaType:    row.MediaType,
		ThumbnailURL: row.ThumbnailURL,
		Filename:     row.Filename,
		SortOrder:    row.SortOrder,
		IsSpoiler:    row.IsSpoiler,
	}, nil
}
