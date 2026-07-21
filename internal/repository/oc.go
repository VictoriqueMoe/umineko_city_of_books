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
	GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.OCCommentRow, int, error)
	GetCommentOCID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
	GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
	LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
	UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

	AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
	UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
	UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
	GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.OCCommentMediaRow, error)
	GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.OCCommentMediaRow, error)
}
