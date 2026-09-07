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

	"github.com/google/uuid"
)

type (
	JournalCommentWriter interface {
		CreateComment(ctx context.Context, s spec.NewJournalComment, tx ...*sql.Tx) (*model.CommentRow, error)
	}

	JournalRepository interface {
		dao.JournalDAO

		DeleteWithMedia(ctx context.Context, s spec.JournalDeletion, tx ...*sql.Tx) ([]string, error)
		DeleteEntryWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error)
		DeleteCommentWithAudit(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) ([]string, error)
	}
)

type journalRepository struct {
	db *sql.DB
	dao.JournalDAO
	audit AuditLogRepository
}

func NewJournalRepo(database *sql.DB, journalDAO dao.JournalDAO, auditRepo AuditLogRepository) (JournalRepository, JournalCommentWriter) {
	repo := &journalRepository{db: database, JournalDAO: journalDAO, audit: auditRepo}

	return repo, repo
}

func (r *journalRepository) Update(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error {
	if s.AsAdmin {
		return r.JournalDAO.UpdateAsAdmin(ctx, s, tx...)
	}

	return r.JournalDAO.Update(ctx, s, tx...)
}

func (r *journalRepository) DeleteWithMedia(ctx context.Context, s spec.JournalDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		entryIDs, err := r.JournalDAO.ListEntryIDs(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		var collected []string
		for _, entryID := range entryIDs {
			entryPaths, err := r.JournalDAO.CollectMediaPaths(ctx, entryID, tx)
			if err != nil {
				return err
			}

			collected = append(collected, entryPaths...)
		}

		commentPaths, err := r.JournalDAO.CollectCommentMediaPaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		collected = append(collected, commentPaths...)

		if s.AsAdmin {
			if err := r.JournalDAO.DeleteAsAdmin(ctx, s.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.JournalDAO.Delete(ctx, spec.OwnedDeletion{ID: s.ID, UserID: s.UserID}, tx); err != nil {
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

func (r *journalRepository) CreateEntry(ctx context.Context, s spec.NewJournalEntry, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	var created *model.JournalEntryRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.JournalDAO.CreateEntry(ctx, s, tx)
		if err != nil {
			return err
		}

		return r.JournalDAO.UpdateLastAuthorActivity(ctx, s.JournalID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *journalRepository) UpdateEntry(ctx context.Context, s spec.JournalEntryUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.JournalDAO.UpdateEntry(ctx, s, tx); err != nil {
			return err
		}

		if !s.RecordAuthorActivity {
			return nil
		}

		return r.JournalDAO.UpdateLastAuthorActivity(ctx, s.JournalID, tx)
	})
}

func (r *journalRepository) DeleteEntryWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		collected, err := r.JournalDAO.CollectMediaPaths(ctx, id, tx)
		if err != nil {
			return err
		}

		commentIDs, err := r.JournalDAO.ListEntryCommentIDs(ctx, id, tx)
		if err != nil {
			return err
		}

		for _, commentID := range commentIDs {
			commentPaths, err := r.JournalDAO.CollectSingleCommentMediaPaths(ctx, commentID, tx)
			if err != nil {
				return err
			}

			collected = append(collected, commentPaths...)
		}

		if err := r.JournalDAO.DeleteEntry(ctx, id, tx); err != nil {
			return err
		}

		paths = dedupePaths(collected)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *journalRepository) CreateComment(ctx context.Context, s spec.NewJournalComment, tx ...*sql.Tx) (*model.CommentRow, error) {
	var created *model.CommentRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.JournalDAO.CreateComment(ctx, s, tx)
		if err != nil {
			return err
		}

		if !s.RecordAuthorActivity {
			return nil
		}

		return r.JournalDAO.UpdateLastAuthorActivity(ctx, s.JournalID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *journalRepository) DeleteCommentWithAudit(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.JournalDAO.CollectSingleCommentMediaPaths(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		action := audit.ActionJournalCommentDelete
		if s.AsAdmin {
			action = audit.ActionJournalCommentDeleteAdmin
		}

		if err := r.JournalDAO.DeleteComment(ctx, s, tx); err != nil {
			return err
		}

		entry := audit.NewEntry{
			ActorID:    s.UserID,
			Action:     action,
			TargetType: audit.TargetJournalComment,
			TargetID:   s.CommentID.String(),
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		paths = mediaPaths

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
