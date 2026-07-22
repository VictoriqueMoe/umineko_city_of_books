package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	journalDAO struct {
		db *sql.DB
		*commentDAO[uuid.UUID]
		*mediaDAO
	}
)

const journalSelectBase = `SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at,
		u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM journal_follows WHERE journal_id = j.id),
		(SELECT COUNT(*) FROM journal_comments WHERE journal_id = j.id),
		(SELECT COUNT(*) FROM journal_entries WHERE journal_id = j.id),
		le.entry_number, le.title, le.body, le.created_at
	FROM journals j
	JOIN users u ON j.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id
	LEFT JOIN LATERAL (
		SELECT entry_number, title, body, created_at
		FROM journal_entries
		WHERE journal_id = j.id AND NOT is_draft
		ORDER BY entry_number DESC
		LIMIT 1
	) le ON TRUE`

func scanJournalRow(scanner interface {
	Scan(dest ...interface{}) error
}, viewerID uuid.UUID, db *sql.DB) (*dto.JournalResponse, error) {
	var j dto.JournalResponse
	var author dto.UserResponse
	var createdAt, lastAuthorActivityAt time.Time
	var updatedAt, archivedAt *time.Time
	var latestEntryNumber *int
	var latestEntryTitle *string
	var latestEntryBody *string
	var latestEntryAt *time.Time
	err := scanner.Scan(
		&j.ID, &j.Title, &j.Work, &createdAt, &updatedAt, &lastAuthorActivityAt, &archivedAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role,
		&j.FollowerCount, &j.CommentCount, &j.EntryCount,
		&latestEntryNumber, &latestEntryTitle, &latestEntryBody, &latestEntryAt,
	)
	if err != nil {
		return nil, err
	}
	j.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	j.UpdatedAt = timePtrToString(updatedAt)
	j.LastAuthorActivityAt = lastAuthorActivityAt.UTC().Format(time.RFC3339)
	j.ArchivedAt = timePtrToString(archivedAt)
	j.Author = author
	j.IsArchived = j.ArchivedAt != nil
	j.LatestEntryNumber = latestEntryNumber
	j.LatestEntryTitle = latestEntryTitle
	j.LatestEntryAt = timePtrToString(latestEntryAt)
	if latestEntryBody != nil {
		excerpt := *latestEntryBody
		if len(excerpt) > 300 {
			excerpt = excerpt[:300] + "..."
		}
		j.LatestEntryExcerpt = excerpt
	}

	if viewerID != uuid.Nil {
		var exists bool
		_ = db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM journal_follows WHERE journal_id = $1 AND user_id = $2)`,
			j.ID, viewerID,
		).Scan(&exists)
		j.IsFollowing = exists
	}
	return &j, nil
}

func (r *journalDAO) Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error) {
	id := uuid.New()
	work := req.Work
	if work == "" {
		work = "general"
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO journals (id, user_id, title, work) VALUES ($1, $2, $3, $4)`,
		id, userID, req.Title, work,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create journal: %w", err)
	}
	return id, nil
}

func (r *journalDAO) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalResponse, error) {
	row := r.db.QueryRowContext(ctx, journalSelectBase+` WHERE j.id = $1`, id)
	j, err := scanJournalRow(row, viewerID, r.db)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal: %w", err)
	}
	return j, nil
}

func (r *journalDAO) List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.JournalResponse, int, error) {
	idx := 1
	next := func() string {
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}
	var conditions []string
	var args []interface{}
	if p.Work != "" {
		conditions = append(conditions, "j.work = "+next())
		args = append(args, p.Work)
	}
	if p.AuthorID != uuid.Nil {
		conditions = append(conditions, "j.user_id = "+next())
		args = append(args, p.AuthorID)
	}
	if p.Search != "" {
		conditions = append(conditions, "j.title ILIKE "+next())
		wildcard := "%" + p.Search + "%"
		args = append(args, wildcard)
	}
	if !p.IncludeArchived {
		conditions = append(conditions, "j.archived_at IS NULL")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	exclSQL, exclArgs := ExcludeClause("j.user_id", excludeUserIDs, idx)
	idx += len(exclArgs)
	if where == "" && exclSQL != "" {
		where = " WHERE 1=1" + exclSQL
	} else {
		where += exclSQL
	}
	args = append(args, exclArgs...)

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM journals j"+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count journals: %w", err)
	}

	var orderBy string
	switch p.Sort {
	case "old":
		orderBy = "ORDER BY j.created_at ASC"
	case "recently_active":
		orderBy = "ORDER BY j.last_author_activity_at DESC"
	case "most_followed":
		orderBy = "ORDER BY (SELECT COUNT(*) FROM journal_follows WHERE journal_id = j.id) DESC, j.created_at DESC"
	default:
		orderBy = "ORDER BY j.created_at DESC"
	}

	limitPH := next()
	offsetPH := next()
	query := journalSelectBase + where + " " + orderBy + " LIMIT " + limitPH + " OFFSET " + offsetPH
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list journals: %w", err)
	}
	defer rows.Close()

	var journals []dto.JournalResponse
	for rows.Next() {
		j, err := scanJournalRow(rows, viewerID, r.db)
		if err != nil {
			return nil, 0, fmt.Errorf("scan journal: %w", err)
		}
		journals = append(journals, *j)
	}
	return journals, total, rows.Err()
}

func (r *journalDAO) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE journals SET title = $1, work = $2, updated_at = NOW(), last_author_activity_at = NOW(), archived_at = NULL WHERE id = $3 AND user_id = $4`,
		req.Title, req.Work, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update journal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("journal not found or not owned")
	}
	return nil
}

func (r *journalDAO) UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateJournalRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE journals SET title = $1, work = $2, updated_at = NOW() WHERE id = $3`,
		req.Title, req.Work, id,
	)
	if err != nil {
		return fmt.Errorf("admin update journal: %w", err)
	}
	return nil
}

func (r *journalDAO) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM journals WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete journal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("journal not found or not owned")
	}
	return nil
}

func (r *journalDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM journals WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete journal: %w", err)
	}
	return nil
}

func (r *journalDAO) GetAuthorID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var authorID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM journals WHERE id = $1`, id).Scan(&authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get journal author: %w", err)
	}
	return authorID, nil
}

func (r *journalDAO) GetTitle(ctx context.Context, id uuid.UUID) (string, error) {
	var title string
	err := r.db.QueryRowContext(ctx, `SELECT title FROM journals WHERE id = $1`, id).Scan(&title)
	if err != nil {
		return "", fmt.Errorf("get journal title: %w", err)
	}
	return title, nil
}

func (r *journalDAO) IsArchived(ctx context.Context, id uuid.UUID) (bool, error) {
	var archivedAt *time.Time
	err := r.db.QueryRowContext(ctx, `SELECT archived_at FROM journals WHERE id = $1`, id).Scan(&archivedAt)
	if err != nil {
		return false, fmt.Errorf("check archived: %w", err)
	}
	return archivedAt != nil, nil
}

func (r *journalDAO) CountUserJournalsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journals WHERE user_id = $1 AND created_at >= NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user journals today: %w", err)
	}
	return count, nil
}

func (r *journalDAO) UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE journals SET last_author_activity_at = NOW(), archived_at = NULL WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("update last author activity: %w", err)
	}
	return nil
}

func (r *journalDAO) ArchiveStale(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM journals WHERE archived_at IS NULL AND last_author_activity_at < $1`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("find stale journals: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan stale journal id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE journals SET archived_at = NOW() WHERE archived_at IS NULL AND last_author_activity_at < $1`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("archive stale journals: %w", err)
	}
	return ids, nil
}

func (r *journalDAO) Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO journal_follows (user_id, journal_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, journalID,
	)
	if err != nil {
		return fmt.Errorf("follow journal: %w", err)
	}
	return nil
}

func (r *journalDAO) Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM journal_follows WHERE user_id = $1 AND journal_id = $2`,
		userID, journalID,
	)
	if err != nil {
		return fmt.Errorf("unfollow journal: %w", err)
	}
	return nil
}

func (r *journalDAO) IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM journal_follows WHERE user_id = $1 AND journal_id = $2)`,
		userID, journalID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check journal follower: %w", err)
	}
	return exists, nil
}

func (r *journalDAO) GetFollowerIDs(ctx context.Context, journalID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM journal_follows WHERE journal_id = $1`,
		journalID,
	)
	if err != nil {
		return nil, fmt.Errorf("get follower ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan follower id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *journalDAO) GetFollowerCount(ctx context.Context, journalID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_follows WHERE journal_id = $1`,
		journalID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get follower count: %w", err)
	}
	return count, nil
}

func (r *journalDAO) ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]dto.JournalResponse, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_follows WHERE user_id = $1`, followerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count followed journals: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		journalSelectBase+`
		JOIN journal_follows jf ON jf.journal_id = j.id
		WHERE jf.user_id = $1
		ORDER BY jf.created_at DESC
		LIMIT $2 OFFSET $3`,
		followerID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list followed journals: %w", err)
	}
	defer rows.Close()

	var journals []dto.JournalResponse
	for rows.Next() {
		j, err := scanJournalRow(rows, viewerID, r.db)
		if err != nil {
			return nil, 0, fmt.Errorf("scan followed journal: %w", err)
		}
		journals = append(journals, *j)
	}
	return journals, total, rows.Err()
}

func (r *journalDAO) CreateEntry(ctx context.Context, id uuid.UUID, journalID uuid.UUID, entryNumber int, title *string, body string, wordCount int, isDraft bool) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO journal_entries (id, journal_id, entry_number, title, body, word_count, is_draft) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, journalID, entryNumber, title, body, wordCount, isDraft,
	)
	if err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	return nil
}

func (r *journalDAO) UpdateEntry(ctx context.Context, id uuid.UUID, title *string, body string, wordCount int, isDraft bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE journal_entries SET title = $1, body = $2, word_count = $3, is_draft = $4, updated_at = NOW() WHERE id = $5`,
		title, body, wordCount, isDraft, id,
	)
	if err != nil {
		return fmt.Errorf("update journal entry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entry not found")
	}
	return nil
}

func (r *journalDAO) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM journal_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete journal entry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entry not found")
	}
	return nil
}

func (r *journalDAO) GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int) (*repository.JournalEntryRow, error) {
	var e repository.JournalEntryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT id, journal_id, entry_number, title, body, word_count, is_draft, created_at, updated_at,
			EXISTS(SELECT 1 FROM journal_entries WHERE journal_id = $1 AND entry_number < $2 AND NOT is_draft),
			EXISTS(SELECT 1 FROM journal_entries WHERE journal_id = $1 AND entry_number > $2 AND NOT is_draft)
		FROM journal_entries
		WHERE journal_id = $1 AND entry_number = $2`,
		journalID, entryNumber,
	).Scan(&e.ID, &e.JournalID, &e.EntryNumber, &e.Title, &e.Body, &e.WordCount, &e.IsDraft, &createdAt, &updatedAt, &e.HasPrev, &e.HasNext)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal entry: %w", err)
	}
	e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	e.UpdatedAt = timePtrToString(updatedAt)
	return &e, nil
}

func (r *journalDAO) GetEntryByID(ctx context.Context, entryID uuid.UUID) (*repository.JournalEntryRow, error) {
	var e repository.JournalEntryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT id, journal_id, entry_number, title, body, word_count, is_draft, created_at, updated_at
		FROM journal_entries
		WHERE id = $1`,
		entryID,
	).Scan(&e.ID, &e.JournalID, &e.EntryNumber, &e.Title, &e.Body, &e.WordCount, &e.IsDraft, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal entry by id: %w", err)
	}
	e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	e.UpdatedAt = timePtrToString(updatedAt)
	return &e, nil
}

func (r *journalDAO) ListEntries(ctx context.Context, journalID uuid.UUID) ([]repository.JournalEntrySummaryRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, entry_number, title, word_count, is_draft, created_at FROM journal_entries WHERE journal_id = $1 ORDER BY entry_number DESC`,
		journalID,
	)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()

	var entries []repository.JournalEntrySummaryRow
	for rows.Next() {
		var s repository.JournalEntrySummaryRow
		var createdAt time.Time
		if err := rows.Scan(&s.ID, &s.EntryNumber, &s.Title, &s.WordCount, &s.IsDraft, &createdAt); err != nil {
			return nil, fmt.Errorf("scan entry summary: %w", err)
		}
		s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entries = append(entries, s)
	}
	return entries, rows.Err()
}

func (r *journalDAO) GetNextEntryNumber(ctx context.Context, journalID uuid.UUID) (int, error) {
	var next int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(entry_number), 0) + 1 FROM journal_entries WHERE journal_id = $1`,
		journalID,
	).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("get next entry number: %w", err)
	}
	return next, nil
}

func (r *journalDAO) GetEntryJournalID(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT journal_id FROM journal_entries WHERE id = $1`, entryID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get entry journal id: %w", err)
	}
	return id, nil
}

func (r *journalDAO) GetEntryAuthorID(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT j.user_id FROM journal_entries e JOIN journals j ON j.id = e.journal_id WHERE e.id = $1`,
		entryID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get entry author id: %w", err)
	}
	return userID, nil
}

func (r *journalDAO) CreateComment(ctx context.Context, id uuid.UUID, journalID uuid.UUID, entryID *uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO journal_comments (id, journal_id, entry_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, journalID, entryID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create journal comment: %w", err)
	}
	return nil
}

func (r *journalDAO) GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]repository.CommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs, 2)
	var total int
	countArgs := []interface{}{journalID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_comments WHERE journal_id = $1 AND entry_id IS NULL`+exclSQL,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count journal comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	queryArgs := []interface{}{viewerID, journalID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.journal_id::text, c.entry_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM journal_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM journal_comment_likes WHERE comment_id = c.id AND user_id = $1)
		FROM journal_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.journal_id = $2 AND c.entry_id IS NULL`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT `+limitPH+` OFFSET `+offsetPH,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get journal comments: %w", err)
	}
	defer rows.Close()

	comments, err := scanJournalCommentRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return comments, total, rows.Err()
}

func (r *journalDAO) GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]repository.CommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs, 2)
	var total int
	countArgs := []interface{}{entryID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_comments WHERE entry_id = $1`+exclSQL,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count entry comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	queryArgs := []interface{}{viewerID, entryID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.journal_id::text, c.entry_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM journal_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM journal_comment_likes WHERE comment_id = c.id AND user_id = $1)
		FROM journal_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.entry_id = $2`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT `+limitPH+` OFFSET `+offsetPH,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get entry comments: %w", err)
	}
	defer rows.Close()

	comments, err := scanJournalCommentRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return comments, total, rows.Err()
}

func scanJournalCommentRows(rows *sql.Rows) ([]repository.CommentRow, error) {
	var comments []repository.CommentRow
	for rows.Next() {
		var c repository.CommentRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.EntityID, &c.EntryID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, fmt.Errorf("scan journal comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		c.UpdatedAt = timePtrToString(updatedAt)
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *journalDAO) GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID) (*int, error) {
	var entryNumber *int
	err := r.db.QueryRowContext(ctx,
		`SELECT e.entry_number
		FROM journal_comments c
		LEFT JOIN journal_entries e ON e.id = c.entry_id
		WHERE c.id = $1`,
		commentID,
	).Scan(&entryNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get comment entry number: %w", err)
	}
	return entryNumber, nil
}
