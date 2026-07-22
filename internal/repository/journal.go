package repository

import (
	"context"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/repository/model"
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
		GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID) (*int, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		AddMedia(ctx context.Context, entryID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMediaBatch(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, id int64, entryID uuid.UUID) (string, error)
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
)

func JournalCommentToDTO(c CommentRow, media []model.PostMediaRow, authorID uuid.UUID) dto.JournalCommentResponse {
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
		Media:     model.MediaRowsToResponse(media),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		IsAuthor:  c.UserID == authorID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func JournalEntryToDTO(e *JournalEntryRow, media []model.PostMediaRow) dto.JournalEntryResponse {
	mediaList := model.MediaRowsToResponse(media)

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

type journalRepository struct {
	dao JournalRepository
}

func NewJournalRepo(dao JournalRepository) JournalRepository {
	return &journalRepository{dao: dao}
}

func (r *journalRepository) Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error) {
	return r.dao.Create(ctx, userID, req)
}

func (r *journalRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalResponse, error) {
	return r.dao.GetByID(ctx, id, viewerID)
}

func (r *journalRepository) List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.JournalResponse, int, error) {
	return r.dao.List(ctx, p, viewerID, excludeUserIDs)
}

func (r *journalRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error {
	return r.dao.Update(ctx, id, userID, req)
}

func (r *journalRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateJournalRequest) error {
	return r.dao.UpdateAsAdmin(ctx, id, req)
}

func (r *journalRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *journalRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *journalRepository) GetAuthorID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, id)
}

func (r *journalRepository) GetTitle(ctx context.Context, id uuid.UUID) (string, error) {
	return r.dao.GetTitle(ctx, id)
}

func (r *journalRepository) IsArchived(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.dao.IsArchived(ctx, id)
}

func (r *journalRepository) CountUserJournalsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUserJournalsToday(ctx, userID)
}

func (r *journalRepository) UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID) error {
	return r.dao.UpdateLastAuthorActivity(ctx, id)
}

func (r *journalRepository) ArchiveStale(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	return r.dao.ArchiveStale(ctx, cutoff)
}

func (r *journalRepository) Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error {
	return r.dao.Follow(ctx, userID, journalID)
}

func (r *journalRepository) Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error {
	return r.dao.Unfollow(ctx, userID, journalID)
}

func (r *journalRepository) IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) (bool, error) {
	return r.dao.IsFollower(ctx, userID, journalID)
}

func (r *journalRepository) GetFollowerIDs(ctx context.Context, journalID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.GetFollowerIDs(ctx, journalID)
}

func (r *journalRepository) GetFollowerCount(ctx context.Context, journalID uuid.UUID) (int, error) {
	return r.dao.GetFollowerCount(ctx, journalID)
}

func (r *journalRepository) ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]dto.JournalResponse, int, error) {
	return r.dao.ListFollowedByUser(ctx, followerID, viewerID, limit, offset)
}

func (r *journalRepository) CreateEntry(ctx context.Context, id uuid.UUID, journalID uuid.UUID, entryNumber int, title *string, body string, wordCount int, isDraft bool) error {
	return r.dao.CreateEntry(ctx, id, journalID, entryNumber, title, body, wordCount, isDraft)
}

func (r *journalRepository) UpdateEntry(ctx context.Context, id uuid.UUID, title *string, body string, wordCount int, isDraft bool) error {
	return r.dao.UpdateEntry(ctx, id, title, body, wordCount, isDraft)
}

func (r *journalRepository) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteEntry(ctx, id)
}

func (r *journalRepository) GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int) (*JournalEntryRow, error) {
	return r.dao.GetEntry(ctx, journalID, entryNumber)
}

func (r *journalRepository) GetEntryByID(ctx context.Context, entryID uuid.UUID) (*JournalEntryRow, error) {
	return r.dao.GetEntryByID(ctx, entryID)
}

func (r *journalRepository) ListEntries(ctx context.Context, journalID uuid.UUID) ([]JournalEntrySummaryRow, error) {
	return r.dao.ListEntries(ctx, journalID)
}

func (r *journalRepository) GetNextEntryNumber(ctx context.Context, journalID uuid.UUID) (int, error) {
	return r.dao.GetNextEntryNumber(ctx, journalID)
}

func (r *journalRepository) GetEntryJournalID(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetEntryJournalID(ctx, entryID)
}

func (r *journalRepository) GetEntryAuthorID(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetEntryAuthorID(ctx, entryID)
}

func (r *journalRepository) CreateComment(ctx context.Context, id uuid.UUID, journalID uuid.UUID, entryID *uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, journalID, entryID, parentID, userID, body)
}

func (r *journalRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *journalRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *journalRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *journalRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *journalRepository) GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, journalID, viewerID, limit, offset, excludeUserIDs)
}

func (r *journalRepository) GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetEntryComments(ctx, entryID, viewerID, limit, offset, excludeUserIDs)
}

func (r *journalRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *journalRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *journalRepository) GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID) (*int, error) {
	return r.dao.GetCommentEntryNumber(ctx, commentID)
}

func (r *journalRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *journalRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *journalRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *journalRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *journalRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *journalRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}

func (r *journalRepository) AddMedia(ctx context.Context, entryID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddMedia(ctx, entryID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *journalRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateMediaURL(ctx, id, mediaURL)
}

func (r *journalRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *journalRepository) GetMediaBatch(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetMediaBatch(ctx, entryIDs)
}

func (r *journalRepository) DeleteMedia(ctx context.Context, id int64, entryID uuid.UUID) (string, error) {
	return r.dao.DeleteMedia(ctx, id, entryID)
}
