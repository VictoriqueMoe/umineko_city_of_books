package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ChatbotBasePromptRepository interface {
		dao.ChatbotBasePromptDAO
	}

	BasePromptInvalidator interface {
		InvalidateList(ctx context.Context)
	}
)

type chatbotBasePromptRepository struct {
	dao.ChatbotBasePromptDAO
	cache *cache.Manager
}

func NewChatbotBasePromptRepo(prompts dao.ChatbotBasePromptDAO, c *cache.Manager) *chatbotBasePromptRepository {
	return &chatbotBasePromptRepository{ChatbotBasePromptDAO: prompts, cache: c}
}

func (r *chatbotBasePromptRepository) List(ctx context.Context, tx ...*sql.Tx) ([]model.ChatbotBasePrompt, error) {
	load := func(ctx context.Context) ([]model.ChatbotBasePrompt, error) {
		return r.ChatbotBasePromptDAO.List(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.ChatbotBasePrompts, load)
}

func (r *chatbotBasePromptRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	load := func(ctx context.Context) (model.ChatbotBasePrompt, error) {
		prompt, err := r.ChatbotBasePromptDAO.GetByID(ctx, id, tx...)
		if err != nil {
			return model.ChatbotBasePrompt{}, err
		}

		return *prompt, nil
	}

	cached, err := r.cache.Load(ctx, cache.ChatbotBasePromptByID, load, id.String())
	if err != nil {
		return nil, err
	}

	return &cached, nil
}

func (r *chatbotBasePromptRepository) Create(ctx context.Context, s spec.NewChatbotBasePrompt, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	created, err := r.ChatbotBasePromptDAO.Create(ctx, s, tx...)
	if err != nil {
		return nil, err
	}

	r.invalidate(ctx, created.ID)

	return created, nil
}

func (r *chatbotBasePromptRepository) Update(ctx context.Context, s spec.ChatbotBasePromptUpdate, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	updated, err := r.ChatbotBasePromptDAO.Update(ctx, s, tx...)
	if err != nil {
		return nil, err
	}

	r.invalidate(ctx, s.ID)

	return updated, nil
}

func (r *chatbotBasePromptRepository) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.ChatbotBasePromptDAO.Delete(ctx, id, tx...); err != nil {
		return err
	}

	r.invalidate(ctx, id)

	return nil
}

func (r *chatbotBasePromptRepository) InvalidateList(ctx context.Context) {
	if err := r.cache.Del(ctx, cache.ChatbotBasePrompts.Key()); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate the chatbot base prompt list cache")
	}
}

func (r *chatbotBasePromptRepository) invalidate(ctx context.Context, id uuid.UUID) {
	keys := []string{cache.ChatbotBasePrompts.Key(), cache.ChatbotBasePromptByID.Key(id.String())}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate chatbot base prompt caches")
	}
}
