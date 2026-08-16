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

const chatbotColumns = `
	c.id, c.user_id, u.username, u.display_name, u.avatar_url,
	c.system_prompt, c.base_prompt_id, COALESCE(b.prompt, ''),
	c.model, c.reasoning_effort, c.verbosity, c.max_output_tokens, c.enabled`

const chatbotJoins = `
	JOIN users u ON u.id = c.user_id
	LEFT JOIN chatbot_base_prompts b ON b.id = c.base_prompt_id`

const chatbotSelectBase = `SELECT ` + chatbotColumns + ` FROM chatbots c` + chatbotJoins

func scanChatbot(row interface{ Scan(...any) error }, bot *repository.Chatbot) error {
	return row.Scan(
		&bot.ID, &bot.UserID, &bot.Username, &bot.DisplayName, &bot.AvatarURL,
		&bot.SystemPrompt, &bot.BasePromptID, &bot.BasePrompt,
		&bot.Model, &bot.ReasoningEffort, &bot.Verbosity, &bot.MaxOutputTokens, &bot.Enabled,
	)
}

func (r *chatbotDAO) ListBots(ctx context.Context, tx ...*sql.Tx) ([]repository.Chatbot, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, chatbotSelectBase+` ORDER BY u.username`)
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

func (r *chatbotDAO) GetBotByUserID(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*repository.Chatbot, error) {
	var bot repository.Chatbot
	err := scanChatbot(txOrDB(r.db, tx).QueryRowContext(ctx, chatbotSelectBase+` WHERE c.user_id = $1`, userID), &bot)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get chatbot by user: %w", err)
	}

	return &bot, nil
}

func (r *chatbotDAO) CreateBot(ctx context.Context, bot repository.Chatbot, tx ...*sql.Tx) (*repository.Chatbot, error) {
	var created repository.Chatbot
	err := scanChatbot(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH c AS (
		     INSERT INTO chatbots (user_id, system_prompt, base_prompt_id, model, reasoning_effort, verbosity, max_output_tokens, enabled)
		     VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		     RETURNING id, user_id, system_prompt, base_prompt_id, model, reasoning_effort, verbosity, max_output_tokens, enabled
		 )
		 SELECT `+chatbotColumns+` FROM c`+chatbotJoins,
		bot.UserID, bot.SystemPrompt, bot.BasePromptID, bot.Model, bot.ReasoningEffort, bot.Verbosity, bot.MaxOutputTokens, bot.Enabled,
	), &created)
	if err != nil {
		return nil, fmt.Errorf("create chatbot: %w", err)
	}

	return &created, nil
}

func (r *chatbotDAO) UpdateBot(ctx context.Context, bot repository.Chatbot, tx ...*sql.Tx) (*repository.Chatbot, error) {
	var updated repository.Chatbot
	err := scanChatbot(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH c AS (
		     UPDATE chatbots
		        SET system_prompt = $2,
		            base_prompt_id = $3,
		            model = $4,
		            reasoning_effort = $5,
		            verbosity = $6,
		            max_output_tokens = $7,
		            enabled = $8,
		            updated_at = NOW()
		      WHERE id = $1
		      RETURNING id, user_id, system_prompt, base_prompt_id, model, reasoning_effort, verbosity, max_output_tokens, enabled
		 )
		 SELECT `+chatbotColumns+` FROM c`+chatbotJoins,
		bot.ID, bot.SystemPrompt, bot.BasePromptID, bot.Model, bot.ReasoningEffort, bot.Verbosity, bot.MaxOutputTokens, bot.Enabled,
	), &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrBotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update chatbot: %w", err)
	}

	return &updated, nil
}

func (r *chatbotDAO) DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
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

func (r *chatbotDAO) CreateInvocation(ctx context.Context, spec repository.NewInvocation, tx ...*sql.Tx) (*repository.ChatbotInvocation, error) {
	var inv repository.ChatbotInvocation
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chatbot_invocations (bot_user_id, user_id, room_id, message_id, channel, model, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		 RETURNING id, bot_user_id, user_id, room_id, message_id, channel, model, status`,
		spec.BotUserID, spec.UserID, spec.RoomID, spec.MessageID, spec.Channel, spec.Model,
	).Scan(&inv.ID, &inv.BotUserID, &inv.UserID, &inv.RoomID, &inv.MessageID, &inv.Channel, &inv.Model, &inv.Status)
	if err != nil {
		return nil, fmt.Errorf("create chatbot invocation: %w", err)
	}

	return &inv, nil
}

func (r *chatbotDAO) CompleteInvocation(ctx context.Context, id uuid.UUID, usage repository.InvocationUsage, status repository.InvocationStatus, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
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

func (r *chatbotDAO) CountUserInvocationsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chatbot_invocations WHERE user_id = $1 AND status <> 'failed' AND created_at > NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user chatbot invocations today: %w", err)
	}

	return count, nil
}

func (r *chatbotDAO) CountInvocationsToday(ctx context.Context, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chatbot_invocations WHERE status <> 'failed' AND created_at > NOW() - INTERVAL '1 day'`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count chatbot invocations today: %w", err)
	}

	return count, nil
}

func (r *chatbotDAO) OldestUserInvocationToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (time.Time, error) {
	var oldest sql.NullTime
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT MIN(created_at) FROM chatbot_invocations WHERE user_id = $1 AND status <> 'failed' AND created_at > NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&oldest)
	if err != nil {
		return time.Time{}, fmt.Errorf("oldest user chatbot invocation today: %w", err)
	}

	return oldest.Time, nil
}

func (r *chatbotDAO) OldestInvocationToday(ctx context.Context, tx ...*sql.Tx) (time.Time, error) {
	var oldest sql.NullTime
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT MIN(created_at) FROM chatbot_invocations WHERE status <> 'failed' AND created_at > NOW() - INTERVAL '1 day'`,
	).Scan(&oldest)
	if err != nil {
		return time.Time{}, fmt.Errorf("oldest chatbot invocation today: %w", err)
	}

	return oldest.Time, nil
}

func (r *chatbotDAO) StatsSince(ctx context.Context, since time.Time, tx ...*sql.Tx) (*repository.ChatbotStats, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT channel,
		        COUNT(*),
		        COALESCE(SUM(prompt_tokens), 0),
		        COALESCE(SUM(cached_prompt_tokens), 0),
		        COALESCE(SUM(cache_write_tokens), 0),
		        COALESCE(SUM(completion_tokens), 0),
		        COALESCE(SUM(reasoning_tokens), 0),
		        COUNT(*) FILTER (WHERE status = 'failed'),
		        COUNT(*) FILTER (WHERE status = 'quota')
		   FROM chatbot_invocations
		  WHERE created_at >= $1
		  GROUP BY channel
		  ORDER BY COUNT(*) DESC, channel`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("chatbot stats since: %w", err)
	}
	defer rows.Close()

	stats := repository.ChatbotStats{Channels: make([]repository.ChatbotChannelStats, 0)}
	for rows.Next() {
		var row repository.ChatbotChannelStats
		var failed, quota int

		if err := rows.Scan(&row.Channel, &row.Invocations, &row.PromptTokens, &row.CachedPromptTokens, &row.CacheWriteTokens, &row.CompletionTokens, &row.ReasoningTokens, &failed, &quota); err != nil {
			return nil, fmt.Errorf("scan chatbot stats: %w", err)
		}

		stats.Invocations += row.Invocations
		stats.PromptTokens += row.PromptTokens
		stats.CachedPromptTokens += row.CachedPromptTokens
		stats.CacheWriteTokens += row.CacheWriteTokens
		stats.CompletionTokens += row.CompletionTokens
		stats.ReasoningTokens += row.ReasoningTokens
		stats.Failed += failed
		stats.Quota += quota

		stats.Channels = append(stats.Channels, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chatbot stats: %w", err)
	}

	return &stats, nil
}
