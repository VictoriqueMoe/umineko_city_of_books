package dao

import (
	"context"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	journalCommentTable struct {
		journalCommentQuerier
	}

	journalRootCommentQuerier struct {
		journalCommentTable
	}

	journalEntryCommentQuerier struct {
		journalCommentTable
	}
)

func (journalCommentTable) Target(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (spec.CommentTarget, error) {
	journalID, err := q.GetJournalCommentEntityID(ctx, id)
	if err != nil {
		return spec.CommentTarget{}, err
	}

	entryNumber, err := q.GetJournalCommentEntryNumber(ctx, id)
	if err != nil {
		return spec.CommentTarget{}, err
	}

	if !entryNumber.Valid {
		return spec.CommentTarget{Kind: spec.CommentKindJournal, ID: journalID}, nil
	}

	return spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: journalID, EntryNumber: int(entryNumber.Int32)}, nil
}

func (journalRootCommentQuerier) Count(ctx context.Context, q *sqlcgen.Queries, target spec.CommentTarget, exclude []uuid.UUID) (int64, error) {
	return q.CountJournalRootComments(ctx, sqlcgen.CountJournalRootCommentsParams{
		JournalID: target.ID,
		Column2:   joinUUIDs(exclude),
	})
}

func (journalRootCommentQuerier) List(ctx context.Context, q *sqlcgen.Queries, s spec.CommentQuery) ([]model.CommentRow, error) {
	rows, err := q.ListJournalRootComments(ctx, sqlcgen.ListJournalRootCommentsParams{
		UserID:    s.ViewerID,
		JournalID: s.Target.ID,
		Column3:   joinUUIDs(s.ExcludeUserIDs),
		Limit:     int32(s.Limit),
		Offset:    int32(s.Offset),
	})
	if err != nil {
		return nil, err
	}

	out := make([]model.CommentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toJournalCommentRow(row))
	}

	return out, nil
}

func (journalEntryCommentQuerier) Create(ctx context.Context, q *sqlcgen.Queries, s spec.NewComment) (*model.CommentRow, error) {
	entryID, err := journalEntryID(ctx, q, s.Target)
	if err != nil {
		return nil, err
	}

	created, err := q.CreateJournalCommentWithEntry(ctx, sqlcgen.CreateJournalCommentWithEntryParams{
		JournalID: s.Target.ID,
		EntryID:   &entryID,
		ParentID:  s.ParentID,
		UserID:    s.UserID,
		Body:      s.Body,
	})
	if err != nil {
		return nil, err
	}

	return new(toJournalCommentRow(journalCommentJoinRow(created))), nil
}

func (journalEntryCommentQuerier) Count(ctx context.Context, q *sqlcgen.Queries, target spec.CommentTarget, exclude []uuid.UUID) (int64, error) {
	entryID, err := journalEntryID(ctx, q, target)
	if err != nil {
		return 0, err
	}

	return q.CountJournalEntryComments(ctx, sqlcgen.CountJournalEntryCommentsParams{
		Column1: entryID,
		Column2: joinUUIDs(exclude),
	})
}

func (journalEntryCommentQuerier) List(ctx context.Context, q *sqlcgen.Queries, s spec.CommentQuery) ([]model.CommentRow, error) {
	entryID, err := journalEntryID(ctx, q, s.Target)
	if err != nil {
		return nil, err
	}

	rows, err := q.ListJournalEntryComments(ctx, sqlcgen.ListJournalEntryCommentsParams{
		UserID:  s.ViewerID,
		Column2: entryID,
		Column3: joinUUIDs(s.ExcludeUserIDs),
		Limit:   int32(s.Limit),
		Offset:  int32(s.Offset),
	})
	if err != nil {
		return nil, err
	}

	out := make([]model.CommentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toJournalCommentRow(journalCommentJoinRow(row)))
	}

	return out, nil
}

func (journalEntryCommentQuerier) Paths(ctx context.Context, q *sqlcgen.Queries, target spec.CommentTarget) ([]string, error) {
	entryID, err := journalEntryID(ctx, q, target)
	if err != nil {
		return nil, err
	}

	rows, err := q.CollectJournalEntryCommentMediaPaths(ctx, entryID)
	if err != nil {
		return nil, err
	}

	out := make([]mediaPathRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaPathRow(row))
	}

	return collectMediaPaths(out), nil
}

func journalEntryID(ctx context.Context, q *sqlcgen.Queries, target spec.CommentTarget) (uuid.UUID, error) {
	entry, err := q.GetJournalEntry(ctx, sqlcgen.GetJournalEntryParams{
		JournalID:   target.ID,
		EntryNumber: int32(target.EntryNumber),
	})
	if err != nil {
		return uuid.Nil, err
	}

	return entry.ID, nil
}
