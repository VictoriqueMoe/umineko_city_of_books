package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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
)

var (
	ErrBasePromptNotFound = errors.New("base prompt not found")
	ErrBasePromptNameUsed = errors.New("that base prompt name is already taken")
	ErrBasePromptInUse    = errors.New("that base prompt is still used by a chatbot")
)

const basePromptSelectBase = `
	SELECT b.id, b.name, b.prompt, b.created_at, b.updated_at,
		(SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = b.id)
	FROM chatbot_base_prompts b`

func scanBasePrompt(row interface{ Scan(...any) error }, prompt *model.ChatbotBasePrompt) error {
	return row.Scan(&prompt.ID, &prompt.Name, &prompt.Prompt, &prompt.CreatedAt, &prompt.UpdatedAt, &prompt.BotCount)
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
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, basePromptSelectBase+` ORDER BY b.name`)
	if err != nil {
		return nil, fmt.Errorf("list base prompts: %w", err)
	}
	defer rows.Close()

	prompts := make([]model.ChatbotBasePrompt, 0)
	for rows.Next() {
		var prompt model.ChatbotBasePrompt
		if err := scanBasePrompt(rows, &prompt); err != nil {
			return nil, fmt.Errorf("scan base prompt: %w", err)
		}

		prompts = append(prompts, prompt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate base prompts: %w", err)
	}

	return prompts, nil
}

func (r *chatbotBasePromptDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	var prompt model.ChatbotBasePrompt
	err := scanBasePrompt(txOrDB(r.db, tx).QueryRowContext(ctx, basePromptSelectBase+` WHERE b.id = $1`, id), &prompt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBasePromptNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get base prompt: %w", err)
	}

	return &prompt, nil
}

func (r *chatbotBasePromptDAO) Create(ctx context.Context, s spec.NewChatbotBasePrompt, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	var created model.ChatbotBasePrompt
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chatbot_base_prompts (id, name, prompt)
		 VALUES (gen_random_uuid(), $1, $2)
		 RETURNING id, name, prompt, created_at, updated_at, 0`,
		s.Name, s.Prompt,
	).Scan(&created.ID, &created.Name, &created.Prompt, &created.CreatedAt, &created.UpdatedAt, &created.BotCount)
	if isUniqueViolation(err) {
		return nil, ErrBasePromptNameUsed
	}
	if err != nil {
		return nil, fmt.Errorf("create base prompt: %w", err)
	}

	return &created, nil
}

func (r *chatbotBasePromptDAO) Update(ctx context.Context, s spec.ChatbotBasePromptUpdate, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	var updated model.ChatbotBasePrompt
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`UPDATE chatbot_base_prompts SET name = $2, prompt = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, prompt, created_at, updated_at,
		   (SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = chatbot_base_prompts.id)`,
		s.ID, s.Name, s.Prompt,
	).Scan(&updated.ID, &updated.Name, &updated.Prompt, &updated.CreatedAt, &updated.UpdatedAt, &updated.BotCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBasePromptNotFound
	}
	if isUniqueViolation(err) {
		return nil, ErrBasePromptNameUsed
	}
	if err != nil {
		return nil, fmt.Errorf("update base prompt: %w", err)
	}

	return &updated, nil
}

func (r *chatbotBasePromptDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chatbot_base_prompts WHERE id = $1`, id)
	if isForeignKeyViolation(err) {
		return ErrBasePromptInUse
	}
	if err != nil {
		return fmt.Errorf("delete base prompt: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete base prompt rows: %w", err)
	}
	if affected == 0 {
		return ErrBasePromptNotFound
	}

	return nil
}
