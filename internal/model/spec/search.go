package spec

import (
	"umineko_city_of_books/internal/model"
)

type (
	SearchQuery struct {
		Query  string
		Types  []model.SearchEntityType
		Limit  int
		Offset int
	}

	QuickSearchQuery struct {
		Query        string
		PerTypeLimit int
	}
)
