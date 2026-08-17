package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/db"
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

	ChatbotInvocation struct {
		ID        uuid.UUID
		BotUserID uuid.UUID
		UserID    uuid.UUID
		RoomID    *uuid.UUID
		MessageID uuid.UUID
		Channel   string
		Model     string
		Status    InvocationStatus
	}

	NewInvocation struct {
		BotUserID uuid.UUID
		UserID    uuid.UUID
		RoomID    *uuid.UUID
		MessageID uuid.UUID
		Channel   string
		Model     string
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

	ChatbotDAO interface {
		ListBots(ctx context.Context, tx ...*sql.Tx) ([]Chatbot, error)
		GetBotByUserID(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*Chatbot, error)
		CreateBot(ctx context.Context, bot Chatbot, tx ...*sql.Tx) (*Chatbot, error)
		UpdateBot(ctx context.Context, bot Chatbot, tx ...*sql.Tx) (*Chatbot, error)
		DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error

		CreateInvocation(ctx context.Context, spec NewInvocation, tx ...*sql.Tx) (*ChatbotInvocation, error)
		CompleteInvocation(ctx context.Context, id uuid.UUID, usage InvocationUsage, status InvocationStatus, tx ...*sql.Tx) error
		CountUserInvocationsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		CountInvocationsToday(ctx context.Context, tx ...*sql.Tx) (int, error)
		OldestUserInvocationToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (time.Time, error)
		OldestInvocationToday(ctx context.Context, tx ...*sql.Tx) (time.Time, error)
		StatsSince(ctx context.Context, since time.Time, tx ...*sql.Tx) (*ChatbotStats, error)
	}

	ChatbotRepository interface {
		ChatbotDAO

		CreateBotWithAccount(ctx context.Context, account NewUser, bot Chatbot, vanityRoleID string, tx ...*sql.Tx) (*Chatbot, error)
		UpdateBotWithAccount(ctx context.Context, bot Chatbot, displayName, avatarURL string, tx ...*sql.Tx) (*Chatbot, error)
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
	db          *sql.DB
	dao         ChatbotDAO
	users       UserRepository
	vanity      VanityRoleRepository
	basePrompts BasePromptInvalidator
	cache       *cache.Manager
}

func NewChatbotRepo(database *sql.DB, dao ChatbotDAO, users UserRepository, vanity VanityRoleRepository, basePrompts BasePromptInvalidator, c *cache.Manager) ChatbotRepository {
	return &chatbotRepository{db: database, dao: dao, users: users, vanity: vanity, basePrompts: basePrompts, cache: c}
}

func (r *chatbotRepository) CreateBotWithAccount(ctx context.Context, account NewUser, bot Chatbot, vanityRoleID string, tx ...*sql.Tx) (*Chatbot, error) {
	var created *Chatbot

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		user, err := r.users.Create(ctx, account, tx)
		if err != nil {
			return fmt.Errorf("create bot account: %w", err)
		}

		if err := r.vanity.AssignToUser(ctx, user.ID, vanityRoleID, tx); err != nil {
			return fmt.Errorf("assign bot badge: %w", err)
		}

		bot.UserID = user.ID

		created, err = r.dao.CreateBot(ctx, bot, tx)
		if err != nil {
			return fmt.Errorf("create chatbot: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.basePrompts.InvalidateList(ctx)

	return created, nil
}

func (r *chatbotRepository) UpdateBotWithAccount(ctx context.Context, bot Chatbot, displayName, avatarURL string, tx ...*sql.Tx) (*Chatbot, error) {
	var updated *Chatbot

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.users.SetDisplayName(ctx, bot.UserID, displayName, tx); err != nil {
			return fmt.Errorf("update bot display name: %w", err)
		}

		if err := r.users.UpdateAvatarURL(ctx, bot.UserID, avatarURL, tx); err != nil {
			return fmt.Errorf("update bot avatar: %w", err)
		}

		var err error
		updated, err = r.dao.UpdateBot(ctx, bot, tx)
		if err != nil {
			return fmt.Errorf("update chatbot: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.basePrompts.InvalidateList(ctx)

	return updated, nil
}

func (r *chatbotRepository) ListBots(ctx context.Context, tx ...*sql.Tx) ([]Chatbot, error) {
	return r.dao.ListBots(ctx, tx...)
}

func (r *chatbotRepository) GetBotByUserID(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*Chatbot, error) {
	return r.dao.GetBotByUserID(ctx, userID, tx...)
}

func (r *chatbotRepository) CreateBot(ctx context.Context, bot Chatbot, tx ...*sql.Tx) (*Chatbot, error) {
	created, err := r.dao.CreateBot(ctx, bot, tx...)
	if err != nil {
		return nil, err
	}

	r.basePrompts.InvalidateList(ctx)

	return created, nil
}

func (r *chatbotRepository) UpdateBot(ctx context.Context, bot Chatbot, tx ...*sql.Tx) (*Chatbot, error) {
	updated, err := r.dao.UpdateBot(ctx, bot, tx...)
	if err != nil {
		return nil, err
	}

	r.basePrompts.InvalidateList(ctx)

	return updated, nil
}

func (r *chatbotRepository) DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	botUserID := r.resolveBotUserID(ctx, id, tx...)

	if err := r.dao.DeleteBot(ctx, id, tx...); err != nil {
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

func (r *chatbotRepository) resolveBotUserID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) uuid.UUID {
	bots, err := r.dao.ListBots(ctx, tx...)
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

func (r *chatbotRepository) CreateInvocation(ctx context.Context, spec NewInvocation, tx ...*sql.Tx) (*ChatbotInvocation, error) {
	return r.dao.CreateInvocation(ctx, spec, tx...)
}

func (r *chatbotRepository) CompleteInvocation(ctx context.Context, id uuid.UUID, usage InvocationUsage, status InvocationStatus, tx ...*sql.Tx) error {
	return r.dao.CompleteInvocation(ctx, id, usage, status, tx...)
}

func (r *chatbotRepository) CountUserInvocationsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUserInvocationsToday(ctx, userID, tx...)
}

func (r *chatbotRepository) OldestUserInvocationToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (time.Time, error) {
	return r.dao.OldestUserInvocationToday(ctx, userID, tx...)
}

func (r *chatbotRepository) OldestInvocationToday(ctx context.Context, tx ...*sql.Tx) (time.Time, error) {
	return r.dao.OldestInvocationToday(ctx, tx...)
}

func (r *chatbotRepository) CountInvocationsToday(ctx context.Context, tx ...*sql.Tx) (int, error) {
	return r.dao.CountInvocationsToday(ctx, tx...)
}

func (r *chatbotRepository) StatsSince(ctx context.Context, since time.Time, tx ...*sql.Tx) (*ChatbotStats, error) {
	return r.dao.StatsSince(ctx, since, tx...)
}
