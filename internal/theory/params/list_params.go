package params

import "github.com/google/uuid"

type (
	ListParams struct {
		Sort     string
		Episode  int
		AuthorID uuid.UUID
		Search   string
		Limit    int
		Offset   int
	}
)

func NewListParams(sort string, episode int, authorID uuid.UUID, search string, limit, offset int) ListParams {
	validSorts := map[string]bool{
		"new": true, "old": true,
		"popular": true, "popular_asc": true,
		"controversial": true, "controversial_asc": true,
		"credibility": true, "credibility_asc": true,
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
		Sort:     sort,
		Episode:  episode,
		AuthorID: authorID,
		Search:   search,
		Limit:    limit,
		Offset:   offset,
	}
}
