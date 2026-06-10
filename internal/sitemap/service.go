package sitemap

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
)

const dateFormat = "2006-01-02"

var staticPaths = []string{
	"",
	"/welcome",
	"/theories",
	"/game-board",
	"/game-board/umineko",
	"/game-board/higurashi",
	"/game-board/ciconia",
	"/game-board/higanbana",
	"/game-board/roseguns",
	"/gallery",
	"/gallery/umineko",
	"/gallery/higurashi",
	"/gallery/ciconia",
	"/quotes",
	"/mysteries",
	"/ships",
	"/fanfiction",
	"/suggestions",
	"/games",
	"/games/live",
	"/games/past",
	"/games/chess",
	"/games/othello",
	"/search",
	"/live",
	"/login",
}

type (
	Entry struct {
		URL     string
		LastMod string
	}

	IndexEntry struct {
		Loc string
	}

	Service interface {
		BaseURL() string
		IndexEntries() []IndexEntry
		StaticEntries(ctx context.Context) []Entry
		Theories(ctx context.Context) ([]Entry, error)
		Posts(ctx context.Context) ([]Entry, error)
		Art(ctx context.Context) ([]Entry, error)
		Users(ctx context.Context) ([]Entry, error)
		Mysteries(ctx context.Context) ([]Entry, error)
		Ships(ctx context.Context) ([]Entry, error)
		Fanfics(ctx context.Context) ([]Entry, error)
		Journals(ctx context.Context) ([]Entry, error)
	}

	service struct {
		repo        repository.SitemapRepository
		settingsSvc settings.Service
		baseURL     string
	}
)

func NewService(repo repository.SitemapRepository, settingsSvc settings.Service, baseURL string) Service {
	return &service{repo: repo, settingsSvc: settingsSvc, baseURL: baseURL}
}

func (s *service) BaseURL() string {
	return s.baseURL
}

func (s *service) IndexEntries() []IndexEntry {
	suffixes := []string{
		"/sitemap-static.xml",
		"/sitemap-theories.xml",
		"/sitemap-posts.xml",
		"/sitemap-art.xml",
		"/sitemap-users.xml",
		"/sitemap-mysteries.xml",
		"/sitemap-ships.xml",
		"/sitemap-fanfics.xml",
		"/sitemap-journals.xml",
	}
	entries := make([]IndexEntry, len(suffixes))
	for i, suf := range suffixes {
		entries[i] = IndexEntry{Loc: s.baseURL + suf}
	}
	return entries
}

func (s *service) StaticEntries(ctx context.Context) []Entry {
	now := time.Now().Format(dateFormat)
	entries := make([]Entry, 0, len(staticPaths)+1)
	for _, p := range staticPaths {
		entries = append(entries, Entry{URL: s.baseURL + p, LastMod: now})
	}
	if s.settingsSvc.Get(ctx, config.SettingRulesPage) != "" {
		entries = append(entries, Entry{URL: s.baseURL + "/rules", LastMod: now})
	}
	return entries
}

func (s *service) entries(rows []repository.SitemapEntry, pathPrefix string) []Entry {
	entries := make([]Entry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, Entry{
			URL:     s.baseURL + pathPrefix + r.ID,
			LastMod: r.LastMod.Format(dateFormat),
		})
	}
	return entries
}

func (s *service) Theories(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListTheories(ctx)
	if err != nil {
		return nil, err
	}
	return s.entries(rows, "/theory/"), nil
}

func (s *service) Posts(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListPosts(ctx)
	if err != nil {
		return nil, err
	}
	return s.entries(rows, "/game-board/"), nil
}

func (s *service) Art(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListArt(ctx)
	if err != nil {
		return nil, err
	}
	return s.entries(rows, "/gallery/art/"), nil
}

func (s *service) Mysteries(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListMysteries(ctx)
	if err != nil {
		return nil, err
	}
	return s.entries(rows, "/mystery/"), nil
}

func (s *service) Ships(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListShips(ctx)
	if err != nil {
		return nil, err
	}
	return s.entries(rows, "/ships/"), nil
}

func (s *service) Fanfics(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListFanfics(ctx)
	if err != nil {
		return nil, err
	}
	return s.entries(rows, "/fanfiction/"), nil
}

func (s *service) Users(ctx context.Context) ([]Entry, error) {
	usernames, err := s.repo.ListUsernames(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(usernames))
	for _, u := range usernames {
		entries = append(entries, Entry{URL: s.baseURL + "/user/" + u})
	}
	return entries, nil
}

func (s *service) Journals(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.ListJournalRows(ctx)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	seenJournals := make(map[string]bool)
	for _, row := range rows {
		if !seenJournals[row.JournalID] {
			entries = append(entries, Entry{
				URL:     s.baseURL + "/journals/" + row.JournalID,
				LastMod: row.JournalUpdatedAt.Format(dateFormat),
			})
			seenJournals[row.JournalID] = true
		}
		if row.EntryNumber.Valid {
			lastMod := row.JournalUpdatedAt
			if row.EntryUpdatedAt.Valid {
				lastMod = row.EntryUpdatedAt.Time
			}
			entries = append(entries, Entry{
				URL:     fmt.Sprintf("%s/journals/%s/entry/%d", s.baseURL, row.JournalID, row.EntryNumber.Int64),
				LastMod: lastMod.Format(dateFormat),
			})
		}
	}
	return entries, nil
}
