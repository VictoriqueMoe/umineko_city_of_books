package cache

import (
	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/cache/engines"
)

func New() *Manager {
	return NewManager(newEngines()...)
}

func newEngines() []engine.Engine {
	return []engine.Engine{
		engines.NewValkey(),
		engines.NewInMemory(0),
	}
}
