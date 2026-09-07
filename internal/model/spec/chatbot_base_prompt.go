package spec

import (
	"github.com/google/uuid"
)

type (
	NewChatbotBasePrompt struct {
		Name   string
		Prompt string
	}

	ChatbotBasePromptUpdate struct {
		ID     uuid.UUID
		Name   string
		Prompt string
	}
)
