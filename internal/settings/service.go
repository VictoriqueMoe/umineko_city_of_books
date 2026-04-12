package settings

import (
	"context"
	"fmt"
	"strconv"
	"sync"

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
		cache      sync.Map
		listeners  []Listener
		listenerMu sync.RWMutex
	}
)

func NewService(repo repository.SettingsRepository) Service {
	return &service{repo: repo}
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
		for k, v := range missing {
			existing[k] = v
		}
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
		s.cache.Store(config.SiteSettingKey(k), v)
	}

	logger.Log.Debug().Msg("settings cache loaded")
	return nil
}

func (s *service) Get(ctx context.Context, def *config.SiteSettingDef) string {
	if v, ok := s.cache.Load(def.Key); ok {
		return v.(string)
	}
	return def.Default
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
	s.cache.Range(func(key, value any) bool {
		result[key.(config.SiteSettingKey)] = value.(string)
		return true
	})
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

	s.cache.Store(setting.Key, value)
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
	for k, v := range values {
		merged[k] = v
	}
	if err := config.ValidateSettings(merged); err != nil {
		return err
	}

	if err := s.repo.SetMultiple(ctx, raw, updatedBy); err != nil {
		return err
	}

	var keys []config.SiteSettingKey
	for k, v := range values {
		s.cache.Store(k, v)
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
