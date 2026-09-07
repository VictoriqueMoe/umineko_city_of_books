package spec

import (
	"github.com/google/uuid"
)

type (
	NewOC struct {
		UserID           uuid.UUID
		Name             string
		Description      string
		Series           string
		CustomSeriesName string
	}

	OCUpdate struct {
		ID               uuid.UUID
		UserID           uuid.UUID
		Name             string
		Description      string
		Series           string
		CustomSeriesName string
		AsAdmin          bool
	}

	OCDeletion struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	OCByID struct {
		ID       uuid.UUID
		ViewerID uuid.UUID
	}

	OCNameLookup struct {
		UserID uuid.UUID
		Name   string
	}

	OCImageUpdate struct {
		ID           uuid.UUID
		ImageURL     string
		ThumbnailURL string
	}

	OCListFilter struct {
		ViewerID         uuid.UUID
		Sort             string
		CrackOCsOnly     bool
		Series           string
		CustomSeriesName string
		OwnerID          uuid.UUID
		Limit            int
		Offset           int
		ExcludeUserIDs   []uuid.UUID
	}

	OCUserListFilter struct {
		UserID   uuid.UUID
		ViewerID uuid.UUID
		Limit    int
		Offset   int
	}

	NewOCGalleryImage struct {
		OCID         uuid.UUID
		ImageURL     string
		ThumbnailURL string
		Caption      string
		SortOrder    int
	}

	OCGalleryImageUpdate struct {
		ID        int64
		OCID      uuid.UUID
		Caption   *string
		SortOrder *int
	}
)
