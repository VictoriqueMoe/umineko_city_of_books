package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	NewOC struct {
		UserID           uuid.UUID
		Name             string
		Description      string
		Series           string
		CustomSeriesName string
	}

	OCUpdate struct {
		ID               uuid.UUID
		UserID           uuid.UUID
		Name             string
		Description      string
		Series           string
		CustomSeriesName string
		AsAdmin          bool
	}

	OCDeletion struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	OCCommentDeletion struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		AsAdmin   bool
	}

	OCDAO interface {
		Create(ctx context.Context, spec NewOC, tx ...*sql.Tx) (*model.OCRow, error)
		Update(ctx context.Context, spec OCUpdate, tx ...*sql.Tx) error
		UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.OCRow, error)
		GetAuthorID(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetImagePaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetGalleryPaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.OCRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.OCRow, int, error)
		ListSummariesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.OCSummaryRow, error)
		HasOC(ctx context.Context, userID uuid.UUID, name string, tx ...*sql.Tx) (bool, error)

		AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int, tx ...*sql.Tx) (int64, error)
		UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string, tx ...*sql.Tx) error
		UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int, tx ...*sql.Tx) error
		DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, tx ...*sql.Tx) error
		GetGallery(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]model.OCImageRow, error)
		GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.OCImageRow, error)

		Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int, tx ...*sql.Tx) error
		Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, tx ...*sql.Tx) error
		Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, tx ...*sql.Tx) error

		CreateComment(ctx context.Context, ocID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
	}

	OCRepository interface {
		OCDAO

		DeleteOC(ctx context.Context, spec OCDeletion, tx ...*sql.Tx) ([]string, error)
		DeleteCommentWithMedia(ctx context.Context, spec OCCommentDeletion, tx ...*sql.Tx) ([]string, error)
	}
)

type ocRepository struct {
	db    *sql.DB
	dao   OCDAO
	audit AuditLogRepository
}

func NewOCRepo(database *sql.DB, dao OCDAO, audit AuditLogRepository) OCRepository {
	return &ocRepository{db: database, dao: dao, audit: audit}
}

func (r *ocRepository) DeleteOC(ctx context.Context, spec OCDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		imagePaths, err := r.dao.GetImagePaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		galleryPaths, err := r.dao.GetGalleryPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.dao.CollectCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		collected := append(imagePaths, galleryPaths...)
		collected = append(collected, commentPaths...)

		if spec.AsAdmin {
			if err := r.dao.DeleteAsAdmin(ctx, spec.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.dao.Delete(ctx, spec.ID, spec.UserID, tx); err != nil {
				return err
			}
		}

		paths = dedupePaths(collected)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *ocRepository) DeleteCommentWithMedia(ctx context.Context, spec OCCommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.dao, r.audit, commentDeleteSpec{
		CommentID:   spec.CommentID,
		UserID:      spec.UserID,
		AsAdmin:     spec.AsAdmin,
		OwnAction:   AuditActionOCCommentDelete,
		AdminAction: AuditActionOCCommentDeleteAdmin,
		TargetType:  AuditTargetOCComment,
	}, tx)
}

func (r *ocRepository) GetImagePaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetImagePaths(ctx, ocID, tx...)
}

func (r *ocRepository) GetGalleryPaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetGalleryPaths(ctx, ocID, tx...)
}

func (r *ocRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *ocRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}

func (r *ocRepository) Create(ctx context.Context, spec NewOC, tx ...*sql.Tx) (*model.OCRow, error) {
	return r.dao.Create(ctx, spec, tx...)
}

func (r *ocRepository) Update(ctx context.Context, spec OCUpdate, tx ...*sql.Tx) error {
	return r.dao.Update(ctx, spec, tx...)
}

func (r *ocRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateImage(ctx, id, imageURL, thumbnailURL, tx...)
}

func (r *ocRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, userID, tx...)
}

func (r *ocRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAsAdmin(ctx, id, tx...)
}

func (r *ocRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.OCRow, error) {
	return r.dao.GetByID(ctx, id, viewerID, tx...)
}

func (r *ocRepository) GetAuthorID(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, ocID, tx...)
}

func (r *ocRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackOCsOnly bool, series string, customSeriesName string, ownerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.OCRow, int, error) {
	return r.dao.List(ctx, viewerID, sort, crackOCsOnly, series, customSeriesName, ownerID, limit, offset, excludeUserIDs, tx...)
}

func (r *ocRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.OCRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset, tx...)
}

func (r *ocRepository) ListSummariesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.OCSummaryRow, error) {
	return r.dao.ListSummariesByUser(ctx, userID, tx...)
}

func (r *ocRepository) HasOC(ctx context.Context, userID uuid.UUID, name string, tx ...*sql.Tx) (bool, error) {
	return r.dao.HasOC(ctx, userID, name, tx...)
}

func (r *ocRepository) AddGalleryImage(ctx context.Context, ocID uuid.UUID, imageURL string, thumbnailURL string, caption string, sortOrder int, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddGalleryImage(ctx, ocID, imageURL, thumbnailURL, caption, sortOrder, tx...)
}

func (r *ocRepository) UpdateGalleryImageURL(ctx context.Context, id int64, imageURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateGalleryImageURL(ctx, id, imageURL, tx...)
}

func (r *ocRepository) UpdateGalleryImageThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateGalleryImageThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *ocRepository) UpdateGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, caption *string, sortOrder *int, tx ...*sql.Tx) error {
	return r.dao.UpdateGalleryImage(ctx, id, ocID, caption, sortOrder, tx...)
}

func (r *ocRepository) DeleteGalleryImage(ctx context.Context, id int64, ocID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteGalleryImage(ctx, id, ocID, tx...)
}

func (r *ocRepository) GetGallery(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]model.OCImageRow, error) {
	return r.dao.GetGallery(ctx, ocID, tx...)
}

func (r *ocRepository) GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.OCImageRow, error) {
	return r.dao.GetGalleryBatch(ctx, ocIDs, tx...)
}

func (r *ocRepository) Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int, tx ...*sql.Tx) error {
	return r.dao.Vote(ctx, userID, ocID, value, tx...)
}

func (r *ocRepository) Favourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Favourite(ctx, userID, ocID, tx...)
}

func (r *ocRepository) Unfavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unfavourite(ctx, userID, ocID, tx...)
}

func (r *ocRepository) CreateComment(ctx context.Context, ocID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, ocID, parentID, userID, body, tx...)
}

func (r *ocRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *ocRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *ocRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *ocRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *ocRepository) GetComments(ctx context.Context, ocID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, ocID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *ocRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *ocRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *ocRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *ocRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *ocRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder, tx...)
}

func (r *ocRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *ocRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *ocRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *ocRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}
