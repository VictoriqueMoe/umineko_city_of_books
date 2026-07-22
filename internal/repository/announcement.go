package repository

import (
	"context"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	AnnouncementRepository interface {
		Create(ctx context.Context, id uuid.UUID, authorID uuid.UUID, title string, body string) error
		Update(ctx context.Context, id uuid.UUID, title string, body string) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*AnnouncementRow, error)
		List(ctx context.Context, limit, offset int) ([]AnnouncementRow, int, error)
		GetLatest(ctx context.Context) (*AnnouncementRow, error)
		SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error

		CreateComment(ctx context.Context, id uuid.UUID, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
	}

	AnnouncementRow struct {
		ID                uuid.UUID
		Title             string
		Body              string
		AuthorID          uuid.UUID
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		Pinned            bool
		CreatedAt         string
		UpdatedAt         string
	}
)

type announcementRepository struct {
	dao AnnouncementRepository
}

func NewAnnouncementRepo(dao AnnouncementRepository) AnnouncementRepository {
	return &announcementRepository{dao: dao}
}

func (r *announcementRepository) Create(ctx context.Context, id uuid.UUID, authorID uuid.UUID, title string, body string) error {
	return r.dao.Create(ctx, id, authorID, title, body)
}

func (r *announcementRepository) Update(ctx context.Context, id uuid.UUID, title string, body string) error {
	return r.dao.Update(ctx, id, title, body)
}

func (r *announcementRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.dao.Delete(ctx, id)
}

func (r *announcementRepository) GetByID(ctx context.Context, id uuid.UUID) (*AnnouncementRow, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *announcementRepository) List(ctx context.Context, limit, offset int) ([]AnnouncementRow, int, error) {
	return r.dao.List(ctx, limit, offset)
}

func (r *announcementRepository) GetLatest(ctx context.Context) (*AnnouncementRow, error) {
	return r.dao.GetLatest(ctx)
}

func (r *announcementRepository) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	return r.dao.SetPinned(ctx, id, pinned)
}

func (r *announcementRepository) CreateComment(ctx context.Context, id uuid.UUID, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, announcementID, parentID, userID, body)
}

func (r *announcementRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *announcementRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *announcementRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *announcementRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *announcementRepository) GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, announcementID, viewerID, limit, offset, excludeUserIDs)
}

func (r *announcementRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *announcementRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *announcementRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *announcementRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *announcementRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *announcementRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *announcementRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *announcementRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}
