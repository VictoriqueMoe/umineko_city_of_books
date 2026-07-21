package repository

import (
	"context"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	JournalRepository interface {
		Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error)
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalResponse, error)
		List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.JournalResponse, int, error)
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateJournalRequest) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetAuthorID(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
		GetTitle(ctx context.Context, id uuid.UUID) (string, error)
		IsArchived(ctx context.Context, id uuid.UUID) (bool, error)
		CountUserJournalsToday(ctx context.Context, userID uuid.UUID) (int, error)
		UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID) error
		ArchiveStale(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error)

		Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error
		Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error
		IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) (bool, error)
		GetFollowerIDs(ctx context.Context, journalID uuid.UUID) ([]uuid.UUID, error)
		GetFollowerCount(ctx context.Context, journalID uuid.UUID) (int, error)
		ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]dto.JournalResponse, int, error)

		CreateEntry(ctx context.Context, id uuid.UUID, journalID uuid.UUID, entryNumber int, title *string, body string, wordCount int, isDraft bool) error
		UpdateEntry(ctx context.Context, id uuid.UUID, title *string, body string, wordCount int, isDraft bool) error
		DeleteEntry(ctx context.Context, id uuid.UUID) error
		GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int) (*JournalEntryRow, error)
		GetEntryByID(ctx context.Context, entryID uuid.UUID) (*JournalEntryRow, error)
		ListEntries(ctx context.Context, journalID uuid.UUID) ([]JournalEntrySummaryRow, error)
		GetNextEntryNumber(ctx context.Context, journalID uuid.UUID) (int, error)
		GetEntryJournalID(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error)
		GetEntryAuthorID(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error)

		CreateComment(ctx context.Context, id uuid.UUID, journalID uuid.UUID, entryID *uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]JournalCommentRow, int, error)
		GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]JournalCommentRow, int, error)
		GetCommentJournalID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID) (*int, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]JournalCommentMediaRow, error)

		AddEntryMedia(ctx context.Context, entryID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateEntryMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateEntryMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetEntryMediaBatch(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID][]JournalEntryMediaRow, error)
		DeleteEntryMedia(ctx context.Context, id int64, entryID uuid.UUID) (string, error)
	}

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

	JournalCommentRow struct {
		ID                uuid.UUID
		JournalID         uuid.UUID
		EntryID           *uuid.UUID
		ParentID          *uuid.UUID
		UserID            uuid.UUID
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		UserLiked         bool
	}

	JournalCommentMediaRow struct {
		ID           int
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	JournalEntryMediaRow struct {
		ID           int
		EntryID      uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}
)

func JournalCommentToDTO(c JournalCommentRow, media []JournalCommentMediaRow, authorID uuid.UUID) dto.JournalCommentResponse {
	mediaList := make([]dto.PostMediaResponse, len(media))
	for i, m := range media {
		mediaList[i] = dto.PostMediaResponse{
			ID:           m.ID,
			MediaURL:     m.MediaURL,
			MediaType:    m.MediaType,
			ThumbnailURL: m.ThumbnailURL,
			SortOrder:    m.SortOrder,
		}
	}
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
		Media:     mediaList,
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		IsAuthor:  c.UserID == authorID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func JournalEntryToDTO(e *JournalEntryRow, media []JournalEntryMediaRow) dto.JournalEntryResponse {
	mediaList := make([]dto.PostMediaResponse, len(media))
	for i, m := range media {
		mediaList[i] = dto.PostMediaResponse{
			ID:           m.ID,
			MediaURL:     m.MediaURL,
			MediaType:    m.MediaType,
			ThumbnailURL: m.ThumbnailURL,
			SortOrder:    m.SortOrder,
		}
	}
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
