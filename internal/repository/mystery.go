package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	MysteryRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool) error
		AddClue(ctx context.Context, mysteryID uuid.UUID, body string, truthType string, sortOrder int, playerID *uuid.UUID) error
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*MysteryRow, error)
		List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID) ([]MysteryRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MysteryRow, int, error)
		GetClues(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryClue, error)
		DeleteClues(ctx context.Context, mysteryID uuid.UUID) error
		DeleteClue(ctx context.Context, clueID int) error
		UpdateClue(ctx context.Context, clueID int, body string) error
		GetAuthorID(ctx context.Context, mysteryID uuid.UUID) (uuid.UUID, error)

		CreateAttempt(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) error
		DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID) error
		GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID) ([]MysteryAttemptRow, error)
		GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error)
		GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error)

		VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int) error

		MarkSolved(ctx context.Context, mysteryID uuid.UUID, attemptID uuid.UUID) error
		IsSolved(ctx context.Context, mysteryID uuid.UUID) (bool, error)
		IsPaused(ctx context.Context, mysteryID uuid.UUID) (bool, error)
		SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool) error
		SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool) error

		GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error)
		GetTopDetectiveIDs(ctx context.Context) ([]string, error)
		GetGMLeaderboard(ctx context.Context, limit int) ([]GMLeaderboardEntry, error)
		GetTopGMIDs(ctx context.Context) ([]string, error)

		CountAttempts(ctx context.Context, mysteryID uuid.UUID) (int, error)
		CountClues(ctx context.Context, mysteryID uuid.UUID) (int, error)
		GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID) ([]uuid.UUID, error)

		CreateComment(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]MysteryCommentRow, error)
		GetCommentMysteryID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]MysteryCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]MysteryCommentMediaRow, error)

		AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int) (int64, error)
		DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID) error
		GetAttachments(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryAttachment, error)
	}

	MysteryRow struct {
		ID                    uuid.UUID
		UserID                uuid.UUID
		Title                 string
		Body                  string
		Difficulty            string
		Solved                bool
		Paused                bool
		GmAway                bool
		FreeForAll            bool
		WinnerID              *uuid.UUID
		WinnerUsername        *string
		WinnerDisplayName     *string
		WinnerAvatarURL       *string
		WinnerRole            *string
		SolvedAt              *string
		PausedAt              *string
		PausedDurationSeconds int
		AuthorUsername        string
		AuthorDisplayName     string
		AuthorAvatarURL       string
		AuthorRole            string
		AttemptCount          int
		ClueCount             int
		CreatedAt             string
		UpdatedAt             string
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

	MysteryCommentRow struct {
		ID                uuid.UUID
		MysteryID         uuid.UUID
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

	MysteryCommentMediaRow struct {
		ID           int
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	LeaderboardEntry struct {
		UserID          uuid.UUID
		Username        string
		DisplayName     string
		AvatarURL       string
		Role            string
		Score           int
		EasySolved      int
		MediumSolved    int
		HardSolved      int
		NightmareSolved int
		ScoreAdjustment int
	}

	GMLeaderboardEntry struct {
		UserID       uuid.UUID
		Username     string
		DisplayName  string
		AvatarURL    string
		Role         string
		Score        int
		MysteryCount int
		PlayerCount  int
	}

	mysteryRepository struct {
		db *sql.DB
	}
)

func (r *MysteryRow) ToResponse() dto.MysteryResponse {
	resp := dto.MysteryResponse{
		ID:                    r.ID,
		Title:                 r.Title,
		Body:                  r.Body,
		Difficulty:            r.Difficulty,
		Solved:                r.Solved,
		Paused:                r.Paused,
		GmAway:                r.GmAway,
		FreeForAll:            r.FreeForAll,
		SolvedAt:              r.SolvedAt,
		PausedAt:              r.PausedAt,
		PausedDurationSeconds: r.PausedDurationSeconds,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		AttemptCount: r.AttemptCount,
		ClueCount:    r.ClueCount,
		CreatedAt:    r.CreatedAt,
	}
	if r.WinnerID != nil && r.WinnerUsername != nil {
		resp.Winner = &dto.UserResponse{
			ID:          *r.WinnerID,
			Username:    *r.WinnerUsername,
			DisplayName: *r.WinnerDisplayName,
			AvatarURL:   *r.WinnerAvatarURL,
			Role:        role.Role(*r.WinnerRole),
		}
	}
	return resp
}

func (r *mysteryRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mysteries (id, user_id, title, body, difficulty, free_for_all) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, title, body, difficulty, freeForAll,
	)
	if err != nil {
		return fmt.Errorf("create mystery: %w", err)
	}
	return nil
}

func (r *mysteryRepository) AddClue(ctx context.Context, mysteryID uuid.UUID, body string, truthType string, sortOrder int, playerID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_clues (mystery_id, body, truth_type, sort_order, player_id) VALUES (?, ?, ?, ?, ?)`,
		mysteryID, body, truthType, sortOrder, playerID,
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

func (r *mysteryRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mysteries SET title = ?, body = ?, difficulty = ?, free_for_all = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, body, difficulty, freeForAll, id,
	)
	if err != nil {
		return fmt.Errorf("update mystery as admin: %w", err)
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
	var solved, paused, freeForAll int
	var gmAway int
	err := r.db.QueryRowContext(ctx,
		`SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
			(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id AND parent_id IS NULL AND user_id != m.user_id),
			(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id)
		FROM mysteries m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		LEFT JOIN users w ON m.winner_id = w.id
		LEFT JOIN user_roles wr ON wr.user_id = w.id
		WHERE m.id = ?`, id,
	).Scan(
		&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &solved, &paused, &gmAway, &freeForAll, &row.SolvedAt, &row.PausedAt, &row.PausedDurationSeconds, &row.CreatedAt, &row.UpdatedAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
		&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
		&row.AttemptCount, &row.ClueCount,
	)
	row.Solved = solved != 0
	row.Paused = paused != 0
	row.GmAway = gmAway != 0
	row.FreeForAll = freeForAll != 0
	if errors.Is(err, sql.ErrNoRows) {
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

	query := `SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
		(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id AND parent_id IS NULL AND user_id != m.user_id),
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
		var solved, paused, gmAway, freeForAll int
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &solved, &paused, &gmAway, &freeForAll, &row.SolvedAt, &row.PausedAt, &row.PausedDurationSeconds, &row.CreatedAt, &row.UpdatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
			&row.AttemptCount, &row.ClueCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mystery: %w", err)
		}
		row.Solved = solved != 0
		row.Paused = paused != 0
		row.GmAway = gmAway != 0
		row.FreeForAll = freeForAll != 0
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *mysteryRepository) GetClues(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryClue, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, body, truth_type, sort_order, player_id FROM mystery_clues WHERE mystery_id = ? ORDER BY sort_order ASC`,
		mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get clues: %w", err)
	}
	defer rows.Close()

	var clues []dto.MysteryClue
	for rows.Next() {
		var c dto.MysteryClue
		if err := rows.Scan(&c.ID, &c.Body, &c.TruthType, &c.SortOrder, &c.PlayerID); err != nil {
			return nil, fmt.Errorf("scan clue: %w", err)
		}
		clues = append(clues, c)
	}
	return clues, rows.Err()
}

func (r *mysteryRepository) DeleteClues(ctx context.Context, mysteryID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mystery_clues WHERE mystery_id = ? AND player_id IS NULL`, mysteryID)
	if err != nil {
		return fmt.Errorf("delete clues: %w", err)
	}
	return nil
}

func (r *mysteryRepository) DeleteClue(ctx context.Context, clueID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mystery_clues WHERE id = ?`, clueID)
	if err != nil {
		return fmt.Errorf("delete clue: %w", err)
	}
	return nil
}

func (r *mysteryRepository) UpdateClue(ctx context.Context, clueID int, body string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mystery_clues SET body = ? WHERE id = ?`, body, clueID)
	if err != nil {
		return fmt.Errorf("update clue: %w", err)
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

func (r *mysteryRepository) MarkSolved(ctx context.Context, mysteryID uuid.UUID, attemptID uuid.UUID) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var attemptUserID uuid.UUID
		var attemptMysteryID uuid.UUID
		if err := tx.QueryRowContext(ctx,
			`SELECT user_id, mystery_id FROM mystery_attempts WHERE id = ?`, attemptID,
		).Scan(&attemptUserID, &attemptMysteryID); err != nil {
			return fmt.Errorf("get attempt for winner: %w", err)
		}
		if attemptMysteryID != mysteryID {
			return fmt.Errorf("attempt does not belong to mystery")
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE mysteries SET solved = 1, winner_id = ?, solved_at = CURRENT_TIMESTAMP WHERE id = ?`,
			attemptUserID, mysteryID,
		); err != nil {
			return fmt.Errorf("mark solved: %w", err)
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE mystery_attempts SET is_winner = 0 WHERE mystery_id = ?`, mysteryID,
		); err != nil {
			return fmt.Errorf("clear previous winner attempts: %w", err)
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE mystery_attempts SET is_winner = 1 WHERE id = ?`, attemptID,
		); err != nil {
			return fmt.Errorf("set winning attempt: %w", err)
		}
		return nil
	})
}

func (r *mysteryRepository) IsSolved(ctx context.Context, mysteryID uuid.UUID) (bool, error) {
	var solved int
	err := r.db.QueryRowContext(ctx, `SELECT solved FROM mysteries WHERE id = ?`, mysteryID).Scan(&solved)
	if err != nil {
		return false, fmt.Errorf("check mystery solved: %w", err)
	}
	return solved != 0, nil
}

func (r *mysteryRepository) IsPaused(ctx context.Context, mysteryID uuid.UUID) (bool, error) {
	var paused int
	err := r.db.QueryRowContext(ctx, `SELECT paused FROM mysteries WHERE id = ?`, mysteryID).Scan(&paused)
	if err != nil {
		return false, fmt.Errorf("check mystery paused: %w", err)
	}
	return paused != 0, nil
}

func (r *mysteryRepository) SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool) error {
	if paused {
		_, err := r.db.ExecContext(ctx,
			`UPDATE mysteries
			 SET paused = 1,
			     paused_at = CASE WHEN paused = 1 THEN paused_at ELSE CURRENT_TIMESTAMP END
			 WHERE id = ?`, mysteryID)
		if err != nil {
			return fmt.Errorf("set mystery paused: %w", err)
		}
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE mysteries
		 SET paused = 0,
		     paused_duration_seconds = paused_duration_seconds + CASE
		         WHEN paused_at IS NOT NULL
		         THEN CAST((julianday(CURRENT_TIMESTAMP) - julianday(paused_at)) * 86400 AS INTEGER)
		         ELSE 0
		     END,
		     paused_at = NULL
		 WHERE id = ?`, mysteryID)
	if err != nil {
		return fmt.Errorf("set mystery unpaused: %w", err)
	}
	return nil
}

func (r *mysteryRepository) SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool) error {
	val := 0
	if away {
		val = 1
	}
	_, err := r.db.ExecContext(ctx, `UPDATE mysteries SET gm_away = ? WHERE id = ?`, val, mysteryID)
	if err != nil {
		return fmt.Errorf("set mystery gm_away: %w", err)
	}
	return nil
}

func (r *mysteryRepository) CountAttempts(ctx context.Context, mysteryID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = ?`, mysteryID).Scan(&count)
	return count, err
}

func (r *mysteryRepository) CountClues(ctx context.Context, mysteryID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = ?`, mysteryID).Scan(&count)
	return count, err
}

func (r *mysteryRepository) GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ma.user_id FROM mystery_attempts ma
		JOIN mysteries m ON m.id = ma.mystery_id
		WHERE ma.mystery_id = ? AND ma.user_id != m.user_id`, mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get player ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan player id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *mysteryRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MysteryRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mysteries WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user mysteries: %w", err)
	}

	query := `SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
		(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id AND parent_id IS NULL AND user_id != m.user_id),
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
		var solved, paused, gmAway, freeForAll int
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &solved, &paused, &gmAway, &freeForAll, &row.SolvedAt, &row.PausedAt, &row.PausedDurationSeconds, &row.CreatedAt, &row.UpdatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
			&row.AttemptCount, &row.ClueCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mystery: %w", err)
		}
		row.Solved = solved != 0
		row.Paused = paused != 0
		row.GmAway = gmAway != 0
		row.FreeForAll = freeForAll != 0
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *mysteryRepository) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, display_name, avatar_url, role, score, easy_solved, medium_solved, hard_solved, nightmare_solved, score_adjustment FROM (
			SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '') AS role,
				COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
					CASE WHEN m.difficulty = 'easy' THEN 2
					     WHEN m.difficulty = 'medium' THEN 4
					     WHEN m.difficulty = 'hard' THEN 6
					     WHEN m.difficulty = 'nightmare' THEN 8
					     ELSE 4 END
				ELSE 0 END), 0) + u.mystery_score_adjustment AS score,
				COALESCE(SUM(CASE WHEN m.difficulty = 'easy' THEN 1 ELSE 0 END), 0) AS easy_solved,
				COALESCE(SUM(CASE WHEN m.difficulty = 'medium' THEN 1 ELSE 0 END), 0) AS medium_solved,
				COALESCE(SUM(CASE WHEN m.difficulty = 'hard' THEN 1 ELSE 0 END), 0) AS hard_solved,
				COALESCE(SUM(CASE WHEN m.difficulty = 'nightmare' THEN 1 ELSE 0 END), 0) AS nightmare_solved,
				u.mystery_score_adjustment AS score_adjustment
			FROM users u
			LEFT JOIN mysteries m ON m.winner_id = u.id AND m.solved = 1
			LEFT JOIN user_roles r ON r.user_id = u.id
			GROUP BY u.id
			HAVING score > 0
		)
		ORDER BY score DESC, display_name ASC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}
	defer rows.Close()

	var result []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.DisplayName, &e.AvatarURL, &e.Role,
			&e.Score, &e.EasySolved, &e.MediumSolved, &e.HardSolved, &e.NightmareSolved, &e.ScoreAdjustment); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *mysteryRepository) GetTopDetectiveIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`WITH ranked AS (
			SELECT u.id AS user_id,
				COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
					CASE WHEN m.difficulty = 'easy' THEN 2
					     WHEN m.difficulty = 'medium' THEN 4
					     WHEN m.difficulty = 'hard' THEN 6
					     WHEN m.difficulty = 'nightmare' THEN 8
					     ELSE 4 END
				ELSE 0 END), 0) + u.mystery_score_adjustment AS score
			FROM users u
			LEFT JOIN mysteries m ON m.winner_id = u.id AND m.solved = 1
			GROUP BY u.id
			HAVING score > 0
		)
		SELECT user_id FROM ranked
		WHERE score = (SELECT MAX(score) FROM ranked)`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *mysteryRepository) GetGMLeaderboard(ctx context.Context, limit int) ([]GMLeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, username, display_name, avatar_url, role, score, mystery_count, player_count FROM (
			SELECT u.id AS user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '') AS role,
				SUM(
					CASE m.difficulty
						WHEN 'easy' THEN 2
						WHEN 'medium' THEN 4
						WHEN 'hard' THEN 6
						WHEN 'nightmare' THEN 8
						ELSE 4
					END
					+ MIN((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
				) + u.gm_score_adjustment AS score,
				COUNT(m.id) AS mystery_count,
				SUM(MIN((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)) AS player_count
			FROM mysteries m
			JOIN users u ON m.user_id = u.id
			LEFT JOIN user_roles r ON r.user_id = u.id
			WHERE m.solved = 1
			GROUP BY u.id
			HAVING score > 0
		)
		ORDER BY score DESC, display_name ASC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get gm leaderboard: %w", err)
	}
	defer rows.Close()

	var result []GMLeaderboardEntry
	for rows.Next() {
		var e GMLeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.DisplayName, &e.AvatarURL, &e.Role,
			&e.Score, &e.MysteryCount, &e.PlayerCount); err != nil {
			return nil, fmt.Errorf("scan gm leaderboard entry: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *mysteryRepository) GetTopGMIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`WITH ranked AS (
			SELECT u.id AS user_id,
				SUM(
					CASE m.difficulty
						WHEN 'easy' THEN 2
						WHEN 'medium' THEN 4
						WHEN 'hard' THEN 6
						WHEN 'nightmare' THEN 8
						ELSE 4
					END
					+ MIN((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
				) + u.gm_score_adjustment AS score
			FROM mysteries m
			JOIN users u ON m.user_id = u.id
			WHERE m.solved = 1
			GROUP BY u.id
			HAVING score > 0
		)
		SELECT user_id FROM ranked
		WHERE score = (SELECT MAX(score) FROM ranked)`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *MysteryCommentRow) ToResponse(media []MysteryCommentMediaRow) dto.MysteryCommentResponse {
	mediaList := make([]dto.PostMediaResponse, len(media))
	for i, m := range media {
		mediaList[i] = dto.PostMediaResponse{
			ID:           m.ID,
			MediaURL:     m.MediaURL,
			MediaType:    m.MediaType,
			ThumbnailURL: m.ThumbnailURL,
			SortOrder:    m.SortOrder,
		}
	}
	return dto.MysteryCommentResponse{
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
		Media:     mediaList,
		LikeCount: r.LikeCount,
		UserLiked: r.UserLiked,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (r *mysteryRepository) CreateComment(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_comments (id, mystery_id, parent_id, user_id, body) VALUES (?, ?, ?, ?, ?)`,
		id, mysteryID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create mystery comment: %w", err)
	}
	return nil
}

func (r *mysteryRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE mystery_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update mystery comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *mysteryRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mystery_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update mystery comment: %w", err)
	}
	return nil
}

func (r *mysteryRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM mystery_comments WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete mystery comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *mysteryRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mystery_comments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete mystery comment: %w", err)
	}
	return nil
}

func (r *mysteryRepository) GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]MysteryCommentRow, error) {
	exclSQL, exclArgs := ExcludeClause("c.user_id", excludeUserIDs)
	args := []interface{}{viewerID, mysteryID}
	args = append(args, exclArgs...)

	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.mystery_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM mystery_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM mystery_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM mystery_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE c.mystery_id = ?`+exclSQL+`
		ORDER BY c.created_at ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get mystery comments: %w", err)
	}
	defer rows.Close()

	var comments []MysteryCommentRow
	for rows.Next() {
		var c MysteryCommentRow
		if err := rows.Scan(
			&c.ID, &c.MysteryID, &c.ParentID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, fmt.Errorf("scan mystery comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *mysteryRepository) GetCommentMysteryID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var mysteryID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT mystery_id FROM mystery_comments WHERE id = ?`, commentID).Scan(&mysteryID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get mystery comment mystery id: %w", err)
	}
	return mysteryID, nil
}

func (r *mysteryRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM mystery_comments WHERE id = ?`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get mystery comment author: %w", err)
	}
	return userID, nil
}

func (r *mysteryRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO mystery_comment_likes (user_id, comment_id) VALUES (?, ?)`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like mystery comment: %w", err)
	}
	return nil
}

func (r *mysteryRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM mystery_comment_likes WHERE user_id = ? AND comment_id = ?`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike mystery comment: %w", err)
	}
	return nil
}

func (r *mysteryRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add mystery comment media: %w", err)
	}
	return res.LastInsertId()
}

func (r *mysteryRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mystery_comment_media SET media_url = ? WHERE id = ?`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update mystery comment media url: %w", err)
	}
	return nil
}

func (r *mysteryRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mystery_comment_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update mystery comment media thumbnail: %w", err)
	}
	return nil
}

func (r *mysteryRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]MysteryCommentMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM mystery_comment_media WHERE comment_id = ? ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get mystery comment media: %w", err)
	}
	defer rows.Close()

	var media []MysteryCommentMediaRow
	for rows.Next() {
		var m MysteryCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan mystery comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *mysteryRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]MysteryCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{commentIDs[0]}
	for _, id := range commentIDs[1:] {
		placeholders += ",?"
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM mystery_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get mystery comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]MysteryCommentMediaRow)
	for rows.Next() {
		var m MysteryCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan mystery comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}

func (r *mysteryRepository) AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO mystery_attachments (mystery_id, file_url, file_name, file_size) VALUES (?, ?, ?, ?)`,
		mysteryID, fileURL, fileName, fileSize,
	)
	if err != nil {
		return 0, fmt.Errorf("add attachment: %w", err)
	}
	return res.LastInsertId()
}

func (r *mysteryRepository) DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM mystery_attachments WHERE id = ? AND mystery_id = ?`,
		id, mysteryID,
	)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("attachment not found")
	}
	return nil
}

func (r *mysteryRepository) GetAttachments(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryAttachment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, file_url, file_name, file_size FROM mystery_attachments WHERE mystery_id = ? ORDER BY created_at`,
		mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []dto.MysteryAttachment
	for rows.Next() {
		var a dto.MysteryAttachment
		if err := rows.Scan(&a.ID, &a.FileURL, &a.FileName, &a.FileSize); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}
