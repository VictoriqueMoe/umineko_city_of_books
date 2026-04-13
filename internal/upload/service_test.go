package upload

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*service, *settings.MockService, string) {
	t.Helper()
	settingsSvc := settings.NewMockService(t)
	dir := t.TempDir()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(dir).Maybe()
	svc := NewService(settingsSvc).(*service)
	return svc, settingsSvc, dir
}

func TestGetUploadDir_ReturnsSettingValue(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)

	// when
	got := svc.GetUploadDir()

	// then
	assert.Equal(t, dir, got)
}

func TestFullDiskPath_StripsUploadsPrefix(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)

	// when
	got := svc.FullDiskPath("/uploads/avatars/pic.png")

	// then
	assert.Equal(t, filepath.Join(dir, "avatars", "pic.png"), got)
}

func TestFullDiskPath_NoPrefixLeftUntouched(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)

	// when
	got := svc.FullDiskPath("custom/path.png")

	// then
	assert.Equal(t, filepath.Join(dir, "custom/path.png"), got)
}

func TestSaveFile_WritesFileAndReturnsURL(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	content := "hello world"

	// when
	url, err := svc.SaveFile("avatars", "a.txt", strings.NewReader(content))

	// then
	require.NoError(t, err)
	assert.Equal(t, "/uploads/avatars/a.txt", url)
	data, err := os.ReadFile(filepath.Join(dir, "avatars", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestSaveFile_CreateDirectoryError(t *testing.T) {
	// given
	settingsSvc := settings.NewMockService(t)
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocked")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(blocker)
	svc := NewService(settingsSvc).(*service)

	// when
	_, err := svc.SaveFile("sub", "f.txt", strings.NewReader("data"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create directory")
}

func TestSaveFile_CreateFileError(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub", "name"), 0755))

	// when
	_, err := svc.SaveFile("sub", "name", strings.NewReader("data"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create file")
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestSaveFile_WriteError(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	_, err := svc.SaveFile("sub", "f.txt", errReader{})

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write file")
}

func TestSaveImage_TooLarge(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, "image/png", 200, 100, strings.NewReader("x"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSaveImage_InvalidType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, "application/pdf", 10, 1024, strings.NewReader("x"))

	// then
	require.ErrorIs(t, err, ErrInvalidFileType)
}

func TestSaveImage_AllAllowedTypes(t *testing.T) {
	cases := []struct {
		contentType string
		wantExt     string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
	}

	for _, tc := range cases {
		t.Run(tc.contentType, func(t *testing.T) {
			// given
			svc, _, dir := newTestService(t)
			id := uuid.New()

			// when
			url, err := svc.SaveImage(context.Background(), "images", id, tc.contentType, 10, 1024, strings.NewReader("data"))

			// then
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(url, "/uploads/images/"))
			assert.True(t, strings.HasSuffix(url, tc.wantExt))
			entries, err := os.ReadDir(filepath.Join(dir, "images"))
			require.NoError(t, err)
			assert.Len(t, entries, 1)
		})
	}
}

func TestSaveImage_ReplacesExistingFileWithSameIDPrefix(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	id := uuid.New()
	imagesDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(imagesDir, 0755))
	oldFile := filepath.Join(imagesDir, id.String()+"_999.png")
	require.NoError(t, os.WriteFile(oldFile, []byte("old"), 0644))

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, "image/png", 10, 1024, strings.NewReader("new"))

	// then
	require.NoError(t, err)
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))
	entries, err := os.ReadDir(imagesDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestSaveVideo_TooLarge(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveVideo(context.Background(), "videos", id, "video/mp4", 200, 100, strings.NewReader("x"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSaveVideo_InvalidType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveVideo(context.Background(), "videos", id, "image/png", 10, 1024, strings.NewReader("x"))

	// then
	require.ErrorIs(t, err, ErrInvalidVideoType)
}

func TestSaveVideo_AllAllowedTypes(t *testing.T) {
	cases := []struct {
		contentType string
		wantExt     string
	}{
		{"video/mp4", ".mp4"},
		{"video/webm", ".webm"},
		{"video/quicktime", ".mov"},
		{"video/x-msvideo", ".avi"},
		{"video/x-matroska", ".mkv"},
		{"video/matroska", ".mkv"},
		{"application/x-matroska", ".mkv"},
	}

	for _, tc := range cases {
		t.Run(tc.contentType, func(t *testing.T) {
			// given
			svc, _, _ := newTestService(t)
			id := uuid.New()

			// when
			url, err := svc.SaveVideo(context.Background(), "videos", id, tc.contentType, 10, 1024, strings.NewReader("data"))

			// then
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(url, tc.wantExt))
		})
	}
}

func TestDelete_EmptyPathNoOp(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	err := svc.Delete("")

	// then
	require.NoError(t, err)
}

func TestDelete_MissingFileNoError(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	err := svc.Delete("/uploads/images/nonexistent.png")

	// then
	require.NoError(t, err)
}

func TestDelete_RemovesFile(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	subDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	target := filepath.Join(subDir, "pic.png")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0644))

	// when
	err := svc.Delete("/uploads/images/pic.png")

	// then
	require.NoError(t, err)
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr))
}

func TestDelete_RemoveErrorWrapped(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	subDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "keep.png"), []byte("x"), 0644))

	// when
	err := svc.Delete("/uploads/images")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete file")
}

func TestDeleteByPrefix_MissingDirectoryNoError(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	err := svc.DeleteByPrefix("missing", "prefix_")

	// then
	require.NoError(t, err)
}

func TestDeleteByPrefix_RemovesMatchingFilesOnly(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	subDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	match1 := filepath.Join(subDir, "abc_1.png")
	match2 := filepath.Join(subDir, "abc_2.png")
	noMatch := filepath.Join(subDir, "xyz_1.png")
	require.NoError(t, os.WriteFile(match1, []byte("x"), 0644))
	require.NoError(t, os.WriteFile(match2, []byte("x"), 0644))
	require.NoError(t, os.WriteFile(noMatch, []byte("x"), 0644))

	// when
	err := svc.DeleteByPrefix("images", "abc_")

	// then
	require.NoError(t, err)
	_, err = os.Stat(match1)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(match2)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(noMatch)
	assert.NoError(t, err)
}

func TestDeleteByPrefix_ReadDirError(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	notADir := filepath.Join(dir, "images")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0644))

	// when
	err := svc.DeleteByPrefix("images", "abc_")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read directory")
}
