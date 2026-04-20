package upload

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
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

var (
	pngMagic  = mustPNGBytes()
	jpegMagic = mustJPEGBytes()
	gifMagic  = mustGIFBytes()
	webpMagic = append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WEBPVP8 ")...)
	mp4Magic  = append([]byte{0, 0, 0, 0x20}, []byte("ftypisom\x00\x00\x00\x00isomiso2avc1mp41")...)
	webmMagic = []byte{0x1A, 0x45, 0xDF, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1F}
	aviMagic  = append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("AVI LIST")...)
	pdfMagic  = []byte("%PDF-1.4\n")
)

func tinyImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	img.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 0, A: 255})
	return img
}

func mustPNGBytes() []byte {
	var buf bytes.Buffer
	err := png.Encode(&buf, tinyImage())
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func mustJPEGBytes() []byte {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, tinyImage(), &jpeg.Options{Quality: 90})
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func mustGIFBytes() []byte {
	var buf bytes.Buffer
	err := gif.Encode(&buf, tinyImage(), nil)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

func TestSaveImage_TooLarge(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, 200, 100, bytesReader(pngMagic))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSaveImage_InvalidType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when — PDF bytes should be rejected from image flow
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pdfMagic)), 1024, bytesReader(pdfMagic))

	// then
	require.ErrorIs(t, err, ErrInvalidFileType)
}

func TestSaveImage_RejectsSpoofedContentType(t *testing.T) {
	// given — the caller used to pass "image/png" which we trusted.
	// Now the bytes are what count, and these bytes are PDF.
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pdfMagic)), 1024, bytesReader(pdfMagic))

	// then
	require.ErrorIs(t, err, ErrInvalidFileType)
}

func TestSaveImage_AllAllowedTypes(t *testing.T) {
	cases := []struct {
		name    string
		body    []byte
		wantExt string
	}{
		{"image/png", pngMagic, ".png"},
		{"image/jpeg", jpegMagic, ".jpg"},
		{"image/gif", gifMagic, ".gif"},
		{"image/webp", webpMagic, ".webp"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _, dir := newTestService(t)
			id := uuid.New()

			// when
			url, err := svc.SaveImage(context.Background(), "images", id, int64(len(tc.body)), 1024, bytesReader(tc.body))

			// then
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(url, "/uploads/images/"))
			assert.True(t, strings.HasSuffix(url, tc.wantExt))
			entries, err := os.ReadDir(filepath.Join(dir, "images"))
			require.NoError(t, err)
			assert.Len(t, entries, 1)
			data, err := os.ReadFile(filepath.Join(dir, "images", entries[0].Name()))
			require.NoError(t, err)
			assert.Equal(t, tc.body, data, "sniffed stream must still write full original bytes to disk")
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
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pngMagic)), 1024, bytesReader(pngMagic))

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
	_, err := svc.SaveVideo(context.Background(), "videos", id, 200, 100, bytesReader(mp4Magic))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSaveVideo_InvalidType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when — image bytes in the video flow
	_, err := svc.SaveVideo(context.Background(), "videos", id, int64(len(pngMagic)), 1024, bytesReader(pngMagic))

	// then
	require.ErrorIs(t, err, ErrInvalidVideoType)
}

func TestSaveVideo_AllAllowedTypes(t *testing.T) {
	cases := []struct {
		name    string
		body    []byte
		wantExt string
	}{
		{"video/mp4", mp4Magic, ".mp4"},
		{"video/webm", webmMagic, ".webm"},
		{"video/x-msvideo", aviMagic, ".avi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _, _ := newTestService(t)
			id := uuid.New()

			// when
			url, err := svc.SaveVideo(context.Background(), "videos", id, int64(len(tc.body)), 1024, bytesReader(tc.body))

			// then
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(url, tc.wantExt))
		})
	}
}

func TestDetectContentType_SniffsKnownFormats(t *testing.T) {
	cases := []struct {
		name string
		body []byte
		want string
	}{
		{"png", pngMagic, "image/png"},
		{"jpeg", jpegMagic, "image/jpeg"},
		{"gif", gifMagic, "image/gif"},
		{"webp", webpMagic, "image/webp"},
		{"mp4", mp4Magic, "video/mp4"},
		{"webm", webmMagic, "video/webm"},
		{"pdf", pdfMagic, "application/pdf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := DetectContentType(bytesReader(tc.body))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDetectContentType_WrappedReaderReplaysFullStream(t *testing.T) {
	body := append([]byte{}, pngMagic...)
	body = append(body, []byte("trailing-content-past-512-byte-sniff")...)

	_, wrapped, err := DetectContentType(bytesReader(body))
	require.NoError(t, err)

	got, err := io.ReadAll(wrapped)
	require.NoError(t, err)
	assert.Equal(t, body, got)
}

func TestDetectContentType_StripsCharsetSuffix(t *testing.T) {
	// text/plain sniff returns "text/plain; charset=utf-8" — we strip the charset.
	got, _, err := DetectContentType(strings.NewReader("just plain text here"))
	require.NoError(t, err)
	assert.Equal(t, "text/plain", got)
}

func TestSaveImage_AviAliasNormalizedForVideo(t *testing.T) {
	// Go sniffs AVI as "video/avi"; our allowlist key is "video/x-msvideo".
	// The alias map must normalize so the file is accepted.
	svc, _, _ := newTestService(t)
	id := uuid.New()

	url, err := svc.SaveVideo(context.Background(), "videos", id, int64(len(aviMagic)), 1024, bytesReader(aviMagic))
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(url, ".avi"))
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
