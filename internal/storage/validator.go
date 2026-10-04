package storage

import (
	"context"
	"fmt"
	"slices"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engines"
)

func SettingsValidator(repo repository.StoredFileRepository, probe func(ctx context.Context, cfg engines.S3Config) error) settings.BatchValidator {
	return func(ctx context.Context, merged map[config.SiteSettingKey]string, changed []config.SiteSettingKey) error {
		s3Changed := slices.ContainsFunc(changed, func(key config.SiteSettingKey) bool {
			return slices.Contains(engines.S3SettingKeys, key)
		})
		if !s3Changed && !slices.Contains(changed, config.SettingStorageBackend.Key) {
			return nil
		}

		cfg := engines.S3ConfigFrom(merged)

		if s3Changed {
			stored, err := repo.CountByBackend(ctx, spec.StoredFileBackend{Backend: config.StorageBackendS3})
			if err != nil {
				return err
			}

			if stored > 0 && !cfg.Complete() {
				return fmt.Errorf("%d uploads are stored in S3, so the S3 connection cannot be cleared while they exist", stored)
			}

			if stored > 0 && slices.Contains(changed, config.SettingS3Endpoint.Key) {
				return fmt.Errorf("%d uploads are stored on the current S3 endpoint and would stop loading if it changed", stored)
			}
		}

		if !cfg.Complete() {
			return nil
		}

		if !s3Changed && config.StorageBackend(merged[config.SettingStorageBackend.Key]) != config.StorageBackendS3 {
			return nil
		}

		if err := probe(ctx, cfg); err != nil {
			return fmt.Errorf("cannot use these S3 settings: %w", err)
		}

		return nil
	}
}
