package model

import (
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	JournalEntryRow struct {
		ID          uuid.UUID
		JournalID   uuid.UUID
		EntryNumber int
		Title       *string
		Body        string
		WordCount   int
		IsDraft     bool
		HasPrev     bool
		HasNext     bool
		PublishedAt *string
		CreatedAt   string
		UpdatedAt   *string
	}

	JournalEntrySummaryRow struct {
		ID          uuid.UUID
		EntryNumber int
		Title       *string
		WordCount   int
		IsDraft     bool
		PublishedAt *string
		CreatedAt   string
	}
)

func JournalEntryToDTO(e *JournalEntryRow, media []PostMediaRow) dto.JournalEntryResponse {
	mediaList := MediaRowsToResponse(media)

	return dto.JournalEntryResponse{
		ID:          e.ID,
		JournalID:   e.JournalID,
		EntryNumber: e.EntryNumber,
		Title:       e.Title,
		Body:        e.Body,
		WordCount:   e.WordCount,
		IsDraft:     e.IsDraft,
		HasPrev:     e.HasPrev,
		HasNext:     e.HasNext,
		PublishedAt: e.PublishedAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Media:       mediaList,
	}
}

func JournalEntrySummaryToDTO(s JournalEntrySummaryRow) dto.JournalEntrySummary {
	return dto.JournalEntrySummary{
		ID:          s.ID,
		EntryNumber: s.EntryNumber,
		Title:       s.Title,
		WordCount:   s.WordCount,
		IsDraft:     s.IsDraft,
		PublishedAt: s.PublishedAt,
		CreatedAt:   s.CreatedAt,
	}
}
