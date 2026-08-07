package dto

import "github.com/google/uuid"

type (
	ChatbotResponse struct {
		ID              uuid.UUID `json:"id"`
		UserID          uuid.UUID `json:"user_id"`
		Username        string    `json:"username"`
		DisplayName     string    `json:"display_name"`
		AvatarURL       string    `json:"avatar_url"`
		SystemPrompt    string    `json:"system_prompt"`
		Model           string    `json:"model"`
		ReasoningEffort string    `json:"reasoning_effort"`
		Verbosity       string    `json:"verbosity"`
		MaxOutputTokens int       `json:"max_output_tokens"`
		Enabled         bool      `json:"enabled"`
	}

	ChatbotUpsertRequest struct {
		Username        string `json:"username"`
		DisplayName     string `json:"display_name"`
		AvatarURL       string `json:"avatar_url"`
		SystemPrompt    string `json:"system_prompt"`
		Model           string `json:"model"`
		ReasoningEffort string `json:"reasoning_effort"`
		Verbosity       string `json:"verbosity"`
		MaxOutputTokens int    `json:"max_output_tokens"`
		Enabled         bool   `json:"enabled"`
	}

	ChatbotUsageResponse struct {
		Invocations        int      `json:"invocations"`
		PromptTokens       int      `json:"prompt_tokens"`
		CachedPromptTokens int      `json:"cached_prompt_tokens"`
		CacheWriteTokens   int      `json:"cache_write_tokens"`
		CompletionTokens   int      `json:"completion_tokens"`
		ReasoningTokens    int      `json:"reasoning_tokens"`
		Failed             int      `json:"failed"`
		Quota              int      `json:"quota"`
		BilledUSD          *float64 `json:"billed_usd"`
	}

	ChatbotModelsResponse struct {
		Models []string `json:"models"`
		Error  string   `json:"error,omitempty"`
	}

	ChatbotTestRequest struct {
		Model string `json:"model"`
	}

	ChatbotTestResponse struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
)
