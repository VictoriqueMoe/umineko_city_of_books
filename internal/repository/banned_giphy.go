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
