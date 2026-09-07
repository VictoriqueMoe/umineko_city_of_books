package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

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
		CreatedAt   string
		UpdatedAt   *string
	}

	JournalEntrySummaryRow struct {
		ID          uuid.UUID
		EntryNumber int
		Title       *string
		WordCount   int
		IsDraft     bool
		CreatedAt   string
	}
)

func JournalCommentToDTO(c CommentRow, media []PostMediaRow, authorID uuid.UUID) dto.JournalCommentResponse {
	return dto.JournalCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		EntryID:  c.EntryID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     MediaRowsToResponse(media),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		IsAuthor:  c.UserID == authorID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

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
		CreatedAt:   s.CreatedAt,
	}
}
