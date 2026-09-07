package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/text"

	"github.com/google/uuid"
)

type (
	JournalDAO interface {
		Create(ctx context.Context, s spec.NewJournal, tx ...*sql.Tx) (*dto.JournalResponse, error)
		GetByID(ctx context.Context, s spec.JournalLookup, tx ...*sql.Tx) (*dto.JournalResponse, error)
		List(ctx context.Context, q spec.JournalQuery, tx ...*sql.Tx) ([]dto.JournalResponse, int, error)
		Update(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error
		UpdateAsAdmin(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListEntryIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		ListEntryCommentIDs(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetAuthorID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetTitle(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (string, error)
		IsArchived(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error)
		CountUserJournalsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		SetPaused(ctx context.Context, s spec.JournalPause, tx ...*sql.Tx) error
		ArchiveStale(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error)

		Follow(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) error
		Unfollow(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) error
		IsFollower(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) (bool, error)
		GetFollowerIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetFollowerCount(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error)
		ListFollowedByUser(ctx context.Context, q spec.JournalFollowedQuery, tx ...*sql.Tx) ([]dto.JournalResponse, int, error)

		CreateEntry(ctx context.Context, s spec.NewJournalEntry, tx ...*sql.Tx) (*model.JournalEntryRow, error)
		UpdateEntry(ctx context.Context, s spec.JournalEntryUpdate, tx ...*sql.Tx) error
		DeleteEntry(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetEntry(ctx context.Context, s spec.JournalEntryLookup, tx ...*sql.Tx) (*model.JournalEntryRow, error)
		GetEntryByID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (*model.JournalEntryRow, error)
		ListEntries(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]model.JournalEntrySummaryRow, error)
		GetNextEntryNumber(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetEntryJournalID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetEntryAuthorID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		CreateComment(ctx context.Context, s spec.NewJournalComment, tx ...*sql.Tx) (*model.CommentRow, error)
		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetEntryComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*int, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)

		AddMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, s spec.MediaDeletion, tx ...*sql.Tx) (string, error)
	}

	journalDAO struct {
		db *sql.DB
		*ownedDAO
		*commentDAO[uuid.UUID]
		*mediaDAO
	}
)

const (
	journalSelectBase = `SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
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
)

func scanJournalRow(ctx context.Context, scanner interface {
	Scan(dest ...any) error
}, viewerID uuid.UUID, db dbtx) (*dto.JournalResponse, error) {
	var j dto.JournalResponse
	var author dto.UserResponse
	var createdAt, lastAuthorActivityAt time.Time
	var updatedAt, archivedAt, pausedAt *time.Time
	var latestEntryNumber *int
	var latestEntryTitle *string
	var latestEntryBody *string
	var latestEntryAt *time.Time
	err := scanner.Scan(
		&j.ID, &j.Title, &j.Work, &createdAt, &updatedAt, &lastAuthorActivityAt, &archivedAt, &pausedAt,
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
	j.IsPaused = pausedAt != nil
	j.LatestEntryNumber = latestEntryNumber
	j.LatestEntryTitle = latestEntryTitle
	j.LatestEntryAt = timePtrToString(latestEntryAt)
	if latestEntryBody != nil {
		excerpt := text.ClampRunes(*latestEntryBody, 300)
		if len(excerpt) != len(*latestEntryBody) {
			excerpt += "..."
		}

		j.LatestEntryExcerpt = excerpt
	}

	if viewerID != uuid.Nil {
		var exists bool
		_ = db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM journal_follows WHERE journal_id = $1 AND user_id = $2)`,
			j.ID, viewerID,
		).Scan(&exists)
		j.IsFollowing = exists
	}
	return &j, nil
}

func (r *journalDAO) Create(ctx context.Context, s spec.NewJournal, tx ...*sql.Tx) (*dto.JournalResponse, error) {
	work := s.Work
	if work == "" {
		work = "general"
	}

	created, err := scanJournalRow(ctx, txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH j AS (
		     INSERT INTO journals (user_id, title, work)
		     VALUES ($1, $2, $3)
		     RETURNING *
		 )
		 SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
		        u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		        0, 0, 0,
		        NULL::int, NULL::text, NULL::text, NULL::timestamptz
		 FROM j
		 JOIN users u ON u.id = j.user_id
		 LEFT JOIN user_roles r ON r.user_id = u.id`,
		s.UserID, s.Title, work,
	), uuid.Nil, txOrDB(r.db, tx))
	if err != nil {
		return nil, fmt.Errorf("create journal: %w", err)
	}

	return created, nil
}

func (r *journalDAO) GetByID(ctx context.Context, s spec.JournalLookup, tx ...*sql.Tx) (*dto.JournalResponse, error) {
	row := txOrDB(r.db, tx).QueryRowContext(ctx, journalSelectBase+` WHERE j.id = $1`, s.ID)
	j, err := scanJournalRow(ctx, row, s.ViewerID, txOrDB(r.db, tx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal: %w", err)
	}
	return j, nil
}

func (r *journalDAO) List(ctx context.Context, q spec.JournalQuery, tx ...*sql.Tx) ([]dto.JournalResponse, int, error) {
	idx := 1
	next := func() string {
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}
	var conditions []string
	var args []any
	if q.Work != "" {
		conditions = append(conditions, "j.work = "+next())
		args = append(args, q.Work)
	}
	if q.AuthorID != uuid.Nil {
		conditions = append(conditions, "j.user_id = "+next())
		args = append(args, q.AuthorID)
	}
	if q.Search != "" {
		conditions = append(conditions, "j.title ILIKE "+next())
		wildcard := "%" + q.Search + "%"
		args = append(args, wildcard)
	}
	if !q.IncludeArchived {
		conditions = append(conditions, "j.archived_at IS NULL")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	exclSQL, exclArgs := ExcludeClause("j.user_id", q.ExcludeUserIDs, idx)
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
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM journals j"+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count journals: %w", err)
	}

	var orderBy string
	switch q.Sort {
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
	args = append(args, q.Limit, q.Offset)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list journals: %w", err)
	}
	defer rows.Close()

	var journals []dto.JournalResponse
	for rows.Next() {
		j, err := scanJournalRow(ctx, rows, q.ViewerID, txOrDB(r.db, tx))
		if err != nil {
			return nil, 0, fmt.Errorf("scan journal: %w", err)
		}
		journals = append(journals, *j)
	}
	return journals, total, rows.Err()
}

func (r *journalDAO) Update(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE journals SET title = $1, work = $2, updated_at = NOW(), last_author_activity_at = NOW(), archived_at = NULL WHERE id = $3 AND user_id = $4`,
		s.Title, s.Work, s.ID, s.UserID,
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

func (r *journalDAO) UpdateAsAdmin(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE journals SET title = $1, work = $2, updated_at = NOW() WHERE id = $3`,
		s.Title, s.Work, s.ID,
	)
	if err != nil {
		return fmt.Errorf("admin update journal: %w", err)
	}
	return nil
}

func (r *journalDAO) ListEntryIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id FROM journal_entries WHERE journal_id = $1 ORDER BY entry_number`,
		journalID,
	)
	if err != nil {
		return nil, fmt.Errorf("list journal entry ids: %w", err)
	}
	return utils.ScanIDs(rows, "journal entry id")
}

func (r *journalDAO) ListEntryCommentIDs(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`WITH RECURSIVE tree AS (
			SELECT id FROM journal_comments WHERE entry_id = $1
			UNION
			SELECT c.id FROM journal_comments c JOIN tree t ON c.parent_id = t.id
		)
		SELECT id FROM tree`,
		entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list entry comment ids: %w", err)
	}
	return utils.ScanIDs(rows, "entry comment id")
}

func (r *journalDAO) GetTitle(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (string, error) {
	var title string
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT title FROM journals WHERE id = $1`, id).Scan(&title)
	if err != nil {
		return "", fmt.Errorf("get journal title: %w", err)
	}
	return title, nil
}

func (r *journalDAO) IsArchived(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var archivedAt *time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT archived_at FROM journals WHERE id = $1`, id).Scan(&archivedAt)
	if err != nil {
		return false, fmt.Errorf("check archived: %w", err)
	}
	return archivedAt != nil, nil
}

func (r *journalDAO) CountUserJournalsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journals WHERE user_id = $1 AND created_at >= NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user journals today: %w", err)
	}
	return count, nil
}

func (r *journalDAO) UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE journals SET last_author_activity_at = NOW(), archived_at = NULL WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("update last author activity: %w", err)
	}
	return nil
}

func (r *journalDAO) ArchiveStale(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`UPDATE journals SET archived_at = NOW()
		 WHERE archived_at IS NULL AND paused_at IS NULL AND last_author_activity_at < $1
		 RETURNING id`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("archive stale journals: %w", err)
	}

	return utils.ScanIDs(rows, "stale journal id")
}

func (r *journalDAO) SetPaused(ctx context.Context, s spec.JournalPause, tx ...*sql.Tx) error {
	var pausedAt any
	if s.Paused {
		pausedAt = time.Now().UTC()
	}

	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE journals SET paused_at = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		pausedAt, s.ID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("set journal paused: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set journal paused rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("journal not found or not owned")
	}

	return nil
}

func (r *journalDAO) Follow(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO journal_follows (user_id, journal_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		s.UserID, s.JournalID,
	)
	if err != nil {
		return fmt.Errorf("follow journal: %w", err)
	}
	return nil
}

func (r *journalDAO) Unfollow(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM journal_follows WHERE user_id = $1 AND journal_id = $2`,
		s.UserID, s.JournalID,
	)
	if err != nil {
		return fmt.Errorf("unfollow journal: %w", err)
	}
	return nil
}

func (r *journalDAO) IsFollower(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) (bool, error) {
	var exists bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM journal_follows WHERE user_id = $1 AND journal_id = $2)`,
		s.UserID, s.JournalID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check journal follower: %w", err)
	}
	return exists, nil
}

func (r *journalDAO) GetFollowerIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT user_id FROM journal_follows WHERE journal_id = $1`,
		journalID,
	)
	if err != nil {
		return nil, fmt.Errorf("get follower ids: %w", err)
	}
	return utils.ScanIDs(rows, "follower id")
}

func (r *journalDAO) GetFollowerCount(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_follows WHERE journal_id = $1`,
		journalID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get follower count: %w", err)
	}
	return count, nil
}

func (r *journalDAO) ListFollowedByUser(ctx context.Context, q spec.JournalFollowedQuery, tx ...*sql.Tx) ([]dto.JournalResponse, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_follows WHERE user_id = $1`, q.FollowerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count followed journals: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		journalSelectBase+`
		JOIN journal_follows jf ON jf.journal_id = j.id
		WHERE jf.user_id = $1
		ORDER BY jf.created_at DESC
		LIMIT $2 OFFSET $3`,
		q.FollowerID, q.Limit, q.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list followed journals: %w", err)
	}
	defer rows.Close()

	var journals []dto.JournalResponse
	for rows.Next() {
		j, err := scanJournalRow(ctx, rows, q.ViewerID, txOrDB(r.db, tx))
		if err != nil {
			return nil, 0, fmt.Errorf("scan followed journal: %w", err)
		}
		journals = append(journals, *j)
	}
	return journals, total, rows.Err()
}

func (r *journalDAO) CreateEntry(ctx context.Context, s spec.NewJournalEntry, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	var e model.JournalEntryRow
	var createdAt time.Time
	var updatedAt *time.Time

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO journal_entries (journal_id, entry_number, title, body, word_count, is_draft)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, journal_id, entry_number, title, body, word_count, is_draft, created_at, updated_at`,
		s.JournalID, s.EntryNumber, s.Title, s.Body, s.WordCount, s.IsDraft,
	).Scan(&e.ID, &e.JournalID, &e.EntryNumber, &e.Title, &e.Body, &e.WordCount, &e.IsDraft, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("create journal entry: %w", err)
	}

	e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	e.UpdatedAt = timePtrToString(updatedAt)

	return &e, nil
}

func (r *journalDAO) UpdateEntry(ctx context.Context, s spec.JournalEntryUpdate, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE journal_entries SET title = $1, body = $2, word_count = $3, is_draft = $4, updated_at = NOW() WHERE id = $5`,
		s.Title, s.Body, s.WordCount, s.IsDraft, s.ID,
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

func (r *journalDAO) DeleteEntry(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM journal_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete journal entry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entry not found")
	}
	return nil
}

func (r *journalDAO) GetEntry(ctx context.Context, s spec.JournalEntryLookup, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	var e model.JournalEntryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id, journal_id, entry_number, title, body, word_count, is_draft, created_at, updated_at,
			EXISTS(SELECT 1 FROM journal_entries WHERE journal_id = $1 AND entry_number < $2 AND NOT is_draft),
			EXISTS(SELECT 1 FROM journal_entries WHERE journal_id = $1 AND entry_number > $2 AND NOT is_draft)
		FROM journal_entries
		WHERE journal_id = $1 AND entry_number = $2`,
		s.JournalID, s.EntryNumber,
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

func (r *journalDAO) GetEntryByID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	var e model.JournalEntryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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

func (r *journalDAO) ListEntries(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]model.JournalEntrySummaryRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, entry_number, title, word_count, is_draft, created_at FROM journal_entries WHERE journal_id = $1 ORDER BY entry_number DESC`,
		journalID,
	)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()

	var entries []model.JournalEntrySummaryRow
	for rows.Next() {
		var e model.JournalEntrySummaryRow
		var createdAt time.Time
		if err := rows.Scan(&e.ID, &e.EntryNumber, &e.Title, &e.WordCount, &e.IsDraft, &createdAt); err != nil {
			return nil, fmt.Errorf("scan entry summary: %w", err)
		}
		e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *journalDAO) GetNextEntryNumber(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var next int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COALESCE(MAX(entry_number), 0) + 1 FROM journal_entries WHERE journal_id = $1`,
		journalID,
	).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("get next entry number: %w", err)
	}
	return next, nil
}

func (r *journalDAO) GetEntryJournalID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var id uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT journal_id FROM journal_entries WHERE id = $1`, entryID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get entry journal id: %w", err)
	}
	return id, nil
}

func (r *journalDAO) GetEntryAuthorID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var userID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT j.user_id FROM journal_entries e JOIN journals j ON j.id = e.journal_id WHERE e.id = $1`,
		entryID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get entry author id: %w", err)
	}
	return userID, nil
}

func (r *journalDAO) CreateComment(ctx context.Context, s spec.NewJournalComment, tx ...*sql.Tx) (*model.CommentRow, error) {
	var c model.CommentRow
	var createdAt time.Time
	var updatedAt *time.Time

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH c AS (
		     INSERT INTO journal_comments (journal_id, entry_id, parent_id, user_id, body)
		     VALUES ($1, $2, $3, $4, $5)
		     RETURNING *
		 )
		 SELECT c.id, c.journal_id::text, c.entry_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
		        u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''), (u.banned_at IS NOT NULL),
		        0, FALSE
		 FROM c
		 JOIN users u ON u.id = c.user_id
		 LEFT JOIN user_roles r ON r.user_id = c.user_id`,
		s.JournalID, s.EntryID, s.ParentID, s.UserID, s.Body,
	).Scan(
		&c.ID, &c.EntityID, &c.EntryID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
		&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole, &c.AuthorBanned,
		&c.LikeCount, &c.UserLiked,
	)
	if err != nil {
		return nil, fmt.Errorf("create journal comment: %w", err)
	}

	c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	c.UpdatedAt = timePtrToString(updatedAt)

	return &c, nil
}

func (r *journalDAO) GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", q.ExcludeUserIDs, 2)
	var total int
	countArgs := []any{q.TargetID}
	countArgs = append(countArgs, exclArgs...)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_comments WHERE journal_id = $1 AND entry_id IS NULL`+exclSQL,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count journal comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", q.ExcludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	queryArgs := []any{q.ViewerID, q.TargetID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, q.Limit, q.Offset)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

func (r *journalDAO) GetEntryComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", q.ExcludeUserIDs, 2)
	var total int
	countArgs := []any{q.TargetID}
	countArgs = append(countArgs, exclArgs...)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_comments WHERE entry_id = $1`+exclSQL,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count entry comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", q.ExcludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	queryArgs := []any{q.ViewerID, q.TargetID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, q.Limit, q.Offset)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

func scanJournalCommentRows(rows *sql.Rows) ([]model.CommentRow, error) {
	var comments []model.CommentRow
	for rows.Next() {
		var c model.CommentRow
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

func (r *journalDAO) GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*int, error) {
	var entryNumber *int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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
