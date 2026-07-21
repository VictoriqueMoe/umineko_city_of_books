package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	SettingsRepository interface {
		Get(ctx context.Context, key string) (string, error)
		GetAll(ctx context.Context) (map[string]string, error)
		Set(ctx context.Context, key, value string, updatedBy uuid.UUID) error
		SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID) error
		Delete(ctx context.Context, key string) error
	}
)
