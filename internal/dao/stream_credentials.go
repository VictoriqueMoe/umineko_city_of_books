package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	StreamCredentialsDAO interface {
		Get(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*model.StreamCredentialsRow, error)
		Upsert(ctx context.Context, s spec.NewStreamCredentials, tx ...*sql.Tx) error
		Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	streamCredentialsDAO struct {
		db *sql.DB
	}
)

func toStreamCredentialsRow(row sqlcgen.GetStreamCredentialsRow) model.StreamCredentialsRow {
	return model.StreamCredentialsRow{
		UserID:    row.UserID,
		IngressID: row.IngressID,
		WhipURL:   row.WhipUrl,
		StreamKey: row.StreamKey,
		Room:      row.Room,
	}
}

func (r *streamCredentialsDAO) Get(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*model.StreamCredentialsRow, error) {
	row, err := genQueries(r.db, tx).GetStreamCredentials(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get stream credentials: %w", err)
	}

	return new(toStreamCredentialsRow(row)), nil
}

func (r *streamCredentialsDAO) Upsert(ctx context.Context, s spec.NewStreamCredentials, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertStreamCredentials(ctx, sqlcgen.UpsertStreamCredentialsParams{
		UserID:    s.UserID,
		IngressID: s.IngressID,
		WhipUrl:   s.WhipURL,
		StreamKey: s.StreamKey,
		Room:      s.Room,
	})
	if err != nil {
		return fmt.Errorf("upsert stream credentials: %w", err)
	}

	return nil
}

func (r *streamCredentialsDAO) Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteStreamCredentials(ctx, userID); err != nil {
		return fmt.Errorf("delete stream credentials: %w", err)
	}

	return nil
}
