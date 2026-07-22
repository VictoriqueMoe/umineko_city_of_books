package repository

import (
	"context"
	"time"
)

type (
	BannedGiphyRepository interface {
		List(ctx context.Context) ([]BannedGiphyRow, error)
		Add(ctx context.Context, kind, value, reason string, createdBy *string) error
		Remove(ctx context.Context, kind, value string) error
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

func (r *bannedGiphyRepository) List(ctx context.Context) ([]BannedGiphyRow, error) {
	return r.dao.List(ctx)
}

func (r *bannedGiphyRepository) Add(ctx context.Context, kind, value, reason string, createdBy *string) error {
	return r.dao.Add(ctx, kind, value, reason, createdBy)
}

func (r *bannedGiphyRepository) Remove(ctx context.Context, kind, value string) error {
	return r.dao.Remove(ctx, kind, value)
}
