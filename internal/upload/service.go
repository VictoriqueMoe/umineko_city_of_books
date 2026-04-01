package upload

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
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
		"video/mp4":              ".mp4",
		"video/webm":             ".webm",
		"video/quicktime":        ".mov",
		"video/x-msvideo":        ".avi",
		"video/x-matroska":       ".mkv",
		"video/matroska":         ".mkv",
		"application/x-matroska": ".mkv",
	}
)

type (
	Service interface {
		SaveFile(subDir string, filename string, reader io.Reader) (string, error)
		SaveImage(ctx context.Context, subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		SaveVideo(ctx context.Context, subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		Delete(urlPath string) error
		DeleteByPrefix(subDir string, prefix string) error
		GetUploadDir() string
		FullDiskPath(urlPath string) string
	}

	service struct {
		settingsSvc settings.Service
	}
)

func NewService(settingsSvc settings.Service) Service {
	return &service{settingsSvc: settingsSvc}
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

func (s *service) saveMedia(subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, allowedTypes map[string]string, typeErr error, reader io.Reader) (string, error) {
	if fileSize > maxSize {
		return "", ErrFileTooLarge
	}

	ext, ok := allowedTypes[contentType]
	if !ok {
		return "", typeErr
	}

	prefix := fmt.Sprintf("%s_", id.String())
	if err := s.DeleteByPrefix(subDir, prefix); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s_%d%s", id.String(), time.Now().UnixMilli(), ext)
	return s.SaveFile(subDir, filename, reader)
}

func (s *service) SaveImage(_ context.Context, subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	return s.saveMedia(subDir, id, contentType, fileSize, maxSize, AllowedImageTypes, ErrInvalidFileType, reader)
}

func (s *service) SaveVideo(_ context.Context, subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	return s.saveMedia(subDir, id, contentType, fileSize, maxSize, AllowedVideoTypes, ErrInvalidVideoType, reader)
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
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
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
