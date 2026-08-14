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
	SecretDAO interface {
		GetFirstSolver(ctx context.Context, secretID string, tx ...*sql.Tx) (*SecretSolver, error)
		GetProgressLeaderboard(ctx context.Context, pieceIDs []string, tx ...*sql.Tx) ([]SecretLeaderboardRow, error)
		GetPieceCountForUser(ctx context.Context, userID uuid.UUID, pieceIDs []string, tx ...*sql.Tx) (int, error)
		GetUserProgressSummary(ctx context.Context, userID uuid.UUID, pieceIDs []string, tx ...*sql.Tx) (*SecretLeaderboardRow, error)
		GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string, tx ...*sql.Tx) ([]SecretSolverRow, error)

		CreateComment(ctx context.Context, secretID string, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		GetComments(ctx context.Context, secretID string, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*CommentRow, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (string, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, spec NewSecretCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID string, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		CountCommentsBySecret(ctx context.Context, secretIDs []string, tx ...*sql.Tx) (map[string]int, error)
		GetCommenterIDs(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	SecretRepository interface {
		SecretDAO

		UpdateCommentBody(ctx context.Context, spec SecretCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, spec SecretCommentDeletion, tx ...*sql.Tx) ([]string, error)
	}

	SecretCommentUpdate struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		Body      string
		AsAdmin   bool
	}

	SecretCommentDeletion struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		AsAdmin   bool
	}

	NewSecretCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	SecretSolver struct {
		UserID      uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
		UnlockedAt  string
	}

	SecretLeaderboardRow struct {
		UserID      uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
		Pieces      int
	}

	SecretSolverRow struct {
		UserID       uuid.UUID
		Username     string
		DisplayName  string
		AvatarURL    string
		Role         string
		SolvedCount  int
		LastSolvedAt string
	}
)

const (
	secretCommentTarget      = "secret_comment"
	secretCommentDeleteLog   = "secret_comment_delete"
	secretCommentDeleteAdmin = "secret_comment_delete_admin"
)

type secretRepository struct {
	db    *sql.DB
	dao   SecretDAO
	audit AuditLogRepository
}

func NewSecretRepo(database *sql.DB, dao SecretDAO, audit AuditLogRepository) SecretRepository {
	return &secretRepository{db: database, dao: dao, audit: audit}
}

func (r *secretRepository) UpdateCommentBody(ctx context.Context, spec SecretCommentUpdate, tx ...*sql.Tx) error {
	if spec.AsAdmin {
		return r.dao.UpdateCommentAsAdmin(ctx, spec.CommentID, spec.Body, tx...)
	}

	return r.dao.UpdateComment(ctx, spec.CommentID, spec.UserID, spec.Body, tx...)
}

func (r *secretRepository) DeleteCommentWithAudit(ctx context.Context, spec SecretCommentDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		action := secretCommentDeleteLog

		mediaPaths, err := r.dao.CollectSingleCommentMediaPaths(ctx, spec.CommentID, tx)
		if err != nil {
			return err
		}

		if spec.AsAdmin {
			if err := r.dao.DeleteCommentAsAdmin(ctx, spec.CommentID, tx); err != nil {
				return err
			}

			action = secretCommentDeleteAdmin
		} else {
			if err := r.dao.DeleteComment(ctx, spec.CommentID, spec.UserID, tx); err != nil {
				return err
			}
		}

		entry := NewAuditEntry{
			ActorID:    spec.UserID,
			Action:     action,
			TargetType: secretCommentTarget,
			TargetID:   spec.CommentID.String(),
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

func (r *secretRepository) GetFirstSolver(ctx context.Context, secretID string, tx ...*sql.Tx) (*SecretSolver, error) {
	return r.dao.GetFirstSolver(ctx, secretID, tx...)
}

func (r *secretRepository) GetProgressLeaderboard(ctx context.Context, pieceIDs []string, tx ...*sql.Tx) ([]SecretLeaderboardRow, error) {
	return r.dao.GetProgressLeaderboard(ctx, pieceIDs, tx...)
}

func (r *secretRepository) GetPieceCountForUser(ctx context.Context, userID uuid.UUID, pieceIDs []string, tx ...*sql.Tx) (int, error) {
	return r.dao.GetPieceCountForUser(ctx, userID, pieceIDs, tx...)
}

func (r *secretRepository) GetUserProgressSummary(ctx context.Context, userID uuid.UUID, pieceIDs []string, tx ...*sql.Tx) (*SecretLeaderboardRow, error) {
	return r.dao.GetUserProgressSummary(ctx, userID, pieceIDs, tx...)
}

func (r *secretRepository) GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string, tx ...*sql.Tx) ([]SecretSolverRow, error) {
	return r.dao.GetSolversLeaderboard(ctx, parentSecretIDs, tx...)
}

func (r *secretRepository) CreateComment(ctx context.Context, secretID string, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, secretID, parentID, userID, body, tx...)
}

func (r *secretRepository) GetComments(ctx context.Context, secretID string, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, secretID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *secretRepository) GetCommentByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.GetCommentByID(ctx, id, tx...)
}

func (r *secretRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *secretRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *secretRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *secretRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *secretRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *secretRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *secretRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *secretRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *secretRepository) AddCommentMedia(ctx context.Context, spec NewSecretCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *secretRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *secretRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *secretRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *secretRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *secretRepository) CollectCommentMediaPaths(ctx context.Context, entityID string, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *secretRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}

func (r *secretRepository) CountCommentsBySecret(ctx context.Context, secretIDs []string, tx ...*sql.Tx) (map[string]int, error) {
	return r.dao.CountCommentsBySecret(ctx, secretIDs, tx...)
}

func (r *secretRepository) GetCommenterIDs(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetCommenterIDs(ctx, secretID, tx...)
}
