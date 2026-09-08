package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatbotBasePromptDAO interface {
		List(ctx context.Context, tx ...*sql.Tx) ([]model.ChatbotBasePrompt, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error)
		Create(ctx context.Context, s spec.NewChatbotBasePrompt, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error)
		Update(ctx context.Context, s spec.ChatbotBasePromptUpdate, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error)
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
	}

	chatbotBasePromptDAO struct {
		db *sql.DB
	}

	basePromptRow = sqlcgen.GetChatbotBasePromptByIDRow
)

var (
	ErrBasePromptNotFound = errors.New("base prompt not found")
	ErrBasePromptNameUsed = errors.New("that base prompt name is already taken")
	ErrBasePromptInUse    = errors.New("that base prompt is still used by a chatbot")
)

func toBasePrompt(row basePromptRow) model.ChatbotBasePrompt {
	return model.ChatbotBasePrompt{
		ID:        row.ID,
		Name:      row.Name,
		Prompt:    row.Prompt,
		BotCount:  int(row.BotCount),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func isUniqueViolation(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "23505"
	}

	return false
}

func isForeignKeyViolation(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "23503"
	}

	return false
}

func (r *chatbotBasePromptDAO) List(ctx context.Context, tx ...*sql.Tx) ([]model.ChatbotBasePrompt, error) {
	rows, err := genQueries(r.db, tx).ListChatbotBasePrompts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list base prompts: %w", err)
	}

	prompts := make([]model.ChatbotBasePrompt, 0, len(rows))
	for _, row := range rows {
		prompts = append(prompts, toBasePrompt(basePromptRow(row)))
	}

	return prompts, nil
}

func (r *chatbotBasePromptDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	row, err := genQueries(r.db, tx).GetChatbotBasePromptByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBasePromptNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get base prompt: %w", err)
	}

	return new(toBasePrompt(row)), nil
}

func (r *chatbotBasePromptDAO) Create(ctx context.Context, s spec.NewChatbotBasePrompt, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	created, err := genQueries(r.db, tx).CreateChatbotBasePrompt(ctx, sqlcgen.CreateChatbotBasePromptParams{
		Name:   s.Name,
		Prompt: s.Prompt,
	})
	if isUniqueViolation(err) {
		return nil, ErrBasePromptNameUsed
	}
	if err != nil {
		return nil, fmt.Errorf("create base prompt: %w", err)
	}

	return new(toBasePrompt(basePromptRow(created))), nil
}

func (r *chatbotBasePromptDAO) Update(ctx context.Context, s spec.ChatbotBasePromptUpdate, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	updated, err := genQueries(r.db, tx).UpdateChatbotBasePrompt(ctx, sqlcgen.UpdateChatbotBasePromptParams{
		ID:     s.ID,
		Name:   s.Name,
		Prompt: s.Prompt,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBasePromptNotFound
	}
	if isUniqueViolation(err) {
		return nil, ErrBasePromptNameUsed
	}
	if err != nil {
		return nil, fmt.Errorf("update base prompt: %w", err)
	}

	return new(toBasePrompt(basePromptRow(updated))), nil
}

func (r *chatbotBasePromptDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteChatbotBasePrompt(ctx, id)
	if isForeignKeyViolation(err) {
		return ErrBasePromptInUse
	}
	if err != nil {
		return fmt.Errorf("delete base prompt: %w", err)
	}

	if affected == 0 {
		return ErrBasePromptNotFound
	}

	return nil
}
