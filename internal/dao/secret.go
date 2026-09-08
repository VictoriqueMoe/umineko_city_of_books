package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	SecretDAO interface {
		GetFirstSolver(ctx context.Context, secretID string, tx ...*sql.Tx) (*model.SecretSolver, error)
		GetProgressLeaderboard(ctx context.Context, pieceIDs []string, tx ...*sql.Tx) ([]model.SecretLeaderboardRow, error)
		GetPieceCountForUser(ctx context.Context, s spec.SecretPieceCount, tx ...*sql.Tx) (int, error)
		GetUserProgressSummary(ctx context.Context, s spec.SecretPieceCount, tx ...*sql.Tx) (*model.SecretLeaderboardRow, error)
		GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string, tx ...*sql.Tx) ([]model.SecretSolverRow, error)

		GetComments(ctx context.Context, q spec.CommentQuery[string], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.CommentRow, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (string, error)
		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID string, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		CountCommentsBySecret(ctx context.Context, secretIDs []string, tx ...*sql.Tx) (map[string]int, error)
		GetCommenterIDs(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	secretDAO struct {
		db *sql.DB
		*commentDAO[string]
	}

	secretCommentJoinRow = sqlcgen.GetSecretCommentWithLikesRow
)

func toSecretCommentRow(row secretCommentJoinRow) model.CommentRow {
	return model.CommentRow{
		ID:                row.ID,
		EntityID:          row.EntityID,
		ParentID:          row.ParentID,
		UserID:            row.UserID,
		Body:              row.Body,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         nullTimeToStringPtr(row.UpdatedAt),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
		AuthorRole:        row.AuthorRole,
		LikeCount:         int(row.LikeCount),
		UserLiked:         row.UserLiked,
	}
}

func (r *secretDAO) GetFirstSolver(ctx context.Context, secretID string, tx ...*sql.Tx) (*model.SecretSolver, error) {
	row, err := genQueries(r.db, tx).GetFirstSecretSolver(ctx, secretID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get first solver: %w", err)
	}

	return &model.SecretSolver{
		UserID:      row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl,
		Role:        row.AuthorRole,
		UnlockedAt:  row.UnlockedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (r *secretDAO) GetProgressLeaderboard(ctx context.Context, pieceIDs []string, tx ...*sql.Tx) ([]model.SecretLeaderboardRow, error) {
	if len(pieceIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetSecretProgressLeaderboard(ctx, strings.Join(pieceIDs, ","))
	if err != nil {
		return nil, fmt.Errorf("leaderboard: %w", err)
	}

	var result []model.SecretLeaderboardRow
	for _, row := range rows {
		result = append(result, model.SecretLeaderboardRow{
			UserID:      row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
			Role:        row.AuthorRole,
			Pieces:      int(row.Pieces),
		})
	}

	return result, nil
}

func (r *secretDAO) GetPieceCountForUser(ctx context.Context, s spec.SecretPieceCount, tx ...*sql.Tx) (int, error) {
	if len(s.PieceIDs) == 0 {
		return 0, nil
	}

	count, err := genQueries(r.db, tx).CountSecretPiecesForUser(ctx, sqlcgen.CountSecretPiecesForUserParams{
		UserID:  s.UserID,
		Column2: strings.Join(s.PieceIDs, ","),
	})
	if err != nil {
		return 0, fmt.Errorf("count user pieces: %w", err)
	}

	return int(count), nil
}

func (r *secretDAO) GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string, tx ...*sql.Tx) ([]model.SecretSolverRow, error) {
	if len(parentSecretIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetSecretSolversLeaderboard(ctx, strings.Join(parentSecretIDs, ","))
	if err != nil {
		return nil, fmt.Errorf("solvers leaderboard: %w", err)
	}

	var result []model.SecretSolverRow
	for _, row := range rows {
		result = append(result, model.SecretSolverRow{
			UserID:       row.ID,
			Username:     row.Username,
			DisplayName:  row.DisplayName,
			AvatarURL:    row.AvatarUrl,
			Role:         row.AuthorRole,
			SolvedCount:  int(row.Solved),
			LastSolvedAt: row.LastSolved.UTC().Format(time.RFC3339),
		})
	}

	return result, nil
}

func (r *secretDAO) GetUserProgressSummary(ctx context.Context, s spec.SecretPieceCount, tx ...*sql.Tx) (*model.SecretLeaderboardRow, error) {
	if len(s.PieceIDs) == 0 {
		return nil, nil
	}

	row, err := genQueries(r.db, tx).GetSecretUserProgressSummary(ctx, sqlcgen.GetSecretUserProgressSummaryParams{
		Column1: strings.Join(s.PieceIDs, ","),
		ID:      s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user progress summary: %w", err)
	}

	return &model.SecretLeaderboardRow{
		UserID:      row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl,
		Role:        row.AuthorRole,
		Pieces:      int(row.Pieces),
	}, nil
}

func (r *secretDAO) GetCommentByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.CommentRow, error) {
	row, err := genQueries(r.db, tx).GetSecretCommentWithLikes(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get secret comment by id: %w", err)
	}

	return new(toSecretCommentRow(row)), nil
}

func (r *secretDAO) GetCommenterIDs(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).GetSecretCommenterIDs(ctx, secretID)
	if err != nil {
		return nil, fmt.Errorf("list commenter ids: %w", err)
	}

	return ids, nil
}

func (r *secretDAO) CountCommentsBySecret(ctx context.Context, secretIDs []string, tx ...*sql.Tx) (map[string]int, error) {
	counts := make(map[string]int)
	if len(secretIDs) == 0 {
		return counts, nil
	}

	rows, err := genQueries(r.db, tx).CountSecretCommentsBySecretIDs(ctx, strings.Join(secretIDs, ","))
	if err != nil {
		return nil, fmt.Errorf("count secret comments: %w", err)
	}

	for _, row := range rows {
		counts[row.SecretID] = int(row.CommentCount)
	}

	return counts, nil
}
