package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	ShipDAO interface {
		Create(ctx context.Context, userID uuid.UUID, title string, description string, tx ...*sql.Tx) (*model.ShipRow, error)
		UpdateDetails(ctx context.Context, spec ShipDetailsUpdate, tx ...*sql.Tx) error
		UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.ShipRow, error)
		GetAuthorID(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetImagePaths(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.ShipRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ShipRow, int, error)

		InsertCharacters(ctx context.Context, shipID uuid.UUID, characters []dto.ShipCharacter, tx ...*sql.Tx) error
		DeleteCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) error
		GetCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]model.ShipCharacterRow, error)
		GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.ShipCharacterRow, error)

		Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int, tx ...*sql.Tx) error

		CreateComment(ctx context.Context, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, spec NewShipCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
	}

	ShipRepository interface {
		ShipDAO

		CreateWithCharacters(ctx context.Context, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter, tx ...*sql.Tx) (*model.ShipRow, error)
		UpdateWithCharacters(ctx context.Context, spec ShipUpdate, tx ...*sql.Tx) error
		DeleteShip(ctx context.Context, spec ShipDeletion, tx ...*sql.Tx) ([]string, error)
		UpdateCommentBody(ctx context.Context, spec ShipCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, spec ShipCommentDeletion, tx ...*sql.Tx) ([]string, error)
	}

	ShipDetailsUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Title       string
		Description string
		AsAdmin     bool
	}

	ShipUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Title       string
		Description string
		AsAdmin     bool
		Characters  []dto.ShipCharacter
	}

	ShipDeletion struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	ShipCommentUpdate struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		Body      string
		AsAdmin   bool
	}

	ShipCommentDeletion struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		AsAdmin   bool
	}

	NewShipCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		Filename     string
		SortOrder    int
	}
)

type shipRepository struct {
	db    *sql.DB
	dao   ShipDAO
	audit AuditLogRepository
}

func NewShipRepo(database *sql.DB, dao ShipDAO, audit AuditLogRepository) ShipRepository {
	return &shipRepository{db: database, dao: dao, audit: audit}
}

func (r *shipRepository) CreateWithCharacters(ctx context.Context, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter, tx ...*sql.Tx) (*model.ShipRow, error) {
	var created *model.ShipRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.Create(ctx, userID, title, description, tx)
		if err != nil {
			return err
		}

		return r.dao.InsertCharacters(ctx, created.ID, characters, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *shipRepository) UpdateWithCharacters(ctx context.Context, spec ShipUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		details := ShipDetailsUpdate{
			ID:          spec.ID,
			UserID:      spec.UserID,
			Title:       spec.Title,
			Description: spec.Description,
			AsAdmin:     spec.AsAdmin,
		}

		if err := r.dao.UpdateDetails(ctx, details, tx); err != nil {
			return err
		}

		if err := r.dao.DeleteCharacters(ctx, spec.ID, tx); err != nil {
			return err
		}

		return r.dao.InsertCharacters(ctx, spec.ID, spec.Characters, tx)
	})
}

func (r *shipRepository) DeleteShip(ctx context.Context, spec ShipDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		imagePaths, err := r.dao.GetImagePaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.dao.CollectCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		collected := append(imagePaths, commentPaths...)

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

func (r *shipRepository) UpdateCommentBody(ctx context.Context, spec ShipCommentUpdate, tx ...*sql.Tx) error {
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
			Action:     AuditActionShipCommentUpdateAdmin,
			TargetType: AuditTargetShipComment,
			TargetID:   spec.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment update: %w", err)
		}

		return nil
	})
}

func (r *shipRepository) DeleteCommentWithAudit(ctx context.Context, spec ShipCommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.dao, r.audit, commentDeleteSpec{
		CommentID:   spec.CommentID,
		UserID:      spec.UserID,
		AsAdmin:     spec.AsAdmin,
		OwnAction:   AuditActionShipCommentDelete,
		AdminAction: AuditActionShipCommentDeleteAdmin,
		TargetType:  AuditTargetShipComment,
	}, tx)
}

func (r *shipRepository) Create(ctx context.Context, userID uuid.UUID, title string, description string, tx ...*sql.Tx) (*model.ShipRow, error) {
	return r.dao.Create(ctx, userID, title, description, tx...)
}

func (r *shipRepository) UpdateDetails(ctx context.Context, spec ShipDetailsUpdate, tx ...*sql.Tx) error {
	return r.dao.UpdateDetails(ctx, spec, tx...)
}

func (r *shipRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateImage(ctx, id, imageURL, thumbnailURL, tx...)
}

func (r *shipRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, userID, tx...)
}

func (r *shipRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAsAdmin(ctx, id, tx...)
}

func (r *shipRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.ShipRow, error) {
	return r.dao.GetByID(ctx, id, viewerID, tx...)
}

func (r *shipRepository) GetAuthorID(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, shipID, tx...)
}

func (r *shipRepository) GetImagePaths(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetImagePaths(ctx, shipID, tx...)
}

func (r *shipRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *shipRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}

func (r *shipRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.ShipRow, int, error) {
	return r.dao.List(ctx, viewerID, sort, crackshipsOnly, series, characterID, limit, offset, excludeUserIDs, tx...)
}

func (r *shipRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ShipRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset, tx...)
}

func (r *shipRepository) InsertCharacters(ctx context.Context, shipID uuid.UUID, characters []dto.ShipCharacter, tx ...*sql.Tx) error {
	return r.dao.InsertCharacters(ctx, shipID, characters, tx...)
}

func (r *shipRepository) DeleteCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCharacters(ctx, shipID, tx...)
}

func (r *shipRepository) GetCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]model.ShipCharacterRow, error) {
	return r.dao.GetCharacters(ctx, shipID, tx...)
}

func (r *shipRepository) GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.ShipCharacterRow, error) {
	return r.dao.GetCharactersBatch(ctx, shipIDs, tx...)
}

func (r *shipRepository) Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int, tx ...*sql.Tx) error {
	return r.dao.Vote(ctx, userID, shipID, value, tx...)
}

func (r *shipRepository) CreateComment(ctx context.Context, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, shipID, parentID, userID, body, tx...)
}

func (r *shipRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *shipRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *shipRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *shipRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *shipRepository) GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, shipID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *shipRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *shipRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *shipRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *shipRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *shipRepository) AddCommentMedia(ctx context.Context, spec NewShipCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *shipRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *shipRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *shipRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *shipRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}
