package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatBannedWordDAO interface {
		Create(ctx context.Context, s spec.ChatBannedWordSpec, tx ...*sql.Tx) (*model.ChatBannedWordRow, error)
		Update(ctx context.Context, s spec.ChatBannedWordUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatBannedWordRow, error)
		ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error)
		ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error)
	}

	chatBannedWordDAO struct {
		db *sql.DB
	}

	chatBannedWordJoinRow = sqlcgen.GetChatBannedWordByIDRow
)

func toChatBannedWordRow(row chatBannedWordJoinRow) model.ChatBannedWordRow {
	return model.ChatBannedWordRow{
		ID:            row.ID,
		Scope:         row.Scope,
		RoomID:        row.RoomID,
		Pattern:       row.Pattern,
		MatchMode:     row.MatchMode,
		CaseSensitive: row.CaseSensitive,
		Action:        row.Action,
		CreatedBy:     row.CreatedBy,
		CreatedByName: row.CreatedByName,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (r *chatBannedWordDAO) Create(ctx context.Context, s spec.ChatBannedWordSpec, tx ...*sql.Tx) (*model.ChatBannedWordRow, error) {
	created, err := genQueries(r.db, tx).CreateChatBannedWord(ctx, sqlcgen.CreateChatBannedWordParams{
		Scope:         s.Scope,
		RoomID:        s.RoomID,
		Pattern:       s.Pattern,
		MatchMode:     s.MatchMode,
		CaseSensitive: s.CaseSensitive,
		Action:        s.Action,
		CreatedBy:     s.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create banned word: %w", err)
	}

	return new(toChatBannedWordRow(chatBannedWordJoinRow(created))), nil
}

func (r *chatBannedWordDAO) Update(ctx context.Context, s spec.ChatBannedWordUpdate, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).UpdateChatBannedWord(ctx, sqlcgen.UpdateChatBannedWordParams{
		Pattern:       s.Pattern,
		MatchMode:     s.MatchMode,
		CaseSensitive: s.CaseSensitive,
		Action:        s.Action,
		ID:            s.ID,
	})
	if err != nil {
		return fmt.Errorf("update banned word: %w", err)
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *chatBannedWordDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteChatBannedWord(ctx, id); err != nil {
		return fmt.Errorf("delete banned word: %w", err)
	}

	return nil
}

func (r *chatBannedWordDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatBannedWordRow, error) {
	row, err := genQueries(r.db, tx).GetChatBannedWordByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get banned word: %w", err)
	}

	return new(toChatBannedWordRow(row)), nil
}

func (r *chatBannedWordDAO) ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	rows, err := genQueries(r.db, tx).ListGlobalChatBannedWords(ctx)
	if err != nil {
		return nil, fmt.Errorf("query banned words: %w", err)
	}

	var result []model.ChatBannedWordRow
	for _, row := range rows {
		result = append(result, toChatBannedWordRow(chatBannedWordJoinRow(row)))
	}

	return result, nil
}

func (r *chatBannedWordDAO) ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	rows, err := genQueries(r.db, tx).ListChatBannedWordsForRoom(ctx, &roomID)
	if err != nil {
		return nil, fmt.Errorf("query banned words: %w", err)
	}

	var result []model.ChatBannedWordRow
	for _, row := range rows {
		result = append(result, toChatBannedWordRow(chatBannedWordJoinRow(row)))
	}

	return result, nil
}

func (r *chatBannedWordDAO) ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	rows, err := genQueries(r.db, tx).ListApplicableChatBannedWords(ctx, &roomID)
	if err != nil {
		return nil, fmt.Errorf("query banned words: %w", err)
	}

	var result []model.ChatBannedWordRow
	for _, row := range rows {
		result = append(result, toChatBannedWordRow(chatBannedWordJoinRow(row)))
	}

	return result, nil
}
