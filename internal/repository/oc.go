package repository

import (
	"context"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type OCRepository interface {
	Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string) error
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string, asAdmin bool) error
	UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.OCRow, error)
	GetAuthorID(ctx context.Context, ocID uuid.UUID) (uuid.UUID, error)
	List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCRow, int, error)
	ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.OCRow, int, error)
	ListSummariesByUser(ctx context.Context, userID uuid.UUID) ([]model.OCSummaryRow, error)
	HasOC(ctx context.Context, userID uuid.UUID, name string) (bool, error)

	AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int) (int64, error)
	UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string) error
	UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string) error
	UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int) error
	DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID) error
	GetGallery(ctx context.Context, ocID uuid.UUID) ([]model.OCImageRow, error)
	GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID) (map[uuid.UUID][]model.OCImageRow, error)

	Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error
	Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error
	Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error

	CreateComment(ctx context.Context, id uuid.UUID, ocID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
	UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
	UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
	DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
	GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
	GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
	GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
	LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
	UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

	AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
	UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
	UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
	GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
	GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
}

type ocRepository struct {
	dao OCRepository
}

func NewOCRepo(dao OCRepository) OCRepository {
	return &ocRepository{dao: dao}
}

func (r *ocRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string) error {
	return r.dao.Create(ctx, id, userID, name, description, series, customSeriesName)
}

func (r *ocRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, series string, customSeriesName string, asAdmin bool) error {
	return r.dao.Update(ctx, id, userID, name, description, series, customSeriesName, asAdmin)
}

func (r *ocRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	return r.dao.UpdateImage(ctx, id, imageURL, thumbnailURL)
}

func (r *ocRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *ocRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *ocRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.OCRow, error) {
	return r.dao.GetByID(ctx, id, viewerID)
}

func (r *ocRepository) GetAuthorID(ctx context.Context, ocID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, ocID)
}

func (r *ocRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCRow, int, error) {
	return r.dao.List(ctx, viewerID, sort, crackOCsOnly, series, customSeriesName, ownerID, limit, offset, excludeUserIDs)
}

func (r *ocRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.OCRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset)
}

func (r *ocRepository) ListSummariesByUser(ctx context.Context, userID uuid.UUID) ([]model.OCSummaryRow, error) {
	return r.dao.ListSummariesByUser(ctx, userID)
}

func (r *ocRepository) HasOC(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	return r.dao.HasOC(ctx, userID, name)
}

func (r *ocRepository) AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int) (int64, error) {
	return r.dao.AddGalleryImage(ctx, ocID, imageURL, thumbnailURL, caption, sortOrder)
}

func (r *ocRepository) UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string) error {
	return r.dao.UpdateGalleryImageURL(ctx, id, imageURL)
}

func (r *ocRepository) UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateGalleryImageThumbnail(ctx, id, thumbnailURL)
}

func (r *ocRepository) UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int) error {
	return r.dao.UpdateGalleryImage(ctx, id, ocID, caption, sortOrder)
}

func (r *ocRepository) DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID) error {
	return r.dao.DeleteGalleryImage(ctx, id, ocID)
}

func (r *ocRepository) GetGallery(ctx context.Context, ocID uuid.UUID) ([]model.OCImageRow, error) {
	return r.dao.GetGallery(ctx, ocID)
}

func (r *ocRepository) GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID) (map[uuid.UUID][]model.OCImageRow, error) {
	return r.dao.GetGalleryBatch(ctx, ocIDs)
}

func (r *ocRepository) Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error {
	return r.dao.Vote(ctx, userID, ocID, value)
}

func (r *ocRepository) Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error {
	return r.dao.Favourite(ctx, userID, ocID)
}

func (r *ocRepository) Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) error {
	return r.dao.Unfavourite(ctx, userID, ocID)
}

func (r *ocRepository) CreateComment(ctx context.Context, id uuid.UUID, ocID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, ocID, parentID, userID, body)
}

func (r *ocRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *ocRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *ocRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *ocRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *ocRepository) GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, ocID, viewerID, limit, offset, excludeUserIDs)
}

func (r *ocRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *ocRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *ocRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *ocRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *ocRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *ocRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *ocRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *ocRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *ocRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}
