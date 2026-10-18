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
		Fallback bool
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

	StoredFileLocations struct {
		Backend   config.StorageBackend
		Keys      []string
		Locations []string
	}

	StoredFilePrefix struct {
		Prefix string
	}

	StoredFileBackend struct {
		Backend config.StorageBackend
	}

	StoredFileMove struct {
		Key          string
		FromBackend  config.StorageBackend
		FromLocation string
		Backend      config.StorageBackend
		Location     string
		Size         int64
	}
)
