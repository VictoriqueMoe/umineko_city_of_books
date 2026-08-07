package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	chatbotDAO struct {
		db *sql.DB
	}
)

const chatbotSelectBase = `
	SELECT c.id, c.user_id, u.username, u.display_name, u.avatar_url,
		c.system_prompt, c.model, c.reasoning_effort, c.verbosity, c.max_output_tokens, c.enabled
	FROM chatbots c
	JOIN users u ON u.id = c.user_id`

func scanChatbot(row interface{ Scan(...any) error }, bot *repository.Chatbot) error {
	return row.Scan(
		&bot.ID, &bot.UserID, &bot.Username, &bot.DisplayName, &bot.AvatarURL,
		&bot.SystemPrompt, &bot.Model, &bot.ReasoningEffort, &bot.Verbosity, &bot.MaxOutputTokens, &bot.Enabled,
	)
}

func (r *chatbotDAO) ListBots(ctx context.Context) ([]repository.Chatbot, error) {
	rows, err := r.db.QueryContext(ctx, chatbotSelectBase+` ORDER BY u.username`)
	if err != nil {
		return nil, fmt.Errorf("list chatbots: %w", err)
	}
	defer rows.Close()

	bots := make([]repository.Chatbot, 0)
	for rows.Next() {
		var bot repository.Chatbot
		if err := scanChatbot(rows, &bot); err != nil {
			return nil, fmt.Errorf("scan chatbot: %w", err)
		}

		bots = append(bots, bot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chatbots: %w", err)
	}

	return bots, nil
}

func (r *chatbotDAO) GetBotByUserID(ctx context.Context, userID uuid.UUID) (*repository.Chatbot, error) {
	var bot repository.Chatbot
	err := scanChatbot(r.db.QueryRowContext(ctx, chatbotSelectBase+` WHERE c.user_id = $1`, userID), &bot)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get chatbot by user: %w", err)
	}

	return &bot, nil
}

func (r *chatbotDAO) CreateBot(ctx context.Context, bot repository.Chatbot) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chatbots (id, user_id, system_prompt, model, reasoning_effort, verbosity, max_output_tokens, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		bot.ID, bot.UserID, bot.SystemPrompt, bot.Model, bot.ReasoningEffort, bot.Verbosity, bot.MaxOutputTokens, bot.Enabled,
	)
	if err != nil {
		return fmt.Errorf("create chatbot: %w", err)
	}

	return nil
}

func (r *chatbotDAO) UpdateBot(ctx context.Context, bot repository.Chatbot) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chatbots
		    SET system_prompt = $2,
		        model = $3,
		        reasoning_effort = $4,
		        verbosity = $5,
		        max_output_tokens = $6,
		        enabled = $7,
		        updated_at = NOW()
		  WHERE id = $1`,
		bot.ID, bot.SystemPrompt, bot.Model, bot.ReasoningEffort, bot.Verbosity, bot.MaxOutputTokens, bot.Enabled,
	)
	if err != nil {
		return fmt.Errorf("update chatbot: %w", err)
	}

	return nil
}

func (r *chatbotDAO) DeleteBot(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM users WHERE is_bot AND id = (SELECT user_id FROM chatbots WHERE id = $1)`, id)
	if err != nil {
		return fmt.Errorf("delete chatbot: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete chatbot rows: %w", err)
	}

	if affected == 0 {
		return repository.ErrBotNotFound
	}

	return nil
}

func (r *chatbotDAO) CreateInvocation(ctx context.Context, id, botUserID, userID uuid.UUID, roomID *uuid.UUID, messageID uuid.UUID, surface, model string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chatbot_invocations (id, bot_user_id, user_id, room_id, message_id, surface, model, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')`,
		id, botUserID, userID, roomID, messageID, surface, model,
	)
	if err != nil {
		return fmt.Errorf("create chatbot invocation: %w", err)
	}

	return nil
}

func (r *chatbotDAO) CompleteInvocation(ctx context.Context, id uuid.UUID, usage repository.InvocationUsage, status repository.InvocationStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chatbot_invocations
		    SET prompt_tokens = $2,
		        cached_prompt_tokens = $3,
		        cache_write_tokens = $4,
		        completion_tokens = $5,
		        reasoning_tokens = $6,
		        status = $7
		  WHERE id = $1`,
		id, usage.PromptTokens, usage.CachedPromptTokens, usage.CacheWriteTokens, usage.CompletionTokens, usage.ReasoningTokens, status,
	)
	if err != nil {
		return fmt.Errorf("complete chatbot invocation: %w", err)
	}

	return nil
}

func (r *chatbotDAO) CountUserInvocationsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chatbot_invocations WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user chatbot invocations today: %w", err)
	}

	return count, nil
}

func (r *chatbotDAO) CountInvocationsToday(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chatbot_invocations WHERE created_at > NOW() - INTERVAL '1 day'`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count chatbot invocations today: %w", err)
	}

	return count, nil
}

func (r *chatbotDAO) StatsSince(ctx context.Context, since time.Time) (*repository.ChatbotStats, error) {
	var stats repository.ChatbotStats
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(prompt_tokens), 0),
		        COALESCE(SUM(cached_prompt_tokens), 0),
		        COALESCE(SUM(cache_write_tokens), 0),
		        COALESCE(SUM(completion_tokens), 0),
		        COALESCE(SUM(reasoning_tokens), 0),
		        COUNT(*) FILTER (WHERE status = 'failed'),
		        COUNT(*) FILTER (WHERE status = 'quota')
		   FROM chatbot_invocations
		  WHERE created_at >= $1`,
		since,
	).Scan(&stats.Invocations, &stats.PromptTokens, &stats.CachedPromptTokens, &stats.CacheWriteTokens, &stats.CompletionTokens, &stats.ReasoningTokens, &stats.Failed, &stats.Quota)
	if err != nil {
		return nil, fmt.Errorf("chatbot stats since: %w", err)
	}

	return &stats, nil
}

func (r *chatbotDAO) CreateBotAccount(ctx context.Context, userID uuid.UUID, username, displayName, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, display_name, avatar_url, is_bot, dms_enabled, email_verified)
		 VALUES ($1, $2, '!', $3, $4, TRUE, TRUE, TRUE)`,
		userID, username, displayName, avatarURL,
	)
	if err != nil {
		return fmt.Errorf("insert bot account: %w", err)
	}

	return nil
}

func (r *chatbotDAO) UpdateBotAccount(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = $2, avatar_url = $3 WHERE id = $1 AND is_bot`,
		userID, displayName, avatarURL,
	)
	if err != nil {
		return fmt.Errorf("update bot account: %w", err)
	}

	return nil
}
