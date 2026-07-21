package repository

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

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
		GetComments(ctx context.Context, secretID string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]SecretCommentRow, error)
		GetCommentByID(ctx context.Context, id uuid.UUID) (*SecretCommentRow, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentSecretID(ctx context.Context, commentID uuid.UUID) (string, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.CommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.CommentMediaRow, error)

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

	SecretCommentRow struct {
		ID                uuid.UUID
		SecretID          string
		ParentID          *uuid.UUID
		UserID            uuid.UUID
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		UserLiked         bool
	}
)

func (r *SecretCommentRow) ToResponse(media []model.CommentMediaRow) dto.SecretCommentResponse {
	return dto.SecretCommentResponse{
		ID:       r.ID,
		ParentID: r.ParentID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Body:      r.Body,
		Media:     model.CommentMediaRowsToResponse(media),
		LikeCount: r.LikeCount,
		UserLiked: r.UserLiked,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
