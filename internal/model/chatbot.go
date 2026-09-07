package model

import (
	"github.com/google/uuid"
)

const (
	InvocationPending InvocationStatus = "pending"
	InvocationReplied InvocationStatus = "replied"
	InvocationRefused InvocationStatus = "refused"
	InvocationFailed  InvocationStatus = "failed"
	InvocationQuota   InvocationStatus = "quota"
)

type (
	InvocationStatus string

	Chatbot struct {
		ID              uuid.UUID
		UserID          uuid.UUID
		Username        string
		DisplayName     string
		AvatarURL       string
		SystemPrompt    string
		BasePromptID    *uuid.UUID
		BasePrompt      string
		Model           string
		ReasoningEffort string
		Verbosity       string
		MaxOutputTokens int
		Enabled         bool
	}

	ChatbotInvocation struct {
		ID        uuid.UUID
		BotUserID uuid.UUID
		UserID    uuid.UUID
		RoomID    *uuid.UUID
		MessageID uuid.UUID
		Channel   string
		Model     string
		Status    InvocationStatus
	}

	ChatbotChannelStats struct {
		Channel            string
		Invocations        int
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
	}

	ChatbotStats struct {
		Invocations        int
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
		Failed             int
		Quota              int
		Channels           []ChatbotChannelStats
	}
)
