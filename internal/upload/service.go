package upload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"

	"github.com/google/uuid"
)

var (
	AllowedImageTypes = map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}
)

type (
	Service interface {
		SaveFile(subDir string, filename string, reader io.Reader) (string, error)
		SaveImage(subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		DeleteByPrefix(subDir string, prefix string) error
		GetUploadDir() string
	}

	service struct{}
)

func NewService() Service {
	return &service{}
}

func (s *service) SaveFile(subDir string, filename string, reader io.Reader) (string, error) {
	dir := filepath.Join(config.Cfg.UploadDir, subDir)
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

func (s *service) SaveImage(subDir string, id uuid.UUID, contentType string, fileSize int64, maxSize int64, reader io.Reader) (string, error) {
	if fileSize > maxSize {
		return "", ErrFileTooLarge
	}

	ext, ok := AllowedImageTypes[contentType]
	if !ok {
		return "", ErrInvalidFileType
	}

	prefix := fmt.Sprintf("%s_", id.String())
	if err := s.DeleteByPrefix(subDir, prefix); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s_%d%s", id.String(), time.Now().UnixMilli(), ext)
	return s.SaveFile(subDir, filename, reader)
}

func (s *service) DeleteByPrefix(subDir string, prefix string) error {
	dir := filepath.Join(config.Cfg.UploadDir, subDir)
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

func (s *service) GetUploadDir() string {
	return config.Cfg.UploadDir
}
