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
	ErrBasePromptNotFound = errors.New("base prompt not found")
	ErrBasePromptNameUsed = errors.New("that base prompt name is already taken")
	ErrBasePromptInUse    = errors.New("that base prompt is still used by a chatbot")
)

type (
	ChatbotBasePrompt struct {
		ID        uuid.UUID
		Name      string
		Prompt    string
		BotCount  int
		CreatedAt time.Time
		UpdatedAt time.Time
	}

	ChatbotBasePromptRepository interface {
		List(ctx context.Context) ([]ChatbotBasePrompt, error)
		GetByID(ctx context.Context, id uuid.UUID) (*ChatbotBasePrompt, error)
		Create(ctx context.Context, prompt ChatbotBasePrompt) error
		Update(ctx context.Context, prompt ChatbotBasePrompt) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	BasePromptInvalidator interface {
		InvalidateList(ctx context.Context)
	}
)

type chatbotBasePromptRepository struct {
	dao   ChatbotBasePromptRepository
	cache *cache.Manager
}

func NewChatbotBasePromptRepo(dao ChatbotBasePromptRepository, c *cache.Manager) *chatbotBasePromptRepository {
	return &chatbotBasePromptRepository{dao: dao, cache: c}
}

func (r *chatbotBasePromptRepository) List(ctx context.Context) ([]ChatbotBasePrompt, error) {
	key := cache.ChatbotBasePrompts.Key()

	if cached, err := cache.Get[[]ChatbotBasePrompt](ctx, r.cache, key); err == nil {
		return cached, nil
	}

	prompts, err := r.dao.List(ctx)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, prompts, cache.ChatbotBasePrompts.TTL)

	return prompts, nil
}

func (r *chatbotBasePromptRepository) GetByID(ctx context.Context, id uuid.UUID) (*ChatbotBasePrompt, error) {
	key := cache.ChatbotBasePromptByID.Key(id.String())

	if cached, err := cache.Get[ChatbotBasePrompt](ctx, r.cache, key); err == nil {
		return &cached, nil
	}

	prompt, err := r.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, *prompt, cache.ChatbotBasePromptByID.TTL)

	return prompt, nil
}

func (r *chatbotBasePromptRepository) Create(ctx context.Context, prompt ChatbotBasePrompt) error {
	if err := r.dao.Create(ctx, prompt); err != nil {
		return err
	}

	r.invalidate(ctx, prompt.ID)

	return nil
}

func (r *chatbotBasePromptRepository) Update(ctx context.Context, prompt ChatbotBasePrompt) error {
	if err := r.dao.Update(ctx, prompt); err != nil {
		return err
	}

	r.invalidate(ctx, prompt.ID)

	return nil
}

func (r *chatbotBasePromptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.dao.Delete(ctx, id); err != nil {
		return err
	}

	r.invalidate(ctx, id)

	return nil
}

func (r *chatbotBasePromptRepository) InvalidateList(ctx context.Context) {
	if err := r.cache.Del(ctx, cache.ChatbotBasePrompts.Key()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to invalidate the chatbot base prompt list cache")
	}
}

func (r *chatbotBasePromptRepository) invalidate(ctx context.Context, id uuid.UUID) {
	keys := []string{cache.ChatbotBasePrompts.Key(), cache.ChatbotBasePromptByID.Key(id.String())}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Log.Error().Err(err).Msg("failed to invalidate chatbot base prompt caches")
	}
}
