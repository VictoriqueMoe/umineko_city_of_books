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
	ChatbotDAO interface {
		ListBots(ctx context.Context, tx ...*sql.Tx) ([]model.Chatbot, error)
		GetBotByUserID(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*model.Chatbot, error)
		CreateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error)
		UpdateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error)
		DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error

		CreateInvocation(ctx context.Context, s spec.NewInvocation, tx ...*sql.Tx) (*model.ChatbotInvocation, error)
		CompleteInvocation(ctx context.Context, s spec.InvocationCompletion, tx ...*sql.Tx) error
		CountUserInvocationsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		CountInvocationsToday(ctx context.Context, tx ...*sql.Tx) (int, error)
		OldestUserInvocationToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (time.Time, error)
		OldestInvocationToday(ctx context.Context, tx ...*sql.Tx) (time.Time, error)
		StatsSince(ctx context.Context, since time.Time, tx ...*sql.Tx) (*model.ChatbotStats, error)
	}

	chatbotDAO struct {
		db *sql.DB
	}

	chatbotRow = sqlcgen.GetChatbotByUserIDRow
)

var (
	ErrBotNotFound = errors.New("chatbot not found")
)

func toChatbot(row chatbotRow) model.Chatbot {
	return model.Chatbot{
		ID:              row.ID,
		UserID:          row.UserID,
		Username:        row.Username,
		DisplayName:     row.DisplayName,
		AvatarURL:       row.AvatarUrl,
		SystemPrompt:    row.SystemPrompt,
		BasePromptID:    row.BasePromptID,
		BasePrompt:      row.BasePrompt,
		Model:           row.Model,
		ReasoningEffort: row.ReasoningEffort,
		Verbosity:       row.Verbosity,
		MaxOutputTokens: int(row.MaxOutputTokens),
		Enabled:         row.Enabled,
	}
}

func (r *chatbotDAO) ListBots(ctx context.Context, tx ...*sql.Tx) ([]model.Chatbot, error) {
	rows, err := genQueries(r.db, tx).ListChatbots(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chatbots: %w", err)
	}

	bots := make([]model.Chatbot, 0, len(rows))
	for _, row := range rows {
		bots = append(bots, toChatbot(chatbotRow(row)))
	}

	return bots, nil
}

func (r *chatbotDAO) GetBotByUserID(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*model.Chatbot, error) {
	row, err := genQueries(r.db, tx).GetChatbotByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get chatbot by user: %w", err)
	}

	return new(toChatbot(row)), nil
}

func (r *chatbotDAO) CreateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error) {
	created, err := genQueries(r.db, tx).CreateChatbot(ctx, sqlcgen.CreateChatbotParams{
		UserID:          bot.UserID,
		SystemPrompt:    bot.SystemPrompt,
		BasePromptID:    bot.BasePromptID,
		Model:           bot.Model,
		ReasoningEffort: bot.ReasoningEffort,
		Verbosity:       bot.Verbosity,
		MaxOutputTokens: int32(bot.MaxOutputTokens),
		Enabled:         bot.Enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("create chatbot: %w", err)
	}

	return new(toChatbot(chatbotRow(created))), nil
}

func (r *chatbotDAO) UpdateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error) {
	updated, err := genQueries(r.db, tx).UpdateChatbot(ctx, sqlcgen.UpdateChatbotParams{
		ID:              bot.ID,
		SystemPrompt:    bot.SystemPrompt,
		BasePromptID:    bot.BasePromptID,
		Model:           bot.Model,
		ReasoningEffort: bot.ReasoningEffort,
		Verbosity:       bot.Verbosity,
		MaxOutputTokens: int32(bot.MaxOutputTokens),
		Enabled:         bot.Enabled,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update chatbot: %w", err)
	}

	return new(toChatbot(chatbotRow(updated))), nil
}

func (r *chatbotDAO) DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteChatbot(ctx, id)
	if err != nil {
		return fmt.Errorf("delete chatbot: %w", err)
	}

	if affected == 0 {
		return ErrBotNotFound
	}

	return nil
}

func (r *chatbotDAO) CreateInvocation(ctx context.Context, s spec.NewInvocation, tx ...*sql.Tx) (*model.ChatbotInvocation, error) {
	row, err := genQueries(r.db, tx).CreateChatbotInvocation(ctx, sqlcgen.CreateChatbotInvocationParams{
		BotUserID: s.BotUserID,
		UserID:    s.UserID,
		RoomID:    s.RoomID,
		MessageID: s.MessageID,
		Channel:   sqlcgen.ChatbotChannel(s.Channel),
		Model:     s.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("create chatbot invocation: %w", err)
	}

	inv := model.ChatbotInvocation{
		ID:        row.ID,
		BotUserID: row.BotUserID,
		UserID:    row.UserID,
		RoomID:    row.RoomID,
		MessageID: row.MessageID,
		Channel:   string(row.Channel),
		Model:     row.Model,
		Status:    model.InvocationStatus(row.Status),
	}

	return &inv, nil
}

func (r *chatbotDAO) CompleteInvocation(ctx context.Context, s spec.InvocationCompletion, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CompleteChatbotInvocation(ctx, sqlcgen.CompleteChatbotInvocationParams{
		ID:                 s.ID,
		PromptTokens:       int32(s.Usage.PromptTokens),
		CachedPromptTokens: int32(s.Usage.CachedPromptTokens),
		CacheWriteTokens:   int32(s.Usage.CacheWriteTokens),
		CompletionTokens:   int32(s.Usage.CompletionTokens),
		ReasoningTokens:    int32(s.Usage.ReasoningTokens),
		Status:             sqlcgen.ChatbotInvocationStatus(s.Status),
	})
	if err != nil {
		return fmt.Errorf("complete chatbot invocation: %w", err)
	}

	return nil
}

func (r *chatbotDAO) CountUserInvocationsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUserChatbotInvocationsToday(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count user chatbot invocations today: %w", err)
	}

	return int(count), nil
}

func (r *chatbotDAO) CountInvocationsToday(ctx context.Context, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountChatbotInvocationsToday(ctx)
	if err != nil {
		return 0, fmt.Errorf("count chatbot invocations today: %w", err)
	}

	return int(count), nil
}

func (r *chatbotDAO) OldestUserInvocationToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (time.Time, error) {
	oldest, err := genQueries(r.db, tx).GetOldestUserChatbotInvocationToday(ctx, userID)
	if err != nil {
		return time.Time{}, fmt.Errorf("oldest user chatbot invocation today: %w", err)
	}

	return oldest, nil
}

func (r *chatbotDAO) OldestInvocationToday(ctx context.Context, tx ...*sql.Tx) (time.Time, error) {
	oldest, err := genQueries(r.db, tx).GetOldestChatbotInvocationToday(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("oldest chatbot invocation today: %w", err)
	}

	return oldest, nil
}

func (r *chatbotDAO) StatsSince(ctx context.Context, since time.Time, tx ...*sql.Tx) (*model.ChatbotStats, error) {
	rows, err := genQueries(r.db, tx).ChatbotStatsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("chatbot stats since: %w", err)
	}

	stats := model.ChatbotStats{Channels: make([]model.ChatbotChannelStats, 0, len(rows))}
	for _, row := range rows {
		channel := model.ChatbotChannelStats{
			Channel:            string(row.Channel),
			Invocations:        int(row.Invocations),
			PromptTokens:       int(row.PromptTokens),
			CachedPromptTokens: int(row.CachedPromptTokens),
			CacheWriteTokens:   int(row.CacheWriteTokens),
			CompletionTokens:   int(row.CompletionTokens),
			ReasoningTokens:    int(row.ReasoningTokens),
		}

		stats.Invocations += channel.Invocations
		stats.PromptTokens += channel.PromptTokens
		stats.CachedPromptTokens += channel.CachedPromptTokens
		stats.CacheWriteTokens += channel.CacheWriteTokens
		stats.CompletionTokens += channel.CompletionTokens
		stats.ReasoningTokens += channel.ReasoningTokens
		stats.Failed += int(row.Failed)
		stats.Quota += int(row.Quota)

		stats.Channels = append(stats.Channels, channel)
	}

	return &stats, nil
}
