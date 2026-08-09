package openai

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
)

type (
	Service interface {
		Enabled() bool
		Complete(ctx context.Context, req CompletionRequest) (*CompletionResult, error)
		Costs(ctx context.Context, since time.Time) (*CostsResult, error)
		Models(ctx context.Context) ([]string, error)
		OnSettingsBatchChanged(keys []config.SiteSettingKey)
	}

	Message struct {
		Role    string
		Content string
	}

	CompletionRequest struct {
		Model           string
		SystemPrompt    string
		Messages        []Message
		ReasoningEffort string
		Verbosity       string
		MaxOutputTokens int
		CacheKey        string
		SafetyID        string
	}

	CompletionResult struct {
		Text               string
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
		Incomplete         bool
	}

	CostsResult struct {
		AmountUSD float64
		Currency  string
	}

	service struct {
		settingsSvc        settings.Service
		baseURL            string
		httpClient         *http.Client
		mu                 sync.RWMutex
		apiKey             string
		adminKey           string
		apiClient          sdk.Client
		adminClient        sdk.Client
		rateLimitedUntilNs atomic.Int64
	}
)

const (
	defaultBaseURL       = "https://api.openai.com/v1/"
	requestTimeout       = 60 * time.Second
	settingsTimeout      = 10 * time.Second
	defaultRateLimitHold = time.Minute
	maxErrorBodyBytes    = 512
	costsBucketLimit     = 180
	chatbotKeyPrefix     = "chatbot_"

	cacheModeExplicit = "explicit"

	typeReasoning   = "reasoning"
	typeOutputText  = "output_text"
	defaultCurrency = "usd"
)

func NewService(settingsSvc settings.Service) Service {
	return newService(settingsSvc, defaultBaseURL)
}

func newService(settingsSvc settings.Service, baseURL string) *service {
	s := &service{
		settingsSvc: settingsSvc,
		baseURL:     baseURL,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}

	s.reload()

	return s
}

func (s *service) OnSettingsBatchChanged(keys []config.SiteSettingKey) {
	if slices.ContainsFunc(keys, isChatbotKey) {
		s.reload()
	}
}

func isChatbotKey(key config.SiteSettingKey) bool {
	return strings.HasPrefix(string(key), chatbotKeyPrefix)
}

func (s *service) reload() {
	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()

	apiKey := strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingChatbotAPIKey))
	adminKey := strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingChatbotAdminKey))

	apiClient := s.newClient(option.WithAPIKey(apiKey))
	adminClient := s.newClient(option.WithAdminAPIKey(adminKey))

	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiKey = apiKey
	s.adminKey = adminKey
	s.apiClient = apiClient
	s.adminClient = adminClient
}

func (s *service) newClient(auth option.RequestOption) sdk.Client {
	return sdk.NewClient(
		option.WithBaseURL(s.baseURL),
		option.WithHTTPClient(s.httpClient),
		option.WithMaxRetries(0),
		auth,
	)
}

func (s *service) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.apiKey != ""
}

func (s *service) apiAuth() (sdk.Client, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.apiClient, s.apiKey
}

func (s *service) adminAuth() (sdk.Client, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.adminClient, s.adminKey
}

func (s *service) Complete(ctx context.Context, req CompletionRequest) (*CompletionResult, error) {
	client, apiKey := s.apiAuth()

	if apiKey == "" {
		return nil, ErrDisabled
	}

	if resetAt, limited := s.rateLimitResetAt(); limited {
		return nil, &RateLimitError{ResetAt: resetAt}
	}

	params := responses.ResponseNewParams{
		Model: req.Model,
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: buildInput(req)},
	}

	if effort := strings.TrimSpace(req.ReasoningEffort); effort != "" {
		params.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(effort)}
	}

	if verbosity := strings.TrimSpace(req.Verbosity); verbosity != "" {
		params.Text = responses.ResponseTextConfigParam{Verbosity: responses.ResponseTextConfigVerbosity(verbosity)}
	}

	if req.MaxOutputTokens > 0 {
		params.MaxOutputTokens = param.NewOpt(int64(req.MaxOutputTokens))
	}

	if cacheKey := strings.TrimSpace(req.CacheKey); cacheKey != "" {
		params.PromptCacheKey = param.NewOpt(cacheKey)
	}

	if strings.TrimSpace(req.SystemPrompt) != "" {
		params.PromptCacheOptions = responses.ResponseNewParamsPromptCacheOptions{Mode: cacheModeExplicit}
	}

	if safetyID := strings.TrimSpace(req.SafetyID); safetyID != "" {
		params.SafetyIdentifier = param.NewOpt(safetyID)
	}

	resp, err := client.Responses.New(ctx, params)
	if err != nil {
		return nil, s.sanitise(err)
	}

	return &CompletionResult{
		Text:               outputText(resp.Output),
		PromptTokens:       int(resp.Usage.InputTokens),
		CachedPromptTokens: int(resp.Usage.InputTokensDetails.CachedTokens),
		CacheWriteTokens:   int(resp.Usage.InputTokensDetails.CacheWriteTokens),
		CompletionTokens:   int(resp.Usage.OutputTokens),
		ReasoningTokens:    int(resp.Usage.OutputTokensDetails.ReasoningTokens),
		Incomplete:         resp.Status == responses.ResponseStatusIncomplete,
	}, nil
}

func (s *service) Models(ctx context.Context) ([]string, error) {
	client, apiKey := s.apiAuth()

	if apiKey == "" {
		return nil, ErrDisabled
	}

	if resetAt, limited := s.rateLimitResetAt(); limited {
		return nil, &RateLimitError{ResetAt: resetAt}
	}

	pager := client.Models.ListAutoPaging(ctx)

	models := make([]sdk.Model, 0)
	for pager.Next() {
		models = append(models, pager.Current())
	}
	if err := pager.Err(); err != nil {
		return nil, s.sanitise(err)
	}

	slices.SortStableFunc(models, func(a, b sdk.Model) int {
		return cmp.Compare(b.Created, a.Created)
	})

	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	return ids, nil
}

func (s *service) Costs(ctx context.Context, since time.Time) (*CostsResult, error) {
	client, adminKey := s.adminAuth()

	if adminKey == "" {
		return nil, nil
	}

	if resetAt, limited := s.rateLimitResetAt(); limited {
		return nil, &RateLimitError{ResetAt: resetAt}
	}

	params := sdk.AdminOrganizationUsageCostsParams{
		StartTime: since.Unix(),
		Limit:     param.NewOpt(int64(costsBucketLimit)),
	}

	page, err := client.Admin.Organization.Usage.Costs(ctx, params)
	if err != nil {
		return nil, s.sanitise(err)
	}

	result := &CostsResult{Currency: defaultCurrency}

	for _, bucket := range page.Data {
		for _, item := range bucket.Results {
			result.AmountUSD += item.Amount.Value

			if item.Amount.Currency != "" {
				result.Currency = item.Amount.Currency
			}
		}
	}

	return result, nil
}

func buildInput(req CompletionRequest) responses.ResponseInputParam {
	input := make(responses.ResponseInputParam, 0, len(req.Messages)+1)

	if prompt := strings.TrimSpace(req.SystemPrompt); prompt != "" {
		input = append(input, responses.ResponseInputItemParamOfMessage(cacheablePrefix(prompt), responses.EasyInputMessageRoleSystem))
	}

	for _, m := range req.Messages {
		input = append(input, responses.ResponseInputItemParamOfMessage(m.Content, responses.EasyInputMessageRole(m.Role)))
	}

	return input
}

func cacheablePrefix(prompt string) responses.ResponseInputMessageContentListParam {
	block := responses.ResponseInputTextParam{Text: prompt}
	block.PromptCacheBreakpoint = responses.ResponseInputTextPromptCacheBreakpointParam{Mode: cacheModeExplicit}

	return responses.ResponseInputMessageContentListParam{
		{OfInputText: &block},
	}
}

func outputText(items []responses.ResponseOutputItemUnion) string {
	var sb strings.Builder

	for _, item := range items {
		if item.Type == typeReasoning {
			continue
		}

		for _, content := range item.Content {
			if content.Type != typeOutputText {
				continue
			}

			sb.WriteString(content.Text)
		}
	}

	return strings.TrimSpace(sb.String())
}

func (s *service) sanitise(err error) error {
	apiErr, ok := errors.AsType[*sdk.Error](err)
	if !ok {
		return fmt.Errorf("call openai: %w", stripRequestURL(err))
	}

	if apiErr.StatusCode == http.StatusTooManyRequests {
		resetAt := parseRateLimitReset(errorHeader(apiErr), time.Now())
		s.rateLimitedUntilNs.Store(resetAt.UnixNano())

		return &RateLimitError{ResetAt: resetAt}
	}

	return &APIError{StatusCode: apiErr.StatusCode, Body: errorBody(apiErr)}
}

func errorHeader(apiErr *sdk.Error) http.Header {
	if apiErr.Response == nil {
		return http.Header{}
	}

	return apiErr.Response.Header
}

func errorBody(apiErr *sdk.Error) string {
	if apiErr.Response != nil && apiErr.Response.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(apiErr.Response.Body, maxErrorBodyBytes))

		if err == nil && len(raw) > 0 {
			return string(raw)
		}
	}

	body := apiErr.RawJSON()
	if len(body) > maxErrorBodyBytes {
		return body[:maxErrorBodyBytes]
	}

	return body
}

func (s *service) rateLimitResetAt() (time.Time, bool) {
	ns := s.rateLimitedUntilNs.Load()
	if ns == 0 {
		return time.Time{}, false
	}

	resetAt := time.Unix(0, ns)
	if time.Now().After(resetAt) {
		return time.Time{}, false
	}

	return resetAt, true
}

func stripRequestURL(err error) error {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		return urlErr.Err
	}

	return err
}

func parseRateLimitReset(h http.Header, now time.Time) time.Time {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return now.Add(time.Duration(secs) * time.Second)
		}

		if t, err := http.ParseTime(v); err == nil {
			return t
		}
	}

	return now.Add(defaultRateLimitHold)
}
