package engines

import (
	"path/filepath"
	"umineko_city_of_books/internal/storage/engine"
)

func ResolvePath(uploadDir string, key string) (string, error) {
	if err := engine.ValidateKey(key); err != nil {
		return "", err
	}

	return filepath.Join(uploadDir, filepath.FromSlash(key)), nil
}

func Root(uploadDir string) (string, error) {
	return filepath.Abs(uploadDir)
}
