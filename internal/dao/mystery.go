package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
)

type (
	mysteryDAO struct {
		db *sql.DB
		*commentDAO[uuid.UUID]
		*mediaDAO
	}
)

func mysteryNullTimePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	return new(t.Time.UTC().Format(time.RFC3339))
}

func (r *mysteryDAO) Create(ctx context.Context, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool, knox dto.KnoxContract, tx ...*sql.Tx) (*repository.MysteryRow, error) {
	var row repository.MysteryRow
	var solvedAt, pausedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH m AS (
			INSERT INTO mysteries (user_id, title, body, difficulty, free_for_all, keep_open_after_solve,
				knox_culprit_named_early, knox_no_supernatural, knox_passages_declared, knox_no_unknown_poison, knox_no_outsider,
				knox_no_lucky_accident, knox_detective_not_culprit, knox_clues_shown, knox_narrator_hides_nothing, knox_no_unannounced_twins, knox_contract_published)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, TRUE)
			RETURNING *
		)
		SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve, m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider, m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins, m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			NULL::uuid, NULL::text, NULL::text, NULL::text, NULL::text,
			0, 0, 0
		FROM m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id`,
		userID, title, body, difficulty, freeForAll, keepOpenAfterSolve,
		knox.CulpritNamedEarly, knox.NoSupernatural, knox.PassagesDeclared, knox.NoUnknownPoison, knox.NoOutsider,
		knox.NoLuckyAccident, knox.DetectiveNotCulprit, knox.CluesShown, knox.NarratorHidesNothing, knox.NoUnannouncedTwins,
	).Scan(
		&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &row.Solved, &row.Paused, &row.GmAway, &row.FreeForAll, &row.KeepOpenAfterSolve, &row.Knox.CulpritNamedEarly, &row.Knox.NoSupernatural, &row.Knox.PassagesDeclared, &row.Knox.NoUnknownPoison, &row.Knox.NoOutsider, &row.Knox.NoLuckyAccident, &row.Knox.DetectiveNotCulprit, &row.Knox.CluesShown, &row.Knox.NarratorHidesNothing, &row.Knox.NoUnannouncedTwins, &row.KnoxPublished, &solvedAt, &pausedAt, &row.PausedDurationSeconds, &createdAt, &updatedAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
		&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
		&row.AttemptCount, &row.ClueCount, &row.SolverCount,
	)
	if err != nil {
		return nil, fmt.Errorf("create mystery: %w", err)
	}

	row.SolvedAt = mysteryNullTimePtr(solvedAt)
	row.PausedAt = mysteryNullTimePtr(pausedAt)
	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	row.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	return &row, nil
}

func (r *mysteryDAO) AddClue(ctx context.Context, mysteryID uuid.UUID, spec repository.NewClue, tx ...*sql.Tx) (*dto.MysteryClue, error) {
	var clue dto.MysteryClue
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO mystery_clues (mystery_id, body, truth_type, sort_order, player_id) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, body, truth_type, sort_order, player_id`,
		mysteryID, spec.Body, spec.TruthType, spec.SortOrder, spec.PlayerID,
	).Scan(&clue.ID, &clue.Body, &clue.TruthType, &clue.SortOrder, &clue.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("add clue: %w", err)
	}

	return &clue, nil
}

func (r *mysteryDAO) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE mysteries SET title = $1, body = $2, difficulty = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 AND user_id = $5`,
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

func (r *mysteryDAO) UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool, knox dto.KnoxContract, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE mysteries SET title = $1, body = $2, difficulty = $3, free_for_all = $4, keep_open_after_solve = $5,
			knox_culprit_named_early = $6, knox_no_supernatural = $7, knox_passages_declared = $8, knox_no_unknown_poison = $9, knox_no_outsider = $10,
			knox_no_lucky_accident = $11, knox_detective_not_culprit = $12, knox_clues_shown = $13, knox_narrator_hides_nothing = $14, knox_no_unannounced_twins = $15, knox_contract_published = TRUE,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $16`,
		title, body, difficulty, freeForAll, keepOpenAfterSolve,
		knox.CulpritNamedEarly, knox.NoSupernatural, knox.PassagesDeclared, knox.NoUnknownPoison, knox.NoOutsider,
		knox.NoLuckyAccident, knox.DetectiveNotCulprit, knox.CluesShown, knox.NarratorHidesNothing, knox.NoUnannouncedTwins,
		id,
	)
	if err != nil {
		return fmt.Errorf("update mystery as admin: %w", err)
	}
	return nil
}

func (r *mysteryDAO) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM mysteries WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete mystery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mystery not found or not owned")
	}
	return nil
}

func (r *mysteryDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM mysteries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete mystery: %w", err)
	}
	return nil
}

func (r *mysteryDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*repository.MysteryRow, error) {
	var row repository.MysteryRow
	var solvedAt, pausedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve, m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider, m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins, m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
			(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id AND parent_id IS NULL AND user_id != m.user_id),
			(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id),
			(SELECT COUNT(DISTINCT user_id) FROM mystery_attempts WHERE mystery_id = m.id AND is_winner = TRUE)
		FROM mysteries m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		LEFT JOIN users w ON m.winner_id = w.id
		LEFT JOIN user_roles wr ON wr.user_id = w.id
		WHERE m.id = $1`, id,
	).Scan(
		&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &row.Solved, &row.Paused, &row.GmAway, &row.FreeForAll, &row.KeepOpenAfterSolve, &row.Knox.CulpritNamedEarly, &row.Knox.NoSupernatural, &row.Knox.PassagesDeclared, &row.Knox.NoUnknownPoison, &row.Knox.NoOutsider, &row.Knox.NoLuckyAccident, &row.Knox.DetectiveNotCulprit, &row.Knox.CluesShown, &row.Knox.NarratorHidesNothing, &row.Knox.NoUnannouncedTwins, &row.KnoxPublished, &solvedAt, &pausedAt, &row.PausedDurationSeconds, &createdAt, &updatedAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
		&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
		&row.AttemptCount, &row.ClueCount, &row.SolverCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get mystery: %w", err)
	}
	row.SolvedAt = mysteryNullTimePtr(solvedAt)
	row.PausedAt = mysteryNullTimePtr(pausedAt)
	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	row.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return &row, nil
}

func (r *mysteryDAO) List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]repository.MysteryRow, int, error) {
	where := ""
	var args []any

	if solved != nil {
		if *solved {
			where = " WHERE m.solved = TRUE"
		} else {
			where = " WHERE m.solved = FALSE"
		}
	}

	exclSQL, exclArgs := ExcludeClause("m.user_id", excludeUserIDs, len(args)+1)
	if where == "" && exclSQL != "" {
		where = " WHERE 1=1" + exclSQL
	} else {
		where += exclSQL
	}
	args = append(args, exclArgs...)

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM mysteries m`+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count mysteries: %w", err)
	}

	orderBy := "ORDER BY m.created_at DESC"
	if sort == "old" {
		orderBy = "ORDER BY m.created_at ASC"
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	query := `SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve, m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider, m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins, m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
		(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id AND parent_id IS NULL AND user_id != m.user_id),
		(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id),
		(SELECT COUNT(DISTINCT user_id) FROM mystery_attempts WHERE mystery_id = m.id AND is_winner = TRUE)
	FROM mysteries m
	JOIN users u ON m.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id
	LEFT JOIN users w ON m.winner_id = w.id
	LEFT JOIN user_roles wr ON wr.user_id = w.id` + where + ` ` + orderBy + ` LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	args = append(args, limit, offset)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list mysteries: %w", err)
	}
	defer rows.Close()

	var result []repository.MysteryRow
	for rows.Next() {
		var row repository.MysteryRow
		var solvedAt, pausedAt sql.NullTime
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &row.Solved, &row.Paused, &row.GmAway, &row.FreeForAll, &row.KeepOpenAfterSolve, &row.Knox.CulpritNamedEarly, &row.Knox.NoSupernatural, &row.Knox.PassagesDeclared, &row.Knox.NoUnknownPoison, &row.Knox.NoOutsider, &row.Knox.NoLuckyAccident, &row.Knox.DetectiveNotCulprit, &row.Knox.CluesShown, &row.Knox.NarratorHidesNothing, &row.Knox.NoUnannouncedTwins, &row.KnoxPublished, &solvedAt, &pausedAt, &row.PausedDurationSeconds, &createdAt, &updatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
			&row.AttemptCount, &row.ClueCount, &row.SolverCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mystery: %w", err)
		}
		row.SolvedAt = mysteryNullTimePtr(solvedAt)
		row.PausedAt = mysteryNullTimePtr(pausedAt)
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *mysteryDAO) GetClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryClue, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, body, truth_type, sort_order, player_id FROM mystery_clues WHERE mystery_id = $1 ORDER BY sort_order ASC`,
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

func (r *mysteryDAO) DeleteClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM mystery_clues WHERE mystery_id = $1 AND player_id IS NULL`, mysteryID)
	if err != nil {
		return fmt.Errorf("delete clues: %w", err)
	}
	return nil
}

func (r *mysteryDAO) DeleteClue(ctx context.Context, clueID int, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM mystery_clues WHERE id = $1`, clueID)
	if err != nil {
		return fmt.Errorf("delete clue: %w", err)
	}
	return nil
}

func (r *mysteryDAO) UpdateClue(ctx context.Context, clueID int, body string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `UPDATE mystery_clues SET body = $1 WHERE id = $2`, body, clueID)
	if err != nil {
		return fmt.Errorf("update clue: %w", err)
	}
	return nil
}

func (r *mysteryDAO) GetAuthorID(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var authorID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT user_id FROM mysteries WHERE id = $1`, mysteryID).Scan(&authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get mystery author: %w", err)
	}
	return authorID, nil
}

func (r *mysteryDAO) CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string, tx ...*sql.Tx) (*repository.MysteryAttemptRow, error) {
	var row repository.MysteryAttemptRow
	var createdAt time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH a AS (
			INSERT INTO mystery_attempts (mystery_id, user_id, parent_id, body) VALUES ($1, $2, $3, $4)
			RETURNING id, mystery_id, user_id, parent_id, body, is_winner, created_at
		)
		SELECT a.id, a.mystery_id, a.user_id, a.parent_id, a.body, a.is_winner, a.created_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			0, 0
		FROM a
		JOIN users u ON a.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id`,
		mysteryID, userID, parentID, body,
	).Scan(
		&row.ID, &row.MysteryID, &row.UserID, &row.ParentID, &row.Body, &row.IsWinner, &createdAt,
		&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
		&row.VoteScore, &row.UserVote,
	)
	if err != nil {
		return nil, fmt.Errorf("create attempt: %w", err)
	}

	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)

	return &row, nil
}

func (r *mysteryDAO) DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM mystery_attempts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete attempt: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("attempt not found or not owned")
	}
	return nil
}

func (r *mysteryDAO) DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM mystery_attempts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete attempt: %w", err)
	}
	return nil
}

func (r *mysteryDAO) GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) ([]repository.MysteryAttemptRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT a.id, a.mystery_id, a.user_id, a.parent_id, a.body, a.is_winner, a.created_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			COALESCE((SELECT SUM(value) FROM mystery_attempt_votes WHERE attempt_id = a.id), 0),
			COALESCE((SELECT value FROM mystery_attempt_votes WHERE attempt_id = a.id AND user_id = $1), 0)
		FROM mystery_attempts a
		JOIN users u ON a.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE a.mystery_id = $2
		ORDER BY a.created_at ASC`,
		viewerID, mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get attempts: %w", err)
	}
	defer rows.Close()

	var result []repository.MysteryAttemptRow
	for rows.Next() {
		var row repository.MysteryAttemptRow
		var createdAt time.Time
		if err := rows.Scan(
			&row.ID, &row.MysteryID, &row.UserID, &row.ParentID, &row.Body, &row.IsWinner, &createdAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.VoteScore, &row.UserVote,
		); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *mysteryDAO) GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var authorID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT user_id FROM mystery_attempts WHERE id = $1`, attemptID).Scan(&authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get attempt author: %w", err)
	}
	return authorID, nil
}

func (r *mysteryDAO) GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var mysteryID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT mystery_id FROM mystery_attempts WHERE id = $1`, attemptID).Scan(&mysteryID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get attempt mystery: %w", err)
	}
	return mysteryID, nil
}

func (r *mysteryDAO) VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int, tx ...*sql.Tx) error {
	if value == 0 {
		_, err := txOrDB(r.db, tx).ExecContext(ctx,
			`DELETE FROM mystery_attempt_votes WHERE user_id = $1 AND attempt_id = $2`,
			userID, attemptID,
		)
		return err
	}
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO mystery_attempt_votes (user_id, attempt_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, attempt_id) DO UPDATE SET value = $4`,
		userID, attemptID, value, value,
	)
	if err != nil {
		return fmt.Errorf("vote attempt: %w", err)
	}
	return nil
}

func (r *mysteryDAO) GetAttemptOwner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error) {
	var attemptUserID uuid.UUID
	var attemptMysteryID uuid.UUID
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT user_id, mystery_id FROM mystery_attempts WHERE id = $1`, attemptID,
	).Scan(&attemptUserID, &attemptMysteryID); err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get attempt for winner: %w", err)
	}

	return attemptUserID, attemptMysteryID, nil
}

func (r *mysteryDAO) SetMysteryWinner(ctx context.Context, mysteryID uuid.UUID, winnerID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE mysteries SET solved = TRUE, winner_id = $1, solved_at = NOW() WHERE id = $2`,
		winnerID, mysteryID,
	); err != nil {
		return fmt.Errorf("mark solved: %w", err)
	}

	return nil
}

func (r *mysteryDAO) SetAttemptWinner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE mystery_attempts SET is_winner = TRUE WHERE id = $1`, attemptID,
	); err != nil {
		return fmt.Errorf("set winning attempt: %w", err)
	}

	return nil
}

func (r *mysteryDAO) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE mysteries SET solved = TRUE, solved_at = NOW() WHERE id = $1 AND solved = FALSE`,
		mysteryID,
	)
	if err != nil {
		return fmt.Errorf("mark permanently solved: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mystery not found or already solved")
	}
	return nil
}

func (r *mysteryDAO) UserHasWinningAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var exists bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM mystery_attempts WHERE mystery_id = $1 AND user_id = $2 AND is_winner = TRUE)`,
		mysteryID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user winning attempt: %w", err)
	}
	return exists, nil
}

func (r *mysteryDAO) GetSolverIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT DISTINCT user_id FROM mystery_attempts WHERE mystery_id = $1 AND is_winner = TRUE`,
		mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get solver ids: %w", err)
	}
	return utils.ScanIDs(rows, "solver id")
}

func (r *mysteryDAO) IsSolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var solved bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT solved FROM mysteries WHERE id = $1`, mysteryID).Scan(&solved)
	if err != nil {
		return false, fmt.Errorf("check mystery solved: %w", err)
	}
	return solved, nil
}

func (r *mysteryDAO) IsPaused(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var paused bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT paused FROM mysteries WHERE id = $1`, mysteryID).Scan(&paused)
	if err != nil {
		return false, fmt.Errorf("check mystery paused: %w", err)
	}
	return paused, nil
}

func (r *mysteryDAO) SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool, tx ...*sql.Tx) error {
	if paused {
		_, err := txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE mysteries
			 SET paused = TRUE,
			     paused_at = CASE WHEN paused = TRUE THEN paused_at ELSE NOW() END
			 WHERE id = $1`, mysteryID)
		if err != nil {
			return fmt.Errorf("set mystery paused: %w", err)
		}
		return nil
	}
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE mysteries
		 SET paused = FALSE,
		     paused_duration_seconds = paused_duration_seconds + CASE
		         WHEN paused_at IS NOT NULL
		         THEN EXTRACT(EPOCH FROM (NOW() - paused_at))::INTEGER
		         ELSE 0
		     END,
		     paused_at = NULL
		 WHERE id = $1`, mysteryID)
	if err != nil {
		return fmt.Errorf("set mystery unpaused: %w", err)
	}
	return nil
}

func (r *mysteryDAO) SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `UPDATE mysteries SET gm_away = $1 WHERE id = $2`, away, mysteryID)
	if err != nil {
		return fmt.Errorf("set mystery gm_away: %w", err)
	}
	return nil
}

func (r *mysteryDAO) CountAttempts(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = $1`, mysteryID).Scan(&count)
	return count, err
}

func (r *mysteryDAO) CountClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = $1`, mysteryID).Scan(&count)
	return count, err
}

func (r *mysteryDAO) GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT DISTINCT ma.user_id FROM mystery_attempts ma
		JOIN mysteries m ON m.id = ma.mystery_id
		WHERE ma.mystery_id = $1 AND ma.user_id != m.user_id`, mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get player ids: %w", err)
	}
	return utils.ScanIDs(rows, "player id")
}

func (r *mysteryDAO) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]repository.MysteryRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM mysteries WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user mysteries: %w", err)
	}

	query := `SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve, m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider, m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins, m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		w.id, w.username, w.display_name, w.avatar_url, COALESCE(wr.role, ''),
		(SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = m.id AND parent_id IS NULL AND user_id != m.user_id),
		(SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = m.id),
		(SELECT COUNT(DISTINCT user_id) FROM mystery_attempts WHERE mystery_id = m.id AND is_winner = TRUE)
	FROM mysteries m
	JOIN users u ON m.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id
	LEFT JOIN users w ON m.winner_id = w.id
	LEFT JOIN user_roles wr ON wr.user_id = w.id
	WHERE m.user_id = $1
	ORDER BY m.created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user mysteries: %w", err)
	}
	defer rows.Close()

	var result []repository.MysteryRow
	for rows.Next() {
		var row repository.MysteryRow
		var solvedAt, pausedAt sql.NullTime
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Title, &row.Body, &row.Difficulty, &row.Solved, &row.Paused, &row.GmAway, &row.FreeForAll, &row.KeepOpenAfterSolve, &row.Knox.CulpritNamedEarly, &row.Knox.NoSupernatural, &row.Knox.PassagesDeclared, &row.Knox.NoUnknownPoison, &row.Knox.NoOutsider, &row.Knox.NoLuckyAccident, &row.Knox.DetectiveNotCulprit, &row.Knox.CluesShown, &row.Knox.NarratorHidesNothing, &row.Knox.NoUnannouncedTwins, &row.KnoxPublished, &solvedAt, &pausedAt, &row.PausedDurationSeconds, &createdAt, &updatedAt,
			&row.AuthorUsername, &row.AuthorDisplayName, &row.AuthorAvatarURL, &row.AuthorRole,
			&row.WinnerID, &row.WinnerUsername, &row.WinnerDisplayName, &row.WinnerAvatarURL, &row.WinnerRole,
			&row.AttemptCount, &row.ClueCount, &row.SolverCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mystery: %w", err)
		}
		row.SolvedAt = mysteryNullTimePtr(solvedAt)
		row.PausedAt = mysteryNullTimePtr(pausedAt)
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *mysteryDAO) GetLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]repository.LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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
			LEFT JOIN mystery_attempts a ON a.user_id = u.id AND a.is_winner = TRUE
			LEFT JOIN mysteries m ON m.id = a.mystery_id
			LEFT JOIN user_roles r ON r.user_id = u.id
			GROUP BY u.id, r.role
			HAVING COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
					CASE WHEN m.difficulty = 'easy' THEN 2
					     WHEN m.difficulty = 'medium' THEN 4
					     WHEN m.difficulty = 'hard' THEN 6
					     WHEN m.difficulty = 'nightmare' THEN 8
					     ELSE 4 END
				ELSE 0 END), 0) + u.mystery_score_adjustment > 0
		) AS lb
		ORDER BY score DESC, display_name ASC
		LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}
	defer rows.Close()

	var result []repository.LeaderboardEntry
	for rows.Next() {
		var e repository.LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.DisplayName, &e.AvatarURL, &e.Role,
			&e.Score, &e.EasySolved, &e.MediumSolved, &e.HardSolved, &e.NightmareSolved, &e.ScoreAdjustment); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *mysteryDAO) GetTopDetectiveIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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
			LEFT JOIN mystery_attempts a ON a.user_id = u.id AND a.is_winner = TRUE
			LEFT JOIN mysteries m ON m.id = a.mystery_id
			GROUP BY u.id
			HAVING COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
					CASE WHEN m.difficulty = 'easy' THEN 2
					     WHEN m.difficulty = 'medium' THEN 4
					     WHEN m.difficulty = 'hard' THEN 6
					     WHEN m.difficulty = 'nightmare' THEN 8
					     ELSE 4 END
				ELSE 0 END), 0) + u.mystery_score_adjustment > 0
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

func (r *mysteryDAO) GetGMLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]repository.GMLeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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
					+ LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
				) + u.gm_score_adjustment AS score,
				COUNT(m.id) AS mystery_count,
				SUM(LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)) AS player_count
			FROM mysteries m
			JOIN users u ON m.user_id = u.id
			LEFT JOIN user_roles r ON r.user_id = u.id
			WHERE m.solved = TRUE
			GROUP BY u.id, r.role
			HAVING SUM(
					CASE m.difficulty
						WHEN 'easy' THEN 2
						WHEN 'medium' THEN 4
						WHEN 'hard' THEN 6
						WHEN 'nightmare' THEN 8
						ELSE 4
					END
					+ LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
				) + u.gm_score_adjustment > 0
		) AS gm_lb
		ORDER BY score DESC, display_name ASC
		LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get gm leaderboard: %w", err)
	}
	defer rows.Close()

	var result []repository.GMLeaderboardEntry
	for rows.Next() {
		var e repository.GMLeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.DisplayName, &e.AvatarURL, &e.Role,
			&e.Score, &e.MysteryCount, &e.PlayerCount); err != nil {
			return nil, fmt.Errorf("scan gm leaderboard entry: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *mysteryDAO) GetTopGMIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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
					+ LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
				) + u.gm_score_adjustment AS score
			FROM mysteries m
			JOIN users u ON m.user_id = u.id
			WHERE m.solved = TRUE
			GROUP BY u.id
			HAVING SUM(
					CASE m.difficulty
						WHEN 'easy' THEN 2
						WHEN 'medium' THEN 4
						WHEN 'hard' THEN 6
						WHEN 'nightmare' THEN 8
						ELSE 4
					END
					+ LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
				) + u.gm_score_adjustment > 0
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

func (r *mysteryDAO) AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int, tx ...*sql.Tx) (int64, error) {
	var id int64
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO mystery_attachments (mystery_id, file_url, file_name, file_size) VALUES ($1, $2, $3, $4) RETURNING id`,
		mysteryID, fileURL, fileName, fileSize,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add attachment: %w", err)
	}
	return id, nil
}

func (r *mysteryDAO) DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM mystery_attachments WHERE id = $1 AND mystery_id = $2`,
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

func (r *mysteryDAO) GetAttachments(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryAttachment, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, file_url, file_name, file_size FROM mystery_attachments WHERE mystery_id = $1 ORDER BY created_at`,
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

func (r *mysteryDAO) AddMedia(ctx context.Context, spec repository.NewMysteryMedia, tx ...*sql.Tx) (int64, error) {
	return r.mediaDAO.AddMedia(ctx, spec.MysteryID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *mysteryDAO) AddCommentMedia(ctx context.Context, spec repository.NewMysteryCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.commentDAO.AddCommentMedia(ctx, spec.CommentID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *mysteryDAO) GetAttachmentPaths(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT file_url FROM mystery_attachments WHERE mystery_id = $1 ORDER BY created_at, id`,
		mysteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get mystery attachment paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var fileURL string
		if err := rows.Scan(&fileURL); err != nil {
			return nil, fmt.Errorf("scan mystery attachment path: %w", err)
		}

		if fileURL != "" {
			paths = append(paths, fileURL)
		}
	}

	return paths, rows.Err()
}
