package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"
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

	journalJoinRow        = sqlcgen.GetJournalByIDRow
	journalEntryJoinRow   = sqlcgen.GetJournalEntryByIDRow
	journalCommentJoinRow = sqlcgen.ListJournalRootCommentsRow

	journalListRow interface {
		sqlcgen.ListJournalsByNewRow |
			sqlcgen.ListJournalsByOldRow |
			sqlcgen.ListJournalsByRecentlyActiveRow |
			sqlcgen.ListJournalsByMostFollowedRow
	}
)

func journalStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}

	return new(ns.String)
}

func journalIntPtr(ni sql.NullInt32) *int {
	if !ni.Valid {
		return nil
	}

	return new(int(ni.Int32))
}

func journalNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *value, Valid: true}
}

func toJournalResponse(row journalJoinRow) dto.JournalResponse {
	out := dto.JournalResponse{
		ID:    row.ID,
		Title: row.Title,
		Work:  row.Work,
		Author: dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarUrl,
			Role:        role.Role(row.AuthorRole),
		},
		FollowerCount:        int(row.FollowerCount),
		IsArchived:           row.ArchivedAt.Valid,
		IsPaused:             row.PausedAt.Valid,
		CommentCount:         int(row.CommentCount),
		EntryCount:           int(row.EntryCount),
		LatestEntryNumber:    journalIntPtr(row.LatestEntryNumber),
		LatestEntryTitle:     journalStringPtr(row.LatestEntryTitle),
		LatestEntryAt:        nullTimeToStringPtr(row.LatestEntryAt),
		CreatedAt:            row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:            nullTimeToStringPtr(row.UpdatedAt),
		LastAuthorActivityAt: row.LastAuthorActivityAt.UTC().Format(time.RFC3339),
		ArchivedAt:           nullTimeToStringPtr(row.ArchivedAt),
	}

	if row.LatestEntryBody.Valid {
		excerpt := text.ClampRunes(row.LatestEntryBody.String, 300)
		if len(excerpt) != len(row.LatestEntryBody.String) {
			excerpt += "..."
		}

		out.LatestEntryExcerpt = excerpt
	}

	return out
}

func toJournalEntryRow(row journalEntryJoinRow) model.JournalEntryRow {
	return model.JournalEntryRow{
		ID:          row.ID,
		JournalID:   row.JournalID,
		EntryNumber: int(row.EntryNumber),
		Title:       journalStringPtr(row.Title),
		Body:        row.Body,
		WordCount:   int(row.WordCount),
		IsDraft:     row.IsDraft,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   new(row.UpdatedAt.UTC().Format(time.RFC3339)),
	}
}

func toJournalCommentRow(row journalCommentJoinRow) model.CommentRow {
	return model.CommentRow{
		ID:                row.ID,
		EntityID:          row.EntityID,
		EntryID:           row.EntryID,
		ParentID:          row.ParentID,
		UserID:            row.UserID,
		Body:              row.Body,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         nullTimeToStringPtr(row.UpdatedAt),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
		AuthorRole:        row.AuthorRole,
		LikeCount:         int(row.LikeCount),
		UserLiked:         row.UserLiked,
	}
}

func applyJournalViewerFollow(ctx context.Context, queries *sqlcgen.Queries, journal *dto.JournalResponse, viewerID uuid.UUID) error {
	if viewerID == uuid.Nil {
		return nil
	}

	following, err := queries.IsJournalFollowedByViewer(ctx, sqlcgen.IsJournalFollowedByViewerParams{
		JournalID: journal.ID,
		UserID:    viewerID,
	})
	if err != nil {
		return fmt.Errorf("check journal follow: %w", err)
	}

	journal.IsFollowing = following

	return nil
}

func journalResponses[T journalListRow](listed []T) []dto.JournalResponse {
	var journals []dto.JournalResponse
	for _, row := range listed {
		journals = append(journals, toJournalResponse(journalJoinRow(row)))
	}

	return journals
}

func (r *journalDAO) Create(ctx context.Context, s spec.NewJournal, tx ...*sql.Tx) (*dto.JournalResponse, error) {
	work := s.Work
	if work == "" {
		work = "general"
	}

	created, err := genQueries(r.db, tx).CreateJournal(ctx, sqlcgen.CreateJournalParams{
		UserID: s.UserID,
		Title:  s.Title,
		Work:   work,
	})
	if err != nil {
		return nil, fmt.Errorf("create journal: %w", err)
	}

	return new(toJournalResponse(journalJoinRow(created))), nil
}

func (r *journalDAO) GetByID(ctx context.Context, s spec.JournalLookup, tx ...*sql.Tx) (*dto.JournalResponse, error) {
	queries := genQueries(r.db, tx)

	row, err := queries.GetJournalByID(ctx, s.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal: %w", err)
	}

	journal := toJournalResponse(row)
	if err := applyJournalViewerFollow(ctx, queries, &journal, s.ViewerID); err != nil {
		return nil, err
	}

	return &journal, nil
}

func (r *journalDAO) List(ctx context.Context, q spec.JournalQuery, tx ...*sql.Tx) ([]dto.JournalResponse, int, error) {
	queries := genQueries(r.db, tx)

	var authorID *uuid.UUID
	if q.AuthorID != uuid.Nil {
		authorID = new(q.AuthorID)
	}

	exclude := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountJournals(ctx, sqlcgen.CountJournalsParams{
		Work:            q.Work,
		AuthorID:        authorID,
		Search:          q.Search,
		IncludeArchived: q.IncludeArchived,
		ExcludeUserIds:  exclude,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count journals: %w", err)
	}

	params := sqlcgen.ListJournalsByNewParams{
		Work:            q.Work,
		AuthorID:        authorID,
		Search:          q.Search,
		IncludeArchived: q.IncludeArchived,
		ExcludeUserIds:  exclude,
		RowOffset:       int32(q.Offset),
		RowLimit:        int32(q.Limit),
	}

	var journals []dto.JournalResponse
	switch q.Sort {
	case "old":
		listed, err := queries.ListJournalsByOld(ctx, sqlcgen.ListJournalsByOldParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list journals: %w", err)
		}

		journals = journalResponses(listed)
	case "recently_active":
		listed, err := queries.ListJournalsByRecentlyActive(ctx, sqlcgen.ListJournalsByRecentlyActiveParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list journals: %w", err)
		}

		journals = journalResponses(listed)
	case "most_followed":
		listed, err := queries.ListJournalsByMostFollowed(ctx, sqlcgen.ListJournalsByMostFollowedParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list journals: %w", err)
		}

		journals = journalResponses(listed)
	default:
		listed, err := queries.ListJournalsByNew(ctx, params)
		if err != nil {
			return nil, 0, fmt.Errorf("list journals: %w", err)
		}

		journals = journalResponses(listed)
	}

	for i := range journals {
		if err := applyJournalViewerFollow(ctx, queries, &journals[i], q.ViewerID); err != nil {
			return nil, 0, err
		}
	}

	return journals, int(total), nil
}

func (r *journalDAO) Update(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).UpdateJournal(ctx, sqlcgen.UpdateJournalParams{
		Title:  s.Title,
		Work:   s.Work,
		ID:     s.ID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update journal: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("journal not found or not owned")
	}

	return nil
}

func (r *journalDAO) UpdateAsAdmin(ctx context.Context, s spec.JournalUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateJournalAsAdmin(ctx, sqlcgen.UpdateJournalAsAdminParams{
		Title: s.Title,
		Work:  s.Work,
		ID:    s.ID,
	})
	if err != nil {
		return fmt.Errorf("admin update journal: %w", err)
	}

	return nil
}

func (r *journalDAO) ListEntryIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).ListJournalEntryIDs(ctx, journalID)
	if err != nil {
		return nil, fmt.Errorf("list journal entry ids: %w", err)
	}

	return ids, nil
}

func (r *journalDAO) ListEntryCommentIDs(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).ListJournalEntryCommentIDs(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("list entry comment ids: %w", err)
	}

	return ids, nil
}

func (r *journalDAO) GetTitle(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (string, error) {
	title, err := genQueries(r.db, tx).GetJournalTitle(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get journal title: %w", err)
	}

	return title, nil
}

func (r *journalDAO) IsArchived(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error) {
	archivedAt, err := genQueries(r.db, tx).GetJournalArchivedAt(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check archived: %w", err)
	}

	return archivedAt.Valid, nil
}

func (r *journalDAO) CountUserJournalsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUserJournalsToday(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count user journals today: %w", err)
	}

	return int(count), nil
}

func (r *journalDAO) UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UpdateJournalLastAuthorActivity(ctx, id); err != nil {
		return fmt.Errorf("update last author activity: %w", err)
	}

	return nil
}

func (r *journalDAO) ArchiveStale(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).ArchiveStaleJournals(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("archive stale journals: %w", err)
	}

	return ids, nil
}

func (r *journalDAO) SetPaused(ctx context.Context, s spec.JournalPause, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).SetJournalPaused(ctx, sqlcgen.SetJournalPausedParams{
		PausedAt: sql.NullTime{Time: time.Now().UTC(), Valid: s.Paused},
		ID:       s.ID,
		UserID:   s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set journal paused: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("journal not found or not owned")
	}

	return nil
}

func (r *journalDAO) Follow(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).FollowJournal(ctx, sqlcgen.FollowJournalParams{
		UserID:    s.UserID,
		JournalID: s.JournalID,
	})
	if err != nil {
		return fmt.Errorf("follow journal: %w", err)
	}

	return nil
}

func (r *journalDAO) Unfollow(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UnfollowJournal(ctx, sqlcgen.UnfollowJournalParams{
		UserID:    s.UserID,
		JournalID: s.JournalID,
	})
	if err != nil {
		return fmt.Errorf("unfollow journal: %w", err)
	}

	return nil
}

func (r *journalDAO) IsFollower(ctx context.Context, s spec.JournalFollow, tx ...*sql.Tx) (bool, error) {
	exists, err := genQueries(r.db, tx).IsJournalFollower(ctx, sqlcgen.IsJournalFollowerParams{
		UserID:    s.UserID,
		JournalID: s.JournalID,
	})
	if err != nil {
		return false, fmt.Errorf("check journal follower: %w", err)
	}

	return exists, nil
}

func (r *journalDAO) GetFollowerIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).GetJournalFollowerIDs(ctx, journalID)
	if err != nil {
		return nil, fmt.Errorf("get follower ids: %w", err)
	}

	return ids, nil
}

func (r *journalDAO) GetFollowerCount(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).GetJournalFollowerCount(ctx, journalID)
	if err != nil {
		return 0, fmt.Errorf("get follower count: %w", err)
	}

	return int(count), nil
}

func (r *journalDAO) ListFollowedByUser(ctx context.Context, q spec.JournalFollowedQuery, tx ...*sql.Tx) ([]dto.JournalResponse, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountFollowedJournals(ctx, q.FollowerID)
	if err != nil {
		return nil, 0, fmt.Errorf("count followed journals: %w", err)
	}

	rows, err := queries.ListFollowedJournals(ctx, sqlcgen.ListFollowedJournalsParams{
		UserID: q.FollowerID,
		Limit:  int32(q.Limit),
		Offset: int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list followed journals: %w", err)
	}

	var journals []dto.JournalResponse
	for _, row := range rows {
		journal := toJournalResponse(journalJoinRow(row))
		if err := applyJournalViewerFollow(ctx, queries, &journal, q.ViewerID); err != nil {
			return nil, 0, err
		}

		journals = append(journals, journal)
	}

	return journals, int(total), nil
}

func (r *journalDAO) CreateEntry(ctx context.Context, s spec.NewJournalEntry, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	created, err := genQueries(r.db, tx).CreateJournalEntry(ctx, sqlcgen.CreateJournalEntryParams{
		JournalID:   s.JournalID,
		EntryNumber: int32(s.EntryNumber),
		Title:       journalNullString(s.Title),
		Body:        s.Body,
		WordCount:   int32(s.WordCount),
		IsDraft:     s.IsDraft,
	})
	if err != nil {
		return nil, fmt.Errorf("create journal entry: %w", err)
	}

	return new(toJournalEntryRow(journalEntryJoinRow(created))), nil
}

func (r *journalDAO) UpdateEntry(ctx context.Context, s spec.JournalEntryUpdate, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).UpdateJournalEntry(ctx, sqlcgen.UpdateJournalEntryParams{
		Title:     journalNullString(s.Title),
		Body:      s.Body,
		WordCount: int32(s.WordCount),
		IsDraft:   s.IsDraft,
		ID:        s.ID,
	})
	if err != nil {
		return fmt.Errorf("update journal entry: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("entry not found")
	}

	return nil
}

func (r *journalDAO) DeleteEntry(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).DeleteJournalEntry(ctx, id)
	if err != nil {
		return fmt.Errorf("delete journal entry: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("entry not found")
	}

	return nil
}

func (r *journalDAO) GetEntry(ctx context.Context, s spec.JournalEntryLookup, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	row, err := genQueries(r.db, tx).GetJournalEntry(ctx, sqlcgen.GetJournalEntryParams{
		JournalID:   s.JournalID,
		EntryNumber: int32(s.EntryNumber),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal entry: %w", err)
	}

	entry := toJournalEntryRow(journalEntryJoinRow{
		ID:          row.ID,
		JournalID:   row.JournalID,
		EntryNumber: row.EntryNumber,
		Title:       row.Title,
		Body:        row.Body,
		WordCount:   row.WordCount,
		IsDraft:     row.IsDraft,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	})
	entry.HasPrev = row.HasPrev
	entry.HasNext = row.HasNext

	return &entry, nil
}

func (r *journalDAO) GetEntryByID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (*model.JournalEntryRow, error) {
	row, err := genQueries(r.db, tx).GetJournalEntryByID(ctx, entryID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get journal entry by id: %w", err)
	}

	return new(toJournalEntryRow(row)), nil
}

func (r *journalDAO) ListEntries(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]model.JournalEntrySummaryRow, error) {
	rows, err := genQueries(r.db, tx).ListJournalEntries(ctx, journalID)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	var entries []model.JournalEntrySummaryRow
	for _, row := range rows {
		entries = append(entries, model.JournalEntrySummaryRow{
			ID:          row.ID,
			EntryNumber: int(row.EntryNumber),
			Title:       journalStringPtr(row.Title),
			WordCount:   int(row.WordCount),
			IsDraft:     row.IsDraft,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return entries, nil
}

func (r *journalDAO) GetNextEntryNumber(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error) {
	next, err := genQueries(r.db, tx).GetNextJournalEntryNumber(ctx, journalID)
	if err != nil {
		return 0, fmt.Errorf("get next entry number: %w", err)
	}

	return int(next), nil
}

func (r *journalDAO) GetEntryJournalID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	id, err := genQueries(r.db, tx).GetJournalEntryJournalID(ctx, entryID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get entry journal id: %w", err)
	}

	return id, nil
}

func (r *journalDAO) GetEntryAuthorID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	userID, err := genQueries(r.db, tx).GetJournalEntryAuthorID(ctx, entryID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get entry author id: %w", err)
	}

	return userID, nil
}

func (r *journalDAO) CreateComment(ctx context.Context, s spec.NewJournalComment, tx ...*sql.Tx) (*model.CommentRow, error) {
	created, err := genQueries(r.db, tx).CreateJournalCommentWithEntry(ctx, sqlcgen.CreateJournalCommentWithEntryParams{
		JournalID: s.JournalID,
		EntryID:   s.EntryID,
		ParentID:  s.ParentID,
		UserID:    s.UserID,
		Body:      s.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("create journal comment: %w", err)
	}

	comment := toJournalCommentRow(journalCommentJoinRow{
		ID:          created.ID,
		EntityID:    created.EntityID,
		EntryID:     created.EntryID,
		ParentID:    created.ParentID,
		UserID:      created.UserID,
		Body:        created.Body,
		CreatedAt:   created.CreatedAt,
		UpdatedAt:   created.UpdatedAt,
		Username:    created.Username,
		DisplayName: created.DisplayName,
		AvatarUrl:   created.AvatarUrl,
		AuthorRole:  created.AuthorRole,
		LikeCount:   created.LikeCount,
		UserLiked:   created.UserLiked,
	})
	comment.AuthorBanned = created.AuthorBanned

	return &comment, nil
}

func (r *journalDAO) GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error) {
	queries := genQueries(r.db, tx)
	exclude := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountJournalRootComments(ctx, sqlcgen.CountJournalRootCommentsParams{
		JournalID: q.TargetID,
		Column2:   exclude,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count journal comments: %w", err)
	}

	rows, err := queries.ListJournalRootComments(ctx, sqlcgen.ListJournalRootCommentsParams{
		UserID:    q.ViewerID,
		JournalID: q.TargetID,
		Column3:   exclude,
		Limit:     int32(q.Limit),
		Offset:    int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get journal comments: %w", err)
	}

	var comments []model.CommentRow
	for _, row := range rows {
		comments = append(comments, toJournalCommentRow(row))
	}

	return comments, int(total), nil
}

func (r *journalDAO) GetEntryComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error) {
	queries := genQueries(r.db, tx)
	exclude := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountJournalEntryComments(ctx, sqlcgen.CountJournalEntryCommentsParams{
		Column1: q.TargetID,
		Column2: exclude,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count entry comments: %w", err)
	}

	rows, err := queries.ListJournalEntryComments(ctx, sqlcgen.ListJournalEntryCommentsParams{
		UserID:  q.ViewerID,
		Column2: q.TargetID,
		Column3: exclude,
		Limit:   int32(q.Limit),
		Offset:  int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get entry comments: %w", err)
	}

	var comments []model.CommentRow
	for _, row := range rows {
		comments = append(comments, toJournalCommentRow(journalCommentJoinRow(row)))
	}

	return comments, int(total), nil
}

func (r *journalDAO) GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*int, error) {
	entryNumber, err := genQueries(r.db, tx).GetJournalCommentEntryNumber(ctx, commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get comment entry number: %w", err)
	}

	return journalIntPtr(entryNumber), nil
}
