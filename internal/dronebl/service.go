package dronebl

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	"golang.org/x/sync/singleflight"
)

const (
	lookupTimeout = 2 * time.Second
	failureTTL    = 5 * time.Minute
)

type (
	Resolver interface {
		LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
	}

	Verdict struct {
		Listed  bool  `json:"listed"`
		Classes []int `json:"classes,omitempty"`
	}

	CrawlerRegistry interface {
		Ranges(ctx context.Context) []netip.Prefix
	}

	Checker struct {
		settingsSvc     settings.Service
		cache           *cache.Manager
		resolver        Resolver
		crawlerRegistry CrawlerRegistry
		group           singleflight.Group
	}
)

func New(settingsSvc settings.Service, cacheMgr *cache.Manager, resolver Resolver, crawlerRegistry CrawlerRegistry) *Checker {
	return &Checker{
		settingsSvc:     settingsSvc,
		cache:           cacheMgr,
		resolver:        resolver,
		crawlerRegistry: crawlerRegistry,
	}
}

func (c *Checker) Enabled(ctx context.Context) bool {
	return c.settingsSvc.GetBool(ctx, config.SettingDroneBLEnabled)
}

func (c *Checker) Allowlisted(ctx context.Context, ip string) bool {
	if allowlisted(parseAllowlist(c.settingsSvc.Get(ctx, config.SettingDroneBLAllowlist)), ip) {
		return true
	}

	if c.crawlerRegistry == nil {
		return false
	}

	return allowlisted(c.crawlerRegistry.Ranges(ctx), ip)
}

func (c *Checker) Blocked(ctx context.Context, ip string) (Verdict, bool) {
	verdict := c.Check(ctx, ip)
	if !verdict.Listed {
		return verdict, false
	}

	ignored := parseClassFilter(c.settingsSvc.Get(ctx, config.SettingDroneBLIgnoredClasses))

	return verdict, slices.ContainsFunc(verdict.Classes, func(class int) bool {
		return !ignored[class]
	})
}

func (c *Checker) Check(ctx context.Context, ip string) Verdict {
	name, ok := queryName(ip)
	if !ok {
		return Verdict{}
	}

	key := cache.DroneBL.Key(ip)
	if cached, err := cache.Get[Verdict](ctx, c.cache, key); err == nil {
		return cached
	}

	result, _, _ := c.group.Do(key, func() (any, error) {
		return c.lookup(context.WithoutCancel(ctx), key, name, ip), nil
	})

	verdict, ok := result.(Verdict)
	if !ok {
		return Verdict{}
	}

	return verdict
}

func (c *Checker) lookup(ctx context.Context, key, name, ip string) Verdict {
	lookupCtx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()

	answers, err := c.resolver.LookupNetIP(lookupCtx, "ip4", name)
	if err != nil && !isNotFound(err) {
		logger.Log.Warn().Err(err).Str("ip", ip).Msg("dronebl lookup failed, treating the address as clean")
		_ = cache.Set(ctx, c.cache, key, Verdict{}, failureTTL)

		return Verdict{}
	}

	classes := classesFrom(answers)
	verdict := Verdict{Listed: len(classes) > 0, Classes: classes}
	_ = cache.Set(ctx, c.cache, key, verdict, cache.DroneBL.TTL)

	return verdict
}

func isNotFound(err error) bool {
	var dnsErr *net.DNSError

	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}
