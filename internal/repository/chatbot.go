package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ChatbotRepository interface {
		dao.ChatbotDAO

		CreateBotWithAccount(ctx context.Context, s spec.NewChatbotWithAccount, tx ...*sql.Tx) (*model.Chatbot, error)
		UpdateBotWithAccount(ctx context.Context, s spec.ChatbotAccountUpdate, tx ...*sql.Tx) (*model.Chatbot, error)
	}
)

type chatbotRepository struct {
	db *sql.DB
	dao.ChatbotDAO
	users       UserRepository
	vanity      VanityRoleRepository
	basePrompts BasePromptInvalidator
	cache       *cache.Manager
}

func NewChatbotRepo(database *sql.DB, chatbots dao.ChatbotDAO, users UserRepository, vanity VanityRoleRepository, basePrompts BasePromptInvalidator, c *cache.Manager) ChatbotRepository {
	return &chatbotRepository{db: database, ChatbotDAO: chatbots, users: users, vanity: vanity, basePrompts: basePrompts, cache: c}
}

func (r *chatbotRepository) CreateBotWithAccount(ctx context.Context, s spec.NewChatbotWithAccount, tx ...*sql.Tx) (*model.Chatbot, error) {
	var created *model.Chatbot

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		user, err := r.users.Create(ctx, s.Account, tx)
		if err != nil {
			return fmt.Errorf("create bot account: %w", err)
		}

		if err := r.vanity.AssignToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: s.VanityRoleID}, tx); err != nil {
			return fmt.Errorf("assign bot badge: %w", err)
		}

		bot := s.Bot
		bot.UserID = user.ID

		created, err = r.ChatbotDAO.CreateBot(ctx, bot, tx)
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

func (r *chatbotRepository) UpdateBotWithAccount(ctx context.Context, s spec.ChatbotAccountUpdate, tx ...*sql.Tx) (*model.Chatbot, error) {
	var updated *model.Chatbot

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.users.SetDisplayName(ctx, spec.UserDisplayNameUpdate{UserID: s.Bot.UserID, DisplayName: s.DisplayName}, tx); err != nil {
			return fmt.Errorf("update bot display name: %w", err)
		}

		if err := r.users.UpdateAvatarURL(ctx, spec.UserAvatarUpdate{UserID: s.Bot.UserID, AvatarURL: s.AvatarURL}, tx); err != nil {
			return fmt.Errorf("update bot avatar: %w", err)
		}

		var err error
		updated, err = r.ChatbotDAO.UpdateBot(ctx, s.Bot, tx)
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

func (r *chatbotRepository) CreateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error) {
	created, err := r.ChatbotDAO.CreateBot(ctx, bot, tx...)
	if err != nil {
		return nil, err
	}

	r.basePrompts.InvalidateList(ctx)

	return created, nil
}

func (r *chatbotRepository) UpdateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error) {
	updated, err := r.ChatbotDAO.UpdateBot(ctx, bot, tx...)
	if err != nil {
		return nil, err
	}

	r.basePrompts.InvalidateList(ctx)

	return updated, nil
}

func (r *chatbotRepository) DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	botUserID := r.resolveBotUserID(ctx, id, tx...)

	if err := r.ChatbotDAO.DeleteBot(ctx, id, tx...); err != nil {
		return err
	}

	r.basePrompts.InvalidateList(ctx)

	keys := []string{cache.VanityAssignments.Key()}
	if botUserID != uuid.Nil {
		keys = append(keys, cache.UserRole.Key(botUserID.String()), cache.UserVanityRoleIDs.Key(botUserID.String()))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate caches after deleting a bot")
	}

	return nil
}

func (r *chatbotRepository) resolveBotUserID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) uuid.UUID {
	bots, err := r.ChatbotDAO.ListBots(ctx, tx...)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("chatbot_id", id.String()).Msg("failed to resolve bot user id before deleting a bot")

		return uuid.Nil
	}

	for _, bot := range bots {
		if bot.ID == id {
			return bot.UserID
		}
	}

	return uuid.Nil
}
