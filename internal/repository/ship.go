package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ShipRepository interface {
		dao.ShipDAO

		CreateWithCharacters(ctx context.Context, s spec.NewShipWithCharacters, tx ...*sql.Tx) (*model.ShipRow, error)
		UpdateWithCharacters(ctx context.Context, update spec.ShipUpdate, tx ...*sql.Tx) error
		DeleteShip(ctx context.Context, deletion spec.ShipDeletion, tx ...*sql.Tx) ([]string, error)
		UpdateCommentBody(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, deletion spec.CommentDeletion, tx ...*sql.Tx) ([]string, error)
	}

	shipRepository struct {
		db *sql.DB
		dao.ShipDAO
		audit AuditLogRepository
	}
)

func NewShipRepo(database *sql.DB, ships dao.ShipDAO, auditLog AuditLogRepository) ShipRepository {
	return &shipRepository{db: database, ShipDAO: ships, audit: auditLog}
}

func (r *shipRepository) CreateWithCharacters(ctx context.Context, s spec.NewShipWithCharacters, tx ...*sql.Tx) (*model.ShipRow, error) {
	var created *model.ShipRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.ShipDAO.Create(ctx, spec.NewShip{
			UserID:      s.UserID,
			Title:       s.Title,
			Description: s.Description,
		}, tx)
		if err != nil {
			return err
		}

		return r.ShipDAO.InsertCharacters(ctx, spec.NewShipCharacters{
			ShipID:     created.ID,
			Characters: s.Characters,
		}, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *shipRepository) UpdateWithCharacters(ctx context.Context, update spec.ShipUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		details := spec.ShipDetailsUpdate{
			ID:          update.ID,
			UserID:      update.UserID,
			Title:       update.Title,
			Description: update.Description,
			AsAdmin:     update.AsAdmin,
		}

		if err := r.ShipDAO.UpdateDetails(ctx, details, tx); err != nil {
			return err
		}

		if err := r.ShipDAO.DeleteCharacters(ctx, update.ID, tx); err != nil {
			return err
		}

		return r.ShipDAO.InsertCharacters(ctx, spec.NewShipCharacters{
			ShipID:     update.ID,
			Characters: update.Characters,
		}, tx)
	})
}

func (r *shipRepository) DeleteShip(ctx context.Context, deletion spec.ShipDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		imagePaths, err := r.ShipDAO.GetImagePaths(ctx, deletion.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.ShipDAO.CollectCommentMediaPaths(ctx, deletion.ID, tx)
		if err != nil {
			return err
		}

		collected := append(imagePaths, commentPaths...)

		if deletion.AsAdmin {
			if err := r.ShipDAO.DeleteAsAdmin(ctx, deletion.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.ShipDAO.Delete(ctx, spec.OwnedDeletion{ID: deletion.ID, UserID: deletion.UserID}, tx); err != nil {
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

func (r *shipRepository) UpdateCommentBody(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		authorID, err := r.ShipDAO.GetCommentAuthorID(ctx, update.CommentID, tx)
		if err != nil {
			return err
		}

		if err := r.ShipDAO.UpdateComment(ctx, update, tx); err != nil {
			return err
		}

		if authorID == update.UserID {
			return nil
		}

		entry := audit.NewEntry{
			ActorID:    update.UserID,
			Action:     audit.ActionShipCommentUpdateAdmin,
			TargetType: audit.TargetShipComment,
			TargetID:   update.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment update: %w", err)
		}

		return nil
	})
}

func (r *shipRepository) DeleteCommentWithAudit(ctx context.Context, deletion spec.CommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.ShipDAO, r.audit, commentDeleteSpec{
		CommentDeletion: deletion,
		OwnAction:       audit.ActionShipCommentDelete,
		AdminAction:     audit.ActionShipCommentDeleteAdmin,
		TargetType:      audit.TargetShipComment,
	}, tx)
}
