package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ChatWatchPartyDAO interface {
		CreateSession(ctx context.Context, row model.ChatWatchPartySessionRow, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error)
		GetByID(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error)
		ListActiveByRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error)
		EndSession(ctx context.Context, s spec.WatchPartySessionEnd, tx ...*sql.Tx) error
		SetControllerID(ctx context.Context, s spec.WatchPartyControllerUpdate, tx ...*sql.Tx) error

		UpsertParticipant(ctx context.Context, s spec.WatchPartyParticipantUpsert, tx ...*sql.Tx) error
		SetParticipantIdentifier(ctx context.Context, s spec.WatchPartyIdentifierUpdate, tx ...*sql.Tx) error
		MarkParticipantLeft(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) error
		MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) error
		GetActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartyParticipantRow, error)
		GetParticipant(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) (*model.ChatWatchPartyParticipantRow, error)
		SetParticipantControl(ctx context.Context, s spec.WatchPartyControlUpdate, tx ...*sql.Tx) error
		CountActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (int, error)

		ListIdleActiveSessions(ctx context.Context, idleBefore string, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error)
	}

	chatWatchPartyDAO struct {
		db *sql.DB
	}

	watchPartySessionRow = sqlcgen.GetWatchPartySessionByIDRow

	watchPartyParticipantRow = sqlcgen.GetWatchPartyParticipantRow
)

func watchPartyNullTime(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}

	return sql.NullString{String: nt.Time.Format(time.RFC3339Nano), Valid: true}
}

func watchPartyUUID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}

	return *id
}

func toWatchPartySessionRow(row watchPartySessionRow) model.ChatWatchPartySessionRow {
	return model.ChatWatchPartySessionRow{
		ID:                  row.ID,
		RoomID:              row.RoomID,
		StartedBy:           watchPartyUUID(row.StartedBy),
		ControllerID:        watchPartyUUID(row.ControllerID),
		HyperbeamSessionID:  row.HyperbeamSessionID,
		HyperbeamAdminToken: row.HyperbeamAdminToken,
		EmbedURL:            row.EmbedUrl,
		VMBaseURL:           row.VmBaseUrl,
		Title:               row.Title,
		Type:                row.Type,
		StartURL:            row.StartUrl,
		Region:              row.Region,
		Status:              row.Status,
		StartedAt:           row.StartedAt.Format(time.RFC3339Nano),
		EndedAt:             watchPartyNullTime(row.EndedAt),
		EndedReason:         row.EndedReason,
	}
}

func toWatchPartyParticipantRow(row watchPartyParticipantRow) model.ChatWatchPartyParticipantRow {
	return model.ChatWatchPartyParticipantRow{
		SessionID:           row.SessionID,
		UserID:              row.UserID,
		Username:            row.Username,
		DisplayName:         row.DisplayName,
		AvatarURL:           row.AvatarUrl,
		HasControl:          row.HasControl,
		HyperbeamIdentifier: row.HyperbeamIdentifier,
		JoinedAt:            row.JoinedAt.Format(time.RFC3339Nano),
		LeftAt:              watchPartyNullTime(row.LeftAt),
	}
}

func (r *chatWatchPartyDAO) CreateSession(ctx context.Context, row model.ChatWatchPartySessionRow, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error) {
	created, err := genQueries(r.db, tx).CreateWatchPartySession(ctx, sqlcgen.CreateWatchPartySessionParams{
		RoomID:              row.RoomID,
		StartedBy:           &row.StartedBy,
		ControllerID:        &row.ControllerID,
		HyperbeamSessionID:  row.HyperbeamSessionID,
		HyperbeamAdminToken: row.HyperbeamAdminToken,
		EmbedUrl:            row.EmbedURL,
		VmBaseUrl:           row.VMBaseURL,
		Title:               row.Title,
		Type:                row.Type,
		StartUrl:            row.StartURL,
		Region:              row.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create watch party session: %w", err)
	}

	return new(toWatchPartySessionRow(watchPartySessionRow(created))), nil
}

func (r *chatWatchPartyDAO) ListActiveByRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error) {
	rows, err := genQueries(r.db, tx).ListActiveWatchPartiesByRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list active watch parties: %w", err)
	}

	var result []model.ChatWatchPartySessionRow
	for _, row := range rows {
		result = append(result, toWatchPartySessionRow(watchPartySessionRow(row)))
	}

	return result, nil
}

func (r *chatWatchPartyDAO) GetByID(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error) {
	row, err := genQueries(r.db, tx).GetWatchPartySessionByID(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan watch party session: %w", err)
	}

	return new(toWatchPartySessionRow(row)), nil
}

func (r *chatWatchPartyDAO) EndSession(ctx context.Context, s spec.WatchPartySessionEnd, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).EndWatchPartySession(ctx, sqlcgen.EndWatchPartySessionParams{
		ID:          s.SessionID,
		EndedReason: sql.NullString{String: s.Reason, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("end watch party session: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) SetControllerID(ctx context.Context, s spec.WatchPartyControllerUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetWatchPartyControllerID(ctx, sqlcgen.SetWatchPartyControllerIDParams{
		ID:           s.SessionID,
		ControllerID: &s.ControllerID,
	})
	if err != nil {
		return fmt.Errorf("set watch party controller: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) UpsertParticipant(ctx context.Context, s spec.WatchPartyParticipantUpsert, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertWatchPartyParticipant(ctx, sqlcgen.UpsertWatchPartyParticipantParams{
		SessionID:           s.SessionID,
		UserID:              s.UserID,
		HasControl:          s.HasControl,
		HyperbeamIdentifier: s.Identifier,
	})
	if err != nil {
		return fmt.Errorf("upsert watch party participant: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) SetParticipantIdentifier(ctx context.Context, s spec.WatchPartyIdentifierUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetWatchPartyParticipantIdentifier(ctx, sqlcgen.SetWatchPartyParticipantIdentifierParams{
		SessionID:           s.SessionID,
		UserID:              s.UserID,
		HyperbeamIdentifier: s.Identifier,
	})
	if err != nil {
		return fmt.Errorf("set watch party participant identifier: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) MarkParticipantLeft(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).MarkWatchPartyParticipantLeft(ctx, sqlcgen.MarkWatchPartyParticipantLeftParams{
		SessionID: s.SessionID,
		UserID:    s.UserID,
	})
	if err != nil {
		return fmt.Errorf("mark watch party participant left: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkAllWatchPartyParticipantsLeft(ctx, sessionID); err != nil {
		return fmt.Errorf("mark all watch party participants left: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) GetActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartyParticipantRow, error) {
	rows, err := genQueries(r.db, tx).GetActiveWatchPartyParticipants(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list watch party participants: %w", err)
	}

	var result []model.ChatWatchPartyParticipantRow
	for _, row := range rows {
		result = append(result, toWatchPartyParticipantRow(watchPartyParticipantRow(row)))
	}

	return result, nil
}

func (r *chatWatchPartyDAO) GetParticipant(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) (*model.ChatWatchPartyParticipantRow, error) {
	row, err := genQueries(r.db, tx).GetWatchPartyParticipant(ctx, sqlcgen.GetWatchPartyParticipantParams{
		SessionID: s.SessionID,
		UserID:    s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan watch party participant: %w", err)
	}

	return new(toWatchPartyParticipantRow(row)), nil
}

func (r *chatWatchPartyDAO) SetParticipantControl(ctx context.Context, s spec.WatchPartyControlUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetWatchPartyParticipantControl(ctx, sqlcgen.SetWatchPartyParticipantControlParams{
		SessionID:  s.SessionID,
		UserID:     s.UserID,
		HasControl: s.HasControl,
	})
	if err != nil {
		return fmt.Errorf("set watch party participant control: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) CountActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (int, error) {
	total, err := genQueries(r.db, tx).CountActiveWatchPartyParticipants(ctx, sessionID)
	if err != nil {
		return 0, fmt.Errorf("count watch party participants: %w", err)
	}

	return int(total), nil
}

func (r *chatWatchPartyDAO) ListIdleActiveSessions(ctx context.Context, idleBefore string, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error) {
	rows, err := genQueries(r.db, tx).ListIdleActiveWatchPartySessions(ctx, idleBefore)
	if err != nil {
		return nil, fmt.Errorf("list idle watch party sessions: %w", err)
	}

	var result []model.ChatWatchPartySessionRow
	for _, row := range rows {
		result = append(result, toWatchPartySessionRow(watchPartySessionRow(row)))
	}

	return result, nil
}
