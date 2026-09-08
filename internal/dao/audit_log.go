package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model/spec"
)

type (
	AuditLogDAO interface {
		Create(ctx context.Context, entry audit.NewEntry, tx ...*sql.Tx) error
		CreateSystem(ctx context.Context, entry audit.NewEntry, tx ...*sql.Tx) error
		List(ctx context.Context, s spec.AuditLogListing, tx ...*sql.Tx) ([]audit.Entry, int, error)
		ListForUser(ctx context.Context, s spec.AuditLogUserListing, tx ...*sql.Tx) ([]audit.Entry, int, error)
	}

	auditLogDAO struct {
		db *sql.DB
	}

	auditLogJoinRow = sqlcgen.ListAuditLogRow
)

func toAuditEntry(row auditLogJoinRow) audit.Entry {
	out := audit.Entry{
		ID:              int(row.ID),
		ActorName:       row.ActorName,
		Action:          audit.Action(row.Action),
		TargetType:      audit.TargetType(row.TargetType),
		TargetID:        row.TargetID.String,
		Details:         row.Details,
		CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
		SubjectID:       row.SubjectID,
		SubjectName:     row.SubjectName,
		SubjectUsername: row.SubjectUsername,
	}

	if row.ActorID != nil {
		out.ActorID = *row.ActorID
	}

	return out
}

func (r *auditLogDAO) Create(ctx context.Context, entry audit.NewEntry, tx ...*sql.Tx) error {
	if err := r.insert(ctx, &entry.ActorID, entry, tx); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

func (r *auditLogDAO) CreateSystem(ctx context.Context, entry audit.NewEntry, tx ...*sql.Tx) error {
	if err := r.insert(ctx, nil, entry, tx); err != nil {
		return fmt.Errorf("create system audit log: %w", err)
	}

	return nil
}

func (r *auditLogDAO) insert(ctx context.Context, actorID *uuid.UUID, entry audit.NewEntry, tx []*sql.Tx) error {
	var subjectID *uuid.UUID
	if entry.SubjectID != uuid.Nil {
		subjectID = &entry.SubjectID
	}

	return genQueries(r.db, tx).CreateAuditLogEntry(ctx, sqlcgen.CreateAuditLogEntryParams{
		ActorID:    actorID,
		Action:     string(entry.Action),
		TargetType: string(entry.TargetType),
		TargetID:   sql.NullString{String: entry.TargetID, Valid: true},
		Details:    entry.Details,
		SubjectID:  subjectID,
	})
}

func (r *auditLogDAO) ListForUser(ctx context.Context, s spec.AuditLogUserListing, tx ...*sql.Tx) ([]audit.Entry, int, error) {
	queries := genQueries(r.db, tx)
	id := s.UserID.String()

	total, err := queries.CountAuditLogForUser(ctx, id)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit log for user: %w", err)
	}

	rows, err := queries.ListAuditLogForUser(ctx, sqlcgen.ListAuditLogForUserParams{
		Column1: id,
		Limit:   int32(s.Page.Limit()),
		Offset:  int32(s.Page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log for user: %w", err)
	}

	var entries []audit.Entry
	for _, row := range rows {
		entries = append(entries, toAuditEntry(auditLogJoinRow(row)))
	}

	return entries, int(total), nil
}

func (r *auditLogDAO) List(ctx context.Context, s spec.AuditLogListing, tx ...*sql.Tx) ([]audit.Entry, int, error) {
	queries := genQueries(r.db, tx)
	action := string(s.Action)

	total, err := queries.CountAuditLog(ctx, action)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	rows, err := queries.ListAuditLog(ctx, sqlcgen.ListAuditLogParams{
		Column1: action,
		Limit:   int32(s.Page.Limit()),
		Offset:  int32(s.Page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log: %w", err)
	}

	var entries []audit.Entry
	for _, row := range rows {
		entries = append(entries, toAuditEntry(row))
	}

	return entries, int(total), nil
}
