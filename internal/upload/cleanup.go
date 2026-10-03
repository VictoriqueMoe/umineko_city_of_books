package upload

import (
	"context"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/storage"
)

const orphanGracePeriod = time.Hour

func CleanOrphanedFiles(ctx context.Context, repo repository.UploadRepository, storedFiles repository.StoredFileRepository, storageSvc storage.Service) (int, error) {
	referenced, err := repo.GetAllReferencedFiles()
	if err != nil {
		return 0, fmt.Errorf("get referenced files: %w", err)
	}

	if len(referenced) == 0 {
		logger.Ctx(ctx).Warn().Msg("orphan cleanup skipped: zero referenced files in DB")
		return 0, nil
	}

	refSet := make(map[string]bool, len(referenced))
	for _, ref := range referenced {
		refSet[ref] = true
	}

	stored, err := storedFiles.ListByPrefix(ctx, spec.StoredFilePrefix{})
	if err != nil {
		return 0, fmt.Errorf("list stored files: %w", err)
	}

	cutoff := time.Now().Add(-orphanGracePeriod)

	removed := 0
	for _, file := range stored {
		if strings.Count(file.Key, "/") != 1 || file.CreatedAt.After(cutoff) {
			continue
		}

		urlPath := urlPrefix + file.Key
		if refSet[urlPath] {
			continue
		}

		if err := storageSvc.Delete(ctx, file.Key); err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("file", urlPath).Str("backend", string(file.Backend)).Msg("failed to remove orphaned file")
			continue
		}

		logger.Ctx(ctx).Info().Str("file", urlPath).Str("backend", string(file.Backend)).Msg("removed orphaned file")
		removed++
	}

	return removed, nil
}
