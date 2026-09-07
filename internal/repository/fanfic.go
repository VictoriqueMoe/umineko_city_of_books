package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	FanficRepository interface {
		dao.FanficDAO

		CreateWithDetails(ctx context.Context, s spec.NewFanficWithDetails, tx ...*sql.Tx) (*model.FanficRow, error)
		UpdateWithDetails(ctx context.Context, s spec.FanficUpdateWithDetails, tx ...*sql.Tx) error
		DeleteFanfic(ctx context.Context, s spec.FanficDelete, tx ...*sql.Tx) ([]string, error)

		CreateChapterWithCount(ctx context.Context, s spec.NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error)
		UpdateChapterWithCount(ctx context.Context, s spec.ChapterUpdate, tx ...*sql.Tx) error
		DeleteChapterWithCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error

		RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error)

		UpdateCommentBody(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, s spec.FanficCommentDelete, tx ...*sql.Tx) ([]string, error)
	}
)

type fanficRepository struct {
	db *sql.DB
	dao.FanficDAO
	audit AuditLogRepository
}

func NewFanficRepo(database *sql.DB, fanficDAO dao.FanficDAO, auditRepo AuditLogRepository) FanficRepository {
	return &fanficRepository{db: database, FanficDAO: fanficDAO, audit: auditRepo}
}

func (r *fanficRepository) CreateWithDetails(ctx context.Context, s spec.NewFanficWithDetails, tx ...*sql.Tx) (*model.FanficRow, error) {
	var created *model.FanficRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.FanficDAO.RegisterSeries(ctx, s.Series, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.RegisterLanguage(ctx, s.Language, tx); err != nil {
			return err
		}

		var err error

		created, err = r.FanficDAO.Create(ctx, s.NewFanfic, tx)
		if err != nil {
			return err
		}

		if err := r.FanficDAO.AddGenres(ctx, spec.FanficGenres{FanficID: created.ID, Genres: s.Genres}, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.AddTags(ctx, spec.FanficTags{FanficID: created.ID, Tags: s.Tags}, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.AddCharacters(ctx, spec.FanficCharacters{FanficID: created.ID, Characters: s.Characters, IsPairing: s.IsPairing}, tx); err != nil {
			return err
		}

		if err := r.registerOCCharacters(ctx, s.UserID, s.Characters, tx); err != nil {
			return err
		}

		if s.FirstChapter == nil {
			return nil
		}

		chapter := *s.FirstChapter
		chapter.FanficID = created.ID

		if _, err := r.FanficDAO.CreateChapter(ctx, chapter, tx); err != nil {
			return err
		}

		return r.FanficDAO.UpdateWordCount(ctx, created.ID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *fanficRepository) registerOCCharacters(ctx context.Context, userID uuid.UUID, characters []dto.FanficCharacter, tx *sql.Tx) error {
	for _, c := range characters {
		name := strings.TrimSpace(c.CharacterName)
		if c.CharacterID != "" || name == "" {
			continue
		}

		if err := r.FanficDAO.RegisterOCCharacter(ctx, spec.NewFanficOCCharacter{Name: name, CreatorID: userID}, tx); err != nil {
			return err
		}
	}

	return nil
}

func (r *fanficRepository) UpdateWithDetails(ctx context.Context, s spec.FanficUpdateWithDetails, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.FanficDAO.RegisterSeries(ctx, s.Series, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.RegisterLanguage(ctx, s.Language, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.Update(ctx, s.FanficUpdate, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.DeleteGenres(ctx, s.ID, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.DeleteTags(ctx, s.ID, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.DeleteCharacters(ctx, s.ID, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.AddGenres(ctx, spec.FanficGenres{FanficID: s.ID, Genres: s.Genres}, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.AddTags(ctx, spec.FanficTags{FanficID: s.ID, Tags: s.Tags}, tx); err != nil {
			return err
		}

		if err := r.FanficDAO.AddCharacters(ctx, spec.FanficCharacters{FanficID: s.ID, Characters: s.Characters, IsPairing: s.IsPairing}, tx); err != nil {
			return err
		}

		return r.registerOCCharacters(ctx, s.UserID, s.Characters, tx)
	})
}

func (r *fanficRepository) DeleteFanfic(ctx context.Context, s spec.FanficDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		coverPaths, err := r.FanficDAO.GetCoverImagePaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.FanficDAO.CollectCommentMediaPaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		collected := append(coverPaths, commentPaths...)

		if s.AsAdmin {
			err = r.FanficDAO.DeleteAsAdmin(ctx, s.ID, tx)
		} else {
			err = r.FanficDAO.Delete(ctx, spec.OwnedDeletion{ID: s.ID, UserID: s.UserID}, tx)
		}
		if err != nil {
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

func (r *fanficRepository) CreateChapterWithCount(ctx context.Context, s spec.NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	var created *model.FanficChapterRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.FanficDAO.CreateChapter(ctx, s, tx)
		if err != nil {
			return err
		}

		return r.FanficDAO.UpdateWordCount(ctx, s.FanficID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *fanficRepository) UpdateChapterWithCount(ctx context.Context, s spec.ChapterUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.FanficDAO.UpdateChapter(ctx, s, tx); err != nil {
			return err
		}

		fanficID, err := r.FanficDAO.GetChapterFanficID(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		return r.FanficDAO.UpdateWordCount(ctx, fanficID, tx)
	})
}

func (r *fanficRepository) DeleteChapterWithCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		fanficID, err := r.FanficDAO.GetChapterFanficID(ctx, id, tx)
		if err != nil {
			return err
		}

		if err := r.FanficDAO.DeleteChapter(ctx, id, tx); err != nil {
			return err
		}

		return r.FanficDAO.UpdateWordCount(ctx, fanficID, tx)
	})
}

func (r *fanficRepository) RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error) {
	var inserted bool

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		ok, err := r.FanficDAO.RecordView(ctx, s, tx)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		if err := r.FanficDAO.IncrementViewCount(ctx, s.TargetID, tx); err != nil {
			return err
		}

		inserted = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return inserted, nil
}

func (r *fanficRepository) UpdateCommentBody(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error {
	return r.FanficDAO.UpdateComment(ctx, s, tx...)
}

func (r *fanficRepository) DeleteCommentWithAudit(ctx context.Context, s spec.FanficCommentDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.FanficDAO.CollectSingleCommentMediaPaths(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		if err := r.FanficDAO.DeleteComment(ctx, s.CommentDeletion, tx); err != nil {
			return err
		}

		if err := r.audit.Create(ctx, s.Audit, tx); err != nil {
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
