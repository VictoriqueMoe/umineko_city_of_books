package chatbot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

func toBasePromptResponse(prompt model.ChatbotBasePrompt) dto.ChatbotBasePromptResponse {
	return dto.ChatbotBasePromptResponse{
		ID:        prompt.ID,
		Name:      prompt.Name,
		Prompt:    prompt.Prompt,
		BotCount:  prompt.BotCount,
		CreatedAt: prompt.CreatedAt,
		UpdatedAt: prompt.UpdatedAt,
	}
}

func validateBasePrompt(req dto.ChatbotBasePromptUpsertRequest) (string, string, error) {
	name := strings.TrimSpace(req.Name)
	prompt := strings.TrimSpace(req.Prompt)

	if name == "" || prompt == "" {
		return "", "", ErrBotInvalid
	}

	return name, prompt, nil
}

func (a *adminService) ListBasePrompts(ctx context.Context) ([]dto.ChatbotBasePromptResponse, error) {
	prompts, err := a.basePromptRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list base prompts: %w", err)
	}

	out := make([]dto.ChatbotBasePromptResponse, 0, len(prompts))
	for i := range prompts {
		out = append(out, toBasePromptResponse(prompts[i]))
	}

	return out, nil
}

func (a *adminService) CreateBasePrompt(ctx context.Context, actorID uuid.UUID, req dto.ChatbotBasePromptUpsertRequest) (*dto.ChatbotBasePromptResponse, error) {
	name, prompt, err := validateBasePrompt(req)
	if err != nil {
		return nil, err
	}

	created, err := a.basePromptRepo.Create(ctx, spec.NewChatbotBasePrompt{Name: name, Prompt: prompt})
	if err != nil {
		return nil, err
	}

	a.audit(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     audit.ActionChatbotBasePromptCreate,
		TargetType: audit.TargetChatbotBasePrompt,
		TargetID:   created.ID.String(),
		Details:    fmt.Sprintf("name=%s", created.Name),
	})

	response := toBasePromptResponse(*created)

	return &response, nil
}

func (a *adminService) UpdateBasePrompt(ctx context.Context, actorID uuid.UUID, id uuid.UUID, req dto.ChatbotBasePromptUpsertRequest) (*dto.ChatbotBasePromptResponse, error) {
	name, prompt, err := validateBasePrompt(req)
	if err != nil {
		return nil, err
	}

	updated, err := a.basePromptRepo.Update(ctx, spec.ChatbotBasePromptUpdate{ID: id, Name: name, Prompt: prompt})
	if err != nil {
		return nil, err
	}

	a.reloader.Reload()

	a.audit(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     audit.ActionChatbotBasePromptUpdate,
		TargetType: audit.TargetChatbotBasePrompt,
		TargetID:   id.String(),
		Details:    fmt.Sprintf("name=%s bots=%d", updated.Name, updated.BotCount),
	})

	response := toBasePromptResponse(*updated)

	return &response, nil
}

func (a *adminService) DeleteBasePrompt(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	doomed, lookupErr := a.basePromptRepo.GetByID(ctx, id)
	if lookupErr != nil && !errors.Is(lookupErr, dao.ErrBasePromptNotFound) {
		logger.Ctx(ctx).Error().Err(lookupErr).Str("base_prompt_id", id.String()).Msg("failed to read the base prompt before deleting it")
	}

	if err := a.basePromptRepo.Delete(ctx, id); err != nil {
		return err
	}

	a.reloader.Reload()

	entry := audit.NewEntry{
		ActorID:    actorID,
		Action:     audit.ActionChatbotBasePromptDelete,
		TargetType: audit.TargetChatbotBasePrompt,
		TargetID:   id.String(),
	}

	if doomed != nil {
		entry.Details = fmt.Sprintf("name=%s bots=%d", doomed.Name, doomed.BotCount)
	}

	a.audit(ctx, entry)

	return nil
}
