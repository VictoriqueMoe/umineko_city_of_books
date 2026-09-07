package model

import (
	"database/sql"
	"time"
)

type (
	SitemapEntry struct {
		ID      string
		LastMod time.Time
	}

	SitemapJournalRow struct {
		JournalID        string
		JournalUpdatedAt time.Time
		EntryNumber      sql.NullInt64
		EntryUpdatedAt   sql.NullTime
	}
)
