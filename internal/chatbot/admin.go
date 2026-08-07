package chatbot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
)

const (
	botVanityRoleID = "bot"

	testSystemPrompt    = "You are a connectivity probe. Reply with a single word."
	testUserMessage     = "ping"
	testMaxOutputTokens = 16
)

var (
	ErrBotNotFound     = errors.New("chatbot not found")
	ErrBotUsernameUsed = errors.New("that username is already taken")
	ErrBotInvalid      = errors.New("a chatbot needs a username, a display name and a system prompt")
	ErrBotUnknownModel = errors.New("the provider does not offer that model")
)

type (
	AdminService interface {
		List(ctx context.Context) ([]dto.ChatbotResponse, error)
		Create(ctx context.Context, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error)
		Update(ctx context.Context, id uuid.UUID, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error)
		Delete(ctx context.Context, id uuid.UUID) error
		Usage(ctx context.Context, since time.Time) (*dto.ChatbotUsageResponse, error)
		Models(ctx context.Context) ([]string, error)
		Test(ctx context.Context, model string) (bool, string, error)
	}

	adminService struct {
		botRepo    repository.ChatbotRepository
		vanityRepo repository.VanityRoleRepository
		userSvc    user.Service
		openaiSvc  openai.Service
		reloader   Service
	}
)

func NewAdminService(botRepo repository.ChatbotRepository, vanityRepo repository.VanityRoleRepository, userSvc user.Service, openaiSvc openai.Service, reloader Service) AdminService {
	return &adminService{
		botRepo:    botRepo,
		vanityRepo: vanityRepo,
		userSvc:    userSvc,
		openaiSvc:  openaiSvc,
		reloader:   reloader,
	}
}

func (a *adminService) List(ctx context.Context) ([]dto.ChatbotResponse, error) {
	bots, err := a.botRepo.ListBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chatbots: %w", err)
	}

	out := make([]dto.ChatbotResponse, 0, len(bots))
	for i := range bots {
		out = append(out, toResponse(bots[i]))
	}

	return out, nil
}

func (a *adminService) Create(ctx context.Context, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error) {
	if err := a.validateUpsert(ctx, req); err != nil {
		return nil, err
	}

	displayName := user.ClampDisplayName(req.DisplayName)
	if displayName == "" {
		return nil, ErrBotInvalid
	}

	if err := a.userSvc.CheckUsernameAvailable(ctx, strings.TrimSpace(req.Username)); err != nil {
		if errors.Is(err, user.ErrUsernameTaken) {
			return nil, ErrBotUsernameUsed
		}

		return nil, fmt.Errorf("check username: %w", err)
	}

	userID := uuid.New()
	if err := a.botRepo.CreateBotAccount(ctx, userID, req.Username, displayName, req.AvatarURL); err != nil {
		return nil, fmt.Errorf("create bot account: %w", err)
	}

	if err := a.vanityRepo.AssignToUser(ctx, userID, botVanityRoleID); err != nil {
		return nil, fmt.Errorf("assign bot badge: %w", err)
	}

	bot := repository.Chatbot{
		ID:              uuid.New(),
		UserID:          userID,
		SystemPrompt:    req.SystemPrompt,
		Model:           req.Model,
		ReasoningEffort: req.ReasoningEffort,
		Verbosity:       req.Verbosity,
		MaxOutputTokens: req.MaxOutputTokens,
		Enabled:         req.Enabled,
	}

	if err := a.botRepo.CreateBot(ctx, bot); err != nil {
		return nil, fmt.Errorf("create chatbot: %w", err)
	}

	a.reloader.Reload()

	return a.byUser(ctx, userID)
}

func (a *adminService) Update(ctx context.Context, id uuid.UUID, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error) {
	if err := a.validateUpsert(ctx, req); err != nil {
		return nil, err
	}

	displayName := user.ClampDisplayName(req.DisplayName)
	if displayName == "" {
		return nil, ErrBotInvalid
	}

	bots, err := a.botRepo.ListBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chatbots: %w", err)
	}

	var current *repository.Chatbot
	for i := range bots {
		if bots[i].ID == id {
			current = &bots[i]

			break
		}
	}
	if current == nil {
		return nil, ErrBotNotFound
	}

	updated := *current
	updated.SystemPrompt = req.SystemPrompt
	updated.Model = req.Model
	updated.ReasoningEffort = req.ReasoningEffort
	updated.Verbosity = req.Verbosity
	updated.MaxOutputTokens = req.MaxOutputTokens
	updated.Enabled = req.Enabled

	if err := a.botRepo.UpdateBot(ctx, updated); err != nil {
		return nil, fmt.Errorf("update chatbot: %w", err)
	}

	if err := a.botRepo.UpdateBotAccount(ctx, current.UserID, displayName, req.AvatarURL); err != nil {
		return nil, fmt.Errorf("update bot profile: %w", err)
	}

	a.reloader.Reload()

	return a.byUser(ctx, current.UserID)
}

func (a *adminService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := a.botRepo.DeleteBot(ctx, id); err != nil {
		if errors.Is(err, repository.ErrBotNotFound) {
			return ErrBotNotFound
		}

		return fmt.Errorf("delete chatbot: %w", err)
	}

	a.reloader.Reload()

	return nil
}

func (a *adminService) Usage(ctx context.Context, since time.Time) (*dto.ChatbotUsageResponse, error) {
	stats, err := a.botRepo.StatsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("chatbot stats: %w", err)
	}

	out := &dto.ChatbotUsageResponse{
		Invocations:        stats.Invocations,
		PromptTokens:       stats.PromptTokens,
		CachedPromptTokens: stats.CachedPromptTokens,
		CacheWriteTokens:   stats.CacheWriteTokens,
		CompletionTokens:   stats.CompletionTokens,
		ReasoningTokens:    stats.ReasoningTokens,
		Failed:             stats.Failed,
		Quota:              stats.Quota,
	}

	costs, costErr := a.openaiSvc.Costs(ctx, since)
	if costErr == nil && costs != nil {
		out.BilledUSD = new(costs.AmountUSD)
	}

	return out, nil
}

func (a *adminService) Models(ctx context.Context) ([]string, error) {
	models, err := a.openaiSvc.Models(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	return models, nil
}

func (a *adminService) Test(ctx context.Context, model string) (bool, string, error) {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return false, "pick a model first", nil
	}

	res, err := a.openaiSvc.Complete(ctx, openai.CompletionRequest{
		Model:           trimmed,
		SystemPrompt:    testSystemPrompt,
		Messages:        []openai.Message{{Role: "user", Content: testUserMessage}},
		MaxOutputTokens: testMaxOutputTokens,
	})
	if err != nil {
		return false, providerMessage(err), nil
	}
	if res == nil {
		return false, "the provider returned no response", nil
	}

	return true, "", nil
}

func providerMessage(err error) string {
	if apiErr, ok := errors.AsType[*openai.APIError](err); ok {
		return apiErr.Message()
	}

	return err.Error()
}

func (a *adminService) byUser(ctx context.Context, userID uuid.UUID) (*dto.ChatbotResponse, error) {
	bot, err := a.botRepo.GetBotByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get chatbot: %w", err)
	}
	if bot == nil {
		return nil, ErrBotNotFound
	}

	resp := toResponse(*bot)

	return &resp, nil
}

func toResponse(bot repository.Chatbot) dto.ChatbotResponse {
	return dto.ChatbotResponse{
		ID:              bot.ID,
		UserID:          bot.UserID,
		Username:        bot.Username,
		DisplayName:     bot.DisplayName,
		AvatarURL:       bot.AvatarURL,
		SystemPrompt:    bot.SystemPrompt,
		Model:           bot.Model,
		ReasoningEffort: bot.ReasoningEffort,
		Verbosity:       bot.Verbosity,
		MaxOutputTokens: bot.MaxOutputTokens,
		Enabled:         bot.Enabled,
	}
}

func (a *adminService) validateUpsert(ctx context.Context, req dto.ChatbotUpsertRequest) error {
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.DisplayName) == "" || strings.TrimSpace(req.SystemPrompt) == "" {
		return ErrBotInvalid
	}

	switch req.ReasoningEffort {
	case "", "none", "low", "medium", "high", "xhigh", "max":
	default:
		return fmt.Errorf("%w: unknown reasoning effort", ErrBotInvalid)
	}

	switch req.Verbosity {
	case "", "low", "medium", "high":
	default:
		return fmt.Errorf("%w: unknown verbosity", ErrBotInvalid)
	}

	return validateModel(ctx, a.openaiSvc, req.Model)
}

func ModelValidator(openaiSvc openai.Service) settings.Validator {
	return func(ctx context.Context, value string) error {
		return validateModel(ctx, openaiSvc, value)
	}
}

func validateModel(ctx context.Context, openaiSvc openai.Service, model string) error {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return nil
	}

	available, err := openaiSvc.Models(ctx)
	if err != nil || len(available) == 0 {
		return nil
	}

	if slices.Contains(available, trimmed) {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrBotUnknownModel, trimmed)
}
