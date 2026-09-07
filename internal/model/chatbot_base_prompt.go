package model

import (
	"time"

	"github.com/google/uuid"
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
)
