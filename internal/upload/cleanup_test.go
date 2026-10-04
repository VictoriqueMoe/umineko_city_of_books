package upload

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCleanOrphanedFiles(t *testing.T) {
	old := time.Now().Add(-2 * orphanGracePeriod)
	fresh := time.Now()

	tests := []struct {
		name        string
		key         string
		createdAt   time.Time
		referenced  []string
		wantRemoved int
		wantDelete  bool
	}{
		{
			name:        "removes old unreferenced file",
			key:         "avatars/orphan.webp",
			createdAt:   old,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 1,
			wantDelete:  true,
		},
		{
			name:        "keeps recently stored unreferenced file",
			key:         "avatars/justuploaded.webp",
			createdAt:   fresh,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 0,
		},
		{
			name:        "keeps referenced file even when old",
			key:         "avatars/keep.webp",
			createdAt:   old,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 0,
		},
		{
			name:        "ignores nested keys",
			key:         "avatars/nested/deep.webp",
			createdAt:   old,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			repo := repository.NewMockUploadRepository(t)
			repo.EXPECT().GetAllReferencedFiles().Return(tt.referenced, nil)
			storedFiles := repository.NewMockStoredFileRepository(t)
			storedFiles.EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{}).Return([]model.StoredFileRow{
				{Key: tt.key, Backend: config.StorageBackendS3, CreatedAt: tt.createdAt},
			}, nil)
			storageSvc := storage.NewMockService(t)
			if tt.wantDelete {
				storageSvc.EXPECT().Delete(mock.Anything, []string{tt.key}).Return(nil)
			}

			// when
			removed, err := CleanOrphanedFiles(context.Background(), repo, storedFiles, storageSvc)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.wantRemoved, removed)
		})
	}
}

func TestCleanOrphanedFiles_SkipsWhenNothingReferenced(t *testing.T) {
	// given
	repo := repository.NewMockUploadRepository(t)
	repo.EXPECT().GetAllReferencedFiles().Return(nil, nil)

	// when
	removed, err := CleanOrphanedFiles(context.Background(), repo, repository.NewMockStoredFileRepository(t), storage.NewMockService(t))

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
}

func TestCleanOrphanedFiles_DeleteFailureIsNotCounted(t *testing.T) {
	// given
	repo := repository.NewMockUploadRepository(t)
	repo.EXPECT().GetAllReferencedFiles().Return([]string{"/uploads/avatars/keep.webp"}, nil)
	storedFiles := repository.NewMockStoredFileRepository(t)
	storedFiles.EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{}).Return([]model.StoredFileRow{
		{Key: "avatars/orphan.webp", Backend: config.StorageBackendLocal, CreatedAt: time.Now().Add(-2 * orphanGracePeriod)},
	}, nil)
	storageSvc := storage.NewMockService(t)
	storageSvc.EXPECT().Delete(mock.Anything, []string{"avatars/orphan.webp"}).Return(errors.New("locked"))

	// when
	removed, err := CleanOrphanedFiles(context.Background(), repo, storedFiles, storageSvc)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
}

func TestCleanOrphanedFiles_ErrorsAreReported(t *testing.T) {
	t.Run("referenced files query fails", func(t *testing.T) {
		// given
		repo := repository.NewMockUploadRepository(t)
		repo.EXPECT().GetAllReferencedFiles().Return(nil, errors.New("db down"))

		// when
		_, err := CleanOrphanedFiles(context.Background(), repo, repository.NewMockStoredFileRepository(t), storage.NewMockService(t))

		// then
		require.Error(t, err)
	})

	t.Run("stored files listing fails", func(t *testing.T) {
		// given
		repo := repository.NewMockUploadRepository(t)
		repo.EXPECT().GetAllReferencedFiles().Return([]string{"/uploads/avatars/keep.webp"}, nil)
		storedFiles := repository.NewMockStoredFileRepository(t)
		storedFiles.EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{}).Return(nil, errors.New("db down"))

		// when
		_, err := CleanOrphanedFiles(context.Background(), repo, storedFiles, storage.NewMockService(t))

		// then
		require.Error(t, err)
	})
}
