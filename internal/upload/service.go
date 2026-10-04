package upload

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage"

	"github.com/google/uuid"
)

const (
	stagingDirName = ".staging"
)

var (
	deleteRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second, 30 * time.Second}

	AllowedImageTypes = map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}

	AllowedVideoTypes = map[string]string{
		"video/mp4":        ".mp4",
		"video/webm":       ".webm",
		"video/x-msvideo":  ".avi",
		"video/x-matroska": ".mkv",
	}

	AllowedAudioTypes = map[string]string{
		"audio/mpeg": ".mp3",
		"audio/mp4":  ".m4a",
		"audio/ogg":  ".ogg",
		"audio/wav":  ".wav",
		"audio/flac": ".flac",
	}

	AllowedAttachmentTypes = map[string]string{
		"application/pdf": ".pdf",
		"text/plain":      ".txt",
		"application/zip": ".docx",
	}

	sniffAliases = map[string]string{
		"video/avi":       "video/x-msvideo",
		"video/matroska":  "video/x-matroska",
		"application/ogg": "audio/ogg",
		"audio/wave":      "audio/wav",
		"audio/x-wav":     "audio/wav",
		"audio/x-flac":    "audio/flac",
		"audio/x-m4a":     "audio/mp4",
	}

	audioMP4Brands = map[string]bool{
		"M4A ": true, "M4B ": true,
	}

	mp4FallbackBrands = map[string]bool{
		"isom": true, "iso2": true, "iso4": true, "iso5": true, "iso6": true,
		"mp41": true, "mp42": true, "mp71": true, "avc1": true, "dash": true,
		"msdh": true, "msix": true, "M4V ": true, "M4A ": true, "qt  ": true,
	}
)

type (
	Service interface {
		SaveFile(ctx context.Context, subDir string, filename string, reader io.Reader) (string, error)
		SaveImage(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		StageMedia(ctx context.Context, mediaType string, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		SaveAudio(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		SaveAttachment(ctx context.Context, subDir string, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		Store(ctx context.Context, subDir string, localPath string) (string, error)
		Discard(localPath string)
		Delete(urlPaths ...string)
		DeleteByPrefix(ctx context.Context, subDir string, prefix string) error
		CleanStaging(olderThan time.Duration) (int, error)
	}

	service struct {
		settingsSvc settings.Service
		storageSvc  storage.Service
		storedFiles repository.StoredFileRepository
		mediaProc   *media.Processor
		liveStaging sync.Map
	}
)

func NewService(settingsSvc settings.Service, storageSvc storage.Service, storedFiles repository.StoredFileRepository, processors ...*media.Processor) Service {
	var mediaProc *media.Processor
	if len(processors) > 0 {
		mediaProc = processors[0]
	}

	return &service{settingsSvc: settingsSvc, storageSvc: storageSvc, storedFiles: storedFiles, mediaProc: mediaProc}
}

func (s *service) SaveFile(ctx context.Context, subDir string, filename string, reader io.Reader) (string, error) {
	staged, err := s.stage(filename, reader)
	if err != nil {
		return "", err
	}
	defer s.Discard(staged)

	return s.Store(ctx, subDir, staged)
}

func (s *service) Store(ctx context.Context, subDir string, localPath string) (string, error) {
	key := subDir + "/" + filepath.Base(localPath)
	if err := s.storageSvc.Store(ctx, key, localPath); err != nil {
		return "", fmt.Errorf("store %s: %w", key, err)
	}

	return urlPrefix + key, nil
}

func (s *service) Discard(localPath string) {
	dir := filepath.Dir(localPath)
	if filepath.Base(filepath.Dir(dir)) != stagingDirName {
		logger.Log.Error().Str("path", localPath).Msg("refusing to discard a path outside the upload staging area")

		return
	}

	s.liveStaging.Delete(dir)

	if err := os.RemoveAll(dir); err != nil {
		logger.Log.Warn().Err(err).Str("dir", dir).Msg("failed to remove staged upload, the staging sweep will retry")
	}
}

func (s *service) StageMedia(
	ctx context.Context,
	mediaType string,
	subDir string,
	id uuid.UUID,
	fileSize int64,
	maxSize int64,
	reader io.Reader,
) (string, error) {
	var allowedTypes map[string]string
	var typeErr error
	switch mediaType {
	case media.MediaTypeImage:
		allowedTypes, typeErr = AllowedImageTypes, ErrInvalidFileType
	case media.MediaTypeVideo:
		allowedTypes, typeErr = AllowedVideoTypes, ErrInvalidVideoType
	case media.MediaTypeAudio:
		allowedTypes, typeErr = AllowedAudioTypes, ErrInvalidAudioType
	default:
		return "", fmt.Errorf("unknown media type %q", mediaType)
	}

	if fileSize > maxSize {
		return "", fmt.Errorf("%w: file size %dMB exceeds maximum %dMB", ErrFileTooLarge, fileSize/(1024*1024), maxSize/(1024*1024))
	}

	sniffed, wrapped, err := DetectContentType(reader)
	if err != nil {
		return "", err
	}
	sniffed = NormaliseSniffedType(sniffed)

	ext, ok := allowedTypes[sniffed]
	if !ok {
		return "", typeErr
	}

	prefix := fmt.Sprintf("%s_", id.String())
	if err := s.DeleteByPrefix(ctx, subDir, prefix); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s_%d%s", id.String(), time.Now().UnixMilli(), ext)

	return s.stage(filename, wrapped)
}

func (s *service) SaveImage(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	staged, err := s.StageMedia(ctx, media.MediaTypeImage, subDir, id, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}
	defer s.Discard(staged)

	maxPixels := s.settingsSvc.GetInt(ctx, config.SettingMaxImagePixels)
	if err := media.CheckImageFileBounds(staged, maxPixels); err != nil {
		return "", err
	}

	if s.mediaProc == nil {
		return s.Store(ctx, subDir, staged)
	}

	encoded, err := s.encodeImage(ctx, subDir, staged)
	if err != nil {
		return "", err
	}

	return s.Store(ctx, subDir, encoded)
}

func (s *service) encodeImage(ctx context.Context, subDir string, staged string) (string, error) {
	job := media.Job{
		Type:      media.JobImage,
		InputPath: staged,
	}
	switch subDir {
	case "avatars":
		job.MaxWidth = media.AvatarMaxWidth
		job.MaxHeight = media.AvatarMaxHeight
		job.Quality = media.AvatarQuality
		job.SquareCrop = true
	case "banners":
		job.MaxWidth = media.BannerMaxWidth
		job.MaxHeight = media.BannerMaxHeight
		job.Quality = media.BannerQuality
	}

	result := make(chan string, 1)
	errCh := make(chan error, 1)
	job.Callback = func(outputPath string) {
		result <- outputPath
	}
	job.ErrorCallback = func(encErr error) {
		errCh <- encErr
	}
	s.mediaProc.Enqueue(job)

	select {
	case outputPath := <-result:
		return outputPath, nil
	case encErr := <-errCh:
		return "", encErr
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (s *service) SaveAudio(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	staged, err := s.StageMedia(ctx, media.MediaTypeAudio, subDir, id, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}
	defer s.Discard(staged)

	return s.Store(ctx, subDir, staged)
}

func (s *service) SaveAttachment(ctx context.Context, subDir string, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	if fileSize > maxSize {
		return "", fmt.Errorf("%w: file size %dMB exceeds maximum %dMB", ErrFileTooLarge, fileSize/(1024*1024), maxSize/(1024*1024))
	}

	sniffed, wrapped, err := DetectContentType(reader)
	if err != nil {
		return "", err
	}

	ext, ok := AllowedAttachmentTypes[NormaliseSniffedType(sniffed)]
	if !ok {
		return "", ErrInvalidAttachmentType
	}

	return s.SaveFile(ctx, subDir, uuid.New().String()+ext, wrapped)
}

func (s *service) Delete(urlPaths ...string) {
	for _, urlPath := range urlPaths {
		if err := s.delete(urlPath); err != nil {
			s.retryDelete(urlPath, err)
		}
	}
}

func (s *service) retryDelete(urlPath string, first error) {
	go func() {
		for _, delay := range deleteRetryDelays {
			time.Sleep(delay)

			if err := s.delete(urlPath); err == nil {
				logger.Log.Debug().Str("path", urlPath).Msg("deleted upload on retry")

				return
			}
		}

		logger.Log.Error().Err(first).Str("path", urlPath).Msg("failed to delete upload after retries")
	}()
}

func (s *service) delete(urlPath string) error {
	key, ok := keyFromURL(urlPath)
	if !ok {
		return nil
	}

	if err := s.storageSvc.Delete(context.Background(), key); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

func (s *service) DeleteByPrefix(ctx context.Context, subDir string, prefix string) error {
	rows, err := s.storedFiles.ListByPrefix(ctx, spec.StoredFilePrefix{Prefix: subDir + "/" + prefix})
	if err != nil {
		return fmt.Errorf("list stored files: %w", err)
	}

	urlPaths := make([]string, 0, len(rows))
	for _, row := range rows {
		urlPaths = append(urlPaths, urlPrefix+row.Key)
	}

	s.Delete(urlPaths...)

	return nil
}

func (s *service) CleanStaging(olderThan time.Duration) (int, error) {
	root := filepath.Join(s.settingsSvc.Get(context.Background(), config.SettingUploadDir), stagingDirName)

	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("read staging directory: %w", err)
	}

	cutoff := time.Now().Add(-olderThan)

	removed := 0
	var errs []error
	for _, entry := range entries {
		info, err := entry.Info()
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}

		dir := filepath.Join(root, entry.Name())
		if _, live := s.liveStaging.Load(dir); live || info.ModTime().After(cutoff) {
			continue
		}

		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, err)
			continue
		}

		removed++
	}

	return removed, errors.Join(errs...)
}

func (s *service) newStagingDir() (string, error) {
	root := filepath.Join(s.settingsSvc.Get(context.Background(), config.SettingUploadDir), stagingDirName)
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}

	dir, err := os.MkdirTemp(root, "upload-")
	if err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}

	s.liveStaging.Store(dir, struct{}{})

	return dir, nil
}

func (s *service) stage(filename string, reader io.Reader) (string, error) {
	dir, err := s.newStagingDir()
	if err != nil {
		return "", err
	}

	staged := filepath.Join(dir, filepath.Base(filename))

	dst, err := os.Create(staged)
	if err != nil {
		s.Discard(staged)

		return "", fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(dst, reader); err != nil {
		_ = dst.Close()
		s.Discard(staged)

		return "", fmt.Errorf("write file: %w", err)
	}

	if err := dst.Close(); err != nil {
		s.Discard(staged)

		return "", fmt.Errorf("close file: %w", err)
	}

	return staged, nil
}

func DetectContentType(reader io.Reader) (string, io.Reader, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(reader, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && err != io.EOF {
		return "", nil, fmt.Errorf("read for sniff: %w", err)
	}
	peek := buf[:n]
	mt := http.DetectContentType(peek)
	if i := strings.Index(mt, ";"); i >= 0 {
		mt = strings.TrimSpace(mt[:i])
	}
	if mt == "application/octet-stream" {
		if alt := sniffVideoFallback(peek); alt != "" {
			mt = alt
		}

		if alt := sniffFLAC(peek); alt != "" {
			mt = alt
		}
	}
	if mt == "video/webm" && bytes.Contains(peek, []byte("matroska")) {
		mt = "video/x-matroska"
	}
	if mt == "video/mp4" && isAudioOnlyMP4(peek) {
		mt = "audio/mp4"
	}
	if mt == "application/ogg" && !isOggAudio(peek) {
		mt = "application/octet-stream"
	}
	return mt, io.MultiReader(bytes.NewReader(peek), reader), nil
}

func NormaliseSniffedType(mediaType string) string {
	if alias, ok := sniffAliases[mediaType]; ok {
		return alias
	}

	return mediaType
}

func sniffVideoFallback(b []byte) string {
	if alt := sniffMP4(b); alt != "" {
		return alt
	}
	if alt := sniffMatroska(b); alt != "" {
		return alt
	}
	return ""
}

func sniffMP4(b []byte) string {
	if len(b) < 12 {
		return ""
	}
	boxSize := int(binary.BigEndian.Uint32(b[:4]))
	if boxSize < 8 || boxSize%4 != 0 || boxSize > len(b) {
		return ""
	}
	if !bytes.Equal(b[4:8], []byte("ftyp")) {
		return ""
	}
	for st := 8; st+4 <= boxSize; st += 4 {
		if st == 12 {
			continue
		}
		if mp4FallbackBrands[string(b[st:st+4])] {
			return "video/mp4"
		}
	}
	return ""
}

func sniffFLAC(b []byte) string {
	if len(b) >= 4 && bytes.Equal(b[:4], []byte("fLaC")) {
		return "audio/flac"
	}

	return ""
}

func isAudioOnlyMP4(b []byte) bool {
	if len(b) < 12 || !bytes.Equal(b[4:8], []byte("ftyp")) {
		return false
	}

	return audioMP4Brands[string(b[8:12])]
}

func isOggAudio(b []byte) bool {
	return bytes.Contains(b, []byte("vorbis")) || bytes.Contains(b, []byte("OpusHead")) || bytes.Contains(b, []byte("FLAC"))
}

func sniffMatroska(b []byte) string {
	if len(b) < 4 || !bytes.Equal(b[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return ""
	}
	if bytes.Contains(b, []byte("matroska")) {
		return "video/x-matroska"
	}
	if bytes.Contains(b, []byte("webm")) {
		return "video/webm"
	}
	return ""
}
