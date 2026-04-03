package upload

import (
	"os"
	"path/filepath"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
)

func CleanOrphanedFiles(repo repository.UploadRepository, uploadDir string) int {
	referenced, err := repo.GetAllReferencedFiles()
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to get referenced files for cleanup")
		return 0
	}

	refSet := make(map[string]bool, len(referenced))
	for _, ref := range referenced {
		refSet[ref] = true
	}

	removed := 0
	topEntries, err := os.ReadDir(uploadDir)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to read upload directory")
		return 0
	}

	for _, topEntry := range topEntries {
		if !topEntry.IsDir() {
			continue
		}
		subDir := topEntry.Name()
		dir := filepath.Join(uploadDir, subDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			urlPath := "/uploads/" + subDir + "/" + entry.Name()
			if !refSet[urlPath] {
				fullPath := filepath.Join(dir, entry.Name())
				if err := os.Remove(fullPath); err != nil {
					logger.Log.Warn().Err(err).Str("file", fullPath).Msg("failed to remove orphaned file")
				} else {
					logger.Log.Info().Str("file", urlPath).Msg("removed orphaned file")
					removed++
				}
			}
		}
	}

	return removed
}
