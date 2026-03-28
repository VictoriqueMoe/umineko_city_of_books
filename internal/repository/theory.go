package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/theory/params"
)

type (
	TheoryRepository interface {
		Create(ctx context.Context, userID int, req dto.CreateTheoryRequest) (int64, error)
		GetByID(ctx context.Context, id int) (*dto.TheoryDetailResponse, error)
		List(ctx context.Context, p params.ListParams, userID int) ([]dto.TheoryResponse, int, error)
		Update(ctx context.Context, id, userID int, title, body string, episode int) error
		Delete(ctx context.Context, id, userID int) error
		GetEvidence(ctx context.Context, theoryID int) ([]dto.EvidenceResponse, error)
		CreateResponse(ctx context.Context, theoryID, userID int, req dto.CreateResponseRequest) (int64, error)
		DeleteResponse(ctx context.Context, id, userID int) error
		GetResponses(ctx context.Context, theoryID int, userID int) ([]dto.ResponseResponse, error)
		GetResponseEvidence(ctx context.Context, responseID int) ([]dto.EvidenceResponse, error)
		VoteTheory(ctx context.Context, userID, theoryID, value int) error
		VoteResponse(ctx context.Context, userID, responseID, value int) error
		GetUserTheoryVote(ctx context.Context, userID, theoryID int) (int, error)
	}

	theoryRepository struct {
		db *sql.DB
	}
)

func (r *theoryRepository) Create(ctx context.Context, userID int, req dto.CreateTheoryRequest) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO theories (user_id, title, body, episode) VALUES (?, ?, ?, ?)`,
		userID, req.Title, req.Body, req.Episode,
	)
	if err != nil {
		return 0, fmt.Errorf("insert theory: %w", err)
	}

	theoryID, _ := result.LastInsertId()

	for i, ev := range req.Evidence {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order) VALUES (?, ?, ?, ?, ?)`,
			theoryID, ev.AudioID, ev.QuoteIndex, ev.Note, i,
		)
		if err != nil {
			return 0, fmt.Errorf("insert evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return theoryID, nil
}

func (r *theoryRepository) GetByID(ctx context.Context, id int) (*dto.TheoryDetailResponse, error) {
	var t dto.TheoryDetailResponse
	var author dto.UserResponse

	err := r.db.QueryRowContext(ctx,
		`SELECT t.id, t.title, t.body, t.episode, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url
		 FROM theories t
		 JOIN users u ON t.user_id = u.id
		 WHERE t.id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Body, &t.Episode, &t.CreatedAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL)

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

func (r *theoryRepository) List(ctx context.Context, p params.ListParams, userID int) ([]dto.TheoryResponse, int, error) {
	var conditions []string
	var args []interface{}
	if p.Episode > 0 {
		conditions = append(conditions, "t.episode = ?")
		args = append(args, p.Episode)
	}
	if p.AuthorID > 0 {
		conditions = append(conditions, "t.user_id = ?")
		args = append(args, p.AuthorID)
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
	case "controversial":
		orderBy = `ORDER BY (SELECT COUNT(*) FROM theory_votes WHERE theory_id = t.id) DESC, t.created_at DESC`
	default:
		orderBy = `ORDER BY t.created_at DESC`
	}

	query := fmt.Sprintf(
		`SELECT t.id, t.title, t.body, t.episode, t.created_at,
		        u.id, u.username, u.display_name, u.avatar_url
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
		if err := rows.Scan(&t.ID, &t.Title, &t.Body, &t.Episode, &t.CreatedAt,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL); err != nil {
			return nil, 0, fmt.Errorf("scan theory: %w", err)
		}
		t.Author = author

		if len(t.Body) > 200 {
			t.Body = t.Body[:200] + "..."
		}

		up, down, _ := r.getTheoryVoteCounts(ctx, t.ID)
		t.VoteScore = up - down

		withLove, withoutLove, _ := r.getResponseSideCounts(ctx, t.ID)
		t.WithLoveCount = withLove
		t.WithoutLoveCount = withoutLove

		if userID > 0 {
			t.UserVote, _ = r.GetUserTheoryVote(ctx, userID, t.ID)
		}

		theories = append(theories, t)
	}

	return theories, total, rows.Err()
}

func (r *theoryRepository) Update(ctx context.Context, id, userID int, title, body string, episode int) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE theories SET title = ?, body = ?, episode = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND user_id = ?`,
		title, body, episode, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update theory: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("theory not found or not owned by user")
	}
	return nil
}

func (r *theoryRepository) Delete(ctx context.Context, id, userID int) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM theories WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete theory: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("theory not found or not owned by user")
	}
	return nil
}

func (r *theoryRepository) GetEvidence(ctx context.Context, theoryID int) ([]dto.EvidenceResponse, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT te.id, te.audio_id, te.quote_index, te.note, te.sort_order
		 FROM theory_evidence te
		 WHERE te.theory_id = ?
		 ORDER BY te.sort_order`, theoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get evidence: %w", err)
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

func (r *theoryRepository) CreateResponse(ctx context.Context, theoryID, userID int, req dto.CreateResponseRequest) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO responses (theory_id, user_id, side, body, parent_id) VALUES (?, ?, ?, ?, ?)`,
		theoryID, userID, req.Side, req.Body, req.ParentID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert response: %w", err)
	}

	responseID, _ := result.LastInsertId()

	for i, ev := range req.Evidence {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO response_evidence (response_id, audio_id, quote_index, note, sort_order) VALUES (?, ?, ?, ?, ?)`,
			responseID, ev.AudioID, ev.QuoteIndex, ev.Note, i,
		)
		if err != nil {
			return 0, fmt.Errorf("insert response evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return responseID, nil
}

func (r *theoryRepository) DeleteResponse(ctx context.Context, id, userID int) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM responses WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete response: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("response not found or not owned by user")
	}
	return nil
}

func (r *theoryRepository) GetResponses(ctx context.Context, theoryID int, userID int) ([]dto.ResponseResponse, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.parent_id, r.side, r.body, r.created_at,
		        u.id, u.username, u.display_name, u.avatar_url
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
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL); err != nil {
			return nil, fmt.Errorf("scan response: %w", err)
		}
		resp.Author = author

		up, down, _ := r.getResponseVoteCounts(ctx, resp.ID)
		resp.VoteScore = up - down

		if userID > 0 {
			resp.UserVote, _ = r.getUserResponseVote(ctx, userID, resp.ID)
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

	return buildResponseTree(all), nil
}

type responseNode struct {
	data     dto.ResponseResponse
	children []*responseNode
}

func buildResponseTree(flat []dto.ResponseResponse) []dto.ResponseResponse {
	nodes := make(map[int]*responseNode)
	for i := range flat {
		nodes[flat[i].ID] = &responseNode{data: flat[i]}
	}

	var roots []*responseNode
	for i := range flat {
		if flat[i].ParentID == nil {
			roots = append(roots, nodes[flat[i].ID])
		} else {
			if parent, ok := nodes[*flat[i].ParentID]; ok {
				parent.children = append(parent.children, nodes[flat[i].ID])
			} else {
				roots = append(roots, nodes[flat[i].ID])
			}
		}
	}

	result := make([]dto.ResponseResponse, len(roots))
	for i, r := range roots {
		result[i] = flattenNode(r)
	}
	return result
}

func flattenNode(n *responseNode) dto.ResponseResponse {
	resp := n.data
	resp.Replies = nil
	for _, child := range n.children {
		resp.Replies = append(resp.Replies, flattenNode(child))
	}
	return resp
}

func (r *theoryRepository) GetResponseEvidence(ctx context.Context, responseID int) ([]dto.EvidenceResponse, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT re.id, re.audio_id, re.quote_index, re.note, re.sort_order
		 FROM response_evidence re
		 WHERE re.response_id = ?
		 ORDER BY re.sort_order`, responseID,
	)
	if err != nil {
		return nil, fmt.Errorf("get response evidence: %w", err)
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

func (r *theoryRepository) VoteTheory(ctx context.Context, userID, theoryID, value int) error {
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

func (r *theoryRepository) VoteResponse(ctx context.Context, userID, responseID, value int) error {
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

func (r *theoryRepository) GetUserTheoryVote(ctx context.Context, userID, theoryID int) (int, error) {
	var value int
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM theory_votes WHERE user_id = ? AND theory_id = ?`, userID, theoryID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func (r *theoryRepository) getTheoryVoteCounts(ctx context.Context, theoryID int) (int, int, error) {
	var up, down int
	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM theory_votes WHERE theory_id = ?`, theoryID,
	).Scan(&up, &down)
	return up, down, nil
}

func (r *theoryRepository) getResponseVoteCounts(ctx context.Context, responseID int) (int, int, error) {
	var up, down int
	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		 FROM response_votes WHERE response_id = ?`, responseID,
	).Scan(&up, &down)
	return up, down, nil
}

func (r *theoryRepository) getUserResponseVote(ctx context.Context, userID, responseID int) (int, error) {
	var value int
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM response_votes WHERE user_id = ? AND response_id = ?`, userID, responseID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func (r *theoryRepository) getResponseSideCounts(ctx context.Context, theoryID int) (int, int, error) {
	var withLove, withoutLove int
	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN side = 'with_love' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN side = 'without_love' THEN 1 ELSE 0 END), 0)
		 FROM responses WHERE theory_id = ?`, theoryID,
	).Scan(&withLove, &withoutLove)
	return withLove, withoutLove, nil
}
