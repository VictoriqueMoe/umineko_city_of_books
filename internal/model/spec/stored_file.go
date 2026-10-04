package spec

import (
	"umineko_city_of_books/internal/config"
)

type (
	NewStoredFile struct {
		Key      string
		Backend  config.StorageBackend
		Location string
		Size     int64
	}

	StoredFileBackfill struct {
		Backend   config.StorageBackend
		Keys      []string
		Locations []string
		Sizes     []int64
	}

	StoredFileKey struct {
		Key string
	}

	StoredFileKeys struct {
		Keys []string
	}

	StoredFilePrefix struct {
		Prefix string
	}

	StoredFileBackend struct {
		Backend config.StorageBackend
	}
)
