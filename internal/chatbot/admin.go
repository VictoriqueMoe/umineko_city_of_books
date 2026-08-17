package chatbot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
)

const (
	botVanityRoleID = "bot"
	botPasswordHash = "!"
	botHomePage     = "landing"

	testSystemPrompt    = "You are a connectivity probe. Reply with a single word."
	testUserMessage     = "ping"
	testMaxOutputTokens = 16
)

type (
	AdminService interface {
		List(ctx context.Context) ([]dto.ChatbotResponse, error)
		Create(ctx context.Context, actorID uuid.UUID, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error)
		Update(ctx context.Context, actorID uuid.UUID, id uuid.UUID, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error)
		Delete(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error
		Usage(ctx context.Context, since time.Time) (*dto.ChatbotUsageResponse, error)
		Models(ctx context.Context) ([]string, error)
		Test(ctx context.Context, model string) (bool, string, error)

		ListBasePrompts(ctx context.Context) ([]dto.ChatbotBasePromptResponse, error)
		CreateBasePrompt(ctx context.Context, actorID uuid.UUID, req dto.ChatbotBasePromptUpsertRequest) (*dto.ChatbotBasePromptResponse, error)
		UpdateBasePrompt(ctx context.Context, actorID uuid.UUID, id uuid.UUID, req dto.ChatbotBasePromptUpsertRequest) (*dto.ChatbotBasePromptResponse, error)
		DeleteBasePrompt(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error
	}

	adminService struct {
		botRepo        repository.ChatbotRepository
		basePromptRepo repository.ChatbotBasePromptRepository
		auditRepo      repository.AuditLogRepository
		userSvc        user.Service
		openaiSvc      openai.Service
		reloader       Service
	}
)

func NewAdminService(
	botRepo repository.ChatbotRepository,
	basePromptRepo repository.ChatbotBasePromptRepository,
	auditRepo repository.AuditLogRepository,
	userSvc user.Service,
	openaiSvc openai.Service,
	reloader Service,
) AdminService {
	return &adminService{
		botRepo:        botRepo,
		basePromptRepo: basePromptRepo,
		auditRepo:      auditRepo,
		userSvc:        userSvc,
		openaiSvc:      openaiSvc,
		reloader:       reloader,
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

func (a *adminService) Create(ctx context.Context, actorID uuid.UUID, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error) {
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

	account := repository.NewUser{
		Username:      strings.TrimSpace(req.Username),
		PasswordHash:  botPasswordHash,
		DisplayName:   displayName,
		AvatarURL:     req.AvatarURL,
		HomePage:      botHomePage,
		IsBot:         true,
		DMsEnabled:    true,
		EmailVerified: true,
	}

	bot := repository.Chatbot{
		SystemPrompt:    req.SystemPrompt,
		BasePromptID:    req.BasePromptID,
		Model:           req.Model,
		ReasoningEffort: req.ReasoningEffort,
		Verbosity:       req.Verbosity,
		MaxOutputTokens: req.MaxOutputTokens,
		Enabled:         req.Enabled,
	}

	created, err := a.botRepo.CreateBotWithAccount(ctx, account, bot, botVanityRoleID)
	if err != nil {
		return nil, fmt.Errorf("create chatbot: %w", err)
	}

	a.reloader.Reload()

	a.audit(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatbotCreate,
		TargetType: repository.AuditTargetChatbot,
		TargetID:   created.ID.String(),
		Details:    fmt.Sprintf("username=%s name=%s model=%s enabled=%t", created.Username, created.DisplayName, created.Model, created.Enabled),
		SubjectID:  created.UserID,
	})

	response := toResponse(*created)

	return &response, nil
}

func (a *adminService) Update(ctx context.Context, actorID uuid.UUID, id uuid.UUID, req dto.ChatbotUpsertRequest) (*dto.ChatbotResponse, error) {
	if err := a.validateUpsert(ctx, req); err != nil {
		return nil, err
	}

	displayName := user.ClampDisplayName(req.DisplayName)
	if displayName == "" {
		return nil, ErrBotInvalid
	}

	current, err := a.findBot(ctx, id)
	if err != nil {
		return nil, err
	}

	spec := *current
	spec.SystemPrompt = req.SystemPrompt
	spec.BasePromptID = req.BasePromptID
	spec.Model = req.Model
	spec.ReasoningEffort = req.ReasoningEffort
	spec.Verbosity = req.Verbosity
	spec.MaxOutputTokens = req.MaxOutputTokens
	spec.Enabled = req.Enabled

	updated, err := a.botRepo.UpdateBotWithAccount(ctx, spec, displayName, req.AvatarURL)
	if err != nil {
		return nil, fmt.Errorf("update chatbot: %w", err)
	}

	a.reloader.Reload()

	a.audit(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatbotUpdate,
		TargetType: repository.AuditTargetChatbot,
		TargetID:   id.String(),
		Details:    chatbotChanges(*current, req, displayName),
		SubjectID:  current.UserID,
	})

	response := toResponse(*updated)

	return &response, nil
}

func (a *adminService) Delete(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	doomed, lookupErr := a.findBot(ctx, id)
	if lookupErr != nil && !errors.Is(lookupErr, ErrBotNotFound) {
		logger.Log.Error().Err(lookupErr).Str("chatbot_id", id.String()).Msg("failed to read the chatbot before deleting it")
	}

	if err := a.botRepo.DeleteBot(ctx, id); err != nil {
		if errors.Is(err, repository.ErrBotNotFound) {
			return ErrBotNotFound
		}

		return fmt.Errorf("delete chatbot: %w", err)
	}

	a.reloader.Reload()

	entry := repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatbotDelete,
		TargetType: repository.AuditTargetChatbot,
		TargetID:   id.String(),
	}

	if doomed != nil {
		entry.Details = fmt.Sprintf("username=%s name=%s", doomed.Username, doomed.DisplayName)
		entry.SubjectID = doomed.UserID
	}

	a.audit(ctx, entry)

	return nil
}

func (a *adminService) findBot(ctx context.Context, id uuid.UUID) (*repository.Chatbot, error) {
	bots, err := a.botRepo.ListBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chatbots: %w", err)
	}

	for i := range bots {
		if bots[i].ID == id {
			return &bots[i], nil
		}
	}

	return nil, ErrBotNotFound
}

func (a *adminService) audit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := a.auditRepo.Create(ctx, entry); err != nil {
		logger.Log.Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func sameBasePrompt(current, next *uuid.UUID) bool {
	if current == nil || next == nil {
		return current == next
	}

	return *current == *next
}

func chatbotChanges(current repository.Chatbot, req dto.ChatbotUpsertRequest, displayName string) string {
	changed := make([]string, 0, 9)

	if current.SystemPrompt != req.SystemPrompt {
		changed = append(changed, "system_prompt")
	}

	if !sameBasePrompt(current.BasePromptID, req.BasePromptID) {
		changed = append(changed, "base_prompt")
	}

	if current.Model != req.Model {
		changed = append(changed, fmt.Sprintf("model=%s", req.Model))
	}

	if current.ReasoningEffort != req.ReasoningEffort {
		changed = append(changed, fmt.Sprintf("reasoning_effort=%s", req.ReasoningEffort))
	}

	if current.Verbosity != req.Verbosity {
		changed = append(changed, fmt.Sprintf("verbosity=%s", req.Verbosity))
	}

	if current.MaxOutputTokens != req.MaxOutputTokens {
		changed = append(changed, fmt.Sprintf("max_output_tokens=%d", req.MaxOutputTokens))
	}

	if current.Enabled != req.Enabled {
		changed = append(changed, fmt.Sprintf("enabled=%t", req.Enabled))
	}

	if current.DisplayName != displayName {
		changed = append(changed, fmt.Sprintf("display_name=%s", displayName))
	}

	if current.AvatarURL != req.AvatarURL {
		changed = append(changed, "avatar")
	}

	if len(changed) == 0 {
		return "unchanged"
	}

	return strings.Join(changed, " ")
}

func (a *adminService) Usage(ctx context.Context, since time.Time) (*dto.ChatbotUsageResponse, error) {
	stats, err := a.botRepo.StatsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("chatbot stats: %w", err)
	}

	channels := make([]dto.ChatbotChannelUsage, 0, len(stats.Channels))
	for i := range stats.Channels {
		channels = append(channels, dto.ChatbotChannelUsage{
			Channel:            stats.Channels[i].Channel,
			Invocations:        stats.Channels[i].Invocations,
			PromptTokens:       stats.Channels[i].PromptTokens,
			CachedPromptTokens: stats.Channels[i].CachedPromptTokens,
			CacheWriteTokens:   stats.Channels[i].CacheWriteTokens,
			CompletionTokens:   stats.Channels[i].CompletionTokens,
			ReasoningTokens:    stats.Channels[i].ReasoningTokens,
		})
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
		Channels:           channels,
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

func toResponse(bot repository.Chatbot) dto.ChatbotResponse {
	return dto.ChatbotResponse{
		ID:              bot.ID,
		UserID:          bot.UserID,
		Username:        bot.Username,
		DisplayName:     bot.DisplayName,
		AvatarURL:       bot.AvatarURL,
		SystemPrompt:    bot.SystemPrompt,
		BasePromptID:    bot.BasePromptID,
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
