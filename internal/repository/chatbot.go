package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ChatbotRepository interface {
		dao.ChatbotDAO

		CreateBotWithAccount(ctx context.Context, s spec.NewChatbotWithAccount, tx ...*sql.Tx) (*model.Chatbot, error)
		UpdateBotWithAccount(ctx context.Context, s spec.ChatbotAccountUpdate, tx ...*sql.Tx) (*model.Chatbot, error)
		DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
	}
)

type chatbotRepository struct {
	db *sql.DB
	dao.ChatbotDAO
	users       UserRepository
	permissions PermissionRepository
	basePrompts ChatbotBasePromptRepository
}

func NewChatbotRepo(database *sql.DB, chatbots dao.ChatbotDAO, users UserRepository, permissions PermissionRepository, basePrompts ChatbotBasePromptRepository) ChatbotRepository {
	return &chatbotRepository{db: database, ChatbotDAO: chatbots, users: users, permissions: permissions, basePrompts: basePrompts}
}

func basePromptIDs(bots ...*model.Chatbot) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(bots))
	for _, bot := range bots {
		if bot != nil && bot.BasePromptID != nil {
			ids = append(ids, *bot.BasePromptID)
		}
	}

	return ids
}

func (r *chatbotRepository) CreateBotWithAccount(ctx context.Context, s spec.NewChatbotWithAccount, tx ...*sql.Tx) (*model.Chatbot, error) {
	var created *model.Chatbot

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		user, err := r.users.Create(ctx, s.Account, tx)
		if err != nil {
			return fmt.Errorf("create bot account: %w", err)
		}

		if err := r.permissions.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: s.VanityRoleID}, tx); err != nil {
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

	r.permissions.InvalidateVanityAssignments(ctx, created.UserID.String())
	r.basePrompts.Invalidate(ctx, basePromptIDs(created)...)

	return created, nil
}

func (r *chatbotRepository) UpdateBotWithAccount(ctx context.Context, s spec.ChatbotAccountUpdate, tx ...*sql.Tx) (*model.Chatbot, error) {
	var previous, updated *model.Chatbot

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error
		previous, err = r.ChatbotDAO.GetBot(ctx, s.Bot.ID, tx)
		if err != nil {
			return fmt.Errorf("read chatbot before update: %w", err)
		}

		if err := r.users.SetDisplayName(ctx, spec.UserDisplayNameUpdate{UserID: s.Bot.UserID, DisplayName: s.DisplayName}, tx); err != nil {
			return fmt.Errorf("update bot display name: %w", err)
		}

		if err := r.users.UpdateAvatarURL(ctx, spec.UserAvatarUpdate{UserID: s.Bot.UserID, AvatarURL: s.AvatarURL}, tx); err != nil {
			return fmt.Errorf("update bot avatar: %w", err)
		}

		updated, err = r.ChatbotDAO.UpdateBot(ctx, s.Bot, tx)
		if err != nil {
			return fmt.Errorf("update chatbot: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.basePrompts.Invalidate(ctx, basePromptIDs(previous, updated)...)

	return updated, nil
}

func (r *chatbotRepository) CreateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error) {
	created, err := r.ChatbotDAO.CreateBot(ctx, bot, tx...)
	if err != nil {
		return nil, err
	}

	r.basePrompts.Invalidate(ctx, basePromptIDs(created)...)

	return created, nil
}

func (r *chatbotRepository) UpdateBot(ctx context.Context, bot model.Chatbot, tx ...*sql.Tx) (*model.Chatbot, error) {
	previous, err := r.ChatbotDAO.GetBot(ctx, bot.ID, tx...)
	if err != nil {
		return nil, err
	}

	updated, err := r.ChatbotDAO.UpdateBot(ctx, bot, tx...)
	if err != nil {
		return nil, err
	}

	r.basePrompts.Invalidate(ctx, basePromptIDs(previous, updated)...)

	return updated, nil
}

func (r *chatbotRepository) DeleteBot(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	bot, err := r.ChatbotDAO.GetBot(ctx, id, tx...)
	if err != nil {
		return err
	}

	if bot == nil {
		return daoerr.ErrBotNotFound
	}

	if err := r.users.AdminDeleteAccount(ctx, bot.UserID, tx...); err != nil {
		return err
	}

	r.basePrompts.Invalidate(ctx, basePromptIDs(bot)...)

	return nil
}
