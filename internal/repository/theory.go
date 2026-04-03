package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/theory/params"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

type (
	TheoryRepository interface {
		Create(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error)
		GetByID(ctx context.Context, id uuid.UUID) (*dto.TheoryDetailResponse, error)
		List(ctx context.Context, p params.ListParams, userID uuid.UUID) ([]dto.TheoryResponse, int, error)
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateTheoryRequest) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateTheoryRequest) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetEvidence(ctx context.Context, theoryID uuid.UUID) ([]dto.EvidenceResponse, error)
		CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error)
		DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID) error
		GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID) ([]dto.ResponseResponse, error)
		GetResponseEvidence(ctx context.Context, responseID uuid.UUID) ([]dto.EvidenceResponse, error)
		VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error
		VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error
		GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID) (int, error)
		GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID) (uuid.UUID, error)
		GetResponseInfo(ctx context.Context, responseID uuid.UUID) (authorID uuid.UUID, theoryID uuid.UUID, err error)
		GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ActivityItem, int, error)
		CountUserTheoriesToday(ctx context.Context, userID uuid.UUID) (int, error)
		CountUserResponsesToday(ctx context.Context, userID uuid.UUID) (int, error)
		UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64) error
		GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID) (withLoveSum float64, withoutLoveSum float64, err error)
		SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64) error
		GetTheoryTitle(ctx context.Context, theoryID uuid.UUID) (string, error)
	}

	theoryRepository struct {
		db *sql.DB
	}
)

func (r *theoryRepository) Create(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	theoryID := uuid.New()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO theories (id, user_id, title, body, episode) VALUES (?, ?, ?, ?, ?)`,
		theoryID, userID, req.Title, req.Body, req.Episode,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert theory: %w", err)
	}

	for i, ev := range req.Evidence {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order) VALUES (?, ?, ?, ?, ?)`,
			theoryID, ev.AudioID, ev.QuoteIndex, ev.Note, i,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("insert evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}

	return theoryID, nil
}

func (r *theoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*dto.TheoryDetailResponse, error) {
	var t dto.TheoryDetailResponse
	var author dto.UserResponse

	err := r.db.QueryRowContext(ctx,
		`SELECT t.id, t.title, t.body, t.episode, t.credibility_score, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM theories t
		 JOIN users u ON t.user_id = u.id
		 WHERE t.id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Body, &t.Episode, &t.CredibilityScore, &t.CreatedAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get theory: %w", err)
	}

	t.Author = author

	up, down, err := r.getTheoryVoteCounts(ctx, id)
	if err != nil {
		return nil, err
	}
	t.VoteScore = up - down

	withLove, withoutLove, err := r.getResponseSideCounts(ctx, id)
	if err != nil {
		return nil, err
	}
	t.WithLoveCount = withLove
	t.WithoutLoveCount = withoutLove

	return &t, nil
}

func (r *theoryRepository) List(ctx context.Context, p params.ListParams, userID uuid.UUID) ([]dto.TheoryResponse, int, error) {
	var conditions []string
	var args []interface{}
	if p.Episode > 0 {
		conditions = append(conditions, "t.episode = ?")
		args = append(args, p.Episode)
	}
	if p.AuthorID != uuid.Nil {
		conditions = append(conditions, "t.user_id = ?")
		args = append(args, p.AuthorID)
	}
	if p.Search != "" {
		conditions = append(conditions, "(t.title LIKE ? OR t.body LIKE ?)")
		wildcard := "%" + p.Search + "%"
		args = append(args, wildcard, wildcard)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM theories t"+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count theories: %w", err)
	}

	var orderBy string
	switch p.Sort {
	case "popular":
		orderBy = `ORDER BY (SELECT COALESCE(SUM(value), 0) FROM theory_votes WHERE theory_id = t.id) DESC, t.created_at DESC`
	case "popular_asc":
		orderBy = `ORDER BY (SELECT COALESCE(SUM(value), 0) FROM theory_votes WHERE theory_id = t.id) ASC, t.created_at ASC`
	case "controversial":
		orderBy = `ORDER BY (SELECT COUNT(*) FROM theory_votes WHERE theory_id = t.id) DESC, t.created_at DESC`
	case "controversial_asc":
		orderBy = `ORDER BY (SELECT COUNT(*) FROM theory_votes WHERE theory_id = t.id) ASC, t.created_at ASC`
	case "credibility":
		orderBy = `ORDER BY t.credibility_score DESC, t.created_at DESC`
	case "credibility_asc":
		orderBy = `ORDER BY t.credibility_score ASC, t.created_at ASC`
	case "old":
		orderBy = `ORDER BY t.created_at ASC`
	default:
		orderBy = `ORDER BY t.created_at DESC`
	}

	query := fmt.Sprintf(
		`SELECT t.id, t.title, t.body, t.episode, t.credibility_score, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM theories t
		 JOIN users u ON t.user_id = u.id
		 %s %s LIMIT ? OFFSET ?`, where, orderBy,
	)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list theories: %w", err)
	}
	defer rows.Close()

	var theories []dto.TheoryResponse
	for rows.Next() {
		var t dto.TheoryResponse
		var author dto.UserResponse
		if err := rows.Scan(&t.ID, &t.Title, &t.Body, &t.Episode, &t.CredibilityScore, &t.CreatedAt,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role); err != nil {
			return nil, 0, fmt.Errorf("scan theory: %w", err)
		}
		t.Author = author

		if len(t.Body) > 200 {
			t.Body = t.Body[:200] + "..."
		}

		up, down, err := r.getTheoryVoteCounts(ctx, t.ID)
		if err != nil {
			logger.Log.Error().Err(err).Str("theory_id", t.ID.String()).Msg("failed to get theory vote counts")
		}
		t.VoteScore = up - down

		withLove, withoutLove, err := r.getResponseSideCounts(ctx, t.ID)
		if err != nil {
			logger.Log.Error().Err(err).Str("theory_id", t.ID.String()).Msg("failed to get response side counts")
		}
		t.WithLoveCount = withLove
		t.WithoutLoveCount = withoutLove

		if userID != uuid.Nil {
			vote, err := r.GetUserTheoryVote(ctx, userID, t.ID)
			if err != nil {
				logger.Log.Error().Err(err).Str("theory_id", t.ID.String()).Msg("failed to get user theory vote")
			}
			t.UserVote = vote
		}

		theories = append(theories, t)
	}

	return theories, total, rows.Err()
}

func (r *theoryRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateTheoryRequest) error {
	return r.updateTheory(ctx, id, &userID, req)
}

func (r *theoryRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateTheoryRequest) error {
	return r.updateTheory(ctx, id, nil, req)
}

func (r *theoryRepository) updateTheory(ctx context.Context, id uuid.UUID, userID *uuid.UUID, req dto.CreateTheoryRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var result sql.Result
	if userID != nil {
		result, err = tx.ExecContext(ctx,
			`UPDATE theories SET title = ?, body = ?, episode = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE id = ? AND user_id = ?`,
			req.Title, req.Body, req.Episode, id, *userID,
		)
	} else {
		result, err = tx.ExecContext(ctx,
			`UPDATE theories SET title = ?, body = ?, episode = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE id = ?`,
			req.Title, req.Body, req.Episode, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update theory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get rows affected for theory update")
	}
	if affected == 0 {
		return fmt.Errorf("theory not found or not owned by user")
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM theory_evidence WHERE theory_id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete old evidence: %w", err)
	}

	for i, ev := range req.Evidence {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order) VALUES (?, ?, ?, ?, ?)`,
			id, ev.AudioID, ev.QuoteIndex, ev.Note, i,
		)
		if err != nil {
			return fmt.Errorf("insert evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (r *theoryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM theories WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete theory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get rows affected for theory delete")
	}
	if affected == 0 {
		return fmt.Errorf("theory not found or not owned by user")
	}
	return nil
}

func (r *theoryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM theories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete theory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get rows affected for admin theory delete")
	}
	if affected == 0 {
		return fmt.Errorf("theory not found")
	}
	return nil
}

func (r *theoryRepository) GetEvidence(ctx context.Context, theoryID uuid.UUID) ([]dto.EvidenceResponse, error) {
	return r.queryEvidence(ctx,
		`SELECT te.id, te.audio_id, te.quote_index, te.note, te.sort_order
		 FROM theory_evidence te
		 WHERE te.theory_id = ?
		 ORDER BY te.sort_order`, theoryID,
	)
}

func (r *theoryRepository) CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	responseID := uuid.New()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO responses (id, theory_id, user_id, side, body, parent_id) VALUES (?, ?, ?, ?, ?, ?)`,
		responseID, theoryID, userID, req.Side, req.Body, req.ParentID,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert response: %w", err)
	}

	for i, ev := range req.Evidence {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO response_evidence (response_id, audio_id, quote_index, note, sort_order) VALUES (?, ?, ?, ?, ?)`,
			responseID, ev.AudioID, ev.QuoteIndex, ev.Note, i,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("insert response evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}

	return responseID, nil
}

func (r *theoryRepository) DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM responses WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete response: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get rows affected for response delete")
	}
	if affected == 0 {
		return fmt.Errorf("response not found or not owned by user")
	}
	return nil
}

func (r *theoryRepository) DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM responses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete response: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get rows affected for admin response delete")
	}
	if affected == 0 {
		return fmt.Errorf("response not found")
	}
	return nil
}

func (r *theoryRepository) GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID) ([]dto.ResponseResponse, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.parent_id, r.side, r.body, r.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM responses r
		 JOIN users u ON r.user_id = u.id
		 WHERE r.theory_id = ?
		 ORDER BY r.created_at ASC`, theoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get responses: %w", err)
	}
	defer rows.Close()

	var all []dto.ResponseResponse
	for rows.Next() {
		var resp dto.ResponseResponse
		var author dto.UserResponse
		if err := rows.Scan(&resp.ID, &resp.ParentID, &resp.Side, &resp.Body, &resp.CreatedAt,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role); err != nil {
			return nil, fmt.Errorf("scan response: %w", err)
		}
		resp.Author = author

		up, down, err := r.getResponseVoteCounts(ctx, resp.ID)
		if err != nil {
			logger.Log.Error().Err(err).Str("response_id", resp.ID.String()).Msg("failed to get response vote counts")
		}
		resp.VoteScore = up - down

		if userID != uuid.Nil {
			vote, err := r.getUserResponseVote(ctx, userID, resp.ID)
			if err != nil {
				logger.Log.Error().Err(err).Str("response_id", resp.ID.String()).Msg("failed to get user response vote")
			}
			resp.UserVote = vote
		}

		evidence, err := r.GetResponseEvidence(ctx, resp.ID)
		if err != nil {
			return nil, err
		}
		resp.Evidence = evidence

		all = append(all, resp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return utils.BuildTree(all,
		func(r dto.ResponseResponse) uuid.UUID { return r.ID },
		func(r dto.ResponseResponse) *uuid.UUID { return r.ParentID },
		func(r *dto.ResponseResponse, replies []dto.ResponseResponse) { r.Replies = replies },
	), nil
}

func (r *theoryRepository) GetResponseEvidence(ctx context.Context, responseID uuid.UUID) ([]dto.EvidenceResponse, error) {
	return r.queryEvidence(ctx,
		`SELECT re.id, re.audio_id, re.quote_index, re.note, re.sort_order
		 FROM response_evidence re
		 WHERE re.response_id = ?
		 ORDER BY re.sort_order`, responseID,
	)
}

func (r *theoryRepository) queryEvidence(ctx context.Context, query string, args ...any) ([]dto.EvidenceResponse, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}
	defer rows.Close()

	var evidence []dto.EvidenceResponse
	for rows.Next() {
		var ev dto.EvidenceResponse
		if err := rows.Scan(&ev.ID, &ev.AudioID, &ev.QuoteIndex, &ev.Note, &ev.SortOrder); err != nil {
			return nil, fmt.Errorf("scan evidence: %w", err)
		}
		evidence = append(evidence, ev)
	}
	return evidence, rows.Err()
}

func (r *theoryRepository) VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error {
	if value == 0 {
		_, err := r.db.ExecContext(ctx,
			`DELETE FROM theory_votes WHERE user_id = ? AND theory_id = ?`, userID, theoryID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO theory_votes (user_id, theory_id, value) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, theory_id) DO UPDATE SET value = excluded.value`,
		userID, theoryID, value,
	)
	return err
}

func (r *theoryRepository) VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error {
	if value == 0 {
		_, err := r.db.ExecContext(ctx,
			`DELETE FROM response_votes WHERE user_id = ? AND response_id = ?`, userID, responseID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO response_votes (user_id, response_id, value) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, response_id) DO UPDATE SET value = excluded.value`,
		userID, responseID, value,
	)
	return err
}

func (r *theoryRepository) GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID) (int, error) {
	var value int
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM theory_votes WHERE user_id = ? AND theory_id = ?`, userID, theoryID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func (r *theoryRepository) getTheoryVoteCounts(ctx context.Context, theoryID uuid.UUID) (int, int, error) {
	var up, down int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM theory_votes WHERE theory_id = ?`, theoryID,
	).Scan(&up, &down)
	return up, down, err
}

func (r *theoryRepository) getResponseVoteCounts(ctx context.Context, responseID uuid.UUID) (int, int, error) {
	var up, down int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM response_votes WHERE response_id = ?`, responseID,
	).Scan(&up, &down)
	return up, down, err
}

func (r *theoryRepository) getUserResponseVote(ctx context.Context, userID uuid.UUID, responseID uuid.UUID) (int, error) {
	var value int
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM response_votes WHERE user_id = ? AND response_id = ?`, userID, responseID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func (r *theoryRepository) getResponseSideCounts(ctx context.Context, theoryID uuid.UUID) (int, int, error) {
	var withLove, withoutLove int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN side = 'with_love' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN side = 'without_love' THEN 1 ELSE 0 END), 0)
		 FROM responses WHERE theory_id = ?`, theoryID,
	).Scan(&withLove, &withoutLove)
	return withLove, withoutLove, err
}

func (r *theoryRepository) GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM theories WHERE id = ?`, theoryID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get theory author: %w", err)
	}
	return userID, nil
}

func (r *theoryRepository) GetResponseInfo(ctx context.Context, responseID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	var authorID, theoryID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, theory_id FROM responses WHERE id = ?`, responseID,
	).Scan(&authorID, &theoryID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get response info: %w", err)
	}
	return authorID, theoryID, nil
}

func (r *theoryRepository) GetTheoryTitle(ctx context.Context, theoryID uuid.UUID) (string, error) {
	var title string
	err := r.db.QueryRowContext(ctx, `SELECT title FROM theories WHERE id = ?`, theoryID).Scan(&title)
	if err != nil {
		return "", fmt.Errorf("get theory title: %w", err)
	}
	return title, nil
}

func (r *theoryRepository) GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ActivityItem, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM theories WHERE user_id = ?) + (SELECT COUNT(*) FROM responses WHERE user_id = ?)`,
		userID, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count activity: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT type, theory_id, theory_title, side, body, created_at FROM (
			SELECT 'theory' as type, t.id as theory_id, t.title as theory_title, '' as side, t.body, t.created_at
			FROM theories t WHERE t.user_id = ?
			UNION ALL
			SELECT 'response' as type, r.theory_id, th.title as theory_title, r.side, r.body, r.created_at
			FROM responses r JOIN theories th ON r.theory_id = th.id WHERE r.user_id = ?
		) combined ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get activity: %w", err)
	}
	defer rows.Close()

	var items []dto.ActivityItem
	for rows.Next() {
		var item dto.ActivityItem
		if err := rows.Scan(&item.Type, &item.TheoryID, &item.TheoryTitle, &item.Side, &item.Body, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan activity: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *theoryRepository) CountUserTheoriesToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = ? AND created_at > datetime('now', '-1 day')`, userID,
	).Scan(&count)
	return count, err
}

func (r *theoryRepository) CountUserResponsesToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = ? AND created_at > datetime('now', '-1 day')`, userID,
	).Scan(&count)
	return count, err
}

func (r *theoryRepository) UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE theories SET credibility_score = ? WHERE id = ?`, score, theoryID,
	)
	if err != nil {
		return fmt.Errorf("update credibility score: %w", err)
	}
	return nil
}

func (r *theoryRepository) GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID) (float64, float64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.side, COALESCE(SUM(re.truth_weight), 0)
		 FROM responses r
		 LEFT JOIN response_evidence re ON r.id = re.response_id
		 WHERE r.theory_id = ? AND r.parent_id IS NULL
		 GROUP BY r.side`, theoryID,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("get evidence weights: %w", err)
	}
	defer rows.Close()

	var withLove, withoutLove float64
	for rows.Next() {
		var side string
		var weight float64
		if err := rows.Scan(&side, &weight); err != nil {
			return 0, 0, fmt.Errorf("scan evidence weight: %w", err)
		}
		if side == "with_love" {
			withLove = weight
		} else {
			withoutLove = weight
		}
	}
	return withLove, withoutLove, rows.Err()
}

func (r *theoryRepository) SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE response_evidence SET truth_weight = ? WHERE id = ?`, weight, evidenceID,
	)
	if err != nil {
		return fmt.Errorf("set evidence truth weight: %w", err)
	}
	return nil
}
