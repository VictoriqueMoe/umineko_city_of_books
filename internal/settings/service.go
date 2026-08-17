package settings

import (
	"context"
	"fmt"
	"maps"
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

	Validator func(ctx context.Context, value string) error

	Service interface {
		Get(ctx context.Context, def *config.SiteSettingDef) string
		GetInt(ctx context.Context, def *config.SiteSettingDef) int
		GetBool(ctx context.Context, def *config.SiteSettingDef) bool
		GetAll(ctx context.Context) map[config.SiteSettingKey]string
		Set(ctx context.Context, setting *config.SiteSettingDef, value string, updatedBy uuid.UUID) error
		SetMultiple(ctx context.Context, values map[config.SiteSettingKey]string, updatedBy uuid.UUID) error
		Subscribe(listener Listener)
		SubscribeBatch(listener BatchListener)
		RegisterValidator(setting *config.SiteSettingDef, validate Validator)
		Refresh(ctx context.Context) error
	}

	service struct {
		repo           repository.SettingsRepository
		listeners      []Listener
		batchListeners []BatchListener
		listenerMu     sync.RWMutex
		validators     map[config.SiteSettingKey]Validator
		validatorMu    sync.RWMutex
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

func (s *service) SubscribeBatch(listener BatchListener) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	s.batchListeners = append(s.batchListeners, listener)
}

func (s *service) RegisterValidator(setting *config.SiteSettingDef, validate Validator) {
	s.validatorMu.Lock()
	defer s.validatorMu.Unlock()

	if s.validators == nil {
		s.validators = make(map[config.SiteSettingKey]Validator)
	}

	s.validators[setting.Key] = validate
}

func (s *service) validatorFor(key config.SiteSettingKey) Validator {
	s.validatorMu.RLock()
	defer s.validatorMu.RUnlock()

	return s.validators[key]
}

func (s *service) validateChanged(ctx context.Context, changed map[config.SiteSettingKey]string) error {
	for key, value := range changed {
		def, ok := config.SettingByKey(key)
		if !ok {
			return fmt.Errorf("unknown setting: %s", key)
		}

		if err := config.ValidateSettingValue(def, value); err != nil {
			return err
		}

		validate := s.validatorFor(key)
		if validate == nil {
			continue
		}

		if err := validate(ctx, value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}

	return nil
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
	for _, l := range s.batchListeners {
		l.OnSettingsBatchChanged(keys)
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

	var stale []string
	for k := range existing {
		if _, ok := config.SettingByKey(config.SiteSettingKey(k)); !ok {
			stale = append(stale, k)
		}
	}

	if len(missing) > 0 || len(stale) > 0 {
		if err := s.repo.Reconcile(ctx, repository.SettingsReconcile{Missing: missing, Stale: stale, UpdatedBy: uuid.Nil}); err != nil {
			return err
		}
	}

	if len(missing) > 0 {
		logger.Log.Info().Int("count", len(missing)).Msg("seeded missing settings with defaults")
	}

	for _, k := range stale {
		logger.Log.Info().Str("key", k).Msg("removed stale setting")
	}

	logger.Log.Debug().Msg("settings reconciled")
	return nil
}

func (s *service) Get(ctx context.Context, def *config.SiteSettingDef) string {
	v, err := s.repo.Get(ctx, string(def.Key))
	if err != nil {
		return def.Default
	}

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
	changed := merged[setting.Key] != value
	merged[setting.Key] = value

	if err := config.ValidateSettings(merged); err != nil {
		return err
	}

	if changed {
		if err := s.validateChanged(ctx, map[config.SiteSettingKey]string{setting.Key: value}); err != nil {
			return err
		}
	}

	if err := s.repo.Set(ctx, string(setting.Key), value, updatedBy); err != nil {
		return err
	}

	s.notify(setting.Key, value)
	logger.Log.Info().Str("key", string(setting.Key)).Str("updated_by", updatedBy.String()).Msg("setting updated")
	return nil
}

func (s *service) SetMultiple(ctx context.Context, values map[config.SiteSettingKey]string, updatedBy uuid.UUID) error {
	merged := s.GetAll(ctx)

	raw := make(map[string]string, len(values))
	keys := make([]config.SiteSettingKey, 0, len(values))
	changed := make(map[config.SiteSettingKey]string)

	for k, v := range values {
		if _, ok := config.SettingByKey(k); !ok {
			return fmt.Errorf("unknown setting: %s", k)
		}

		raw[string(k)] = v
		keys = append(keys, k)

		if merged[k] != v {
			changed[k] = v
		}
	}

	maps.Copy(merged, values)

	if err := config.ValidateSettings(merged); err != nil {
		return err
	}

	if err := s.validateChanged(ctx, changed); err != nil {
		return err
	}

	if err := s.repo.SetMultiple(ctx, raw, updatedBy); err != nil {
		return err
	}

	for k, v := range values {
		s.notify(k, v)
	}

	s.notifyBatch(keys)
	logger.Log.Info().Int("count", len(values)).Str("updated_by", updatedBy.String()).Msg("settings updated")
	return nil
}
