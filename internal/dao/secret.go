package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type secretDAO struct {
	db *sql.DB
	*commentDAO[string]
}

func secretIDPlaceholders(ids []string, startIndex int) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := fmt.Sprintf("$%d", startIndex)
	args := []interface{}{ids[0]}
	for i := 1; i < len(ids); i++ {
		placeholders += fmt.Sprintf(",$%d", startIndex+i)
		args = append(args, ids[i])
	}
	return placeholders, args
}

func (r *secretDAO) GetFirstSolver(ctx context.Context, secretID string) (*repository.SecretSolver, error) {
	var s repository.SecretSolver
	var unlockedAt time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''), us.unlocked_at
		 FROM user_secrets us
		 JOIN users u ON us.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE us.secret_id = $1
		 ORDER BY us.unlocked_at ASC
		 LIMIT 1`,
		secretID,
	).Scan(&s.UserID, &s.Username, &s.DisplayName, &s.AvatarURL, &s.Role, &unlockedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get first solver: %w", err)
	}
	s.UnlockedAt = unlockedAt.UTC().Format(time.RFC3339)
	return &s, nil
}

func (r *secretDAO) GetProgressLeaderboard(ctx context.Context, pieceIDs []string) ([]repository.SecretLeaderboardRow, error) {
	if len(pieceIDs) == 0 {
		return nil, nil
	}
	placeholders, args := secretIDPlaceholders(pieceIDs, 1)

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''), COUNT(*) AS pieces
		 FROM user_secrets us
		 JOIN users u ON us.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE us.secret_id IN (`+placeholders+`)
		 GROUP BY u.id, u.username, u.display_name, u.avatar_url, r.role
		 ORDER BY pieces DESC, u.display_name ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("leaderboard: %w", err)
	}
	defer rows.Close()

	var result []repository.SecretLeaderboardRow
	for rows.Next() {
		var row repository.SecretLeaderboardRow
		if err := rows.Scan(&row.UserID, &row.Username, &row.DisplayName, &row.AvatarURL, &row.Role, &row.Pieces); err != nil {
			return nil, fmt.Errorf("scan leaderboard row: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *secretDAO) GetPieceCountForUser(ctx context.Context, userID uuid.UUID, pieceIDs []string) (int, error) {
	if len(pieceIDs) == 0 {
		return 0, nil
	}
	placeholders, args := secretIDPlaceholders(pieceIDs, 2)
	args = append([]interface{}{userID}, args...)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_secrets WHERE user_id = $1 AND secret_id IN (`+placeholders+`)`,
		args...,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user pieces: %w", err)
	}
	return count, nil
}

func (r *secretDAO) GetSolversLeaderboard(ctx context.Context, parentSecretIDs []string) ([]repository.SecretSolverRow, error) {
	if len(parentSecretIDs) == 0 {
		return nil, nil
	}
	placeholders, args := secretIDPlaceholders(parentSecretIDs, 1)

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			COUNT(*) AS solved,
			MAX(us.unlocked_at) AS last_solved
		 FROM user_secrets us
		 JOIN users u ON us.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE us.secret_id IN (`+placeholders+`)
		 GROUP BY u.id, u.username, u.display_name, u.avatar_url, r.role
		 ORDER BY solved DESC, last_solved ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("solvers leaderboard: %w", err)
	}
	defer rows.Close()

	var result []repository.SecretSolverRow
	for rows.Next() {
		var row repository.SecretSolverRow
		var lastSolvedAt time.Time
		if err := rows.Scan(&row.UserID, &row.Username, &row.DisplayName, &row.AvatarURL, &row.Role, &row.SolvedCount, &lastSolvedAt); err != nil {
			return nil, fmt.Errorf("scan solver row: %w", err)
		}
		row.LastSolvedAt = lastSolvedAt.UTC().Format(time.RFC3339)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *secretDAO) GetUserProgressSummary(ctx context.Context, userID uuid.UUID, pieceIDs []string) (*repository.SecretLeaderboardRow, error) {
	if len(pieceIDs) == 0 {
		return nil, nil
	}
	placeholders, args := secretIDPlaceholders(pieceIDs, 1)
	userIDPH := fmt.Sprintf("$%d", len(pieceIDs)+1)
	queryArgs := append(args, userID)

	var row repository.SecretLeaderboardRow
	err := r.db.QueryRowContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM user_secrets us WHERE us.user_id = u.id AND us.secret_id IN (`+placeholders+`))
		 FROM users u
		 LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE u.id = `+userIDPH,
		queryArgs...,
	).Scan(&row.UserID, &row.Username, &row.DisplayName, &row.AvatarURL, &row.Role, &row.Pieces)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user progress summary: %w", err)
	}
	return &row, nil
}

func (r *secretDAO) GetCommentByID(ctx context.Context, id uuid.UUID) (*repository.CommentRow, error) {
	var c repository.CommentRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT c.id, c.secret_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM secret_comment_likes WHERE comment_id = c.id),
			FALSE
		 FROM secret_comments c
		 JOIN users u ON c.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE c.id = $1`,
		id,
	).Scan(
		&c.ID, &c.EntityID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
		&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
		&c.LikeCount, &c.UserLiked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get secret comment by id: %w", err)
	}
	c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	c.UpdatedAt = timePtrToString(updatedAt)
	return &c, nil
}

func (r *secretDAO) GetCommenterIDs(ctx context.Context, secretID string) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT user_id FROM secret_comments WHERE secret_id = $1`,
		secretID,
	)
	if err != nil {
		return nil, fmt.Errorf("list commenter ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan commenter id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *secretDAO) CountCommentsBySecret(ctx context.Context, secretIDs []string) (map[string]int, error) {
	result := make(map[string]int)
	if len(secretIDs) == 0 {
		return result, nil
	}
	placeholders, args := secretIDPlaceholders(secretIDs, 1)
	rows, err := r.db.QueryContext(ctx,
		`SELECT secret_id, COUNT(*) FROM secret_comments WHERE secret_id IN (`+placeholders+`) GROUP BY secret_id`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("count secret comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, fmt.Errorf("scan secret comment count: %w", err)
		}
		result[id] = count
	}
	return result, rows.Err()
}
