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
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

var (
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

	sniffAliases = map[string]string{
		"video/avi":      "video/x-msvideo",
		"video/matroska": "video/x-matroska",
	}

	mp4FallbackBrands = map[string]bool{
		"isom": true, "iso2": true, "iso4": true, "iso5": true, "iso6": true,
		"mp41": true, "mp42": true, "mp71": true, "avc1": true, "dash": true,
		"msdh": true, "msix": true, "M4V ": true, "M4A ": true, "qt  ": true,
	}
)

type (
	Service interface {
		SaveFile(subDir string, filename string, reader io.Reader) (string, error)
		SaveImage(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		SaveVideo(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		Delete(urlPath string) error
		DeleteByPrefix(subDir string, prefix string) error
		GetUploadDir() string
		FullDiskPath(urlPath string) string
	}

	service struct {
		settingsSvc settings.Service
		mediaProc   *media.Processor
	}
)

func NewService(settingsSvc settings.Service, processors ...*media.Processor) Service {
	var mediaProc *media.Processor
	if len(processors) > 0 {
		mediaProc = processors[0]
	}

	return &service{settingsSvc: settingsSvc, mediaProc: mediaProc}
}

func (s *service) GetUploadDir() string {
	return s.settingsSvc.Get(context.Background(), config.SettingUploadDir)
}

func (s *service) SaveFile(subDir string, filename string, reader io.Reader) (string, error) {
	dir := filepath.Join(s.GetUploadDir(), subDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	destPath := filepath.Join(dir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, reader); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("/uploads/%s/%s", subDir, filename), nil
}

func (s *service) saveMedia(
	subDir string,
	id uuid.UUID,
	fileSize int64,
	maxSize int64,
	allowedTypes map[string]string,
	typeErr error,
	reader io.Reader,
) (string, error) {
	if fileSize > maxSize {
		return "", fmt.Errorf("file size %dMB exceeds maximum %dMB", fileSize/(1024*1024), maxSize/(1024*1024))
	}

	sniffed, wrapped, err := DetectContentType(reader)
	if err != nil {
		return "", err
	}
	if alias, ok := sniffAliases[sniffed]; ok {
		sniffed = alias
	}

	ext, ok := allowedTypes[sniffed]
	if !ok {
		return "", typeErr
	}

	prefix := fmt.Sprintf("%s_", id.String())
	if err := s.DeleteByPrefix(subDir, prefix); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s_%d%s", id.String(), time.Now().UnixMilli(), ext)
	return s.SaveFile(subDir, filename, wrapped)
}

func (s *service) SaveImage(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	urlPath, err := s.saveMedia(subDir, id, fileSize, maxSize, AllowedImageTypes, ErrInvalidFileType, reader)
	if err != nil {
		return "", err
	}

	if s.mediaProc == nil || strings.HasSuffix(strings.ToLower(urlPath), ".webp") {
		return urlPath, nil
	}

	job := media.Job{
		Type:      media.JobImage,
		InputPath: s.FullDiskPath(urlPath),
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
		return fmt.Sprintf("/uploads/%s/%s", subDir, filepath.Base(outputPath)), nil
	case encErr := <-errCh:
		_ = os.Remove(job.InputPath)
		return "", encErr
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (s *service) SaveVideo(_ context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	return s.saveMedia(subDir, id, fileSize, maxSize, AllowedVideoTypes, ErrInvalidVideoType, reader)
}

func (s *service) Delete(urlPath string) error {
	if urlPath == "" {
		return nil
	}
	path := s.FullDiskPath(urlPath)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (s *service) FullDiskPath(urlPath string) string {
	rel := strings.TrimPrefix(urlPath, "/uploads/")
	return filepath.Join(s.GetUploadDir(), rel)
}

func (s *service) DeleteByPrefix(subDir string, prefix string) error {
	dir := filepath.Join(s.GetUploadDir(), subDir)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("read directory: path is not a directory: %s", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("remove file: %w", err)
			}
		}
	}
	return nil
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
	}
	if mt == "video/webm" && bytes.Contains(peek, []byte("matroska")) {
		mt = "video/x-matroska"
	}
	return mt, io.MultiReader(bytes.NewReader(peek), reader), nil
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
