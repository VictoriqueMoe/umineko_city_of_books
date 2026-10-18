package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	serviceFixture struct {
		svc       Service
		repo      *repository.MockStoredFileRepository
		local     *engine.MockEngine
		s3        *engine.MockEngine
		uploadDir string
	}
)

var (
	fallbackStoredAt = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
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
	uploadDir := t.TempDir()
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingStorageBackend).Return(string(active)).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(uploadDir).Maybe()

	local := newMockEngine(t, config.StorageBackendLocal, true)
	s3 := newMockEngine(t, config.StorageBackendS3, s3Enabled)
	repo := repository.NewMockStoredFileRepository(t)
	manager := NewProviderManager(NewFactory(local, s3), settingsSvc)

	return &serviceFixture{svc: NewService(manager, repo, settingsSvc), repo: repo, local: local, s3: s3, uploadDir: uploadDir}
}

func writeFallbackFile(t *testing.T, f *serviceFixture, key string, content string) string {
	t.Helper()
	path := filepath.Join(f.uploadDir, filepath.FromSlash(key))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
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

func s3OperationError(err error) error {
	return fmt.Errorf("put posts/a.png: %w", &smithy.OperationError{ServiceID: "S3", OperationName: "PutObject", Err: err})
}

func s3StatusError(status int, code string) error {
	return s3OperationError(&awshttp.ResponseError{
		ResponseError: &smithyhttp.ResponseError{
			Response: &smithyhttp.Response{Response: &http.Response{StatusCode: status}},
			Err:      &smithy.GenericAPIError{Code: code},
		},
		RequestID: "req-1",
	})
}

func s3NoResponseError(err error) error {
	return s3OperationError(&awshttp.ResponseError{
		ResponseError: &smithyhttp.ResponseError{
			Response: &smithyhttp.Response{Response: &http.Response{}},
			Err:      err,
		},
	})
}

func localRow(key string) model.StoredFileRow {
	return model.StoredFileRow{Key: key, Backend: config.StorageBackendLocal, Location: key, Size: 5, CreatedAt: fallbackStoredAt, Fallback: true}
}

func flushMove(key string) spec.StoredFileMove {
	return spec.StoredFileMove{Key: key, FromBackend: config.StorageBackendLocal, FromLocation: key, Backend: config.StorageBackendS3, Location: "bucket/" + key, Size: 5}
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

func TestStore_AFailedFallbackRegistrationStillNamesTheS3Outage(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	localPath := writeTempFile(t, "hello")
	outage := s3StatusError(http.StatusServiceUnavailable, "ServiceUnavailable")
	dbDown := errors.New("db down")
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil).Once()
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("", outage)
	f.local.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("posts/a.png", nil)
	f.repo.EXPECT().Upsert(mock.Anything, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 5, Fallback: true}).Return(dbDown)
	f.local.EXPECT().Delete(mock.Anything, []string{"posts/a.png"}).Return(nil)

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", localPath)

	// then the caller sees both the outage that caused the fallback and the registry failure
	require.ErrorIs(t, err, outage)
	require.ErrorIs(t, err, dbDown)
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

func TestStore_FallsBackToLocalOnlyForProviderSideS3Failures(t *testing.T) {
	tests := []struct {
		name         string
		putErr       error
		wantFallback bool
	}{
		{name: "503 service unavailable", putErr: s3StatusError(http.StatusServiceUnavailable, "ServiceUnavailable"), wantFallback: true},
		{name: "500 internal error", putErr: s3StatusError(http.StatusInternalServerError, "InternalError"), wantFallback: true},
		{name: "retry quota drained by the outage", putErr: errors.Join(s3StatusError(http.StatusServiceUnavailable, "SlowDown"), ratelimit.QuotaExceededError{Available: 0, Requested: 5}), wantFallback: true},
		{name: "connection refused", putErr: s3NoResponseError(&smithyhttp.RequestSendError{Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}}), wantFallback: true},
		{name: "response read timeout", putErr: s3NoResponseError(&awshttp.ResponseTimeoutError{TimeoutDur: time.Second}), wantFallback: true},
		{name: "403 access denied", putErr: s3StatusError(http.StatusForbidden, "AccessDenied")},
		{name: "403 bad credentials", putErr: s3StatusError(http.StatusForbidden, "InvalidAccessKeyId")},
		{name: "404 missing bucket", putErr: s3StatusError(http.StatusNotFound, "NoSuchBucket")},
		{name: "unclassified failure", putErr: errors.New("checksum mismatch")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newServiceFixture(t, config.StorageBackendS3, true)
			localPath := writeTempFile(t, "hello")
			f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
			f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("", tc.putErr)
			if tc.wantFallback {
				f.local.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("posts/a.png", nil)
				f.repo.EXPECT().Upsert(mock.Anything, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 5, Fallback: true}).Return(nil)
			}

			// when
			err := f.svc.Store(context.Background(), "posts/a.png", localPath)

			// then a provider outage lands on local disk and anything else fails loudly
			if tc.wantFallback {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.putErr)
			}
		})
	}
}

func TestStore_CallerCancellationNeverFallsBack(t *testing.T) {
	tests := []struct {
		name   string
		putErr error
	}{
		{name: "sdk reports the cancellation", putErr: s3OperationError(&smithy.CanceledError{Err: context.Canceled})},
		{name: "transport error surfaced after the caller gave up", putErr: s3NoResponseError(&smithyhttp.RequestSendError{Err: errors.New("use of closed network connection")})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newServiceFixture(t, config.StorageBackendS3, true)
			localPath := writeTempFile(t, "hello")
			ctx, cancel := context.WithCancel(context.Background())
			f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
			f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).RunAndReturn(func(context.Context, string, string) (string, error) {
				cancel()
				return "", tc.putErr
			})

			// when
			err := f.svc.Store(ctx, "posts/a.png", localPath)

			// then nothing was written to local disk
			require.ErrorIs(t, err, tc.putErr)
		})
	}
}

func TestStore_LocalActiveNeverFallsBack(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, true)
	localPath := writeTempFile(t, "hello")
	putErr := s3StatusError(http.StatusServiceUnavailable, "ServiceUnavailable")
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
	f.local.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("", putErr).Once()

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", localPath)

	// then the error is returned as it is today
	require.ErrorIs(t, err, putErr)
}

func TestStore_ReportsBothErrorsWhenTheFallbackFails(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	localPath := writeTempFile(t, "hello")
	putErr := s3StatusError(http.StatusServiceUnavailable, "ServiceUnavailable")
	f.repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: "posts/a.png"}).Return(nil, nil)
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("", putErr)
	f.local.EXPECT().PutFile(mock.Anything, "posts/a.png", localPath).Return("", errors.New("disk full"))

	// when
	err := f.svc.Store(context.Background(), "posts/a.png", localPath)

	// then
	require.ErrorIs(t, err, putErr)
	assert.Contains(t, err.Error(), "fall back to local storage: disk full")
}

func TestFlushFallback_PutsEveryFallbackFileFromItsLocalPath(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	keys := []string{"avatars/b.webp", "posts/a.png", "posts/nested/c.png"}
	rows := make([]model.StoredFileRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, localRow(key))
		path := writeFallbackFile(t, f, key, "hello")
		f.s3.EXPECT().PutFile(mock.Anything, key, path).Return("bucket/"+key, nil).Once()
		f.repo.EXPECT().Move(mock.Anything, flushMove(key)).Return(true, nil).Once()
		f.local.EXPECT().Delete(mock.Anything, []string{key}).Return(nil).Once()
	}
	f.repo.EXPECT().ListFallback(mock.Anything).Return(rows, nil).Once()

	// when
	moved, err := f.svc.FlushFallback(context.Background())

	// then every file went to s3 straight from the upload directory and its local copy was removed
	require.NoError(t, err)
	assert.Equal(t, len(keys), moved)
}

func TestFlushFallback_ReturnsTheListingError(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	dbDown := errors.New("db down")
	f.repo.EXPECT().ListFallback(mock.Anything).Return(nil, dbDown).Once()

	// when
	moved, err := f.svc.FlushFallback(context.Background())

	// then the next run of the job retries it
	require.ErrorIs(t, err, dbDown)
	assert.Zero(t, moved)
}

func TestFlushFallback_StopsOnAProviderSideError(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	outage := s3StatusError(http.StatusServiceUnavailable, "ServiceUnavailable")
	pathA := writeFallbackFile(t, f, "posts/a.png", "hello")
	writeFallbackFile(t, f, "posts/b.png", "world")
	f.repo.EXPECT().ListFallback(mock.Anything).Return([]model.StoredFileRow{localRow("posts/a.png"), localRow("posts/b.png")}, nil).Once()
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", pathA).Return("", outage).Once()

	// when
	moved, err := f.svc.FlushFallback(context.Background())

	// then the second row is never attempted and nothing falls back to local again
	require.ErrorIs(t, err, ErrFlushInterrupted)
	require.ErrorIs(t, err, outage)
	assert.Zero(t, moved)
}

func TestFlushFallback_KeepsGoingPastAFileItCannotMove(t *testing.T) {
	rejected := s3StatusError(http.StatusForbidden, "AccessDenied")

	tests := []struct {
		name    string
		onDisk  bool
		putErr  error
		wantErr error
	}{
		{name: "the file is missing from local disk", wantErr: fs.ErrNotExist},
		{name: "s3 rejects the file", onDisk: true, putErr: rejected, wantErr: rejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newServiceFixture(t, config.StorageBackendS3, true)
			if tc.onDisk {
				pathA := writeFallbackFile(t, f, "posts/a.png", "hello")
				f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", pathA).Return("", tc.putErr).Once()
			}
			pathB := writeFallbackFile(t, f, "posts/b.png", "world")
			f.repo.EXPECT().ListFallback(mock.Anything).Return([]model.StoredFileRow{localRow("posts/a.png"), localRow("posts/b.png")}, nil).Once()
			f.s3.EXPECT().PutFile(mock.Anything, "posts/b.png", pathB).Return("bucket/posts/b.png", nil).Once()
			f.repo.EXPECT().Move(mock.Anything, flushMove("posts/b.png")).Return(true, nil).Once()
			f.local.EXPECT().Delete(mock.Anything, []string{"posts/b.png"}).Return(nil).Once()

			// when
			moved, err := f.svc.FlushFallback(context.Background())

			// then the failed file keeps its fallback mark for the next run and the rest still move
			require.ErrorIs(t, err, tc.wantErr)
			assert.NotErrorIs(t, err, ErrFlushInterrupted)
			assert.Equal(t, 1, moved)
		})
	}
}

func TestFlushFallback_RollsBackTheS3CopyWhenTheRowChangedMeanwhile(t *testing.T) {
	// given a fallback file deleted while the flush was uploading it
	f := newServiceFixture(t, config.StorageBackendS3, true)
	path := writeFallbackFile(t, f, "posts/a.png", "hello")
	f.repo.EXPECT().ListFallback(mock.Anything).Return([]model.StoredFileRow{localRow("posts/a.png")}, nil).Once()
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", path).Return("bucket/posts/a.png", nil).Once()
	f.repo.EXPECT().Move(mock.Anything, flushMove("posts/a.png")).Return(false, nil).Once()
	f.s3.EXPECT().Delete(mock.Anything, []string{"bucket/posts/a.png"}).Return(nil).Once()

	// when
	moved, err := f.svc.FlushFallback(context.Background())

	// then the s3 copy is removed and the local copy is left alone
	require.NoError(t, err)
	assert.Zero(t, moved)
}

func TestFlushFallback_AFailedMoveLeavesBothCopiesForTheNextRun(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendS3, true)
	moveErr := errors.New("db down")
	path := writeFallbackFile(t, f, "posts/a.png", "hello")
	f.repo.EXPECT().ListFallback(mock.Anything).Return([]model.StoredFileRow{localRow("posts/a.png")}, nil).Once()
	f.s3.EXPECT().PutFile(mock.Anything, "posts/a.png", path).Return("bucket/posts/a.png", nil).Once()
	f.repo.EXPECT().Move(mock.Anything, flushMove("posts/a.png")).Return(false, moveErr).Once()

	// when
	moved, err := f.svc.FlushFallback(context.Background())

	// then the strict engine mocks prove nothing was deleted
	require.ErrorIs(t, err, moveErr)
	assert.Zero(t, moved)
}

func TestFlushFallback_DoesNothingWhenLocalIsActive(t *testing.T) {
	// given
	f := newServiceFixture(t, config.StorageBackendLocal, true)

	// when
	moved, err := f.svc.FlushFallback(context.Background())

	// then no legacy local file is pushed into s3
	require.NoError(t, err)
	assert.Zero(t, moved)
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
	f.repo.EXPECT().DeleteAt(mock.Anything, spec.StoredFileLocations{Backend: config.StorageBackendS3, Keys: []string{"a/2.png", "a/3.png"}, Locations: []string{"bucket/a/2.png", "old/a/3.png"}}).Return([]string{"a/2.png", "a/3.png"}, nil)

	// when
	err := f.svc.Delete(context.Background(), "a/1.png", "a/2.png", "a/3.png", "a/gone.png")

	// then the local failure is reported and its row survives for the retry
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file locked")
	assert.NotErrorIs(t, err, ErrDeleteRaced)
}

func TestDelete_ARowMovedByAFlushMeanwhileIsDeletedWhereItNowLives(t *testing.T) {
	// given a row the flush moves to s3 between the read and the row delete
	f := newServiceFixture(t, config.StorageBackendS3, true)
	onLocal := model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png"}
	onS3 := model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "bucket/posts/a.png"}
	f.repo.EXPECT().ListByKeys(mock.Anything, spec.StoredFileKeys{Keys: []string{"posts/a.png"}}).Return([]model.StoredFileRow{onLocal}, nil).Once()
	f.local.EXPECT().Delete(mock.Anything, []string{"posts/a.png"}).Return(nil).Once()
	f.repo.EXPECT().DeleteAt(mock.Anything, spec.StoredFileLocations{Backend: config.StorageBackendLocal, Keys: []string{"posts/a.png"}, Locations: []string{"posts/a.png"}}).Return(nil, nil).Once()
	f.repo.EXPECT().ListByKeys(mock.Anything, spec.StoredFileKeys{Keys: []string{"posts/a.png"}}).Return([]model.StoredFileRow{onS3}, nil).Once()
	f.s3.EXPECT().Delete(mock.Anything, []string{"bucket/posts/a.png"}).Return(nil).Once()
	f.repo.EXPECT().DeleteAt(mock.Anything, spec.StoredFileLocations{Backend: config.StorageBackendS3, Keys: []string{"posts/a.png"}, Locations: []string{"bucket/posts/a.png"}}).Return([]string{"posts/a.png"}, nil).Once()

	// when
	err := f.svc.Delete(context.Background(), "posts/a.png")

	// then the s3 copy the flush made is gone along with its row
	require.NoError(t, err)
}

func TestDelete_ARowThatNeverSettlesIsReported(t *testing.T) {
	// given a row whose conditional delete never matches
	f := newServiceFixture(t, config.StorageBackendS3, true)
	row := model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png"}
	f.repo.EXPECT().ListByKeys(mock.Anything, mock.Anything).Return([]model.StoredFileRow{row}, nil).Times(deleteAttempts)
	f.local.EXPECT().Delete(mock.Anything, []string{"posts/a.png"}).Return(nil).Times(deleteAttempts)
	f.repo.EXPECT().DeleteAt(mock.Anything, mock.Anything).Return(nil, nil).Times(deleteAttempts)

	// when
	err := f.svc.Delete(context.Background(), "posts/a.png")

	// then
	require.ErrorIs(t, err, ErrDeleteRaced)
	assert.Contains(t, err.Error(), "posts/a.png")
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
