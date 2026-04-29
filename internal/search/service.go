package search

import (
	"context"
	"strings"

	"umineko_city_of_books/internal/repository"
)

type (
	Result struct {
		repository.SearchResult
		URL string
	}

	Service interface {
		Search(ctx context.Context, query string, types []repository.SearchEntityType, limit, offset int) ([]Result, int, error)
		QuickSearch(ctx context.Context, query string, perTypeLimit int) ([]Result, error)
		ChildEntityTypes() []repository.SearchEntityType
		ParseTypes(raw string) []repository.SearchEntityType
	}

	service struct {
		repo repository.SearchRepository
	}
)

func NewService(repo repository.SearchRepository) Service {
	return &service{repo: repo}
}

func (s *service) Search(ctx context.Context, query string, types []repository.SearchEntityType, limit, offset int) ([]Result, int, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, 0, nil
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

	rows, total, err := s.repo.Search(ctx, q, types, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return decorate(rows), total, nil
}

func (s *service) QuickSearch(ctx context.Context, query string, perTypeLimit int) ([]Result, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if perTypeLimit <= 0 {
		perTypeLimit = 3
	}
	if perTypeLimit > 10 {
		perTypeLimit = 10
	}

	rows, err := s.repo.QuickSearch(ctx, q, perTypeLimit)
	if err != nil {
		return nil, err
	}
	return decorate(rows), nil
}

func (s *service) ChildEntityTypes() []repository.SearchEntityType {
	return ChildEntityTypes()
}

func (s *service) ParseTypes(raw string) []repository.SearchEntityType {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]repository.SearchEntityType, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "comments" {
			out = append(out, ChildEntityTypes()...)
			continue
		}
		out = append(out, repository.SearchEntityType(p))
	}
	return out
}

func decorate(rows []repository.SearchResult) []Result {
	out := make([]Result, len(rows))
	for i, r := range rows {
		out[i] = Result{SearchResult: r, URL: BuildURL(r)}
	}
	return out
}
