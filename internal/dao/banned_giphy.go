package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	BannedGiphyDAO interface {
		List(ctx context.Context, tx ...*sql.Tx) ([]model.BannedGiphyRow, error)
		Add(ctx context.Context, s spec.NewBannedGiphy, tx ...*sql.Tx) error
		Remove(ctx context.Context, s spec.BannedGiphyDeletion, tx ...*sql.Tx) error
	}

	bannedGiphyDAO struct {
		db *sql.DB
	}
)

func (r *bannedGiphyDAO) List(ctx context.Context, tx ...*sql.Tx) ([]model.BannedGiphyRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT kind, value, created_at, created_by, reason FROM banned_giphy ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list banned giphy: %w", err)
	}
	defer rows.Close()

	var result []model.BannedGiphyRow

	for rows.Next() {
		var row model.BannedGiphyRow
		var reason sql.NullString

		if err := rows.Scan(&row.Kind, &row.Value, &row.CreatedAt, &row.CreatedBy, &reason); err != nil {
			return nil, fmt.Errorf("scan banned giphy: %w", err)
		}

		if reason.Valid {
			row.Reason = reason.String
		}

		result = append(result, row)
	}

	return result, rows.Err()
}

func (r *bannedGiphyDAO) Add(ctx context.Context, s spec.NewBannedGiphy, tx ...*sql.Tx) error {
	var reasonVal any
	if s.Reason != "" {
		reasonVal = s.Reason
	}

	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO banned_giphy (kind, value, created_by, reason) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
		s.Kind, s.Value, s.CreatedBy, reasonVal,
	)
	if err != nil {
		return fmt.Errorf("add banned giphy: %w", err)
	}

	return nil
}

func (r *bannedGiphyDAO) Remove(ctx context.Context, s spec.BannedGiphyDeletion, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM banned_giphy WHERE kind = $1 AND value = $2`,
		s.Kind, s.Value,
	)
	if err != nil {
		return fmt.Errorf("remove banned giphy: %w", err)
	}

	return nil
}
