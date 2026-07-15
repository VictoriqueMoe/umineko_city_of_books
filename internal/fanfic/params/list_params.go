package params

import "umineko_city_of_books/internal/bounds"

type (
	ListParams struct {
		Sort       string
		Series     string
		Rating     string
		GenreA     string
		GenreB     string
		Language   string
		Status     string
		Tag        string
		CharacterA string
		CharacterB string
		CharacterC string
		CharacterD string
		IsPairing  bool
		ShowLemons bool
		Search     string
		Limit      int
		Offset     int
	}
)

func NewListParams(filters ListParams, page bounds.Page) ListParams {
	if filters.Sort == "" {
		filters.Sort = "updated"
	}

	filters.Limit = page.Limit()
	filters.Offset = page.Offset()

	return filters
}
