package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	JournalRepository interface {
		Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error)
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalResponse, error)
		List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.JournalResponse, int, error)
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateJournalRequest) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetAuthorID(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
		GetTitle(ctx context.Context, id uuid.UUID) (string, error)
		IsArchived(ctx context.Context, id uuid.UUID) (bool, error)
		CountUserJournalsToday(ctx context.Context, userID uuid.UUID) (int, error)
		UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID) error
		ArchiveStale(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error)

		Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error
		Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error
		IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) (bool, error)
		GetFollowerIDs(ctx context.Context, journalID uuid.UUID) ([]uuid.UUID, error)
		GetFollowerCount(ctx context.Context, journalID uuid.UUID) (int, error)
		ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]dto.JournalResponse, int, error)

		CreateComment(ctx context.Context, id uuid.UUID, journalID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]JournalCommentRow, int, error)
		GetCommentJournalID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]JournalCommentMediaRow, error)
	}

	JournalCommentRow struct {
		ID                uuid.UUID
		JournalID         uuid.UUID
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

	JournalCommentMediaRow struct {
		ID           int
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	journalRepository struct {
		db *sql.DB
	}
)

const journalSelectBase = `SELECT j.id, j.title, j.body, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at,
		u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM journal_follows WHERE journal_id = j.id),
		(SELECT COUNT(*) FROM journal_comments WHERE journal_id = j.id)
	FROM journals j
	JOIN users u ON j.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id`

func scanJournalRow(scanner interface {
	Scan(dest ...interface{}) error
}, viewerID uuid.UUID, db *sql.DB) (*dto.JournalResponse, error) {
	var j dto.JournalResponse
	var author dto.UserResponse
	err := scanner.Scan(
		&j.ID, &j.Title, &j.Body, &j.Work, &j.CreatedAt, &j.UpdatedAt, &j.LastAuthorActivityAt, &j.ArchivedAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Role,
		&j.FollowerCount, &j.CommentCount,
	)
	if err != nil {
		return nil, err
	}
	j.Author = author
	j.IsArchived = j.ArchivedAt != nil

	if viewerID != uuid.Nil {
		var exists int
		_ = db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM journal_follows WHERE journal_id = ? AND user_id = ?)`,
			j.ID, viewerID,
		).Scan(&exists)
		j.IsFollowing = exists == 1
	}
	return &j, nil
}

func (r *journalRepository) Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error) {
	id := uuid.New()
	work := req.Work
	if work == "" {
		work = "general"
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO journals (id, user_id, title, body, work) VALUES (?, ?, ?, ?, ?)`,
		id, userID, req.Title, req.Body, work,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create journal: %w", err)
	}
	return id, nil
}

func (r *journalRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalResponse, error) {
	row := r.db.QueryRowContext(ctx, journalSelectBase+` WHERE j.id = ?`, id)
	j, err := scanJournalRow(row, viewerID, r.db)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal: %w", err)
	}
	return j, nil
}

func (r *journalRepository) List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.JournalResponse, int, error) {
	var conditions []string
	var args []interface{}
	if p.Work != "" {
		conditions = append(conditions, "j.work = ?")
		args = append(args, p.Work)
	}
	if p.AuthorID != uuid.Nil {
		conditions = append(conditions, "j.user_id = ?")
		args = append(args, p.AuthorID)
	}
	if p.Search != "" {
		conditions = append(conditions, "(j.title LIKE ? OR j.body LIKE ?)")
		wildcard := "%" + p.Search + "%"
		args = append(args, wildcard, wildcard)
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

	exclSQL, exclArgs := ExcludeClause("j.user_id", excludeUserIDs)
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

	query := journalSelectBase + where + " " + orderBy + " LIMIT ? OFFSET ?"
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
		if len(j.Body) > 300 {
			j.Body = j.Body[:300] + "..."
		}
		journals = append(journals, *j)
	}
	return journals, total, rows.Err()
}

func (r *journalRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE journals SET title = ?, body = ?, work = ?, updated_at = CURRENT_TIMESTAMP, last_author_activity_at = CURRENT_TIMESTAMP, archived_at = NULL WHERE id = ? AND user_id = ?`,
		req.Title, req.Body, req.Work, id, userID,
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

func (r *journalRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateJournalRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE journals SET title = ?, body = ?, work = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		req.Title, req.Body, req.Work, id,
	)
	if err != nil {
		return fmt.Errorf("admin update journal: %w", err)
	}
	return nil
}

func (r *journalRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM journals WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete journal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("journal not found or not owned")
	}
	return nil
}

func (r *journalRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM journals WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete journal: %w", err)
	}
	return nil
}

func (r *journalRepository) GetAuthorID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var authorID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM journals WHERE id = ?`, id).Scan(&authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get journal author: %w", err)
	}
	return authorID, nil
}

func (r *journalRepository) GetTitle(ctx context.Context, id uuid.UUID) (string, error) {
	var title string
	err := r.db.QueryRowContext(ctx, `SELECT title FROM journals WHERE id = ?`, id).Scan(&title)
	if err != nil {
		return "", fmt.Errorf("get journal title: %w", err)
	}
	return title, nil
}

func (r *journalRepository) IsArchived(ctx context.Context, id uuid.UUID) (bool, error) {
	var archivedAt sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT archived_at FROM journals WHERE id = ?`, id).Scan(&archivedAt)
	if err != nil {
		return false, fmt.Errorf("check archived: %w", err)
	}
	return archivedAt.Valid, nil
}

func (r *journalRepository) CountUserJournalsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journals WHERE user_id = ? AND created_at >= datetime('now', '-1 day')`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user journals today: %w", err)
	}
	return count, nil
}

func (r *journalRepository) UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE journals SET last_author_activity_at = CURRENT_TIMESTAMP, archived_at = NULL WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("update last author activity: %w", err)
	}
	return nil
}

func (r *journalRepository) ArchiveStale(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	cutoffStr := cutoff.UTC().Format("2006-01-02 15:04:05")

	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM journals WHERE archived_at IS NULL AND last_author_activity_at < ?`,
		cutoffStr,
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
		`UPDATE journals SET archived_at = CURRENT_TIMESTAMP WHERE archived_at IS NULL AND last_author_activity_at < ?`,
		cutoffStr,
	)
	if err != nil {
		return nil, fmt.Errorf("archive stale journals: %w", err)
	}
	return ids, nil
}

func (r *journalRepository) Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO journal_follows (user_id, journal_id) VALUES (?, ?)`,
		userID, journalID,
	)
	if err != nil {
		return fmt.Errorf("follow journal: %w", err)
	}
	return nil
}

func (r *journalRepository) Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM journal_follows WHERE user_id = ? AND journal_id = ?`,
		userID, journalID,
	)
	if err != nil {
		return fmt.Errorf("unfollow journal: %w", err)
	}
	return nil
}

func (r *journalRepository) IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM journal_follows WHERE user_id = ? AND journal_id = ?)`,
		userID, journalID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check journal follower: %w", err)
	}
	return exists == 1, nil
}

func (r *journalRepository) GetFollowerIDs(ctx context.Context, journalID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM journal_follows WHERE journal_id = ?`,
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

func (r *journalRepository) GetFollowerCount(ctx context.Context, journalID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_follows WHERE journal_id = ?`,
		journalID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get follower count: %w", err)
	}
	return count, nil
}

func (r *journalRepository) ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]dto.JournalResponse, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_follows WHERE user_id = ?`, followerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count followed journals: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		journalSelectBase+`
		JOIN journal_follows jf ON jf.journal_id = j.id
		WHERE jf.user_id = ?
		ORDER BY jf.created_at DESC
		LIMIT ? OFFSET ?`,
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
		if len(j.Body) > 300 {
			j.Body = j.Body[:300] + "..."
		}
		journals = append(journals, *j)
	}
	return journals, total, rows.Err()
}

func (r *journalRepository) CreateComment(ctx context.Context, id uuid.UUID, journalID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO journal_comments (id, journal_id, parent_id, user_id, body) VALUES (?, ?, ?, ?, ?)`,
		id, journalID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create journal comment: %w", err)
	}
	return nil
}

func (r *journalRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE journal_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update journal comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *journalRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE journal_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update journal comment: %w", err)
	}
	return nil
}

func (r *journalRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM journal_comments WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete journal comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *journalRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM journal_comments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete journal comment: %w", err)
	}
	return nil
}

func (r *journalRepository) GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]JournalCommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs)
	var total int
	countArgs := []interface{}{journalID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM journal_comments WHERE journal_id = ?`+exclSQL,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count journal comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs)
	queryArgs := []interface{}{viewerID, journalID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.journal_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM journal_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM journal_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM journal_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.journal_id = ?`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get journal comments: %w", err)
	}
	defer rows.Close()

	var comments []JournalCommentRow
	for rows.Next() {
		var c JournalCommentRow
		var userLikedInt int
		if err := rows.Scan(
			&c.ID, &c.JournalID, &c.ParentID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &userLikedInt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan journal comment: %w", err)
		}
		c.UserLiked = userLikedInt == 1
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *journalRepository) GetCommentJournalID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT journal_id FROM journal_comments WHERE id = ?`, commentID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get journal comment journal id: %w", err)
	}
	return id, nil
}

func (r *journalRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM journal_comments WHERE id = ?`, commentID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get journal comment author: %w", err)
	}
	return userID, nil
}

func (r *journalRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO journal_comment_likes (user_id, comment_id) VALUES (?, ?)`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like journal comment: %w", err)
	}
	return nil
}

func (r *journalRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM journal_comment_likes WHERE user_id = ? AND comment_id = ?`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike journal comment: %w", err)
	}
	return nil
}

func (r *journalRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO journal_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add journal comment media: %w", err)
	}
	return res.LastInsertId()
}

func (r *journalRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE journal_comment_media SET media_url = ? WHERE id = ?`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update journal comment media url: %w", err)
	}
	return nil
}

func (r *journalRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE journal_comment_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update journal comment media thumbnail: %w", err)
	}
	return nil
}

func (r *journalRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]JournalCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{commentIDs[0]}
	for _, id := range commentIDs[1:] {
		placeholders += ", ?"
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM journal_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get journal comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]JournalCommentMediaRow)
	for rows.Next() {
		var m JournalCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan journal comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}

func JournalCommentToDTO(c JournalCommentRow, media []JournalCommentMediaRow, authorID uuid.UUID) dto.JournalCommentResponse {
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
	return dto.JournalCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     mediaList,
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		IsAuthor:  c.UserID == authorID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
