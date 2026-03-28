package params

type (
	ListParams struct {
		Sort     string
		Episode  int
		AuthorID int
		Limit    int
		Offset   int
	}
)

func NewListParams(sort string, episode, authorID, limit, offset int) ListParams {
	if sort != "popular" && sort != "controversial" {
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
		Limit:    limit,
		Offset:   offset,
	}
}
