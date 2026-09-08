package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao/sqlcgen"
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

	bannedGiphyRow = sqlcgen.BannedGiphy
)

func toBannedGiphyRow(row bannedGiphyRow) model.BannedGiphyRow {
	out := model.BannedGiphyRow{
		Kind:      row.Kind,
		Value:     row.Value,
		CreatedAt: row.CreatedAt,
	}

	if row.CreatedBy.Valid {
		out.CreatedBy = new(row.CreatedBy.String)
	}

	if row.Reason.Valid {
		out.Reason = row.Reason.String
	}

	return out
}

func (r *bannedGiphyDAO) List(ctx context.Context, tx ...*sql.Tx) ([]model.BannedGiphyRow, error) {
	rows, err := genQueries(r.db, tx).ListBannedGiphy(ctx)
	if err != nil {
		return nil, fmt.Errorf("list banned giphy: %w", err)
	}

	var result []model.BannedGiphyRow
	for _, row := range rows {
		result = append(result, toBannedGiphyRow(row))
	}

	return result, nil
}

func (r *bannedGiphyDAO) Add(ctx context.Context, s spec.NewBannedGiphy, tx ...*sql.Tx) error {
	var createdBy sql.NullString
	if s.CreatedBy != nil {
		createdBy = sql.NullString{String: *s.CreatedBy, Valid: true}
	}

	var reason sql.NullString
	if s.Reason != "" {
		reason = sql.NullString{String: s.Reason, Valid: true}
	}

	err := genQueries(r.db, tx).AddBannedGiphy(ctx, sqlcgen.AddBannedGiphyParams{
		Kind:      s.Kind,
		Value:     s.Value,
		CreatedBy: createdBy,
		Reason:    reason,
	})
	if err != nil {
		return fmt.Errorf("add banned giphy: %w", err)
	}

	return nil
}

func (r *bannedGiphyDAO) Remove(ctx context.Context, s spec.BannedGiphyDeletion, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).RemoveBannedGiphy(ctx, sqlcgen.RemoveBannedGiphyParams{
		Kind:  s.Kind,
		Value: s.Value,
	})
	if err != nil {
		return fmt.Errorf("remove banned giphy: %w", err)
	}

	return nil
}
