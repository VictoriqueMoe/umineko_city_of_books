package storage

import (
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"
	"umineko_city_of_books/internal/storage/engines"
)

type (
	Factory struct {
		engines []engine.Engine
	}
)

func NewEngines(settingsSvc settings.Service) []engine.Engine {
	return []engine.Engine{
		engines.NewLocal(settingsSvc),
		engines.NewS3(settingsSvc),
	}
}

func NewFactory(candidates ...engine.Engine) *Factory {
	return &Factory{engines: candidates}
}

func (f *Factory) Engines() []engine.Engine {
	return f.engines
}

func (f *Factory) EnabledEngines() []engine.Engine {
	enabled := make([]engine.Engine, 0, len(f.engines))
	for _, candidate := range f.engines {
		if candidate.Enabled() {
			enabled = append(enabled, candidate)
		}
	}

	return enabled
}

func (f *Factory) Engine(backend config.StorageBackend) (engine.Engine, bool) {
	for _, candidate := range f.engines {
		if candidate.ID() == backend && candidate.Enabled() {
			return candidate, true
		}
	}

	return nil, false
}
