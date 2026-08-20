package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
)

const (
	fetchTimeout = 15 * time.Second
	maxBytes     = 2 << 20
)

type (
	Source struct {
		Name string
		URL  string
	}

	document struct {
		Prefixes []struct {
			IPv4Prefix string `json:"ipv4Prefix"`
			IPv6Prefix string `json:"ipv6Prefix"`
		} `json:"prefixes"`
	}

	Service struct {
		settingsSvc settings.Service
		cache       *cache.Manager
		client      *http.Client
	}
)

func New(settingsSvc settings.Service, cacheMgr *cache.Manager) *Service {
	return &Service{
		settingsSvc: settingsSvc,
		cache:       cacheMgr,
		client:      &http.Client{Timeout: fetchTimeout},
	}
}

func (f *Service) byFeed(ctx context.Context) map[string][]netip.Prefix {
	stored, err := cache.Get[map[string][]netip.Prefix](ctx, f.cache, cache.CrawlerRanges.Key())
	if err != nil {
		return map[string][]netip.Prefix{}
	}

	return stored
}

func (f *Service) Ranges(ctx context.Context) []netip.Prefix {
	stored := f.byFeed(ctx)

	var merged []netip.Prefix
	for _, ranges := range stored {
		merged = append(merged, ranges...)
	}

	return merged
}

func (f *Service) OnSettingChanged(key config.SiteSettingKey, _ string) {
	if key != config.SettingCrawlerFeeds.Key {
		return
	}

	go f.Refresh(context.Background())
}

func (f *Service) Refresh(ctx context.Context) (int, error) {
	sources := parseSources(f.settingsSvc.Get(ctx, config.SettingCrawlerFeeds))
	if len(sources) == 0 {
		_ = f.cache.Del(ctx, cache.CrawlerRanges.Key())

		return 0, nil
	}

	previous := f.byFeed(ctx)
	current := make(map[string][]netip.Prefix, len(sources))
	total := 0
	failures := 0

	for _, source := range sources {
		ranges, err := f.fetch(ctx, source.URL)
		if err != nil {
			failures++
			logger.Log.Error().Err(err).Str("feed", source.Name).Msg("crawler range feed refresh failed, keeping the ranges it gave us last time")

			ranges = previous[source.Name]
		}

		if len(ranges) == 0 {
			continue
		}

		current[source.Name] = ranges
		total += len(ranges)
	}

	if failures == len(sources) && len(current) == 0 {
		return 0, fmt.Errorf("every crawler range feed failed")
	}

	if err := cache.Set(ctx, f.cache, cache.CrawlerRanges.Key(), current, cache.CrawlerRanges.TTL); err != nil {
		return 0, fmt.Errorf("store crawler ranges: %w", err)
	}

	return total, nil
}

func (f *Service) fetch(ctx context.Context, url string) ([]netip.Prefix, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch: unexpected status %d", resp.StatusCode)
	}

	var doc document
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBytes)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	prefixes := prefixesFrom(doc)
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("no usable ipv4Prefix or ipv6Prefix entries")
	}

	return prefixes, nil
}

func Validator(svc *Service) settings.Validator {
	return func(ctx context.Context, value string) error {
		if strings.TrimSpace(value) == "" {
			return nil
		}

		parsed := parseSources(value)
		if len(parsed) == 0 {
			return fmt.Errorf("each entry must read name=https://example.com/ranges.json")
		}

		for _, source := range parsed {
			if _, err := svc.fetch(ctx, source.URL); err != nil {
				return fmt.Errorf("%s: %w", source.Name, err)
			}
		}

		return nil
	}
}

func parseSources(raw string) []Source {
	var sources []Source

	for entry := range strings.SplitSeq(raw, "\n") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		name, url, found := strings.Cut(entry, "=")
		if !found {
			continue
		}

		name = strings.TrimSpace(name)
		url = strings.TrimSpace(url)
		if name == "" || !strings.HasPrefix(url, "https://") {
			continue
		}

		sources = append(sources, Source{Name: name, URL: url})
	}

	return sources
}

func prefixesFrom(doc document) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(doc.Prefixes))

	for _, entry := range doc.Prefixes {
		raw := entry.IPv4Prefix
		if raw == "" {
			raw = entry.IPv6Prefix
		}
		if raw == "" {
			continue
		}

		if prefix, err := netip.ParsePrefix(strings.TrimSpace(raw)); err == nil {
			prefixes = append(prefixes, prefix)
		}
	}

	return prefixes
}
