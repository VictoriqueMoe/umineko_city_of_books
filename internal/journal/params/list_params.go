package params

import "github.com/google/uuid"

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
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return ListParams{
		Sort:            sort,
		Work:            work,
		AuthorID:        authorID,
		Search:          search,
		IncludeArchived: includeArchived,
		Limit:           limit,
		Offset:          offset,
	}
}
