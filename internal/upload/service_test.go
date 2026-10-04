package upload

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	storedObject struct {
		key  string
		data []byte
	}
)

func newTestService(t *testing.T, processors ...*media.Processor) (*service, *storage.MockService, string) {
	t.Helper()
	settingsSvc := settings.NewMockService(t)
	dir := t.TempDir()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(dir).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()
	storageSvc := storage.NewMockService(t)
	storedFiles := repository.NewMockStoredFileRepository(t)
	svc := NewService(settingsSvc, storageSvc, storedFiles, processors...).(*service)
	return svc, storageSvc, dir
}

func storedFilesOf(svc *service) *repository.MockStoredFileRepository {
	return svc.storedFiles.(*repository.MockStoredFileRepository)
}

func captureStore(t *testing.T, storageSvc *storage.MockService) *[]storedObject {
	t.Helper()
	stored := new([]storedObject)
	storageSvc.EXPECT().Store(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string, localPath string) error {
		data, err := os.ReadFile(localPath)
		require.NoError(t, err)
		*stored = append(*stored, storedObject{key: key, data: data})
		return nil
	})
	return stored
}

func expectNoPreviousUploads(svc *service, subDir string, id uuid.UUID) {
	storedFilesOf(svc).EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{Prefix: subDir + "/" + id.String() + "_"}).Return(nil, nil)
}

func assertStagingEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, stagingDirName))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	require.NoError(t, err)
	assert.Empty(t, entries, "staged uploads must be cleaned up")
}

func TestSaveFile_StoresFileAndReturnsURL(t *testing.T) {
	// given
	svc, storageSvc, dir := newTestService(t)
	stored := captureStore(t, storageSvc)
	content := "hello world"

	// when
	url, err := svc.SaveFile(context.Background(), "avatars", "a.txt", strings.NewReader(content))

	// then
	require.NoError(t, err)
	assert.Equal(t, "/uploads/avatars/a.txt", url)
	require.Len(t, *stored, 1)
	assert.Equal(t, "avatars/a.txt", (*stored)[0].key)
	assert.Equal(t, content, string((*stored)[0].data))
	assertStagingEmpty(t, dir)
}

func TestSaveFile_StagingDirectoryError(t *testing.T) {
	// given
	settingsSvc := settings.NewMockService(t)
	blocker := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(blocker)
	svc := NewService(settingsSvc, storage.NewMockService(t), repository.NewMockStoredFileRepository(t)).(*service)

	// when
	_, err := svc.SaveFile(context.Background(), "sub", "f.txt", strings.NewReader("data"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create staging directory")
}

func TestSaveFile_WriteErrorCleansStaging(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)

	// when
	_, err := svc.SaveFile(context.Background(), "sub", "f.txt", iotest.ErrReader(io.ErrUnexpectedEOF))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write file")
	assertStagingEmpty(t, dir)
}

func TestSaveFile_StoreErrorPropagatesAndCleansStaging(t *testing.T) {
	// given
	svc, storageSvc, dir := newTestService(t)
	storageSvc.EXPECT().Store(mock.Anything, "sub/f.txt", mock.Anything).Return(storage.ErrBackendUnavailable)

	// when
	_, err := svc.SaveFile(context.Background(), "sub", "f.txt", strings.NewReader("data"))

	// then
	require.ErrorIs(t, err, storage.ErrBackendUnavailable)
	assertStagingEmpty(t, dir)
}

var (
	pngMagic      = mustPNGBytes()
	jpegMagic     = mustJPEGBytes()
	gifMagic      = mustGIFBytes()
	webpMagic     = append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WEBPVP8 ")...)
	mp4Magic      = append([]byte{0, 0, 0, 0x20}, []byte("ftypisom\x00\x00\x00\x00isomiso2avc1mp41")...)
	mp4IsomOnly   = append([]byte{0, 0, 0, 0x18}, []byte("ftypisom\x00\x00\x00\x01isomiso4")...)
	webmMagic     = []byte{0x1A, 0x45, 0xDF, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1F}
	webmFullMagic = append([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x82, 0x84}, []byte("webm")...)
	mkvMagic      = append([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x82, 0x88}, []byte("matroska")...)
	aviMagic      = append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("AVI LIST")...)
	pdfMagic      = []byte("%PDF-1.4\n")
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

func TestSaveImage_TooLarge(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, 200, 100, bytes.NewReader(pngMagic))

	// then the controllers can tell an oversized file from a server failure
	require.ErrorIs(t, err, ErrFileTooLarge)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSaveImage_InvalidType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when: PDF bytes should be rejected from image flow
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pdfMagic)), 1024, bytes.NewReader(pdfMagic))

	// then
	require.ErrorIs(t, err, ErrInvalidFileType)
}

func TestSaveImage_RejectsSpoofedContentType(t *testing.T) {
	// given: the caller used to pass "image/png" which we trusted.
	// Now the bytes are what count, and these bytes are PDF.
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pdfMagic)), 1024, bytes.NewReader(pdfMagic))

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
			svc, storageSvc, dir := newTestService(t)
			id := uuid.New()
			expectNoPreviousUploads(svc, "images", id)
			stored := captureStore(t, storageSvc)

			// when
			url, err := svc.SaveImage(context.Background(), "images", id, int64(len(tc.body)), 1024, bytes.NewReader(tc.body))

			// then
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(url, "/uploads/images/"+id.String()+"_"))
			assert.True(t, strings.HasSuffix(url, tc.wantExt))
			require.Len(t, *stored, 1)
			assert.Equal(t, strings.TrimPrefix(url, "/uploads/"), (*stored)[0].key)
			assert.Equal(t, tc.body, (*stored)[0].data, "sniffed stream must still store the full original bytes")
			assertStagingEmpty(t, dir)
		})
	}
}

func TestSaveImage_ReplacesExistingFileWithSameIDPrefix(t *testing.T) {
	// given
	svc, storageSvc, _ := newTestService(t)
	id := uuid.New()
	oldKey := "images/" + id.String() + "_999.png"
	storedFilesOf(svc).EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{Prefix: "images/" + id.String() + "_"}).Return([]model.StoredFileRow{{Key: oldKey}}, nil)
	storageSvc.EXPECT().Delete(mock.Anything, []string{oldKey}).Return(nil)
	stored := captureStore(t, storageSvc)

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pngMagic)), 1024, bytes.NewReader(pngMagic))

	// then
	require.NoError(t, err)
	require.Len(t, *stored, 1)
	assert.NotEqual(t, oldKey, (*stored)[0].key)
}

func TestSaveImage_PrefixListingErrorAbortsUpload(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()
	storedFilesOf(svc).EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{Prefix: "images/" + id.String() + "_"}).Return(nil, errors.New("db down"))

	// when
	_, err := svc.SaveImage(context.Background(), "images", id, int64(len(pngMagic)), 1024, bytes.NewReader(pngMagic))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list stored files")
}

func TestStageMedia_VideoTooLarge(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.StageMedia(context.Background(), media.MediaTypeVideo, "videos", id, 200, 100, bytes.NewReader(mp4Magic))

	// then
	require.ErrorIs(t, err, ErrFileTooLarge)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSaveAttachment_TooLarge(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	_, err := svc.SaveAttachment(context.Background(), "attachments", 200, 100, bytes.NewReader([]byte("x")))

	// then
	require.ErrorIs(t, err, ErrFileTooLarge)
}

func TestSaveAttachment_StoresPDF(t *testing.T) {
	// given
	svc, storageSvc, _ := newTestService(t)
	stored := captureStore(t, storageSvc)

	// when
	url, err := svc.SaveAttachment(context.Background(), "attachments", int64(len(pdfMagic)), 1024, bytes.NewReader(pdfMagic))

	// then
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "/uploads/attachments/"))
	assert.True(t, strings.HasSuffix(url, ".pdf"))
	require.Len(t, *stored, 1)
	assert.Equal(t, pdfMagic, (*stored)[0].data)
}

func TestStageMedia_VideoInvalidType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	id := uuid.New()

	// when: image bytes in the video flow
	_, err := svc.StageMedia(context.Background(), media.MediaTypeVideo, "videos", id, int64(len(pngMagic)), 1024, bytes.NewReader(pngMagic))

	// then
	require.ErrorIs(t, err, ErrInvalidVideoType)
}

func TestStageMedia_UnknownMediaType(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	_, err := svc.StageMedia(context.Background(), "hologram", "posts", uuid.New(), 1, 1024, bytes.NewReader(pngMagic))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown media type")
}

func TestStageMedia_AllAllowedVideoTypes(t *testing.T) {
	cases := []struct {
		name    string
		body    []byte
		wantExt string
	}{
		{"video/mp4", mp4Magic, ".mp4"},
		{"video/mp4 isom-only fallback", mp4IsomOnly, ".mp4"},
		{"video/webm", webmMagic, ".webm"},
		{"video/x-matroska", mkvMagic, ".mkv"},
		{"video/x-msvideo", aviMagic, ".avi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _, dir := newTestService(t)
			id := uuid.New()
			expectNoPreviousUploads(svc, "videos", id)

			// when
			staged, err := svc.StageMedia(context.Background(), media.MediaTypeVideo, "videos", id, int64(len(tc.body)), 1024, bytes.NewReader(tc.body))

			// then the caller owns a staged copy with the full bytes until it discards it
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(staged, tc.wantExt))
			data, err := os.ReadFile(staged)
			require.NoError(t, err)
			assert.Equal(t, tc.body, data)

			svc.Discard(staged)
			assertStagingEmpty(t, dir)
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
		{"mp4 isom-only fallback", mp4IsomOnly, "video/mp4"},
		{"webm", webmMagic, "video/webm"},
		{"webm via fallback doctype", webmFullMagic, "video/webm"},
		{"matroska", mkvMagic, "video/x-matroska"},
		{"pdf", pdfMagic, "application/pdf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := DetectContentType(bytes.NewReader(tc.body))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDetectContentType_WrappedReaderReplaysFullStream(t *testing.T) {
	body := append([]byte{}, pngMagic...)
	body = append(body, []byte("trailing-content-past-512-byte-sniff")...)

	_, wrapped, err := DetectContentType(bytes.NewReader(body))
	require.NoError(t, err)

	got, err := io.ReadAll(wrapped)
	require.NoError(t, err)
	assert.Equal(t, body, got)
}

func TestDetectContentType_StripsCharsetSuffix(t *testing.T) {
	// text/plain sniff returns "text/plain; charset=utf-8": we strip the charset.
	got, _, err := DetectContentType(strings.NewReader("just plain text here"))
	require.NoError(t, err)
	assert.Equal(t, "text/plain", got)
}

func TestSaveImage_AviAliasNormalizedForVideo(t *testing.T) {
	// Go sniffs AVI as "video/avi"; our allowlist key is "video/x-msvideo".
	// The alias map must normalize so the file is accepted.
	svc, _, _ := newTestService(t)
	id := uuid.New()
	expectNoPreviousUploads(svc, "videos", id)

	staged, err := svc.StageMedia(context.Background(), media.MediaTypeVideo, "videos", id, int64(len(aviMagic)), 1024, bytes.NewReader(aviMagic))
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(staged, ".avi"))
	svc.Discard(staged)
}

func TestDelete_IgnoresPathsThatAreNotUploads(t *testing.T) {
	cases := []struct {
		name    string
		urlPath string
	}{
		{"empty path", ""},
		{"bare prefix", "/uploads/"},
		{"external url", "https://media.giphy.com/media/abc/giphy.gif"},
		{"relative path", "avatars/pic.png"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _, _ := newTestService(t)

			// when
			err := svc.delete(tc.urlPath)

			// then
			require.NoError(t, err)
		})
	}
}

func TestDelete_DeletesTheStoredKey(t *testing.T) {
	// given
	svc, storageSvc, _ := newTestService(t)
	storageSvc.EXPECT().Delete(mock.Anything, []string{"images/pic.png"}).Return(nil)

	// when
	svc.Delete("/uploads/images/pic.png")

	// then the strict mock asserts the delete happened
}

func TestDelete_StorageErrorWrapped(t *testing.T) {
	// given
	svc, storageSvc, _ := newTestService(t)
	storageSvc.EXPECT().Delete(mock.Anything, []string{"images/pic.png"}).Return(errors.New("bucket gone"))

	// when
	err := svc.delete("/uploads/images/pic.png")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete file")
	assert.Contains(t, err.Error(), "bucket gone")
}

func TestDeleteByPrefix_DeletesEveryMatch(t *testing.T) {
	// given
	svc, storageSvc, _ := newTestService(t)
	storedFilesOf(svc).EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{Prefix: "images/abc_"}).Return([]model.StoredFileRow{
		{Key: "images/abc_1.png", Backend: config.StorageBackendLocal},
		{Key: "images/abc_2.png", Backend: config.StorageBackendS3},
	}, nil)
	storageSvc.EXPECT().Delete(mock.Anything, []string{"images/abc_1.png"}).Return(nil)
	storageSvc.EXPECT().Delete(mock.Anything, []string{"images/abc_2.png"}).Return(nil)

	// when
	err := svc.DeleteByPrefix(context.Background(), "images", "abc_")

	// then
	require.NoError(t, err)
}

func TestDeleteByPrefix_NoMatchesIsNoOp(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	storedFilesOf(svc).EXPECT().ListByPrefix(mock.Anything, spec.StoredFilePrefix{Prefix: "missing/prefix_"}).Return(nil, nil)

	// when
	err := svc.DeleteByPrefix(context.Background(), "missing", "prefix_")

	// then
	require.NoError(t, err)
}

func TestDiscard_RefusesPathsOutsideStaging(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)
	outside := filepath.Join(t.TempDir(), "keep", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(outside), 0755))
	require.NoError(t, os.WriteFile(outside, []byte("x"), 0644))

	// when
	svc.Discard(outside)

	// then
	_, err := os.Stat(outside)
	assert.NoError(t, err)
}

func TestCleanStaging(t *testing.T) {
	// given
	svc, _, dir := newTestService(t)
	root := filepath.Join(dir, stagingDirName)
	oldDir := filepath.Join(root, "upload-old")
	freshDir := filepath.Join(root, "upload-fresh")
	require.NoError(t, os.MkdirAll(oldDir, 0755))
	require.NoError(t, os.MkdirAll(freshDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "f"), []byte("x"), 0644))
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldDir, old, old))

	// when
	removed, err := svc.CleanStaging(time.Hour)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	_, err = os.Stat(oldDir)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(freshDir)
	assert.NoError(t, err)
}

func TestCleanStaging_NeverRemovesAStagedFileStillInUse(t *testing.T) {
	// given a video staged for a transcode that is still queued after the cutoff
	svc, _, dir := newTestService(t)
	id := uuid.New()
	expectNoPreviousUploads(svc, "videos", id)
	staged, err := svc.StageMedia(context.Background(), media.MediaTypeVideo, "videos", id, int64(len(mp4Magic)), 1024, bytes.NewReader(mp4Magic))
	require.NoError(t, err)
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Dir(staged), old, old))

	// when
	removed, err := svc.CleanStaging(time.Hour)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
	_, statErr := os.Stat(staged)
	assert.NoError(t, statErr)

	// when the job finishes and releases it
	svc.Discard(staged)

	// then
	assertStagingEmpty(t, dir)
}

func TestCleanStaging_MissingRootIsNoOp(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	removed, err := svc.CleanStaging(time.Hour)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
}

func makeWebP(t *testing.T, width, height int) []byte {
	t.Helper()

	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not installed, skipping webp pipeline test")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.png")
	outPath := filepath.Join(dir, "src.webp")

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	f, err := os.Create(srcPath)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())

	out, err := exec.Command("cwebp", "-q", "90", srcPath, "-o", outPath).CombinedOutput()
	require.NoError(t, err, string(out))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	return data
}

func TestSaveImage_WebPAvatarIsResizedNotSkipped(t *testing.T) {
	// given
	body := makeWebP(t, 400, 400)
	svc, storageSvc, dir := newTestService(t, media.NewProcessor(1))
	id := uuid.New()
	expectNoPreviousUploads(svc, "avatars", id)
	stored := captureStore(t, storageSvc)

	// when
	urlPath, err := svc.SaveImage(context.Background(), "avatars", id, int64(len(body)), 10<<20, bytes.NewReader(body))

	// then
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(strings.ToLower(urlPath), ".webp"), "expected a webp, got %q", urlPath)
	require.Len(t, *stored, 1)

	cfg, _, err := image.DecodeConfig(bytes.NewReader((*stored)[0].data))
	require.NoError(t, err)
	assert.Equal(t, media.AvatarMaxWidth, cfg.Width, "webp avatar should be resized to the avatar cap")
	assert.Equal(t, media.AvatarMaxWidth, cfg.Height, "webp avatar should be square cropped")
	assertStagingEmpty(t, dir)
}

func TestSaveImage_AnimatedWebPIsAcceptedNotRejected(t *testing.T) {
	// given
	if _, err := exec.LookPath("img2webp"); err != nil {
		t.Skip("img2webp not installed, skipping animated webp test")
	}

	src := t.TempDir()
	frames := make([]string, 0, 2)
	for i := range 2 {
		img := image.NewRGBA(image.Rect(0, 0, 120, 120))
		for y := range 120 {
			for x := range 120 {
				img.Set(x, y, color.RGBA{R: uint8(i * 120), G: uint8(x), B: uint8(y), A: 255})
			}
		}
		p := filepath.Join(src, "f"+string(rune('0'+i))+".png")
		f, err := os.Create(p)
		require.NoError(t, err)
		require.NoError(t, png.Encode(f, img))
		require.NoError(t, f.Close())
		frames = append(frames, p)
	}

	animPath := filepath.Join(src, "anim.webp")
	args := append([]string{"-loop", "0"}, frames...)
	args = append(args, "-o", animPath)
	out, err := exec.Command("img2webp", args...).CombinedOutput()
	require.NoError(t, err, string(out))
	body, err := os.ReadFile(animPath)
	require.NoError(t, err)

	svc, storageSvc, _ := newTestService(t, media.NewProcessor(1))
	id := uuid.New()
	expectNoPreviousUploads(svc, "avatars", id)
	stored := captureStore(t, storageSvc)

	// when
	_, err = svc.SaveImage(context.Background(), "avatars", id, int64(len(body)), 10<<20, bytes.NewReader(body))

	// then
	require.NoError(t, err, "animated webp upload must not fail")
	require.Len(t, *stored, 1, "animated webp should still be stored")
}
