package repository

import (
	"context"
	"errors"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/logger"

	"github.com/google/uuid"
)

var (
	ErrBotNotFound = errors.New("chatbot not found")
)

type (
	InvocationStatus string

	Chatbot struct {
		ID              uuid.UUID
		UserID          uuid.UUID
		Username        string
		DisplayName     string
		AvatarURL       string
		SystemPrompt    string
		BasePromptID    *uuid.UUID
		BasePrompt      string
		Model           string
		ReasoningEffort string
		Verbosity       string
		MaxOutputTokens int
		Enabled         bool
	}

	InvocationUsage struct {
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
	}

	ChatbotChannelStats struct {
		Channel            string
		Invocations        int
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
	}

	ChatbotStats struct {
		Invocations        int
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
		Failed             int
		Quota              int
		Channels           []ChatbotChannelStats
	}

	ChatbotRepository interface {
		ListBots(ctx context.Context) ([]Chatbot, error)
		GetBotByUserID(ctx context.Context, userID uuid.UUID) (*Chatbot, error)
		CreateBot(ctx context.Context, bot Chatbot) error
		UpdateBot(ctx context.Context, bot Chatbot) error
		DeleteBot(ctx context.Context, id uuid.UUID) error
		CreateBotAccount(ctx context.Context, userID uuid.UUID, username, displayName, avatarURL string) error
		UpdateBotAccount(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) error

		CreateInvocation(ctx context.Context, id, botUserID, userID uuid.UUID, roomID *uuid.UUID, messageID uuid.UUID, channel, model string) error
		CompleteInvocation(ctx context.Context, id uuid.UUID, usage InvocationUsage, status InvocationStatus) error
		CountUserInvocationsToday(ctx context.Context, userID uuid.UUID) (int, error)
		CountInvocationsToday(ctx context.Context) (int, error)
		OldestUserInvocationToday(ctx context.Context, userID uuid.UUID) (time.Time, error)
		OldestInvocationToday(ctx context.Context) (time.Time, error)
		StatsSince(ctx context.Context, since time.Time) (*ChatbotStats, error)
	}
)

const (
	InvocationPending InvocationStatus = "pending"
	InvocationReplied InvocationStatus = "replied"
	InvocationRefused InvocationStatus = "refused"
	InvocationFailed  InvocationStatus = "failed"
	InvocationQuota   InvocationStatus = "quota"
)

type chatbotRepository struct {
	dao         ChatbotRepository
	basePrompts BasePromptInvalidator
	cache       *cache.Manager
}

func NewChatbotRepo(dao ChatbotRepository, basePrompts BasePromptInvalidator, c *cache.Manager) ChatbotRepository {
	return &chatbotRepository{dao: dao, basePrompts: basePrompts, cache: c}
}

func (r *chatbotRepository) ListBots(ctx context.Context) ([]Chatbot, error) {
	return r.dao.ListBots(ctx)
}

func (r *chatbotRepository) GetBotByUserID(ctx context.Context, userID uuid.UUID) (*Chatbot, error) {
	return r.dao.GetBotByUserID(ctx, userID)
}

func (r *chatbotRepository) CreateBot(ctx context.Context, bot Chatbot) error {
	if err := r.dao.CreateBot(ctx, bot); err != nil {
		return err
	}

	r.basePrompts.InvalidateList(ctx)

	return nil
}

func (r *chatbotRepository) UpdateBot(ctx context.Context, bot Chatbot) error {
	if err := r.dao.UpdateBot(ctx, bot); err != nil {
		return err
	}

	r.basePrompts.InvalidateList(ctx)

	return nil
}

func (r *chatbotRepository) DeleteBot(ctx context.Context, id uuid.UUID) error {
	botUserID := r.resolveBotUserID(ctx, id)

	if err := r.dao.DeleteBot(ctx, id); err != nil {
		return err
	}

	r.basePrompts.InvalidateList(ctx)

	keys := []string{cache.VanityAssignments.Key()}
	if botUserID != uuid.Nil {
		keys = append(keys, cache.UserRole.Key(botUserID.String()), cache.UserVanityRoleIDs.Key(botUserID.String()))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Log.Error().Err(err).Msg("failed to invalidate caches after deleting a bot")
	}

	return nil
}

func (r *chatbotRepository) resolveBotUserID(ctx context.Context, id uuid.UUID) uuid.UUID {
	bots, err := r.dao.ListBots(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Str("chatbot_id", id.String()).Msg("failed to resolve bot user id before deleting a bot")

		return uuid.Nil
	}

	for i := range bots {
		if bots[i].ID == id {
			return bots[i].UserID
		}
	}

	return uuid.Nil
}

func (r *chatbotRepository) CreateInvocation(ctx context.Context, id, botUserID, userID uuid.UUID, roomID *uuid.UUID, messageID uuid.UUID, channel, model string) error {
	return r.dao.CreateInvocation(ctx, id, botUserID, userID, roomID, messageID, channel, model)
}

func (r *chatbotRepository) CompleteInvocation(ctx context.Context, id uuid.UUID, usage InvocationUsage, status InvocationStatus) error {
	return r.dao.CompleteInvocation(ctx, id, usage, status)
}

func (r *chatbotRepository) CountUserInvocationsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUserInvocationsToday(ctx, userID)
}

func (r *chatbotRepository) OldestUserInvocationToday(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	return r.dao.OldestUserInvocationToday(ctx, userID)
}

func (r *chatbotRepository) OldestInvocationToday(ctx context.Context) (time.Time, error) {
	return r.dao.OldestInvocationToday(ctx)
}

func (r *chatbotRepository) CountInvocationsToday(ctx context.Context) (int, error) {
	return r.dao.CountInvocationsToday(ctx)
}

func (r *chatbotRepository) StatsSince(ctx context.Context, since time.Time) (*ChatbotStats, error) {
	return r.dao.StatsSince(ctx, since)
}

func (r *chatbotRepository) CreateBotAccount(ctx context.Context, userID uuid.UUID, username, displayName, avatarURL string) error {
	return r.dao.CreateBotAccount(ctx, userID, username, displayName, avatarURL)
}

func (r *chatbotRepository) UpdateBotAccount(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) error {
	return r.dao.UpdateBotAccount(ctx, userID, displayName, avatarURL)
}
