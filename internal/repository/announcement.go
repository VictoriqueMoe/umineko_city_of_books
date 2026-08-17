package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	AnnouncementDAO interface {
		Create(ctx context.Context, authorID uuid.UUID, title string, body string, tx ...*sql.Tx) (*AnnouncementRow, error)
		Update(ctx context.Context, id uuid.UUID, title string, body string, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*AnnouncementRow, error)
		List(ctx context.Context, limit, offset int, tx ...*sql.Tx) ([]AnnouncementRow, int, error)
		GetLatest(ctx context.Context, tx ...*sql.Tx) (*AnnouncementRow, error)
		SetPinned(ctx context.Context, id uuid.UUID, pinned bool, tx ...*sql.Tx) error

		CreateComment(ctx context.Context, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, spec NewAnnouncementCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	AnnouncementRepository interface {
		AnnouncementDAO

		UpdateCommentBody(ctx context.Context, spec AnnouncementCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, spec AnnouncementCommentDeletion, tx ...*sql.Tx) ([]string, error)
		DeleteWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	AnnouncementCommentUpdate struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		Body      string
		AsAdmin   bool
	}

	AnnouncementCommentDeletion struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		AsAdmin   bool
	}

	NewAnnouncementCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
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
	db    *sql.DB
	dao   AnnouncementDAO
	audit AuditLogRepository
}

func NewAnnouncementRepo(database *sql.DB, dao AnnouncementDAO, audit AuditLogRepository) AnnouncementRepository {
	return &announcementRepository{db: database, dao: dao, audit: audit}
}

func (r *announcementRepository) UpdateCommentBody(ctx context.Context, spec AnnouncementCommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		authorID, err := r.dao.GetCommentAuthorID(ctx, spec.CommentID, tx)
		if err != nil {
			return err
		}

		if spec.AsAdmin {
			if err := r.dao.UpdateCommentAsAdmin(ctx, spec.CommentID, spec.Body, tx); err != nil {
				return err
			}
		} else {
			if err := r.dao.UpdateComment(ctx, spec.CommentID, spec.UserID, spec.Body, tx); err != nil {
				return err
			}
		}

		if authorID == spec.UserID {
			return nil
		}

		entry := NewAuditEntry{
			ActorID:    spec.UserID,
			Action:     AuditActionAnnouncementCommentUpdateAdmin,
			TargetType: AuditTargetAnnouncementComment,
			TargetID:   spec.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment update: %w", err)
		}

		return nil
	})
}

func (r *announcementRepository) DeleteCommentWithAudit(ctx context.Context, spec AnnouncementCommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.dao, r.audit, commentDeleteSpec{
		CommentID:   spec.CommentID,
		UserID:      spec.UserID,
		AsAdmin:     spec.AsAdmin,
		OwnAction:   AuditActionAnnouncementCommentDelete,
		AdminAction: AuditActionAnnouncementCommentDeleteAdmin,
		TargetType:  AuditTargetAnnouncementComment,
	}, tx)
}

func (r *announcementRepository) DeleteWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.dao.CollectCommentMediaPaths(ctx, id, tx)
		if err != nil {
			return err
		}

		if err := r.dao.Delete(ctx, id, tx); err != nil {
			return err
		}

		paths = mediaPaths

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *announcementRepository) Create(ctx context.Context, authorID uuid.UUID, title string, body string, tx ...*sql.Tx) (*AnnouncementRow, error) {
	return r.dao.Create(ctx, authorID, title, body, tx...)
}

func (r *announcementRepository) Update(ctx context.Context, id uuid.UUID, title string, body string, tx ...*sql.Tx) error {
	return r.dao.Update(ctx, id, title, body, tx...)
}

func (r *announcementRepository) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, tx...)
}

func (r *announcementRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*AnnouncementRow, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *announcementRepository) List(ctx context.Context, limit, offset int, tx ...*sql.Tx) ([]AnnouncementRow, int, error) {
	return r.dao.List(ctx, limit, offset, tx...)
}

func (r *announcementRepository) GetLatest(ctx context.Context, tx ...*sql.Tx) (*AnnouncementRow, error) {
	return r.dao.GetLatest(ctx, tx...)
}

func (r *announcementRepository) SetPinned(ctx context.Context, id uuid.UUID, pinned bool, tx ...*sql.Tx) error {
	return r.dao.SetPinned(ctx, id, pinned, tx...)
}

func (r *announcementRepository) CreateComment(ctx context.Context, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, announcementID, parentID, userID, body, tx...)
}

func (r *announcementRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *announcementRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *announcementRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *announcementRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *announcementRepository) GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, announcementID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *announcementRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *announcementRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *announcementRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *announcementRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *announcementRepository) AddCommentMedia(ctx context.Context, spec NewAnnouncementCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *announcementRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *announcementRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *announcementRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *announcementRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *announcementRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}
