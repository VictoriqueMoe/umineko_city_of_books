package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"
	"umineko_city_of_books/internal/storage/engines"

	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const (
	backfillBatchSize = 1000
	rollbackTimeout   = 30 * time.Second
	deleteAttempts    = 3
)

type (
	Service interface {
		Stat(ctx context.Context, key string) (engine.ObjectInfo, error)
		Open(ctx context.Context, info engine.ObjectInfo, rng *engine.ByteRange) (io.ReadCloser, error)
		Fetch(ctx context.Context, info engine.ObjectInfo, localPath string) error
		Store(ctx context.Context, key string, localPath string) error
		Delete(ctx context.Context, keys ...string) error
		BackfillLocal(ctx context.Context) (int, error)
		FlushFallback(ctx context.Context) (int, error)
	}

	service struct {
		manager     *ProviderManager
		repo        repository.StoredFileRepository
		settingsSvc settings.Service
	}

	backendObjects struct {
		keys      []string
		locations []string
	}

	timeoutError interface {
		error
		Timeout() bool
	}
)

func NewService(manager *ProviderManager, repo repository.StoredFileRepository, settingsSvc settings.Service) Service {
	return &service{manager: manager, repo: repo, settingsSvc: settingsSvc}
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

	out, err := os.Create(filepath.Clean(localPath))
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

	target := active
	location, putErr := active.PutFile(ctx, key, localPath)
	if putErr != nil {
		target, location, err = s.fallBack(ctx, active, key, localPath, putErr)
		if err != nil {
			return err
		}
	}

	stored := spec.NewStoredFile{Key: key, Backend: target.ID(), Location: location, Size: info.Size(), Fallback: putErr != nil}
	if err := s.repo.Upsert(ctx, stored); err != nil {
		return errors.Join(putErr, err, s.rollBack(ctx, target, key, location))
	}

	if previous != nil && (previous.Backend != target.ID() || previous.Location != location) {
		s.deleteSuperseded(ctx, previous.Backend, key, previous.Location)
	}

	if putErr != nil {
		logger.Ctx(ctx).Warn().Err(putErr).Str("key", key).Str("backend", string(active.ID())).Str("fallback_backend", string(target.ID())).Msg("active storage backend is unavailable, stored the file on local disk until it recovers")
	}

	return nil
}

func (s *service) Delete(ctx context.Context, keys ...string) error {
	pending := keys
	var errs []error

	for range deleteAttempts {
		if len(pending) == 0 {
			break
		}

		raced, err := s.deletePass(ctx, pending)
		if err != nil {
			errs = append(errs, err)
		}

		pending = raced
	}

	if len(pending) > 0 {
		errs = append(errs, fmt.Errorf("%w: %v", ErrDeleteRaced, pending))
	}

	return errors.Join(errs...)
}

func (s *service) deletePass(ctx context.Context, keys []string) ([]string, error) {
	rows, err := s.repo.ListByKeys(ctx, spec.StoredFileKeys{Keys: keys})
	if err != nil {
		return nil, err
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

	var raced []string
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

		deleted, err := s.repo.DeleteAt(ctx, spec.StoredFileLocations{Backend: backend, Keys: objects.keys, Locations: objects.locations})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		raced = append(raced, notDeleted(objects.keys, deleted)...)
	}

	return raced, errors.Join(errs...)
}

func notDeleted(keys []string, deleted []string) []string {
	gone := make(map[string]struct{}, len(deleted))
	for _, key := range deleted {
		gone[key] = struct{}{}
	}

	var remaining []string
	for _, key := range keys {
		if _, ok := gone[key]; !ok {
			remaining = append(remaining, key)
		}
	}

	return remaining
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

func (s *service) fallBack(ctx context.Context, active engine.Engine, key string, localPath string, cause error) (engine.Engine, string, error) {
	if active.ID() != config.StorageBackendS3 || !isProviderOutage(ctx, cause) {
		return nil, "", cause
	}

	local, err := s.manager.EngineFor(config.StorageBackendLocal)
	if err != nil {
		return nil, "", errors.Join(cause, err)
	}

	location, err := local.PutFile(ctx, key, localPath)
	if err != nil {
		return nil, "", errors.Join(cause, fmt.Errorf("fall back to local storage: %w", err))
	}

	return local, location, nil
}

func (s *service) FlushFallback(ctx context.Context) (int, error) {
	active, err := s.manager.Active(ctx)
	if err != nil {
		return 0, err
	}

	if active.ID() != config.StorageBackendS3 {
		return 0, nil
	}

	root, err := engines.Root(s.settingsSvc.Get(ctx, config.SettingUploadDir))
	if err != nil {
		return 0, fmt.Errorf("resolve the upload directory: %w", err)
	}

	rows, err := s.repo.ListFallback(ctx)
	if err != nil {
		return 0, err
	}

	moved := 0
	var errs []error
	for _, row := range rows {
		ok, err := s.flushFile(ctx, active, root, row)
		if errors.Is(err, ErrFlushInterrupted) {
			return moved, errors.Join(append(errs, err)...)
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}

		if ok {
			moved++
		}
	}

	return moved, errors.Join(errs...)
}

func (s *service) flushFile(ctx context.Context, active engine.Engine, root string, row model.StoredFileRow) (bool, error) {
	localPath, err := engines.ResolvePath(root, row.Location)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", localPath, err)
	}

	location, err := active.PutFile(ctx, row.Key, localPath)
	if err != nil && isProviderOutage(ctx, err) {
		return false, fmt.Errorf("%w: %w", ErrFlushInterrupted, err)
	}

	if err != nil {
		return false, err
	}

	moved, err := s.repo.Move(ctx, spec.StoredFileMove{
		Key:          row.Key,
		FromBackend:  row.Backend,
		FromLocation: row.Location,
		Backend:      active.ID(),
		Location:     location,
		Size:         info.Size(),
	})
	if err != nil {
		return false, err
	}

	if !moved {
		return false, s.rollBack(ctx, active, row.Key, location)
	}

	s.deleteSuperseded(ctx, row.Backend, row.Key, row.Location)

	return true, nil
}

func (s *service) rollBack(ctx context.Context, active engine.Engine, key string, location string) error {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()

	if err := active.Delete(rollbackCtx, location); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("key", key).Str("location", location).Str("backend", string(active.ID())).Msg("stored object is stranded: no registry row points at it and deleting it failed")

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

func isProviderOutage(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) {
		return false
	}

	if _, ok := errors.AsType[*smithyhttp.RequestSendError](err); ok {
		return true
	}

	if respErr, ok := errors.AsType[*smithyhttp.ResponseError](err); ok && respErr.HTTPStatusCode() != 0 {
		return respErr.HTTPStatusCode() >= http.StatusInternalServerError
	}

	timeout, ok := errors.AsType[timeoutError](err)

	return ok && timeout.Timeout()
}
