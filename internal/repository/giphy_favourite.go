package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	GiphyFavouriteRepository interface {
		Add(ctx context.Context, userID uuid.UUID, fav GiphyFavourite) error
		Remove(ctx context.Context, userID uuid.UUID, giphyID string) error
		List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]GiphyFavourite, int, error)
		ListIDs(ctx context.Context, userID uuid.UUID) ([]string, error)
	}

	GiphyFavourite struct {
		GiphyID    string
		URL        string
		Title      string
		PreviewURL string
		Width      int
		Height     int
		CreatedAt  time.Time
	}
)
