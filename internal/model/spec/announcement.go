package spec

import (
	"github.com/google/uuid"
)

type (
	NewAnnouncement struct {
		AuthorID uuid.UUID
		Title    string
		Body     string
	}

	AnnouncementUpdate struct {
		ID    uuid.UUID
		Title string
		Body  string
	}

	AnnouncementListQuery struct {
		Limit  int
		Offset int
	}

	AnnouncementPinUpdate struct {
		ID     uuid.UUID
		Pinned bool
	}
)
