package testutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	settingssvc "umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage"
	"umineko_city_of_books/internal/storage/engines"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	LocalStorage struct {
		t       *testing.T
		Service storage.Service
		Repo    *repository.MockStoredFileRepository
		Dir     string
	}
)

func NewLocalStorage(t *testing.T) *LocalStorage {
	t.Helper()

	dir := t.TempDir()
	settingsSvc := settingssvc.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(dir).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingStorageBackend).Return(string(config.StorageBackendLocal)).Maybe()

	repo := repository.NewMockStoredFileRepository(t)
	manager := storage.NewProviderManager(storage.NewFactory(engines.NewLocal(settingsSvc)), settingsSvc)

	return &LocalStorage{t: t, Service: storage.NewService(manager, repo), Repo: repo, Dir: dir}
}

func (l *LocalStorage) Put(key string, content string, modTime time.Time) {
	l.t.Helper()

	fullPath := filepath.Join(l.Dir, filepath.FromSlash(key))
	require.NoError(l.t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(l.t, os.WriteFile(fullPath, []byte(content), 0644))
	require.NoError(l.t, os.Chtimes(fullPath, modTime, modTime))

	row := &model.StoredFileRow{Key: key, Backend: config.StorageBackendLocal, Location: key, Size: int64(len(content)), CreatedAt: modTime}
	l.Repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: key}).Return(row, nil).Maybe()
}

func (l *LocalStorage) ExpectOnUnconfiguredS3(key string) {
	row := &model.StoredFileRow{Key: key, Backend: config.StorageBackendS3, Location: "bucket/" + key, Size: 1}
	l.Repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: key}).Return(row, nil).Maybe()
}

func (l *LocalStorage) ExpectMissing(key string) {
	l.Repo.EXPECT().Get(mock.Anything, spec.StoredFileKey{Key: key}).Return(nil, nil).Maybe()
}
