package spec

import (
	"github.com/google/uuid"
)

type (
	NewGiphyFavourite struct {
		UserID     uuid.UUID
		GiphyID    string
		URL        string
		Title      string
		PreviewURL string
		Width      int
		Height     int
	}

	GiphyFavouriteDeletion struct {
		UserID  uuid.UUID
		GiphyID string
	}

	GiphyFavouritePage struct {
		UserID uuid.UUID
		Limit  int
		Offset int
	}
)
