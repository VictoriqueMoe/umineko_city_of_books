package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type (
	chatbotBasePromptDAO struct {
		db *sql.DB
	}
)

const basePromptSelectBase = `
	SELECT b.id, b.name, b.prompt, b.created_at, b.updated_at,
		(SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = b.id)
	FROM chatbot_base_prompts b`

func scanBasePrompt(row interface{ Scan(...any) error }, prompt *repository.ChatbotBasePrompt) error {
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

func (r *chatbotBasePromptDAO) List(ctx context.Context, tx ...*sql.Tx) ([]repository.ChatbotBasePrompt, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, basePromptSelectBase+` ORDER BY b.name`)
	if err != nil {
		return nil, fmt.Errorf("list base prompts: %w", err)
	}
	defer rows.Close()

	prompts := make([]repository.ChatbotBasePrompt, 0)
	for rows.Next() {
		var prompt repository.ChatbotBasePrompt
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

func (r *chatbotBasePromptDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*repository.ChatbotBasePrompt, error) {
	var prompt repository.ChatbotBasePrompt
	err := scanBasePrompt(txOrDB(r.db, tx).QueryRowContext(ctx, basePromptSelectBase+` WHERE b.id = $1`, id), &prompt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrBasePromptNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get base prompt: %w", err)
	}

	return &prompt, nil
}

func (r *chatbotBasePromptDAO) Create(ctx context.Context, name, prompt string, tx ...*sql.Tx) (*repository.ChatbotBasePrompt, error) {
	var created repository.ChatbotBasePrompt
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chatbot_base_prompts (id, name, prompt)
		 VALUES (gen_random_uuid(), $1, $2)
		 RETURNING id, name, prompt, created_at, updated_at, 0`,
		name, prompt,
	).Scan(&created.ID, &created.Name, &created.Prompt, &created.CreatedAt, &created.UpdatedAt, &created.BotCount)
	if isUniqueViolation(err) {
		return nil, repository.ErrBasePromptNameUsed
	}
	if err != nil {
		return nil, fmt.Errorf("create base prompt: %w", err)
	}

	return &created, nil
}

func (r *chatbotBasePromptDAO) Update(ctx context.Context, id uuid.UUID, name, prompt string, tx ...*sql.Tx) (*repository.ChatbotBasePrompt, error) {
	var updated repository.ChatbotBasePrompt
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`UPDATE chatbot_base_prompts SET name = $2, prompt = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, prompt, created_at, updated_at,
		   (SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = chatbot_base_prompts.id)`,
		id, name, prompt,
	).Scan(&updated.ID, &updated.Name, &updated.Prompt, &updated.CreatedAt, &updated.UpdatedAt, &updated.BotCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrBasePromptNotFound
	}
	if isUniqueViolation(err) {
		return nil, repository.ErrBasePromptNameUsed
	}
	if err != nil {
		return nil, fmt.Errorf("update base prompt: %w", err)
	}

	return &updated, nil
}

func (r *chatbotBasePromptDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chatbot_base_prompts WHERE id = $1`, id)
	if isForeignKeyViolation(err) {
		return repository.ErrBasePromptInUse
	}
	if err != nil {
		return fmt.Errorf("delete base prompt: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete base prompt rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrBasePromptNotFound
	}

	return nil
}
