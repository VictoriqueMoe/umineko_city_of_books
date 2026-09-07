package spec

import (
	"umineko_city_of_books/internal/audit"

	"github.com/google/uuid"
)

type (
	NewArt struct {
		UserID       uuid.UUID
		Corner       string
		ArtType      string
		Title        string
		Description  string
		ImageURL     string
		ThumbnailURL string
		IsSpoiler    bool
	}

	NewArtWithTags struct {
		NewArt

		Tags []string
	}

	ArtUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Title       string
		Description string
		IsSpoiler   bool
		AsAdmin     bool
	}

	ArtUpdateWithTags struct {
		ArtUpdate

		Tags []string
	}

	ArtDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
		Audit   audit.NewEntry
	}

	ArtCommentDeletion struct {
		CommentDeletion

		Audit audit.NewEntry
	}

	ArtLookup struct {
		ID       uuid.UUID
		ViewerID uuid.UUID
	}

	ArtFilter struct {
		ViewerID       uuid.UUID
		Corner         string
		ArtType        string
		Search         string
		Tag            string
		Sort           string
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
	}

	ArtUserFilter struct {
		UserID   uuid.UUID
		ViewerID uuid.UUID
		Limit    int
		Offset   int
	}

	ArtTagInsert struct {
		ArtID uuid.UUID
		Tags  []string
	}

	PopularTagFilter struct {
		Corner string
		Limit  int
	}

	ArtGalleryAssignment struct {
		ArtID     uuid.UUID
		UserID    uuid.UUID
		GalleryID *uuid.UUID
	}

	GalleryRef struct {
		GalleryID uuid.UUID
		UserID    uuid.UUID
	}

	NewGallery struct {
		UserID      uuid.UUID
		Name        string
		Description string
	}

	GalleryUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Name        string
		Description string
	}

	GalleryCoverUpdate struct {
		GalleryID  uuid.UUID
		UserID     uuid.UUID
		CoverArtID *uuid.UUID
	}

	GalleryPreviewFilter struct {
		GalleryID uuid.UUID
		Limit     int
	}

	GalleryArtFilter struct {
		GalleryID uuid.UUID
		ViewerID  uuid.UUID
		Limit     int
		Offset    int
	}
)
