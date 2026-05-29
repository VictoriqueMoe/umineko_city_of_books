package search

import (
	"context"
	"sort"
	"strings"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	Result struct {
		repository.SearchResult
		URL string
	}

	ChatSearcher interface {
		SearchMessagesForViewer(ctx context.Context, viewerID, roomID uuid.UUID, query string, limit, offset int) ([]repository.SearchResult, int, error)
	}

	Service interface {
		Search(ctx context.Context, query string, types []repository.SearchEntityType, limit, offset int, viewerID, roomID uuid.UUID) ([]Result, int, error)
		QuickSearch(ctx context.Context, query string, perTypeLimit int, viewerID uuid.UUID) ([]Result, error)
		ChildEntityTypes() []repository.SearchEntityType
		ParseTypes(raw string) []repository.SearchEntityType
	}

	service struct {
		repo repository.SearchRepository
		chat ChatSearcher
	}
)

func NewService(repo repository.SearchRepository, chat ChatSearcher) Service {
	return &service{repo: repo, chat: chat}
}

func (s *service) Search(ctx context.Context, query string, types []repository.SearchEntityType, limit, offset int, viewerID, roomID uuid.UUID) ([]Result, int, error) {
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

	repoTypes, chatRequested, explicit := splitChatType(types)
	includeChat := chatRequested && viewerID != uuid.Nil

	var merged []repository.SearchResult
	var total int

	window := offset + limit
	if !explicit || len(repoTypes) > 0 {
		rows, repoTotal, err := s.repo.Search(ctx, q, repoTypes, window, 0)
		if err != nil {
			return nil, 0, err
		}
		merged = append(merged, rows...)
		total += repoTotal
	}

	if includeChat {
		rows, chatTotal, err := s.chat.SearchMessagesForViewer(ctx, viewerID, roomID, q, window, 0)
		if err != nil {
			return nil, 0, err
		}
		merged = append(merged, rows...)
		total += chatTotal
	}

	sortByRank(merged)
	return decorate(pageOf(merged, offset, limit)), total, nil
}

func (s *service) QuickSearch(ctx context.Context, query string, perTypeLimit int, viewerID uuid.UUID) ([]Result, error) {
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

	if viewerID != uuid.Nil {
		chatRows, _, err := s.chat.SearchMessagesForViewer(ctx, viewerID, uuid.Nil, q, perTypeLimit, 0)
		if err != nil {
			return nil, err
		}
		rows = append(rows, chatRows...)
	}

	sortByRank(rows)
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

func splitChatType(types []repository.SearchEntityType) (repoTypes []repository.SearchEntityType, chatRequested, explicit bool) {
	explicit = len(types) > 0
	if !explicit {
		return nil, true, false
	}
	repoTypes = make([]repository.SearchEntityType, 0, len(types))
	for _, t := range types {
		if t == repository.SearchEntityChatMessage {
			chatRequested = true
			continue
		}
		repoTypes = append(repoTypes, t)
	}
	return repoTypes, chatRequested, explicit
}

func sortByRank(rows []repository.SearchResult) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Rank != rows[j].Rank {
			return rows[i].Rank > rows[j].Rank
		}
		return rows[i].CreatedAt > rows[j].CreatedAt
	})
}

func pageOf(rows []repository.SearchResult, offset, limit int) []repository.SearchResult {
	if offset >= len(rows) {
		return nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func decorate(rows []repository.SearchResult) []Result {
	out := make([]Result, len(rows))
	for i, r := range rows {
		out[i] = Result{SearchResult: r, URL: BuildURL(r)}
	}
	return out
}
