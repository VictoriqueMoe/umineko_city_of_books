package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	serviceFixture struct {
		svc   Service
		repo  *repository.MockStoredFileRepository
		local *engine.MockEngine
		s3    *engine.MockEngine
	}
)

func newMockEngine(t *testing.T, id config.StorageBackend, enabled bool) *engine.MockEngine {
	t.Helper()
	candidate := engine.NewMockEngine(t)
	candidate.EXPECT().ID().Return(id).Maybe()
	candidate.EXPECT().Enabled().Return(enabled).Maybe()
	return candidate
}

func newServiceFixture(t *testing.T, active config.StorageBackend, s3Enabled bool) *serviceFixture {
	t.Helper()
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingStorageBackend).Return(string(active)).Maybe()

	local := newMockEngine(t, config.StorageBackendLocal, true)
	s3 := newMockEngine(t, config.StorageBackendS3, s3Enabled)
	repo := repository.NewMockStoredFileRepository(t)
	manager := NewProviderManager(NewFactory(local, s3), settingsSvc)

	return &serviceFixture{svc: NewService(manager, repo), repo: repo, local: local, s3: s3}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file.bin")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func objectsSeq(count int, failure error) iter.Seq2[engine.ObjectInfo, error] {
	return func(yield func(engine.ObjectInfo, error) bool) {
		for i := range count {
			key := fmt.Sprintf("posts/%d.png", i)
			if !yield(engine.ObjectInfo{Key: key, Location: key, Size: int64(i), Backend: config.StorageBackendLocal}, nil) {
				return
			}

			if failure != nil && i == 0 {
				if !yield(engine.ObjectInfo{}, failure) {
					return
				}
			}
		}
	}
}

func TestStat_RoutesToTheRecordedBackendAndLocation(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/old.png"}).Return(&model.StoredFileRow{Key: "posts/old.png", Backend: config.StorageBackendLocal, Location: "posts/old.png"}, nil)
	f.local.EXPECT().Head(mock.Anything, "posts/old.png").Return(engine.ObjectInfo{Size: 3, Backend: config.StorageBackendLocal}, nil)

	// when
	got, err := f.svc.Stat(context.Background(), "posts/old.png")

	// then the file stays readable from local disk even though s3 is now active
	require.NoError(t, err)
	assert.Equal(t, engine.ObjectInfo{Key: "posts/old.png", Location: "posts/old.png", Size: 3, Backend: config.StorageBackendLocal}, got)
}

func TestStat_UsesTheLocationRecordedAtWriteTime(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(&model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "old-bucket/old-prefix/posts/a.png"}, nil)
	f.s3.EXPECT().Head(mock.Anything, "old-bucket/old-prefix/posts/a.png").Return(engine.ObjectInfo{Size: 1, Backend: config.StorageBackendS3}, nil)

	// when
	got, err := f.svc.Stat(context.Background(), "posts/a.png")

	// then
	require.NoError(t, err)
	assert.Equal(t, "old-bucket/old-prefix/posts/a.png", got.Location)
}

func TestStat_Failures(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		row     *model.StoredFileRow
		lookup  bool
		wantErr error
	}{
		{name: "invalid key never reaches the database", key: "../etc/passwd", wantErr: engine.ErrInvalidKey},
		{name: "unregistered key", key: "posts/missing.png", lookup: true, wantErr: engine.ErrNotFound},
		{name: "recorded backend is not configured", key: "posts/a.png", lookup: true, row: &model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "b/posts/a.png"}, wantErr: ErrBackendUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newServiceFixture(t, config.StorageBackendLocal, false)
			if tc.lookup {
				f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: tc.key}).Return(tc.row, nil)
			}

			// when
			_, err := f.svc.Stat(context.Background(), tc.key)

			// then
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestStore_WritesToActiveEngineAndRecordsLocation(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	localPath := writeTempFile(t, "hello")
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("bucket/site/posts/a.png", nil)
	f.repo.EXPECT().Upsert(mock.Anything, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "bucket/site/posts/a.png", Size: 5}).Return(nil)

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", localPath)

	// then
	require.NoError(t, err)
}

func TestStore_RemovesSupersededCopy(t *testing.T) {
	tests := []struct {
		name     string
		previous model.StoredFileRow
	}{
		{name: "copy on another backend", previous: model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png"}},
		{name: "copy at an older location on the same backend", previous: model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "old-bucket/posts/a.png"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newServiceFixture(t, config.StorageBackendS3, true)
			localPath := writeTempFile(t, "hello")
			f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(new(tc.previous), nil)
			f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("bucket/posts/a.png", nil)
			f.repo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
			owner := f.s3
			if tc.previous.Backend == config.StorageBackendLocal {
				owner = f.local
			}
			owner.EXPECT().Delete(mock.Anything, []string{tc.previous.Location}).Return(nil)

			// when
			err := f.svc.Store(context.Background(), "posts/a.png", localPath)

			// then
			require.NoError(t, err)
		})
	}
}

func TestStore_RollbackSurvivesACancelledRequest(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	localPath := writeTempFile(t, "hello")
	ctx, cancel := context.WithCancel(context.Background())
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("bucket/posts/a.png", nil)
	f.repo.EXPECT().Upsert(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, spec.NewStoredFile, ...*sql.Tx) error {
		cancel()
		return context.Canceled
	})
	f.s3.EXPECT().Delete(mock.Anything, []string{"bucket/posts/a.png"}).RunAndReturn(func(rollbackCtx context.Context, _ ...string) error {
		return rollbackCtx.Err()
	})

	// when
	err := f.svc.Store(ctx, "posts/a.png", localPath)

	// then the registry failure is reported but the rollback itself still ran with a live context
	require.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, err.Error(), "roll back")
}

func TestStore_ReportsAFailedRollback(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	localPath := writeTempFile(t, "hello")
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("bucket/posts/a.png", nil)
	f.repo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(errors.New("db down"))
	f.s3.EXPECT().Delete(mock.Anything, []string{"bucket/posts/a.png"}).Return(errors.New("bucket gone"))

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", localPath)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
	assert.Contains(t, err.Error(), "roll back stored object: bucket gone")
}

func TestStore_FailsWhenActiveBackendIsNotConfigured(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, false)

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", writeTempFile(t, "hello"))

	// then nothing was written anywhere
	require.ErrorIs(t, err, ErrBackendUnavailable)
}

func TestStore_PutFailureLeavesRegistryUntouched(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, false)
	localPath := writeTempFile(t, "hello")
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
	f.local.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("", errors.New("disk full"))

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", localPath)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestDelete_GroupsByBackendAndDropsRowsOnlyForSuccessfulDeletes(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	f.repo.EXPECT().ListByKeys(mock.Anything, spec.StoredFileKeys{Keys: []string{"a/1.png", "a/2.png", "a/3.png", "a/gone.png"}}).Return([]model.StoredFileRow{
		{Key: "a/1.png", Backend: config.StorageBackendLocal, Location: "a/1.png"},
		{Key: "a/2.png", Backend: config.StorageBackendS3, Location: "bucket/a/2.png"},
		{Key: "a/3.png", Backend: config.StorageBackendS3, Location: "old/a/3.png"},
	}, nil)
	f.local.EXPECT().Delete(mock.Anything, []string{"a/1.png"}).Return(errors.New("file locked"))
	f.s3.EXPECT().Delete(mock.Anything, []string{"bucket/a/2.png", "old/a/3.png"}).Return(nil)
	f.repo.EXPECT().Delete(mock.Anything, spec.StoredFileKeys{Keys: []string{"a/2.png", "a/3.png"}}).Return(nil)

	// when
	err := f.svc.Delete(context.Background(), "a/1.png", "a/2.png", "a/3.png", "a/gone.png")

	// then the local failure is reported and its row survives for the retry
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file locked")
}

func TestDelete_NoKeysIsNoOp(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, false)

	// when
	err := f.svc.Delete(context.Background())

	// then
	require.NoError(t, err)
}

func TestFetch_ReadsFromTheObjectsLocation(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, true)
	info := engine.ObjectInfo{Key: "posts/clip.mp4", Location: "bucket/posts/clip.mp4", Size: 4, Backend: config.StorageBackendS3}
	f.s3.EXPECT().Get(mock.Anything, "bucket/posts/clip.mp4", (*engine.ByteRange)(nil)).Return(io.NopCloser(strings.NewReader("clip")), nil)
	localPath := filepath.Join(t.TempDir(), "clip.mp4")

	// when
	err := f.svc.Fetch(context.Background(), info, localPath)

	// then
	require.NoError(t, err)
	data, err := os.ReadFile(localPath)
	require.NoError(t, err)
	assert.Equal(t, "clip", string(data))
}

func TestFetch_RemovesPartialFileOnFailure(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, false)
	info := engine.ObjectInfo{Key: "posts/clip.mp4", Location: "posts/clip.mp4", Backend: config.StorageBackendLocal}
	body := io.NopCloser(io.MultiReader(strings.NewReader("par"), iotest.ErrReader(errors.New("connection reset"))))
	f.local.EXPECT().Get(mock.Anything, "posts/clip.mp4", (*engine.ByteRange)(nil)).Return(body, nil)
	localPath := filepath.Join(t.TempDir(), "clip.mp4")

	// when
	err := f.svc.Fetch(context.Background(), info, localPath)

	// then
	require.Error(t, err)
	_, statErr := os.Stat(localPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestBackfillLocal_RegistersEveryLocalFileInBatches(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, false)
	total := backfillBatchSize + 2
	f.local.EXPECT().List(mock.Anything, "").Return(objectsSeq(total, nil))
	var batchSizes []int
	f.repo.EXPECT().Backfill(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s spec.StoredFileBackfill, _ ...*sql.Tx) (int64, error) {
		assert.Equal(t, config.StorageBackendLocal, s.Backend)
		assert.Equal(t, s.Keys, s.Locations)
		assert.Len(t, s.Sizes, len(s.Keys))
		batchSizes = append(batchSizes, len(s.Keys))
		return int64(len(s.Keys)), nil
	})

	// when
	inserted, err := f.svc.BackfillLocal(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, total, inserted)
	assert.Equal(t, []int{backfillBatchSize, 2}, batchSizes)
}

func TestBackfillLocal_KeepsGoingPastUnreadableEntries(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, false)
	f.local.EXPECT().List(mock.Anything, "").Return(objectsSeq(3, errors.New("permission denied")))
	f.repo.EXPECT().Backfill(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s spec.StoredFileBackfill, _ ...*sql.Tx) (int64, error) {
		return int64(len(s.Keys)), nil
	})

	// when
	inserted, err := f.svc.BackfillLocal(context.Background())

	// then every readable file is registered and the unreadable entry is still reported
	assert.Equal(t, 3, inserted)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}
