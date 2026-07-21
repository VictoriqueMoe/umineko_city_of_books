package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type secretDAO struct {
	db *sql.DB
}

type secretRepository struct {
	repository.SecretRepository
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

func (r *secretDAO) CreateComment(ctx context.Context, id uuid.UUID, secretID string, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO secret_comments (id, secret_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, secretID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create secret comment: %w", err)
	}
	return nil
}

func (r *secretDAO) GetComments(ctx context.Context, secretID string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]repository.SecretCommentRow, error) {
	exclSQL, exclArgs := repository.ExcludeClause("c.user_id", excludeUserIDs, 3)
	args := []interface{}{viewerID, secretID}
	args = append(args, exclArgs...)

	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.secret_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM secret_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM secret_comment_likes WHERE comment_id = c.id AND user_id = $1)
		 FROM secret_comments c
		 JOIN users u ON c.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE c.secret_id = $2`+exclSQL+`
		 ORDER BY c.created_at ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get secret comments: %w", err)
	}
	defer rows.Close()

	var comments []repository.SecretCommentRow
	for rows.Next() {
		var c repository.SecretCommentRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.SecretID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, fmt.Errorf("scan secret comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		c.UpdatedAt = timePtrToString(updatedAt)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *secretDAO) GetCommentByID(ctx context.Context, id uuid.UUID) (*repository.SecretCommentRow, error) {
	var c repository.SecretCommentRow
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
		&c.ID, &c.SecretID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
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

func (r *secretDAO) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM secret_comments WHERE id = $1`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get secret comment author: %w", err)
	}
	return userID, nil
}

func (r *secretDAO) GetCommentSecretID(ctx context.Context, commentID uuid.UUID) (string, error) {
	var secretID string
	err := r.db.QueryRowContext(ctx, `SELECT secret_id FROM secret_comments WHERE id = $1`, commentID).Scan(&secretID)
	if err != nil {
		return "", fmt.Errorf("get secret comment secret id: %w", err)
	}
	return secretID, nil
}

func (r *secretDAO) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE secret_comments SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update secret comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *secretDAO) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE secret_comments SET body = $1, updated_at = NOW() WHERE id = $2`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update secret comment: %w", err)
	}
	return nil
}

func (r *secretDAO) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM secret_comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete secret comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *secretDAO) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM secret_comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete secret comment: %w", err)
	}
	return nil
}

func (r *secretDAO) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO secret_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like secret comment: %w", err)
	}
	return nil
}

func (r *secretDAO) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM secret_comment_likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike secret comment: %w", err)
	}
	return nil
}

func (r *secretDAO) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO secret_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add secret comment media: %w", err)
	}
	return id, nil
}

func (r *secretDAO) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE secret_comment_media SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update secret comment media url: %w", err)
	}
	return nil
}

func (r *secretDAO) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE secret_comment_media SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update secret comment media thumbnail: %w", err)
	}
	return nil
}

func (r *secretDAO) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.CommentMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM secret_comment_media WHERE comment_id = $1 ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get secret comment media: %w", err)
	}
	defer rows.Close()

	var media []model.CommentMediaRow
	for rows.Next() {
		var m model.CommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan secret comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *secretDAO) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.CommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}
	placeholders := "$1"
	args := []interface{}{commentIDs[0]}
	for i := 1; i < len(commentIDs); i++ {
		placeholders += fmt.Sprintf(",$%d", i+1)
		args = append(args, commentIDs[i])
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM secret_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get secret comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.CommentMediaRow)
	for rows.Next() {
		var m model.CommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan secret comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
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
