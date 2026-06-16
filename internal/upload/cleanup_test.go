package upload

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeUploadFile(t *testing.T, uploadDir, subDir, name string, modTime time.Time) string {
	t.Helper()
	dir := filepath.Join(uploadDir, subDir)
	require.NoError(t, os.MkdirAll(dir, 0755))

	full := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(full, []byte("data"), 0644))
	require.NoError(t, os.Chtimes(full, modTime, modTime))

	return full
}

func TestCleanOrphanedFiles(t *testing.T) {
	old := time.Now().Add(-2 * orphanGracePeriod)
	fresh := time.Now()

	tests := []struct {
		name        string
		subDir      string
		fileName    string
		modTime     time.Time
		referenced  []string
		wantRemoved int
		wantExists  bool
	}{
		{
			name:        "removes old unreferenced file",
			subDir:      "avatars",
			fileName:    "orphan.webp",
			modTime:     old,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 1,
			wantExists:  false,
		},
		{
			name:        "keeps recently written unreferenced file",
			subDir:      "avatars",
			fileName:    "justuploaded.webp",
			modTime:     fresh,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 0,
			wantExists:  true,
		},
		{
			name:        "keeps referenced file even when old",
			subDir:      "avatars",
			fileName:    "keep.webp",
			modTime:     old,
			referenced:  []string{"/uploads/avatars/keep.webp"},
			wantRemoved: 0,
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uploadDir := t.TempDir()
			full := writeUploadFile(t, uploadDir, tt.subDir, tt.fileName, tt.modTime)

			repo := repository.NewMockUploadRepository(t)
			repo.EXPECT().GetAllReferencedFiles().Return(tt.referenced, nil)

			removed := CleanOrphanedFiles(repo, uploadDir)

			assert.Equal(t, tt.wantRemoved, removed)
			_, statErr := os.Stat(full)
			if tt.wantExists {
				assert.NoError(t, statErr)
			} else {
				assert.True(t, os.IsNotExist(statErr))
			}
		})
	}
}

func TestCleanOrphanedFiles_SkipsWhenNoReferences(t *testing.T) {
	uploadDir := t.TempDir()
	full := writeUploadFile(t, uploadDir, "avatars", "orphan.webp", time.Now().Add(-2*orphanGracePeriod))

	repo := repository.NewMockUploadRepository(t)
	repo.EXPECT().GetAllReferencedFiles().Return(nil, nil)

	removed := CleanOrphanedFiles(repo, uploadDir)

	assert.Equal(t, 0, removed)
	_, statErr := os.Stat(full)
	assert.NoError(t, statErr)
}
