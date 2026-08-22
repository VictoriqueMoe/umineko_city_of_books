package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

var (
	ErrArtNotOwned = errors.New("art or gallery not found or not owned")
)

type (
	ArtDAO interface {
		CreateArt(ctx context.Context, spec NewArt, tx ...*sql.Tx) (*model.ArtRow, error)
		UpdateArt(ctx context.Context, spec ArtUpdate, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.ArtRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.ArtRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ArtRow, int, error)
		GetArtAuthorID(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetImageURL(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetArtImagePaths(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListGalleryArtImages(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]ArtImageRef, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		Like(ctx context.Context, userID uuid.UUID, artID uuid.UUID, tx ...*sql.Tx) error
		Unlike(ctx context.Context, userID uuid.UUID, artID uuid.UUID, tx ...*sql.Tx) error
		GetLikedBy(ctx context.Context, artID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, artID uuid.UUID, viewerHash string, tx ...*sql.Tx) (bool, error)

		InsertTags(ctx context.Context, artID uuid.UUID, tags []string, tx ...*sql.Tx) error
		DeleteTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) error
		GetTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetTagsBatch(ctx context.Context, artIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)
		GetPopularTags(ctx context.Context, corner string, limit int, tx ...*sql.Tx) ([]model.TagCount, error)

		GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error)
		CountUserArtToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)

		CreateComment(ctx context.Context, artID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, spec NewArtCommentMedia, tx ...*sql.Tx) (int64, error)
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error

		SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID, tx ...*sql.Tx) error

		CreateGallery(ctx context.Context, userID uuid.UUID, name string, description string, tx ...*sql.Tx) (*model.GalleryRow, error)
		UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, tx ...*sql.Tx) error
		SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID, tx ...*sql.Tx) error
		DeleteArtInGallery(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteGalleryRow(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		GetGalleryByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GalleryRow, error)
		ListGalleriesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.GalleryRow, error)
		ListAllGalleries(ctx context.Context, corner string, tx ...*sql.Tx) ([]model.GalleryRow, error)
		GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int, tx ...*sql.Tx) ([]PreviewImage, error)
		ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ArtRow, int, error)
	}

	ArtRepository interface {
		ArtDAO

		CreateWithTags(ctx context.Context, spec NewArtWithTags, tx ...*sql.Tx) (*model.ArtRow, error)
		UpdateWithTags(ctx context.Context, spec ArtUpdateWithTags, tx ...*sql.Tx) error
		DeleteWithImage(ctx context.Context, spec ArtDelete, tx ...*sql.Tx) ([]string, error)
		UpdateCommentWithDetails(ctx context.Context, spec ArtCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, spec ArtCommentDelete, tx ...*sql.Tx) ([]string, error)
		DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	NewArt struct {
		UserID       uuid.UUID
		Corner       string
		ArtType      string
		Title        string
		Description  string
		ImageURL     string
		ThumbnailURL string
		IsSpoiler    bool
	}

	NewArtWithTags struct {
		NewArt

		Tags []string
	}

	ArtUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Title       string
		Description string
		IsSpoiler   bool
		AsAdmin     bool
	}

	ArtUpdateWithTags struct {
		ArtUpdate

		Tags []string
	}

	ArtDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
		Audit   NewAuditEntry
	}

	ArtCommentUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Body    string
		AsAdmin bool
	}

	ArtCommentDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
		Audit   NewAuditEntry
	}

	NewArtCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		Filename     string
		SortOrder    int
	}

	PreviewImage struct {
		ThumbnailURL string
		ImageURL     string
	}

	ArtImageRef struct {
		ArtID        uuid.UUID
		ImageURL     string
		ThumbnailURL string
	}
)

type artRepository struct {
	db    *sql.DB
	dao   ArtDAO
	posts PostRepository
	audit AuditLogRepository
}

func NewArtRepo(database *sql.DB, dao ArtDAO, posts PostRepository, audit AuditLogRepository) ArtRepository {
	return &artRepository{db: database, dao: dao, posts: posts, audit: audit}
}

func (r *artRepository) CreateWithTags(ctx context.Context, spec NewArtWithTags, tx ...*sql.Tx) (*model.ArtRow, error) {
	var created *model.ArtRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateArt(ctx, spec.NewArt, tx)
		if err != nil {
			return err
		}

		return r.dao.InsertTags(ctx, created.ID, spec.Tags, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *artRepository) UpdateWithTags(ctx context.Context, spec ArtUpdateWithTags, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.UpdateArt(ctx, spec.ArtUpdate, tx); err != nil {
			return err
		}

		if err := r.dao.DeleteTags(ctx, spec.ID, tx); err != nil {
			return err
		}

		return r.dao.InsertTags(ctx, spec.ID, spec.Tags, tx)
	})
}

func (r *artRepository) DeleteWithImage(ctx context.Context, spec ArtDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		imagePaths, err := r.dao.GetArtImagePaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.dao.CollectCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		paths = append(imagePaths, commentPaths...)

		if spec.AsAdmin {
			err = r.dao.DeleteAsAdmin(ctx, spec.ID, tx)
		} else {
			err = r.dao.Delete(ctx, spec.ID, spec.UserID, tx)
		}
		if err != nil {
			return err
		}

		if err := r.audit.Create(ctx, spec.Audit, tx); err != nil {
			return fmt.Errorf("audit art delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *artRepository) UpdateCommentWithDetails(ctx context.Context, spec ArtCommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		if spec.AsAdmin {
			err = r.dao.UpdateCommentAsAdmin(ctx, spec.ID, spec.Body, tx)
		} else {
			err = r.dao.UpdateComment(ctx, spec.ID, spec.UserID, spec.Body, tx)
		}
		return err
	})
}

func (r *artRepository) DeleteCommentWithAudit(ctx context.Context, spec ArtCommentDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.dao.CollectSingleCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		paths = dedupePaths(mediaPaths)

		if spec.AsAdmin {
			err = r.dao.DeleteCommentAsAdmin(ctx, spec.ID, tx)
		} else {
			err = r.dao.DeleteComment(ctx, spec.ID, spec.UserID, tx)
		}
		if err != nil {
			return err
		}

		if err := r.audit.Create(ctx, spec.Audit, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *artRepository) CreateArt(ctx context.Context, spec NewArt, tx ...*sql.Tx) (*model.ArtRow, error) {
	return r.dao.CreateArt(ctx, spec, tx...)
}

func (r *artRepository) UpdateArt(ctx context.Context, spec ArtUpdate, tx ...*sql.Tx) error {
	return r.dao.UpdateArt(ctx, spec, tx...)
}

func (r *artRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.ArtRow, error) {
	return r.dao.GetByID(ctx, id, viewerID, tx...)
}

func (r *artRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, userID, tx...)
}

func (r *artRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAsAdmin(ctx, id, tx...)
}

func (r *artRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	return r.dao.ListAll(ctx, viewerID, corner, artType, search, tag, sort, limit, offset, excludeUserIDs, tx...)
}

func (r *artRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset, tx...)
}

func (r *artRepository) GetArtAuthorID(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetArtAuthorID(ctx, artID, tx...)
}

func (r *artRepository) GetImageURL(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetImageURL(ctx, artID, tx...)
}

func (r *artRepository) GetArtImagePaths(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetArtImagePaths(ctx, artID, tx...)
}

func (r *artRepository) ListGalleryArtImages(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]ArtImageRef, error) {
	return r.dao.ListGalleryArtImages(ctx, galleryID, userID, tx...)
}

func (r *artRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *artRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}

func (r *artRepository) Like(ctx context.Context, userID uuid.UUID, artID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Like(ctx, userID, artID, tx...)
}

func (r *artRepository) Unlike(ctx context.Context, userID uuid.UUID, artID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unlike(ctx, userID, artID, tx...)
}

func (r *artRepository) GetLikedBy(ctx context.Context, artID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostLikeUser, error) {
	return r.dao.GetLikedBy(ctx, artID, excludeUserIDs, tx...)
}

func (r *artRepository) RecordView(ctx context.Context, artID uuid.UUID, viewerHash string, tx ...*sql.Tx) (bool, error) {
	return r.dao.RecordView(ctx, artID, viewerHash, tx...)
}

func (r *artRepository) InsertTags(ctx context.Context, artID uuid.UUID, tags []string, tx ...*sql.Tx) error {
	return r.dao.InsertTags(ctx, artID, tags, tx...)
}

func (r *artRepository) DeleteTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteTags(ctx, artID, tx...)
}

func (r *artRepository) GetTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetTags(ctx, artID, tx...)
}

func (r *artRepository) GetTagsBatch(ctx context.Context, artIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	return r.dao.GetTagsBatch(ctx, artIDs, tx...)
}

func (r *artRepository) GetPopularTags(ctx context.Context, corner string, limit int, tx ...*sql.Tx) ([]model.TagCount, error) {
	return r.dao.GetPopularTags(ctx, corner, limit, tx...)
}

func (r *artRepository) GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error) {
	return r.dao.GetCornerCounts(ctx, tx...)
}

func (r *artRepository) CountUserArtToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUserArtToday(ctx, userID, tx...)
}

func (r *artRepository) CreateComment(ctx context.Context, artID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, artID, parentID, userID, body, tx...)
}

func (r *artRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *artRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *artRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *artRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *artRepository) GetComments(ctx context.Context, artID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, artID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *artRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *artRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *artRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *artRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *artRepository) AddCommentMedia(ctx context.Context, spec NewArtCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *artRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *artRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *artRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *artRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *artRepository) SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetGallery(ctx, artID, userID, galleryID, tx...)
}

func (r *artRepository) CreateGallery(ctx context.Context, userID uuid.UUID, name string, description string, tx ...*sql.Tx) (*model.GalleryRow, error) {
	return r.dao.CreateGallery(ctx, userID, name, description, tx...)
}

func (r *artRepository) UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, tx ...*sql.Tx) error {
	return r.dao.UpdateGallery(ctx, id, userID, name, description, tx...)
}

func (r *artRepository) SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetGalleryCover(ctx, galleryID, userID, coverArtID, tx...)
}

func (r *artRepository) DeleteArtInGallery(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteArtInGallery(ctx, galleryID, userID, tx...)
}

func (r *artRepository) DeleteGalleryRow(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteGalleryRow(ctx, id, userID, tx...)
}

func (r *artRepository) DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		images, err := r.dao.ListGalleryArtImages(ctx, id, userID, tx)
		if err != nil {
			return err
		}

		var collected []string

		for i := range images {
			if images[i].ImageURL != "" {
				collected = append(collected, images[i].ImageURL)
			}

			if images[i].ThumbnailURL != "" {
				collected = append(collected, images[i].ThumbnailURL)
			}

			commentPaths, commentErr := r.dao.CollectCommentMediaPaths(ctx, images[i].ArtID, tx)
			if commentErr != nil {
				return commentErr
			}

			collected = append(collected, commentPaths...)
		}

		paths = dedupePaths(collected)

		if err := r.dao.DeleteArtInGallery(ctx, id, userID, tx); err != nil {
			return err
		}

		return r.dao.DeleteGalleryRow(ctx, id, userID, tx)
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *artRepository) GetGalleryByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GalleryRow, error) {
	return r.dao.GetGalleryByID(ctx, id, tx...)
}

func (r *artRepository) ListGalleriesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.GalleryRow, error) {
	return r.dao.ListGalleriesByUser(ctx, userID, tx...)
}

func (r *artRepository) ListAllGalleries(ctx context.Context, corner string, tx ...*sql.Tx) ([]model.GalleryRow, error) {
	return r.dao.ListAllGalleries(ctx, corner, tx...)
}

func (r *artRepository) GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int, tx ...*sql.Tx) ([]PreviewImage, error) {
	return r.dao.GetGalleryPreviewImages(ctx, galleryID, limit, tx...)
}

func (r *artRepository) ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	return r.dao.ListArtInGallery(ctx, galleryID, viewerID, limit, offset, tx...)
}
