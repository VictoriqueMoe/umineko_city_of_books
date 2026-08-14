package repository

import (
	"context"
	"database/sql"
	"time"
)

type (
	BannedGiphyRepository interface {
		List(ctx context.Context, tx ...*sql.Tx) ([]BannedGiphyRow, error)
		Add(ctx context.Context, kind, value, reason string, createdBy *string, tx ...*sql.Tx) error
		Remove(ctx context.Context, kind, value string, tx ...*sql.Tx) error
	}

	BannedGiphyRow struct {
		Kind      string
		Value     string
		CreatedAt time.Time
		CreatedBy *string
		Reason    string
	}
)

type bannedGiphyRepository struct {
	dao BannedGiphyRepository
}

func NewBannedGiphyRepo(dao BannedGiphyRepository) BannedGiphyRepository {
	return &bannedGiphyRepository{dao: dao}
}

func (r *bannedGiphyRepository) List(ctx context.Context, tx ...*sql.Tx) ([]BannedGiphyRow, error) {
	return r.dao.List(ctx, tx...)
}

func (r *bannedGiphyRepository) Add(ctx context.Context, kind, value, reason string, createdBy *string, tx ...*sql.Tx) error {
	return r.dao.Add(ctx, kind, value, reason, createdBy, tx...)
}

func (r *bannedGiphyRepository) Remove(ctx context.Context, kind, value string, tx ...*sql.Tx) error {
	return r.dao.Remove(ctx, kind, value, tx...)
}
