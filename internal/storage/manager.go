package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"
)

var (
	ErrBackendUnavailable = errors.New("storage: backend is not enabled")
)

type (
	ProviderManager struct {
		factory     *Factory
		settingsSvc settings.Service
	}
)

func NewProviderManager(factory *Factory, settingsSvc settings.Service) *ProviderManager {
	return &ProviderManager{factory: factory, settingsSvc: settingsSvc}
}

func (m *ProviderManager) Active(ctx context.Context) (engine.Engine, error) {
	backend := config.StorageBackend(strings.TrimSpace(m.settingsSvc.Get(ctx, config.SettingStorageBackend)))

	active, ok := m.factory.Engine(backend)
	if !ok {
		return nil, fmt.Errorf("%w: storage_backend is %q but that engine is not configured", ErrBackendUnavailable, backend)
	}

	return active, nil
}

func (m *ProviderManager) EngineFor(backend config.StorageBackend) (engine.Engine, error) {
	candidate, ok := m.factory.Engine(backend)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrBackendUnavailable, backend)
	}

	return candidate, nil
}
