package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/theory/params"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

type (
	theoryDAO struct {
		db *sql.DB
	}
)

func (r *theoryDAO) InsertTheory(ctx context.Context, spec repository.NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	series := spec.Series
	if series == "" {
		series = "umineko"
	}

	var created dto.TheoryDetailResponse
	var author dto.UserResponse
	var createdAt time.Time
	if err := getDb(r.db, tx).QueryRowContext(ctx,
		`WITH t AS (
		     INSERT INTO theories (user_id, title, body, episode, series) VALUES ($1, $2, $3, $4, $5)
		     RETURNING id, user_id, title, body, episode, series, credibility_score, status, created_at
		 )
		 SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM t
		 JOIN users u ON t.user_id = u.id`,
		spec.UserID, spec.Title, spec.Body, spec.Episode, series,
	).Scan(&created.ID, &created.Title, &created.Body, &created.Episode, &created.Series, &created.CredibilityScore, &created.Status, &createdAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role); err != nil {
		return nil, fmt.Errorf("insert theory: %w", err)
	}

	created.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	created.Author = author

	return &created, nil
}

func (r *theoryDAO) InsertTheoryEvidence(ctx context.Context, theoryID uuid.UUID, ev dto.EvidenceInput, sortOrder int, tx ...*sql.Tx) (*dto.EvidenceResponse, error) {
	var stored dto.EvidenceResponse
	if err := getDb(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order, lang) VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, audio_id, quote_index, note, sort_order, lang`,
		theoryID, ev.AudioID, ev.QuoteIndex, ev.Note, sortOrder, langOrDefault(ev.Lang),
	).Scan(&stored.ID, &stored.AudioID, &stored.QuoteIndex, &stored.Note, &stored.SortOrder, &stored.Lang); err != nil {
		return nil, fmt.Errorf("insert evidence: %w", err)
	}

	return &stored, nil
}

func (r *theoryDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	var t dto.TheoryDetailResponse
	var author dto.UserResponse
	var createdAt time.Time

	var refutedByResponseID *uuid.UUID
	var refutedAt *time.Time
	var refuterID *uuid.UUID
	var refuter dto.UserResponse

	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.refuted_by_response_id, t.refuted_at, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), ''),
		        ru.id, COALESCE(ru.username, ''), COALESCE(ru.display_name, ''), COALESCE(ru.avatar_url, ''),
		        COALESCE((SELECT role FROM user_roles WHERE user_id = ru.id LIMIT 1), '')
		 FROM theories t
		 JOIN users u ON t.user_id = u.id
		 LEFT JOIN users ru ON t.refuted_by_user_id = ru.id
		 WHERE t.id = $1`, id,
	).Scan(&t.ID, &t.Title, &t.Body, &t.Episode, &t.Series, &t.CredibilityScore, &t.Status, &refutedByResponseID, &refutedAt, &createdAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role,
		&refuterID, &refuter.Username, &refuter.DisplayName, &refuter.AvatarURL, &refuter.Role)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get theory: %w", err)
	}

	t.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	t.Author = author

	t.RefutedByResponseID = refutedByResponseID
	if refutedAt != nil {
		t.RefutedAt = refutedAt.UTC().Format(time.RFC3339)
	}
	if refuterID != nil {
		refuter.ID = *refuterID
		t.RefutedBy = &refuter
	}

	up, down, err := r.getTheoryVoteCounts(ctx, id, tx...)
	if err != nil {
		return nil, err
	}
	t.VoteScore = up - down

	withLove, withoutLove, err := r.getResponseSideCounts(ctx, id, tx...)
	if err != nil {
		return nil, err
	}
	t.WithLoveCount = withLove
	t.WithoutLoveCount = withoutLove

	return &t, nil
}

func (r *theoryDAO) List(ctx context.Context, p params.ListParams, userID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]dto.TheoryResponse, int, error) {
	idx := 1
	next := func() string {
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}
	var conditions []string
	var args []any
	if p.Series != "" {
		conditions = append(conditions, "t.series = "+next())
		args = append(args, p.Series)
	}
	if p.Episode > 0 {
		conditions = append(conditions, "t.episode = "+next())
		args = append(args, p.Episode)
	}
	if p.AuthorID != uuid.Nil {
		conditions = append(conditions, "t.user_id = "+next())
		args = append(args, p.AuthorID)
	}
	if p.Search != "" {
		conditions = append(conditions, "(t.title LIKE "+next()+" OR t.body LIKE "+next()+")")
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

	exclSQL, exclArgs := ExcludeClause("t.user_id", excludeUserIDs, idx)
	idx += len(exclArgs)
	if where == "" && exclSQL != "" {
		where = " WHERE 1=1" + exclSQL
	} else {
		where += exclSQL
	}
	args = append(args, exclArgs...)

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := getDb(r.db, tx).QueryRowContext(ctx,
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

	limitPH := next()
	offsetPH := next()

	query := fmt.Sprintf(
		`SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM theories t
		 JOIN users u ON t.user_id = u.id
		 %s %s LIMIT %s OFFSET %s`, where, orderBy, limitPH, offsetPH,
	)
	args = append(args, p.Limit, p.Offset)

	rows, err := getDb(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list theories: %w", err)
	}
	defer rows.Close()

	var theories []dto.TheoryResponse
	for rows.Next() {
		var t dto.TheoryResponse
		var author dto.UserResponse
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.Title, &t.Body, &t.Episode, &t.Series, &t.CredibilityScore, &t.Status, &createdAt,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role); err != nil {
			return nil, 0, fmt.Errorf("scan theory: %w", err)
		}
		t.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		t.Author = author

		if len(t.Body) > 200 {
			t.Body = t.Body[:200] + "..."
		}

		up, down, err := r.getTheoryVoteCounts(ctx, t.ID, tx...)
		if err != nil {
			logger.Log.Error().Err(err).Str("theory_id", t.ID.String()).Msg("failed to get theory vote counts")
		}
		t.VoteScore = up - down

		withLove, withoutLove, err := r.getResponseSideCounts(ctx, t.ID, tx...)
		if err != nil {
			logger.Log.Error().Err(err).Str("theory_id", t.ID.String()).Msg("failed to get response side counts")
		}
		t.WithLoveCount = withLove
		t.WithoutLoveCount = withoutLove

		if userID != uuid.Nil {
			vote, err := r.GetUserTheoryVote(ctx, userID, t.ID, tx...)
			if err != nil {
				logger.Log.Error().Err(err).Str("theory_id", t.ID.String()).Msg("failed to get user theory vote")
			}
			t.UserVote = vote
		}

		theories = append(theories, t)
	}

	return theories, total, rows.Err()
}

func (r *theoryDAO) UpdateTheory(ctx context.Context, spec repository.TheoryUpdate, tx ...*sql.Tx) error {
	var result sql.Result
	var err error
	if spec.AsAdmin {
		result, err = getDb(r.db, tx).ExecContext(ctx,
			`UPDATE theories SET title = $1, body = $2, episode = $3, updated_at = NOW()
			 WHERE id = $4`,
			spec.Title, spec.Body, spec.Episode, spec.ID,
		)
	} else {
		result, err = getDb(r.db, tx).ExecContext(ctx,
			`UPDATE theories SET title = $1, body = $2, episode = $3, updated_at = NOW()
			 WHERE id = $4 AND user_id = $5`,
			spec.Title, spec.Body, spec.Episode, spec.ID, spec.UserID,
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

	return nil
}

func (r *theoryDAO) ReplaceTheoryEvidence(ctx context.Context, theoryID uuid.UUID, evidence []dto.EvidenceInput, tx ...*sql.Tx) error {
	if _, err := getDb(r.db, tx).ExecContext(ctx, `DELETE FROM theory_evidence WHERE theory_id = $1`, theoryID); err != nil {
		return fmt.Errorf("delete old evidence: %w", err)
	}

	for i, ev := range evidence {
		if _, err := getDb(r.db, tx).ExecContext(ctx,
			`INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order) VALUES ($1, $2, $3, $4, $5)`,
			theoryID, ev.AudioID, ev.QuoteIndex, ev.Note, i,
		); err != nil {
			return fmt.Errorf("insert evidence: %w", err)
		}
	}

	return nil
}

func (r *theoryDAO) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	result, err := getDb(r.db, tx).ExecContext(ctx,
		`DELETE FROM theories WHERE id = $1 AND user_id = $2`, id, userID,
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

func (r *theoryDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	result, err := getDb(r.db, tx).ExecContext(ctx, `DELETE FROM theories WHERE id = $1`, id)
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

func (r *theoryDAO) GetEvidence(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error) {
	return r.queryEvidence(ctx, tx,
		`SELECT te.id, te.audio_id, te.quote_index, te.note, te.sort_order, te.lang
		 FROM theory_evidence te
		 WHERE te.theory_id = $1
		 ORDER BY te.sort_order`, theoryID,
	)
}

func (r *theoryDAO) InsertResponse(ctx context.Context, spec repository.NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error) {
	var created dto.ResponseResponse
	var author dto.UserResponse
	var createdAt time.Time
	if err := getDb(r.db, tx).QueryRowContext(ctx,
		`WITH resp AS (
		     INSERT INTO responses (theory_id, user_id, side, body, parent_id) VALUES ($1, $2, $3, $4, $5)
		     RETURNING id, user_id, parent_id, side, body, created_at
		 )
		 SELECT resp.id, resp.parent_id, resp.side, resp.body, resp.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM resp
		 JOIN users u ON resp.user_id = u.id`,
		spec.TheoryID, spec.UserID, spec.Side, spec.Body, spec.ParentID,
	).Scan(&created.ID, &created.ParentID, &created.Side, &created.Body, &createdAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role); err != nil {
		return nil, fmt.Errorf("insert response: %w", err)
	}

	created.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	created.Author = author

	return &created, nil
}

func (r *theoryDAO) InsertResponseEvidence(ctx context.Context, responseID uuid.UUID, ev dto.EvidenceInput, sortOrder int, tx ...*sql.Tx) (*dto.EvidenceResponse, error) {
	var stored dto.EvidenceResponse
	if err := getDb(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO response_evidence (response_id, audio_id, quote_index, note, sort_order, lang) VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, audio_id, quote_index, note, sort_order, lang`,
		responseID, ev.AudioID, ev.QuoteIndex, ev.Note, sortOrder, langOrDefault(ev.Lang),
	).Scan(&stored.ID, &stored.AudioID, &stored.QuoteIndex, &stored.Note, &stored.SortOrder, &stored.Lang); err != nil {
		return nil, fmt.Errorf("insert response evidence: %w", err)
	}

	return &stored, nil
}

func (r *theoryDAO) DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	result, err := getDb(r.db, tx).ExecContext(ctx,
		`DELETE FROM responses WHERE id = $1 AND user_id = $2`, id, userID,
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

func (r *theoryDAO) DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	result, err := getDb(r.db, tx).ExecContext(ctx, `DELETE FROM responses WHERE id = $1`, id)
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

func (r *theoryDAO) GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]dto.ResponseResponse, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx,
		`SELECT r.id, r.parent_id, r.side, r.body, r.created_at,
		        u.id, u.username, u.display_name, u.avatar_url,
		        COALESCE((SELECT role FROM user_roles WHERE user_id = u.id LIMIT 1), '')
		 FROM responses r
		 JOIN users u ON r.user_id = u.id
		 WHERE r.theory_id = $1
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
		var createdAt time.Time
		if err := rows.Scan(&resp.ID, &resp.ParentID, &resp.Side, &resp.Body, &createdAt,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role); err != nil {
			return nil, fmt.Errorf("scan response: %w", err)
		}
		resp.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		resp.Author = author

		up, down, err := r.getResponseVoteCounts(ctx, resp.ID, tx...)
		if err != nil {
			logger.Log.Error().Err(err).Str("response_id", resp.ID.String()).Msg("failed to get response vote counts")
		}
		resp.VoteScore = up - down

		if userID != uuid.Nil {
			vote, err := r.getUserResponseVote(ctx, userID, resp.ID, tx...)
			if err != nil {
				logger.Log.Error().Err(err).Str("response_id", resp.ID.String()).Msg("failed to get user response vote")
			}
			resp.UserVote = vote
		}

		evidence, err := r.GetResponseEvidence(ctx, resp.ID, tx...)
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

func (r *theoryDAO) GetResponseEvidence(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error) {
	return r.queryEvidence(ctx, tx,
		`SELECT re.id, re.audio_id, re.quote_index, re.note, re.sort_order, re.lang
		 FROM response_evidence re
		 WHERE re.response_id = $1
		 ORDER BY re.sort_order`, responseID,
	)
}

func (r *theoryDAO) queryEvidence(ctx context.Context, tx []*sql.Tx, query string, args ...any) ([]dto.EvidenceResponse, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}
	defer rows.Close()

	var evidence []dto.EvidenceResponse
	for rows.Next() {
		var ev dto.EvidenceResponse
		if err := rows.Scan(&ev.ID, &ev.AudioID, &ev.QuoteIndex, &ev.Note, &ev.SortOrder, &ev.Lang); err != nil {
			return nil, fmt.Errorf("scan evidence: %w", err)
		}
		evidence = append(evidence, ev)
	}
	return evidence, rows.Err()
}

func (r *theoryDAO) VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int, tx ...*sql.Tx) error {
	if value == 0 {
		_, err := getDb(r.db, tx).ExecContext(ctx,
			`DELETE FROM theory_votes WHERE user_id = $1 AND theory_id = $2`, userID, theoryID,
		)
		return err
	}
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`INSERT INTO theory_votes (user_id, theory_id, value) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, theory_id) DO UPDATE SET value = EXCLUDED.value`,
		userID, theoryID, value,
	)
	return err
}

func (r *theoryDAO) VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int, tx ...*sql.Tx) error {
	if value == 0 {
		_, err := getDb(r.db, tx).ExecContext(ctx,
			`DELETE FROM response_votes WHERE user_id = $1 AND response_id = $2`, userID, responseID,
		)
		return err
	}
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`INSERT INTO response_votes (user_id, response_id, value) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, response_id) DO UPDATE SET value = EXCLUDED.value`,
		userID, responseID, value,
	)
	return err
}

func (r *theoryDAO) GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var value int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT value FROM theory_votes WHERE user_id = $1 AND theory_id = $2`, userID, theoryID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func (r *theoryDAO) getTheoryVoteCounts(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (int, int, error) {
	var up, down int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM theory_votes WHERE theory_id = $1`, theoryID,
	).Scan(&up, &down)
	return up, down, err
}

func (r *theoryDAO) getResponseVoteCounts(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (int, int, error) {
	var up, down int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM response_votes WHERE response_id = $1`, responseID,
	).Scan(&up, &down)
	return up, down, err
}

func (r *theoryDAO) getUserResponseVote(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var value int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT value FROM response_votes WHERE user_id = $1 AND response_id = $2`, userID, responseID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func (r *theoryDAO) getResponseSideCounts(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (int, int, error) {
	var withLove, withoutLove int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN side = 'with_love' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN side = 'without_love' THEN 1 ELSE 0 END), 0)
		 FROM responses WHERE theory_id = $1 AND parent_id IS NULL`, theoryID,
	).Scan(&withLove, &withoutLove)
	return withLove, withoutLove, err
}

func (r *theoryDAO) GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var userID uuid.UUID
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT user_id FROM theories WHERE id = $1`, theoryID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get theory author: %w", err)
	}
	return userID, nil
}

func (r *theoryDAO) GetResponseInfo(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error) {
	var authorID, theoryID uuid.UUID
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT user_id, theory_id FROM responses WHERE id = $1`, responseID,
	).Scan(&authorID, &theoryID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get response info: %w", err)
	}
	return authorID, theoryID, nil
}

func (r *theoryDAO) GetTheorySeries(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	var series string
	err := getDb(r.db, tx).QueryRowContext(ctx, `SELECT series FROM theories WHERE id = $1`, theoryID).Scan(&series)
	if err != nil {
		return "", fmt.Errorf("get theory series: %w", err)
	}
	return series, nil
}

func (r *theoryDAO) GetTheoryTitle(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	var title string
	err := getDb(r.db, tx).QueryRowContext(ctx, `SELECT title FROM theories WHERE id = $1`, theoryID).Scan(&title)
	if err != nil {
		return "", fmt.Errorf("get theory title: %w", err)
	}
	return title, nil
}

func (r *theoryDAO) GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]dto.ActivityItem, int, error) {
	var total int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM theories WHERE user_id = $1) + (SELECT COUNT(*) FROM responses WHERE user_id = $2)`,
		userID, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count activity: %w", err)
	}

	rows, err := getDb(r.db, tx).QueryContext(ctx,
		`SELECT type, theory_id, theory_title, side, body, created_at FROM (
			SELECT 'theory' as type, t.id as theory_id, t.title as theory_title, '' as side, t.body, t.created_at
			FROM theories t WHERE t.user_id = $1
			UNION ALL
			SELECT 'response' as type, r.theory_id, th.title as theory_title, r.side, r.body, r.created_at
			FROM responses r JOIN theories th ON r.theory_id = th.id WHERE r.user_id = $2
		) combined ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get activity: %w", err)
	}
	defer rows.Close()

	var items []dto.ActivityItem
	for rows.Next() {
		var item dto.ActivityItem
		var createdAt time.Time
		if err := rows.Scan(&item.Type, &item.TheoryID, &item.TheoryTitle, &item.Side, &item.Body, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan activity: %w", err)
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *theoryDAO) CountUserTheoriesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day'`, userID,
	).Scan(&count)
	return count, err
}

func (r *theoryDAO) CountUserResponsesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day'`, userID,
	).Scan(&count)
	return count, err
}

func (r *theoryDAO) UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64, tx ...*sql.Tx) error {
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`UPDATE theories SET credibility_score = $1 WHERE id = $2`, score, theoryID,
	)
	if err != nil {
		return fmt.Errorf("update credibility score: %w", err)
	}
	return nil
}

func (r *theoryDAO) GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (float64, float64, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx,
		`SELECT r.side, COALESCE(SUM(re.truth_weight), 0)
		 FROM responses r
		 LEFT JOIN response_evidence re ON r.id = re.response_id
		 WHERE r.theory_id = $1 AND r.parent_id IS NULL
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

func (r *theoryDAO) SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64, tx ...*sql.Tx) error {
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`UPDATE response_evidence SET truth_weight = $1 WHERE id = $2`, weight, evidenceID,
	)
	if err != nil {
		return fmt.Errorf("set evidence truth weight: %w", err)
	}
	return nil
}

func langOrDefault(lang string) string {
	if lang == "" {
		return "en"
	}
	return lang
}

func (r *theoryDAO) RecomputeStatus(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) error {
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`UPDATE theories t SET status = CASE
			WHEN EXISTS (SELECT 1 FROM responses r WHERE r.theory_id = t.id AND r.parent_id IS NULL) THEN 'contested'::theory_status
			ELSE 'open'::theory_status
		 END
		 WHERE t.id = $1 AND t.status <> 'refuted'`, theoryID,
	)
	if err != nil {
		return fmt.Errorf("recompute theory status: %w", err)
	}
	return nil
}

func (r *theoryDAO) MarkRefuted(ctx context.Context, theoryID uuid.UUID, responseID uuid.UUID, tx ...*sql.Tx) error {
	res, err := getDb(r.db, tx).ExecContext(ctx,
		`UPDATE theories t SET status = 'refuted', refuted_by_response_id = r.id, refuted_by_user_id = r.user_id, refuted_at = NOW()
		 FROM responses r
		 WHERE t.id = $1 AND r.id = $2 AND r.theory_id = t.id AND r.parent_id IS NULL AND r.side = 'without_love' AND r.user_id <> t.user_id AND t.status <> 'refuted'`,
		theoryID, responseID,
	)
	if err != nil {
		return fmt.Errorf("mark theory refuted: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark theory refuted rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrRefutationRejected
	}
	return nil
}

func (r *theoryDAO) GetResponseMeta(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (repository.ResponseMeta, error) {
	var meta repository.ResponseMeta
	err := getDb(r.db, tx).QueryRowContext(ctx,
		`SELECT user_id, theory_id, side, parent_id FROM responses WHERE id = $1`, responseID,
	).Scan(&meta.AuthorID, &meta.TheoryID, &meta.Side, &meta.ParentID)
	if err != nil {
		return repository.ResponseMeta{}, fmt.Errorf("get response meta: %w", err)
	}
	return meta, nil
}
