package repository

import (
	"context"
	"database/sql"

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
