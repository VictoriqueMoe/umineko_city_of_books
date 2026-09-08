package dao

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	SearchDAO interface {
		Search(ctx context.Context, s spec.SearchQuery, tx ...*sql.Tx) ([]model.SearchResult, int, error)
		QuickSearch(ctx context.Context, s spec.QuickSearchQuery, tx ...*sql.Tx) ([]model.SearchResult, error)
	}
)
