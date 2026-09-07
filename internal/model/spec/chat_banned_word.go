package spec

import (
	"umineko_city_of_books/internal/audit"

	"github.com/google/uuid"
)

type (
	ChatBannedWordSpec struct {
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
	}

	ChatBannedWordUpdate struct {
		ID            uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
	}

	ChatBannedWordCreation struct {
		ChatBannedWordSpec

		Audit audit.NewEntry
	}

	ChatBannedWordDeletion struct {
		ID    uuid.UUID
		Audit audit.NewEntry
	}
)
