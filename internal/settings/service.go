package settings

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"sync"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	Listener interface {
		OnSettingChanged(key config.SiteSettingKey, value string)
	}

	BatchListener interface {
		OnSettingsBatchChanged(keys []config.SiteSettingKey)
	}

	Service interface {
		Get(ctx context.Context, def *config.SiteSettingDef) string
		GetInt(ctx context.Context, def *config.SiteSettingDef) int
		GetBool(ctx context.Context, def *config.SiteSettingDef) bool
		GetAll(ctx context.Context) map[config.SiteSettingKey]string
		Set(ctx context.Context, setting *config.SiteSettingDef, value string, updatedBy uuid.UUID) error
		SetMultiple(ctx context.Context, values map[config.SiteSettingKey]string, updatedBy uuid.UUID) error
		Subscribe(listener Listener)
		Refresh(ctx context.Context) error
	}

	service struct {
		repo       repository.SettingsRepository
		cache      *cache.Manager
		listeners  []Listener
		listenerMu sync.RWMutex
	}
)

func NewService(repo repository.SettingsRepository, cacheMgr *cache.Manager) Service {
	return &service{repo: repo, cache: cacheMgr}
}

func (s *service) Subscribe(listener Listener) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *service) notify(key config.SiteSettingKey, value string) {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	for _, l := range s.listeners {
		l.OnSettingChanged(key, value)
	}
}

func (s *service) notifyBatch(keys []config.SiteSettingKey) {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	for _, l := range s.listeners {
		if bl, ok := l.(BatchListener); ok {
			bl.OnSettingsBatchChanged(keys)
		}
	}
}

func (s *service) Refresh(ctx context.Context) error {
	existing, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	missing := make(map[string]string)
	for _, def := range config.AllSiteSettings {
		if _, ok := existing[string(def.Key)]; !ok {
			missing[string(def.Key)] = def.Default
		}
	}

	if len(missing) > 0 {
		if err := s.repo.SetMultiple(ctx, missing, uuid.Nil); err != nil {
			return err
		}
		maps.Copy(existing, missing)

		logger.Log.Info().Int("count", len(missing)).Msg("seeded missing settings with defaults")
	}

	valid := validKeys()
	for k, v := range existing {
		if !valid[config.SiteSettingKey(k)] {
			if err := s.repo.Delete(ctx, k); err != nil {
				logger.Log.Error().Err(err).Str("key", k).Msg("failed to delete stale setting")
			} else {
				logger.Log.Info().Str("key", k).Msg("removed stale setting")
			}
			continue
		}
		_ = cache.Set(ctx, s.cache, cache.Setting.Key(k), v, cache.Setting.TTL)
	}

	logger.Log.Debug().Msg("settings cache loaded")
	return nil
}

func (s *service) Get(ctx context.Context, def *config.SiteSettingDef) string {
	key := cache.Setting.Key(string(def.Key))

	if v, err := cache.Get[string](ctx, s.cache, key); err == nil {
		return v
	}

	v, err := s.repo.Get(ctx, string(def.Key))
	if err != nil {
		return def.Default
	}

	_ = cache.Set(ctx, s.cache, key, v, cache.Setting.TTL)
	return v
}

func (s *service) GetInt(ctx context.Context, def *config.SiteSettingDef) int {
	v, err := strconv.Atoi(s.Get(ctx, def))
	if err != nil {
		return 0
	}
	return v
}

func (s *service) GetBool(ctx context.Context, def *config.SiteSettingDef) bool {
	return s.Get(ctx, def) == "true"
}

func (s *service) GetAll(ctx context.Context) map[config.SiteSettingKey]string {
	result := make(map[config.SiteSettingKey]string)
	for _, def := range config.AllSiteSettings {
		result[def.Key] = def.Default
	}

	stored, err := s.repo.GetAll(ctx)
	if err == nil {
		for k, v := range stored {
			result[config.SiteSettingKey(k)] = v
		}
	}

	return result
}

func (s *service) Set(ctx context.Context, setting *config.SiteSettingDef, value string, updatedBy uuid.UUID) error {
	merged := s.GetAll(ctx)
	merged[setting.Key] = value
	if err := config.ValidateSettings(merged); err != nil {
		return err
	}

	if err := s.repo.Set(ctx, string(setting.Key), value, updatedBy); err != nil {
		return err
	}

	_ = cache.Set(ctx, s.cache, cache.Setting.Key(string(setting.Key)), value, cache.Setting.TTL)
	s.notify(setting.Key, value)
	logger.Log.Info().Str("key", string(setting.Key)).Str("updated_by", updatedBy.String()).Msg("setting updated")
	return nil
}

func (s *service) SetMultiple(ctx context.Context, values map[config.SiteSettingKey]string, updatedBy uuid.UUID) error {
	valid := validKeys()

	raw := make(map[string]string, len(values))
	for k, v := range values {
		if !valid[k] {
			return fmt.Errorf("unknown setting: %s", k)
		}
		raw[string(k)] = v
	}

	merged := s.GetAll(ctx)
	maps.Copy(merged, values)

	if err := config.ValidateSettings(merged); err != nil {
		return err
	}

	if err := s.repo.SetMultiple(ctx, raw, updatedBy); err != nil {
		return err
	}

	var keys []config.SiteSettingKey
	for k, v := range values {
		_ = cache.Set(ctx, s.cache, cache.Setting.Key(string(k)), v, cache.Setting.TTL)
		s.notify(k, v)
		keys = append(keys, k)
	}
	s.notifyBatch(keys)
	logger.Log.Info().Int("count", len(values)).Str("updated_by", updatedBy.String()).Msg("settings updated")
	return nil
}

func validKeys() map[config.SiteSettingKey]bool {
	m := make(map[config.SiteSettingKey]bool, len(config.AllSiteSettings))
	for _, def := range config.AllSiteSettings {
		m[def.Key] = true
	}
	return m
}
