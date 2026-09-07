package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	MysteryRepository interface {
		dao.MysteryDAO

		MarkSolved(ctx context.Context, s spec.MysterySolve, tx ...*sql.Tx) error
		CreateWithClues(ctx context.Context, s spec.NewMysteryWithClues, tx ...*sql.Tx) (*model.MysteryRow, error)
		UpdateWithClues(ctx context.Context, s spec.MysteryUpdateWithClues, tx ...*sql.Tx) error
		DeleteWithFiles(ctx context.Context, s spec.MysteryDelete, tx ...*sql.Tx) ([]string, error)
		DeleteCommentWithAudit(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) ([]string, error)
	}
)

type mysteryRepository struct {
	db *sql.DB
	dao.MysteryDAO
	audit AuditLogRepository
	cache *cache.Manager
}

func NewMysteryRepo(database *sql.DB, mysteryDAO dao.MysteryDAO, auditRepo AuditLogRepository, c *cache.Manager) MysteryRepository {
	return &mysteryRepository{db: database, MysteryDAO: mysteryDAO, audit: auditRepo, cache: c}
}

func (r *mysteryRepository) CreateWithClues(ctx context.Context, s spec.NewMysteryWithClues, tx ...*sql.Tx) (*model.MysteryRow, error) {
	var created *model.MysteryRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.MysteryDAO.Create(ctx, s.NewMystery, tx)
		if err != nil {
			return err
		}

		return r.addClues(ctx, created.ID, s.Clues, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *mysteryRepository) UpdateWithClues(ctx context.Context, s spec.MysteryUpdateWithClues, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.MysteryDAO.UpdateAsAdmin(ctx, s.MysteryUpdate, tx); err != nil {
			return err
		}

		if err := r.MysteryDAO.DeleteClues(ctx, s.ID, tx); err != nil {
			return err
		}

		return r.addClues(ctx, s.ID, s.Clues, tx)
	})
	if err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) addClues(ctx context.Context, mysteryID uuid.UUID, clues []spec.NewClue, tx *sql.Tx) error {
	for _, clue := range clues {
		if _, err := r.MysteryDAO.AddClue(ctx, spec.NewMysteryClue{MysteryID: mysteryID, NewClue: clue}, tx); err != nil {
			return err
		}
	}

	return nil
}

func (r *mysteryRepository) Update(ctx context.Context, s spec.MysteryOwnerUpdate, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.Update(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) UpdateAsAdmin(ctx context.Context, s spec.MysteryUpdate, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.UpdateAsAdmin(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.Delete(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.DeleteAsAdmin(ctx, id, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) DeleteWithFiles(ctx context.Context, s spec.MysteryDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		row, err := r.MysteryDAO.GetByID(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		collected, err := r.collectFilePaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		paths = dedupePaths(collected)

		if s.AsAdmin {
			if err := r.MysteryDAO.DeleteAsAdmin(ctx, s.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.MysteryDAO.Delete(ctx, spec.OwnedDeletion{ID: s.ID, UserID: s.UserID}, tx); err != nil {
				return err
			}
		}

		if row == nil {
			return nil
		}

		action := audit.ActionMysteryDelete
		if row.UserID != s.UserID {
			action = audit.ActionMysteryDeleteAdmin
		}

		if err := r.audit.Create(ctx, audit.NewEntry{
			ActorID:    s.UserID,
			Action:     action,
			TargetType: audit.TargetMystery,
			TargetID:   s.ID.String(),
			Details:    fmt.Sprintf("title=%q attempts=%d", row.Title, row.AttemptCount),
			SubjectID:  row.UserID,
		}, tx); err != nil {
			return fmt.Errorf("audit mystery delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.invalidateLeaderboards(ctx)

	return paths, nil
}

func (r *mysteryRepository) collectFilePaths(ctx context.Context, mysteryID uuid.UUID, tx *sql.Tx) ([]string, error) {
	mediaPaths, err := r.MysteryDAO.CollectMediaPaths(ctx, mysteryID, tx)
	if err != nil {
		return nil, err
	}

	attachmentPaths, err := r.MysteryDAO.GetAttachmentPaths(ctx, mysteryID, tx)
	if err != nil {
		return nil, err
	}

	commentMediaPaths, err := r.MysteryDAO.CollectCommentMediaPaths(ctx, mysteryID, tx)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(mediaPaths)+len(attachmentPaths)+len(commentMediaPaths))
	paths = append(paths, mediaPaths...)
	paths = append(paths, attachmentPaths...)
	paths = append(paths, commentMediaPaths...)

	return dedupePaths(paths), nil
}

func (r *mysteryRepository) invalidateLeaderboards(ctx context.Context) {
	if err := r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key()); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate mystery leaderboard caches after write")
	}
}

func (r *mysteryRepository) DeleteAttempt(ctx context.Context, s spec.MysteryAttemptDeletion, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.DeleteAttempt(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.DeleteAttemptAsAdmin(ctx, id, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) MarkSolved(ctx context.Context, s spec.MysterySolve, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		attemptUserID, attemptMysteryID, err := r.MysteryDAO.GetAttemptOwner(ctx, s.AttemptID, tx)
		if err != nil {
			return err
		}

		if attemptMysteryID != s.MysteryID {
			return fmt.Errorf("attempt does not belong to mystery")
		}

		if s.LockMystery {
			if err := r.MysteryDAO.SetMysteryWinner(ctx, spec.MysteryWinner{MysteryID: s.MysteryID, WinnerID: attemptUserID}, tx); err != nil {
				return err
			}
		}

		return r.MysteryDAO.SetAttemptWinner(ctx, s.AttemptID, tx)
	})
	if err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key())
}

func (r *mysteryRepository) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	if err := r.MysteryDAO.MarkPermanentlySolved(ctx, mysteryID, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key())
}

func (r *mysteryRepository) GetTopDetectiveIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	load := func(ctx context.Context) ([]string, error) {
		return r.MysteryDAO.GetTopDetectiveIDs(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.MysteryTopDetectives, load)
}

func (r *mysteryRepository) GetTopGMIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	load := func(ctx context.Context) ([]string, error) {
		return r.MysteryDAO.GetTopGMIDs(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.MysteryTopGMs, load)
}

func (r *mysteryRepository) DeleteCommentWithAudit(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		authorID, err := r.MysteryDAO.GetCommentAuthorID(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		collected, err := r.MysteryDAO.CollectSingleCommentMediaPaths(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		paths = dedupePaths(collected)

		if err := r.MysteryDAO.DeleteComment(ctx, s, tx); err != nil {
			return err
		}

		action := audit.ActionMysteryCommentDelete
		if authorID != s.UserID {
			action = audit.ActionMysteryCommentDeleteAdmin
		}

		if err := r.audit.Create(ctx, audit.NewEntry{
			ActorID:    s.UserID,
			Action:     action,
			TargetType: audit.TargetMysteryComment,
			TargetID:   s.CommentID.String(),
			SubjectID:  authorID,
		}, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
