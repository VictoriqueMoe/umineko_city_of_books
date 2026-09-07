package spec

import (
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
)

type (
	NewInvocation struct {
		BotUserID uuid.UUID
		UserID    uuid.UUID
		RoomID    *uuid.UUID
		MessageID uuid.UUID
		Channel   string
		Model     string
	}

	InvocationUsage struct {
		PromptTokens       int
		CachedPromptTokens int
		CacheWriteTokens   int
		CompletionTokens   int
		ReasoningTokens    int
	}

	InvocationCompletion struct {
		ID     uuid.UUID
		Usage  InvocationUsage
		Status model.InvocationStatus
	}

	NewChatbotWithAccount struct {
		Account      NewUser
		Bot          model.Chatbot
		VanityRoleID string
	}

	ChatbotAccountUpdate struct {
		Bot         model.Chatbot
		DisplayName string
		AvatarURL   string
	}
)
