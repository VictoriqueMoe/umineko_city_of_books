package search

import "umineko_city_of_books/internal/model"

func AllEntityTypes() []model.SearchEntityType {
	srcs := model.SearchSources()
	out := make([]model.SearchEntityType, len(srcs))
	for i, s := range srcs {
		out[i] = s.Type
	}
	return out
}

func ChildEntityTypes() []model.SearchEntityType {
	srcs := model.SearchSources()
	out := make([]model.SearchEntityType, 0, len(srcs))
	for _, s := range srcs {
		if s.ParentIDExpr != "" {
			out = append(out, s.Type)
		}
	}
	return out
}
