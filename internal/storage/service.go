package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/storage/engine"
)

const (
	backfillBatchSize = 1000
	rollbackTimeout   = 30 * time.Second
)

type (
	Service interface {
		Stat(ctx context.Context, key string) (engine.ObjectInfo, error)
		Open(ctx context.Context, info engine.ObjectInfo, rng *engine.ByteRange) (io.ReadCloser, error)
		Fetch(ctx context.Context, info engine.ObjectInfo, localPath string) error
		Store(ctx context.Context, key string, localPath string) error
		Delete(ctx context.Context, keys ...string) error
		BackfillLocal(ctx context.Context) (int, error)
	}

	service struct {
		manager *ProviderManager
		repo    repository.StoredFileRepository
	}

	backendObjects struct {
		keys      []string
		locations []string
	}
)

func NewService(manager *ProviderManager, repo repository.StoredFileRepository) Service {
	return &service{manager: manager, repo: repo}
}

func (s *service) Stat(ctx context.Context, key string) (engine.ObjectInfo, error) {
	if err := engine.ValidateKey(key); err != nil {
		return engine.ObjectInfo{}, err
	}

	row, err := s.repo.Get(ctx, spec.StoredFileKey{Key: key})
	if err != nil {
		return engine.ObjectInfo{}, err
	}

	if row == nil {
		return engine.ObjectInfo{}, fmt.Errorf("%w: %s", engine.ErrNotFound, key)
	}

	owner, err := s.manager.EngineFor(row.Backend)
	if err != nil {
		return engine.ObjectInfo{}, err
	}

	info, err := owner.Head(ctx, row.Location)
	if err != nil {
		return engine.ObjectInfo{}, err
	}

	info.Key = key
	info.Location = row.Location

	return info, nil
}

func (s *service) Open(ctx context.Context, info engine.ObjectInfo, rng *engine.ByteRange) (io.ReadCloser, error) {
	owner, err := s.manager.EngineFor(info.Backend)
	if err != nil {
		return nil, err
	}

	return owner.Get(ctx, info.Location, rng)
}

func (s *service) Fetch(ctx context.Context, info engine.ObjectInfo, localPath string) error {
	body, err := s.Open(ctx, info, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", localPath, err)
	}

	if _, err := io.Copy(out, body); err != nil {
		_ = out.Close()
		_ = os.Remove(localPath)

		return fmt.Errorf("fetch %s: %w", info.Key, err)
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(localPath)

		return fmt.Errorf("close %s: %w", localPath, err)
	}

	return nil
}

func (s *service) Store(ctx context.Context, key string, localPath string) error {
	active, err := s.manager.Active(ctx)
	if err != nil {
		return err
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", localPath, err)
	}

	previous, err := s.repo.Get(ctx, spec.StoredFileKey{Key: key})
	if err != nil {
		return err
	}

	location, err := active.PutFile(ctx, key, localPath)
	if err != nil {
		return err
	}

	if err := s.repo.Upsert(ctx, spec.NewStoredFile{Key: key, Backend: active.ID(), Location: location, Size: info.Size()}); err != nil {
		return errors.Join(err, s.rollBack(ctx, active, key, location))
	}

	if previous != nil && (previous.Backend != active.ID() || previous.Location != location) {
		s.deleteSuperseded(ctx, previous.Backend, key, previous.Location)
	}

	return nil
}

func (s *service) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	rows, err := s.repo.ListByKeys(ctx, spec.StoredFileKeys{Keys: keys})
	if err != nil {
		return err
	}

	byBackend := make(map[config.StorageBackend]*backendObjects)
	for _, row := range rows {
		objects, ok := byBackend[row.Backend]
		if !ok {
			objects = new(backendObjects)
			byBackend[row.Backend] = objects
		}

		objects.keys = append(objects.keys, row.Key)
		objects.locations = append(objects.locations, row.Location)
	}

	var errs []error
	for backend, objects := range byBackend {
		owner, err := s.manager.EngineFor(backend)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := owner.Delete(ctx, objects.locations...); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := s.repo.Delete(ctx, spec.StoredFileKeys{Keys: objects.keys}); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *service) BackfillLocal(ctx context.Context) (int, error) {
	local, err := s.manager.EngineFor(config.StorageBackendLocal)
	if err != nil {
		return 0, err
	}

	batch := spec.StoredFileBackfill{Backend: config.StorageBackendLocal}
	inserted := 0
	var errs []error

	flush := func() {
		n, err := s.repo.Backfill(ctx, batch)
		if err != nil {
			errs = append(errs, err)
		}

		inserted += int(n)
		batch.Keys = batch.Keys[:0]
		batch.Locations = batch.Locations[:0]
		batch.Sizes = batch.Sizes[:0]
	}

	for info, err := range local.List(ctx, "") {
		if err != nil {
			errs = append(errs, err)
			continue
		}

		batch.Keys = append(batch.Keys, info.Key)
		batch.Locations = append(batch.Locations, info.Location)
		batch.Sizes = append(batch.Sizes, info.Size)

		if len(batch.Keys) >= backfillBatchSize {
			flush()
		}
	}

	if len(batch.Keys) > 0 {
		flush()
	}

	return inserted, errors.Join(errs...)
}

func (s *service) rollBack(ctx context.Context, active engine.Engine, key string, location string) error {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()

	if err := active.Delete(rollbackCtx, location); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("key", key).Str("location", location).Str("backend", string(active.ID())).Msg("stored object is stranded: its registry write failed and so did the rollback")

		return fmt.Errorf("roll back stored object: %w", err)
	}

	return nil
}

func (s *service) deleteSuperseded(ctx context.Context, backend config.StorageBackend, key string, location string) {
	owner, err := s.manager.EngineFor(backend)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("key", key).Str("location", location).Str("backend", string(backend)).Msg("cannot remove superseded copy of a re-stored file")

		return
	}

	if err := owner.Delete(ctx, location); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("key", key).Str("location", location).Str("backend", string(backend)).Msg("failed to remove superseded copy of a re-stored file")
	}
}
