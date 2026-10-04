package model

import (
	"time"

	"umineko_city_of_books/internal/config"
)

type (
	StoredFileRow struct {
		Key       string
		Backend   config.StorageBackend
		Location  string
		Size      int64
		CreatedAt time.Time
	}
)
