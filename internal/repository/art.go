package repository

import (
	"context"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	ArtRepository interface {
		CreateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, artType string, title string, description string, imageURL string, thumbnailURL string, tags []string, isSpoiler bool) error
		UpdateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, tags []string, isSpoiler bool, asAdmin bool) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ArtRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error)
		GetArtAuthorID(ctx context.Context, artID uuid.UUID) (uuid.UUID, error)
		GetImageURL(ctx context.Context, artID uuid.UUID) (string, error)

		Like(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		Unlike(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		GetLikedBy(ctx context.Context, artID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, artID uuid.UUID, viewerHash string) (bool, error)

		GetTags(ctx context.Context, artID uuid.UUID) ([]string, error)
		GetTagsBatch(ctx context.Context, artIDs []uuid.UUID) (map[uuid.UUID][]string, error)
		GetPopularTags(ctx context.Context, corner string, limit int) ([]model.TagCount, error)

		GetCornerCounts(ctx context.Context) (map[string]int, error)
		CountUserArtToday(ctx context.Context, userID uuid.UUID) (int, error)

		CreateComment(ctx context.Context, id uuid.UUID, artID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtCommentRow, int, error)
		GetCommentArtID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error

		SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error

		CreateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error
		UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error
		SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error
		DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		GetGalleryByID(ctx context.Context, id uuid.UUID) (*model.GalleryRow, error)
		ListGalleriesByUser(ctx context.Context, userID uuid.UUID) ([]model.GalleryRow, error)
		ListAllGalleries(ctx context.Context, corner string) ([]model.GalleryRow, error)
		GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int) ([]PreviewImage, error)
		ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error)
	}

	PreviewImage struct {
		ThumbnailURL string
		ImageURL     string
	}
)
