package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	MysteryRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error
		AddClue(ctx context.Context, mysteryID uuid.UUID, body string, truthType string, sortOrder int) error
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*MysteryRow, error)
		List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID) ([]MysteryRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MysteryRow, int, error)
		GetClues(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryClue, error)
		DeleteClues(ctx context.Context, mysteryID uuid.UUID) error
		GetAuthorID(ctx context.Context, mysteryID uuid.UUID) (uuid.UUID, error)

		CreateAttempt(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) error
		DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID) error
		GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID) ([]MysteryAttemptRow, error)
		GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error)
		GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error)

		VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int) error

		MarkSolved(ctx context.Context, mysteryID uuid.UUID, winnerID uuid.UUID) error
		Unsolve(ctx context.Context, mysteryID uuid.UUID) error

		GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error)

		CountAttempts(ctx context.Context, mysteryID uuid.UUID) (int, error)
	}

	MysteryRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Title             string
		Body              string
		Difficulty        string
		Solved            bool
		WinnerID          *uuid.UUID
		WinnerUsername    *string
		WinnerDisplayName *string
		WinnerAvatarURL   *string
		WinnerRole        *string
		SolvedAt          *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		AttemptCount      int
		ClueCount         int
		CreatedAt         string
		UpdatedAt         string
	}

	MysteryAttemptRow struct {
		ID                uuid.UUID
		MysteryID         uuid.UUID
		UserID            uuid.UUID
		ParentID          *uuid.UUID
		Body              string
		IsWinner          bool
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		VoteScore         int
		UserVote          int
		CreatedAt         string
	}

	LeaderboardEntry struct {
		UserID         uuid.UUID
		Username       string
		DisplayName    string
		AvatarURL      string
		Role           string
		SolvedCount    int
	}

	mysteryRepository struct {
		db *sql.DB
	}
)

func (r *mysteryRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mysteries (id, user_id, title, body, difficulty) VALUES (?, ?, ?, ?, ?)`,
		id, userID, title, body, difficulty,
	)
	if err != nil {
		return fmt.Errorf("create mystery: %w", err)
	}
	return nil
}

func (r *mysteryRepository) AddClue(ctx context.Context, mysteryID uuid.UUID, body string, truthType string, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_clues (mystery_id, body, truth_type, sort_order) VALUES (?, ?, ?, ?)`,
		mysteryID, body, truthType, sortOrder,
	)
	if err != nil {
		return fmt.Errorf("add clue: %w", err)
	}
	return nil
}

func (r *mysteryRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE mysteries SET title = ?, body = ?, difficulty = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		title, body, difficulty, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update mystery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mystery not found or not owned")
	}
	return nil
}

func (r *mysteryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM mysteries WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete mystery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mystery not found or not owned")
	}
	return nil
}

func (r *mysteryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mysteries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete mystery: %w", err)
	}
	return nil
}

func (r *mysteryRepository) GetByID(ctx context.Context, id uuid.UUID) (*MysteryRow, error) {
	var row MysteryRow
	var solved int
	err := r.db.QueryRowContext(ctx,
		`SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.solved_at, m.created_at, m.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
			(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id),
			(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id)
		FROM mysteries m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		LEFT JOIN users w ON m.winner_id = w.id
		LEFT JOIN user_roles wr ON wr.user_id = w.id
		WHERE m.id = ?`, id,
	).Scan(
		&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &solved, &row.SolvedAt, &row.CreatedAt, &row.UpdatedAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
		&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
		&row.AttemptCount, &row.ClueCount,
	)
	row.Solved = solved != 0
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get mystery: %w", err)
	}
	return &row, nil
}

func (r *mysteryRepository) List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID) ([]MysteryRow, int, error) {
	where := ""
	var args []interface{}

	if solved != nil {
		if *solved {
			where = " WHERE m.solved = 1"
		} else {
			where = " WHERE m.solved = 0"
		}
	}

	exclSQL, exclArgs := ExcludeClause("m.user_id", excludeUserIDs)
	if where == "" && exclSQL != "" {
		where = " WHERE 1=1" + exclSQL
	} else {
		where += exclSQL
	}
	args = append(args, exclArgs...)

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mysteries m`+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count mysteries: %w", err)
	}

	orderBy := "ORDER BY m.created_at DESC"
	if sort == "old" {
		orderBy = "ORDER BY m.created_at ASC"
	}

	query := `SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.solved_at, m.created_at, m.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
		(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id),
		(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id)
	FROM mysteries m
	JOIN users u ON m.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id
	LEFT JOIN users w ON m.winner_id = w.id
	LEFT JOIN user_roles wr ON wr.user_id = w.id` + where + ` ` + orderBy + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list mysteries: %w", err)
	}
	defer rows.Close()

	var result []MysteryRow
	for rows.Next() {
		var row MysteryRow
		var solved int
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &solved, &row.SolvedAt, &row.CreatedAt, &row.UpdatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
			&row.AttemptCount, &row.ClueCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mystery: %w", err)
		}
		row.Solved = solved != 0
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *mysteryRepository) GetClues(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryClue, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, body, truth_type, sort_order FROM mystery_clues WHERE mystery_id = ? ORDER BY sort_order ASC`,
		mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get clues: %w", err)
	}
	defer rows.Close()

	var clues []dto.MysteryClue
	for rows.Next() {
		var c dto.MysteryClue
		if err := rows.Scan(&c.ID, &c.Body, &c.TruthType, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan clue: %w", err)
		}
		clues = append(clues, c)
	}
	return clues, rows.Err()
}

func (r *mysteryRepository) DeleteClues(ctx context.Context, mysteryID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mystery_clues WHERE mystery_id = ?`, mysteryID)
	if err != nil {
		return fmt.Errorf("delete clues: %w", err)
	}
	return nil
}

func (r *mysteryRepository) GetAuthorID(ctx context.Context, mysteryID uuid.UUID) (uuid.UUID, error) {
	var authorID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM mysteries WHERE id = ?`, mysteryID).Scan(&authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get mystery author: %w", err)
	}
	return authorID, nil
}

func (r *mysteryRepository) CreateAttempt(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_attempts (id, mystery_id, user_id, parent_id, body) VALUES (?, ?, ?, ?, ?)`,
		id, mysteryID, userID, parentID, body,
	)
	if err != nil {
		return fmt.Errorf("create attempt: %w", err)
	}
	return nil
}

func (r *mysteryRepository) DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM mystery_attempts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete attempt: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("attempt not found or not owned")
	}
	return nil
}

func (r *mysteryRepository) DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mystery_attempts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete attempt: %w", err)
	}
	return nil
}

func (r *mysteryRepository) GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID) ([]MysteryAttemptRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.mystery_id, a.user_id, a.parent_id, a.body, a.is_winner, a.created_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			COALESCE((SELECT SUM(value) FROM mystery_attempt_votes WHERE attempt_id = a.id), 0),
			COALESCE((SELECT value FROM mystery_attempt_votes WHERE attempt_id = a.id AND user_id = ?), 0)
		FROM mystery_attempts a
		JOIN users u ON a.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE a.mystery_id = ?
		ORDER BY a.created_at ASC`,
		viewerID, mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get attempts: %w", err)
	}
	defer rows.Close()

	var result []MysteryAttemptRow
	for rows.Next() {
		var row MysteryAttemptRow
		var isWinner int
		if err := rows.Scan(
			&row.ID, &row.MysteryID, &row.UserID, &row.ParentID, &row.Body, &isWinner, &row.CreatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.VoteScore, &row.UserVote,
		); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		row.IsWinner = isWinner != 0
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *mysteryRepository) GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error) {
	var authorID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM mystery_attempts WHERE id = ?`, attemptID).Scan(&authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get attempt author: %w", err)
	}
	return authorID, nil
}

func (r *mysteryRepository) GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error) {
	var mysteryID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT mystery_id FROM mystery_attempts WHERE id = ?`, attemptID).Scan(&mysteryID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get attempt mystery: %w", err)
	}
	return mysteryID, nil
}

func (r *mysteryRepository) VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int) error {
	if value == 0 {
		_, err := r.db.ExecContext(ctx,
			`DELETE FROM mystery_attempt_votes WHERE user_id = ? AND attempt_id = ?`,
			userID, attemptID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_attempt_votes (user_id, attempt_id, value) VALUES (?, ?, ?)
		ON CONFLICT(user_id, attempt_id) DO UPDATE SET value = ?`,
		userID, attemptID, value, value,
	)
	if err != nil {
		return fmt.Errorf("vote attempt: %w", err)
	}
	return nil
}

func (r *mysteryRepository) MarkSolved(ctx context.Context, mysteryID uuid.UUID, winnerID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mysteries SET solved = 1, winner_id = ?, solved_at = CURRENT_TIMESTAMP WHERE id = ?`,
		winnerID, mysteryID,
	)
	if err != nil {
		return fmt.Errorf("mark solved: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE mystery_attempts SET is_winner = (CASE WHEN user_id = ? THEN 1 ELSE 0 END) WHERE mystery_id = ?`,
		winnerID, mysteryID,
	)
	if err != nil {
		return fmt.Errorf("set winner attempts: %w", err)
	}
	return nil
}

func (r *mysteryRepository) Unsolve(ctx context.Context, mysteryID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mysteries SET solved = 0, winner_id = NULL, solved_at = NULL WHERE id = ?`,
		mysteryID,
	)
	if err != nil {
		return fmt.Errorf("unsolve: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `UPDATE mystery_attempts SET is_winner = 0 WHERE mystery_id = ?`, mysteryID)
	return err
}

func (r *mysteryRepository) CountAttempts(ctx context.Context, mysteryID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = ?`, mysteryID).Scan(&count)
	return count, err
}

func (r *mysteryRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MysteryRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mysteries WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user mysteries: %w", err)
	}

	query := `SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.solved_at, m.created_at, m.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
		(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id),
		(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id)
	FROM mysteries m
	JOIN users u ON m.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id
	LEFT JOIN users w ON m.winner_id = w.id
	LEFT JOIN user_roles wr ON wr.user_id = w.id
	WHERE m.user_id = ?
	ORDER BY m.created_at DESC
	LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user mysteries: %w", err)
	}
	defer rows.Close()

	var result []MysteryRow
	for rows.Next() {
		var row MysteryRow
		var solved int
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &solved, &row.SolvedAt, &row.CreatedAt, &row.UpdatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
			&row.AttemptCount, &row.ClueCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mystery: %w", err)
		}
		row.Solved = solved != 0
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *mysteryRepository) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			COUNT(m.id) AS solved_count
		FROM mysteries m
		JOIN users u ON m.winner_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE m.solved = 1 AND m.winner_id IS NOT NULL
		GROUP BY u.id
		ORDER BY solved_count DESC, u.display_name ASC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}
	defer rows.Close()

	var result []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.DisplayName, &e.AvatarURL, &e.Role, &e.SolvedCount); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
