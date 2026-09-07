package spec

import (
	"github.com/google/uuid"
)

type (
	NewJournal struct {
		UserID uuid.UUID
		Title  string
		Work   string
	}

	JournalLookup struct {
		ID       uuid.UUID
		ViewerID uuid.UUID
	}

	JournalQuery struct {
		Sort            string
		Work            string
		AuthorID        uuid.UUID
		Search          string
		IncludeArchived bool
		Limit           int
		Offset          int
		ViewerID        uuid.UUID
		ExcludeUserIDs  []uuid.UUID
	}

	JournalUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Title   string
		Work    string
		AsAdmin bool
	}

	JournalDeletion struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	JournalPause struct {
		ID     uuid.UUID
		UserID uuid.UUID
		Paused bool
	}

	JournalFollow struct {
		UserID    uuid.UUID
		JournalID uuid.UUID
	}

	JournalFollowedQuery struct {
		FollowerID uuid.UUID
		ViewerID   uuid.UUID
		Limit      int
		Offset     int
	}

	NewJournalEntry struct {
		JournalID   uuid.UUID
		EntryNumber int
		Title       *string
		Body        string
		WordCount   int
		IsDraft     bool
	}

	JournalEntryUpdate struct {
		ID                   uuid.UUID
		JournalID            uuid.UUID
		Title                *string
		Body                 string
		WordCount            int
		IsDraft              bool
		RecordAuthorActivity bool
	}

	JournalEntryLookup struct {
		JournalID   uuid.UUID
		EntryNumber int
	}

	NewJournalComment struct {
		JournalID            uuid.UUID
		EntryID              *uuid.UUID
		ParentID             *uuid.UUID
		UserID               uuid.UUID
		Body                 string
		RecordAuthorActivity bool
	}
)
