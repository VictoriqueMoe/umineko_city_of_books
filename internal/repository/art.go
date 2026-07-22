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
		GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
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

type artRepository struct {
	dao ArtRepository
}

func NewArtRepo(dao ArtRepository) ArtRepository {
	return &artRepository{dao: dao}
}

func (r *artRepository) CreateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, artType string, title string, description string, imageURL string, thumbnailURL string, tags []string, isSpoiler bool) error {
	return r.dao.CreateWithTags(ctx, id, userID, corner, artType, title, description, imageURL, thumbnailURL, tags, isSpoiler)
}

func (r *artRepository) UpdateWithTags(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, tags []string, isSpoiler bool, asAdmin bool) error {
	return r.dao.UpdateWithTags(ctx, id, userID, title, description, tags, isSpoiler, asAdmin)
}

func (r *artRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ArtRow, error) {
	return r.dao.GetByID(ctx, id, viewerID)
}

func (r *artRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *artRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *artRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ArtRow, int, error) {
	return r.dao.ListAll(ctx, viewerID, corner, artType, search, tag, sort, limit, offset, excludeUserIDs)
}

func (r *artRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset)
}

func (r *artRepository) GetArtAuthorID(ctx context.Context, artID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetArtAuthorID(ctx, artID)
}

func (r *artRepository) GetImageURL(ctx context.Context, artID uuid.UUID) (string, error) {
	return r.dao.GetImageURL(ctx, artID)
}

func (r *artRepository) Like(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	return r.dao.Like(ctx, userID, artID)
}

func (r *artRepository) Unlike(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	return r.dao.Unlike(ctx, userID, artID)
}

func (r *artRepository) GetLikedBy(ctx context.Context, artID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error) {
	return r.dao.GetLikedBy(ctx, artID, excludeUserIDs)
}

func (r *artRepository) RecordView(ctx context.Context, artID uuid.UUID, viewerHash string) (bool, error) {
	return r.dao.RecordView(ctx, artID, viewerHash)
}

func (r *artRepository) GetTags(ctx context.Context, artID uuid.UUID) ([]string, error) {
	return r.dao.GetTags(ctx, artID)
}

func (r *artRepository) GetTagsBatch(ctx context.Context, artIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	return r.dao.GetTagsBatch(ctx, artIDs)
}

func (r *artRepository) GetPopularTags(ctx context.Context, corner string, limit int) ([]model.TagCount, error) {
	return r.dao.GetPopularTags(ctx, corner, limit)
}

func (r *artRepository) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	return r.dao.GetCornerCounts(ctx)
}

func (r *artRepository) CountUserArtToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUserArtToday(ctx, userID)
}

func (r *artRepository) CreateComment(ctx context.Context, id uuid.UUID, artID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, artID, parentID, userID, body)
}

func (r *artRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *artRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *artRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *artRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *artRepository) GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, artID, viewerID, limit, offset, excludeUserIDs)
}

func (r *artRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *artRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *artRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *artRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *artRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *artRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *artRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}

func (r *artRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *artRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *artRepository) SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error {
	return r.dao.SetGallery(ctx, artID, userID, galleryID)
}

func (r *artRepository) CreateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error {
	return r.dao.CreateGallery(ctx, id, userID, name, description)
}

func (r *artRepository) UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string) error {
	return r.dao.UpdateGallery(ctx, id, userID, name, description)
}

func (r *artRepository) SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error {
	return r.dao.SetGalleryCover(ctx, galleryID, userID, coverArtID)
}

func (r *artRepository) DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteGallery(ctx, id, userID)
}

func (r *artRepository) GetGalleryByID(ctx context.Context, id uuid.UUID) (*model.GalleryRow, error) {
	return r.dao.GetGalleryByID(ctx, id)
}

func (r *artRepository) ListGalleriesByUser(ctx context.Context, userID uuid.UUID) ([]model.GalleryRow, error) {
	return r.dao.ListGalleriesByUser(ctx, userID)
}

func (r *artRepository) ListAllGalleries(ctx context.Context, corner string) ([]model.GalleryRow, error) {
	return r.dao.ListAllGalleries(ctx, corner)
}

func (r *artRepository) GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int) ([]PreviewImage, error) {
	return r.dao.GetGalleryPreviewImages(ctx, galleryID, limit)
}

func (r *artRepository) ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ArtRow, int, error) {
	return r.dao.ListArtInGallery(ctx, galleryID, viewerID, limit, offset)
}
