package params

import (
	"umineko_city_of_books/internal/bounds"

	"github.com/google/uuid"
)

type (
	ListParams struct {
		Sort            string
		Work            string
		AuthorID        uuid.UUID
		Search          string
		IncludeArchived bool
		Limit           int
		Offset          int
	}
)

func NewListParams(sort string, work string, authorID uuid.UUID, search string, includeArchived bool, limit, offset int) ListParams {
	validSorts := map[string]bool{
		"new":             true,
		"old":             true,
		"recently_active": true,
		"most_followed":   true,
	}
	if !validSorts[sort] {
		sort = "new"
	}
	page := bounds.NewPage(limit, offset)

	return ListParams{
		Sort:            sort,
		Work:            work,
		AuthorID:        authorID,
		Search:          search,
		IncludeArchived: includeArchived,
		Limit:           page.Limit(),
		Offset:          page.Offset(),
	}
}
