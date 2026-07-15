package params

import (
	"umineko_city_of_books/internal/bounds"

	"github.com/google/uuid"
)

type (
	ListParams struct {
		Sort     string
		Episode  int
		AuthorID uuid.UUID
		Search   string
		Series   string
		Limit    int
		Offset   int
	}
)

func NewListParams(sort string, episode int, authorID uuid.UUID, search string, series string, limit, offset int) ListParams {
	validSorts := map[string]bool{
		"new": true, "old": true,
		"popular": true, "popular_asc": true,
		"controversial": true, "controversial_asc": true,
		"credibility": true, "credibility_asc": true,
	}
	if !validSorts[sort] {
		sort = "new"
	}
	if series == "" {
		series = "umineko"
	}

	page := bounds.NewPage(limit, offset)

	return ListParams{
		Sort:     sort,
		Episode:  episode,
		AuthorID: authorID,
		Search:   search,
		Series:   series,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	}
}
