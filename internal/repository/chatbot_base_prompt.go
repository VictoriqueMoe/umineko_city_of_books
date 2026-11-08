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

		Invalidate(ctx context.Context, ids ...uuid.UUID)
	}
)

type chatbotBasePromptRepository struct {
	dao.ChatbotBasePromptDAO
	cache *cache.Manager
}

func NewChatbotBasePromptRepo(prompts dao.ChatbotBasePromptDAO, c *cache.Manager) ChatbotBasePromptRepository {
	return &chatbotBasePromptRepository{ChatbotBasePromptDAO: prompts, cache: c}
}

func (r *chatbotBasePromptRepository) List(ctx context.Context, tx ...*sql.Tx) ([]model.ChatbotBasePrompt, error) {
	load := func(ctx context.Context) ([]model.ChatbotBasePrompt, error) {
		return r.ChatbotBasePromptDAO.List(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.ChatbotBasePrompts.Key(), load)
}

func (r *chatbotBasePromptRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	load := func(ctx context.Context) (model.ChatbotBasePrompt, error) {
		prompt, err := r.ChatbotBasePromptDAO.GetByID(ctx, id, tx...)
		if err != nil {
			return model.ChatbotBasePrompt{}, err
		}

		return *prompt, nil
	}

	cached, err := r.cache.Load(ctx, cache.ChatbotBasePromptByID.Key(id.String()), load)
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

	r.Invalidate(ctx, created.ID)

	return created, nil
}

func (r *chatbotBasePromptRepository) Update(ctx context.Context, s spec.ChatbotBasePromptUpdate, tx ...*sql.Tx) (*model.ChatbotBasePrompt, error) {
	updated, err := r.ChatbotBasePromptDAO.Update(ctx, s, tx...)
	if err != nil {
		return nil, err
	}

	r.Invalidate(ctx, s.ID)

	return updated, nil
}

func (r *chatbotBasePromptRepository) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.ChatbotBasePromptDAO.Delete(ctx, id, tx...); err != nil {
		return err
	}

	r.Invalidate(ctx, id)

	return nil
}

func (r *chatbotBasePromptRepository) Invalidate(ctx context.Context, ids ...uuid.UUID) {
	keys := make([]cache.Key, 0, len(ids)+1)
	keys = append(keys, cache.ChatbotBasePrompts.Key())
	for _, id := range ids {
		keys = append(keys, cache.ChatbotBasePromptByID.Key(id.String()))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate chatbot base prompt caches")
	}
}
