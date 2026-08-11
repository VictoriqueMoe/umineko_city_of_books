package chatbot

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

func toBasePromptResponse(prompt repository.ChatbotBasePrompt) dto.ChatbotBasePromptResponse {
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

func (a *adminService) CreateBasePrompt(ctx context.Context, req dto.ChatbotBasePromptUpsertRequest) (*dto.ChatbotBasePromptResponse, error) {
	name, prompt, err := validateBasePrompt(req)
	if err != nil {
		return nil, err
	}

	created := repository.ChatbotBasePrompt{ID: uuid.New(), Name: name, Prompt: prompt}
	if err := a.basePromptRepo.Create(ctx, created); err != nil {
		return nil, err
	}

	return a.basePromptByID(ctx, created.ID)
}

func (a *adminService) UpdateBasePrompt(ctx context.Context, id uuid.UUID, req dto.ChatbotBasePromptUpsertRequest) (*dto.ChatbotBasePromptResponse, error) {
	name, prompt, err := validateBasePrompt(req)
	if err != nil {
		return nil, err
	}

	if err := a.basePromptRepo.Update(ctx, repository.ChatbotBasePrompt{ID: id, Name: name, Prompt: prompt}); err != nil {
		return nil, err
	}

	a.reloader.Reload()

	return a.basePromptByID(ctx, id)
}

func (a *adminService) DeleteBasePrompt(ctx context.Context, id uuid.UUID) error {
	if err := a.basePromptRepo.Delete(ctx, id); err != nil {
		return err
	}

	a.reloader.Reload()

	return nil
}

func (a *adminService) basePromptByID(ctx context.Context, id uuid.UUID) (*dto.ChatbotBasePromptResponse, error) {
	prompt, err := a.basePromptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	response := toBasePromptResponse(*prompt)

	return &response, nil
}
