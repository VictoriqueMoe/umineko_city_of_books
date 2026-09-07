package spec

import (
	"github.com/google/uuid"
)

type (
	NewMedia struct {
		TargetID     uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		Filename     string
		SortOrder    int
		IsSpoiler    bool
	}

	MediaURLUpdate struct {
		ID  int64
		URL string
	}

	MediaDeletion struct {
		ID       int64
		TargetID uuid.UUID
	}
)
