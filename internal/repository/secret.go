package repository

import (
	"context"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	SecretRepository interface {
		GetFirstSolver(ctx context.Context, secretID string) (*SecretSolver, error)
		GetProgressLeaderboard(ctx context.Context, pieceIDs []string) ([]SecretLeaderboardRow, error)
		GetPieceCountForUser(ctx context.Context, userID uuid.UUID, pieceIDs []string) (int, error)
		GetUserProgressSummary(ctx context.Context, userID uuid.UUID, pieceIDs []string) (*SecretLeaderboardRow, error)
		GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string) ([]SecretSolverRow, error)

		CreateComment(ctx context.Context, id uuid.UUID, secretID string, parentID *uuid.UUID, userID uuid.UUID, body string) error
		GetComments(ctx context.Context, secretID string, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentByID(ctx context.Context, id uuid.UUID) (*CommentRow, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (string, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		CountCommentsBySecret(ctx context.Context, secretIDs []string) (map[string]int, error)
		GetCommenterIDs(ctx context.Context, secretID string) ([]uuid.UUID, error)
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

type secretRepository struct {
	dao SecretRepository
}

func NewSecretRepo(dao SecretRepository) SecretRepository {
	return &secretRepository{dao: dao}
}

func (r *secretRepository) GetFirstSolver(ctx context.Context, secretID string) (*SecretSolver, error) {
	return r.dao.GetFirstSolver(ctx, secretID)
}

func (r *secretRepository) GetProgressLeaderboard(ctx context.Context, pieceIDs []string) ([]SecretLeaderboardRow, error) {
	return r.dao.GetProgressLeaderboard(ctx, pieceIDs)
}

func (r *secretRepository) GetPieceCountForUser(ctx context.Context, userID uuid.UUID, pieceIDs []string) (int, error) {
	return r.dao.GetPieceCountForUser(ctx, userID, pieceIDs)
}

func (r *secretRepository) GetUserProgressSummary(ctx context.Context, userID uuid.UUID, pieceIDs []string) (*SecretLeaderboardRow, error) {
	return r.dao.GetUserProgressSummary(ctx, userID, pieceIDs)
}

func (r *secretRepository) GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string) ([]SecretSolverRow, error) {
	return r.dao.GetSolversLeaderboard(ctx, parentSecretIDs)
}

func (r *secretRepository) CreateComment(ctx context.Context, id uuid.UUID, secretID string, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, secretID, parentID, userID, body)
}

func (r *secretRepository) GetComments(ctx context.Context, secretID string, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, secretID, viewerID, limit, offset, excludeUserIDs)
}

func (r *secretRepository) GetCommentByID(ctx context.Context, id uuid.UUID) (*CommentRow, error) {
	return r.dao.GetCommentByID(ctx, id)
}

func (r *secretRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *secretRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (string, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *secretRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *secretRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *secretRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *secretRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *secretRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *secretRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *secretRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *secretRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *secretRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *secretRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *secretRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}

func (r *secretRepository) CountCommentsBySecret(ctx context.Context, secretIDs []string) (map[string]int, error) {
	return r.dao.CountCommentsBySecret(ctx, secretIDs)
}

func (r *secretRepository) GetCommenterIDs(ctx context.Context, secretID string) ([]uuid.UUID, error) {
	return r.dao.GetCommenterIDs(ctx, secretID)
}
