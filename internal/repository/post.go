package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	PostRepository interface {
		dao.PostDAO

		RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error)
		CreateWithDetails(ctx context.Context, post spec.NewPost, tx ...*sql.Tx) (*model.PostRow, error)
		UpdateWithDetails(ctx context.Context, update spec.PostUpdate, tx ...*sql.Tx) error
		DeleteWithSharedContent(ctx context.Context, deletion spec.PostDelete, tx ...*sql.Tx) (*model.SharedContentRef, []string, error)
		UpdateCommentWithDetails(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, deletion spec.PostCommentDelete, tx ...*sql.Tx) ([]string, error)
		CreatePollWithOptions(ctx context.Context, poll spec.NewPoll, tx ...*sql.Tx) (*model.PollRow, error)
	}
)

type postRepository struct {
	db *sql.DB
	dao.PostDAO
	audit AuditLogRepository
}

func NewPostRepo(database *sql.DB, postDAO dao.PostDAO, auditLog AuditLogRepository) PostRepository {
	return &postRepository{db: database, PostDAO: postDAO, audit: auditLog}
}

func (r *postRepository) RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error) {
	var inserted bool

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		ok, err := r.PostDAO.RecordView(ctx, s, tx)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		if err := r.PostDAO.IncrementViewCount(ctx, s.TargetID, tx); err != nil {
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

func (r *postRepository) CreateWithDetails(ctx context.Context, post spec.NewPost, tx ...*sql.Tx) (*model.PostRow, error) {
	var created *model.PostRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.PostDAO.Create(ctx, post, tx)
		if err != nil {
			return err
		}

		if post.Poll == nil {
			return nil
		}

		poll := *post.Poll
		poll.PostID = created.ID

		_, err = r.createPoll(ctx, poll, tx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *postRepository) UpdateWithDetails(ctx context.Context, update spec.PostUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		return r.PostDAO.UpdatePost(ctx, update, tx)
	})
}

func (r *postRepository) DeleteWithSharedContent(ctx context.Context, deletion spec.PostDelete, tx ...*sql.Tx) (*model.SharedContentRef, []string, error) {
	var (
		shared *model.SharedContentRef
		paths  []string
	)

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		contentID, contentType, err := r.PostDAO.GetSharedContentFields(ctx, deletion.ID, tx)
		if err != nil {
			return err
		}

		if contentID != nil && contentType != nil {
			shared = &model.SharedContentRef{ID: *contentID, Type: *contentType}
		}

		mediaPaths, err := r.PostDAO.CollectMediaPaths(ctx, deletion.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.PostDAO.CollectCommentMediaPaths(ctx, deletion.ID, tx)
		if err != nil {
			return err
		}

		paths = append(mediaPaths, commentPaths...)

		if deletion.AsAdmin {
			err = r.PostDAO.DeleteAsAdmin(ctx, deletion.ID, tx)
		} else {
			err = r.PostDAO.Delete(ctx, spec.OwnedDeletion{ID: deletion.ID, UserID: deletion.UserID}, tx)
		}
		if err != nil {
			return err
		}

		if err := r.audit.Create(ctx, deletion.Audit, tx); err != nil {
			return fmt.Errorf("audit post delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return shared, paths, nil
}

func (r *postRepository) UpdateCommentWithDetails(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		return r.PostDAO.UpdateComment(ctx, update, tx)
	})
}

func (r *postRepository) DeleteCommentWithAudit(ctx context.Context, deletion spec.PostCommentDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.PostDAO.CollectSingleCommentMediaPaths(ctx, deletion.CommentID, tx)
		if err != nil {
			return err
		}

		paths = mediaPaths

		if err := r.PostDAO.DeleteComment(ctx, deletion.CommentDeletion, tx); err != nil {
			return err
		}

		if err := r.audit.Create(ctx, deletion.Audit, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *postRepository) CreatePollWithOptions(ctx context.Context, poll spec.NewPoll, tx ...*sql.Tx) (*model.PollRow, error) {
	var created *model.PollRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.createPoll(ctx, poll, tx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *postRepository) createPoll(ctx context.Context, poll spec.NewPoll, tx *sql.Tx) (*model.PollRow, error) {
	created, err := r.PostDAO.CreatePoll(ctx, spec.NewPostPoll{
		PostID:          poll.PostID,
		DurationSeconds: poll.DurationSeconds,
		ExpiresAt:       poll.ExpiresAt,
	}, tx)
	if err != nil {
		return nil, err
	}

	for i, label := range poll.Options {
		option := spec.NewPostPollOption{PollID: created.ID, Label: label, SortOrder: i}

		if err := r.PostDAO.AddPollOption(ctx, option, tx); err != nil {
			return nil, err
		}
	}

	return created, nil
}
