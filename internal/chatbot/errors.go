package chatbot

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"
)

var (
	ErrBotNotFound     = errors.New("chatbot not found")
	ErrBotUsernameUsed = errors.New("that username is already taken")
	ErrBotInvalid      = errors.New("a chatbot needs a username, a display name and a system prompt")
	ErrBotUnknownModel = errors.New("the provider does not offer that model")

	ErrOptInRoleNotFound  = errors.New("that role does not exist")
	ErrOptInRoleIsSystem  = errors.New("auto-managed roles cannot be handed out on opt in")
	ErrOptInRoleNoChatbot = errors.New("that role does not carry the summon chatbots permission")
)

func classifyProvider(ctx context.Context, err error) outcome {
	if rateErr, ok := errors.AsType[*openai.RateLimitError](err); ok {
		return outcome{reason: reasonProviderLimited, stage: stagePostModel, status: repository.InvocationFailed, clearsAt: rateErr.ResetAt, err: err}
	}

	if errors.Is(err, openai.ErrDisabled) {
		return outcome{reason: reasonNotConfigured, stage: stagePostModel, status: repository.InvocationFailed, err: err}
	}

	if ctx.Err() != nil {
		return outcome{reason: reasonTimeout, stage: stagePostModel, status: repository.InvocationFailed, err: err}
	}

	return outcome{reason: reasonProviderDown, stage: stagePostModel, status: repository.InvocationFailed, detail: openai.Reason(err), err: err}
}

func classifyDelivery(err error) outcome {
	if _, banned := errors.AsType[*chat.ErrBannedWordMatch](err); banned {
		return outcome{reason: reasonFiltered, stage: stagePostModel, status: repository.InvocationRefused, err: err}
	}

	return outcome{reason: reasonUndeliverable, stage: stagePostModel, status: repository.InvocationRefused, err: err}
}
