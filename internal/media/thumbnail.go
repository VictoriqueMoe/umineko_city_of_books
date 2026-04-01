package media

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"umineko_city_of_books/internal/logger"
)

const thumbnailAPIBase = "https://thumbnails.waifuvault.moe"

func GenerateThumbnail(fileURL string, outputDir string, filename string) (string, error) {
	reqURL := fmt.Sprintf("%s/generateThumbnail/ext/fromURL?url=%s", thumbnailAPIBase, url.QueryEscape(fileURL))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("thumbnail request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("thumbnail API returned %d: %s", resp.StatusCode, string(body))
	}

	thumbFilename := "thumb_" + replaceExt(filename, ".webp")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create thumbnail dir: %w", err)
	}

	destPath := filepath.Join(outputDir, thumbFilename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create thumbnail file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return "", fmt.Errorf("write thumbnail: %w", err)
	}

	logger.Log.Debug().Str("url", fileURL).Str("thumb", destPath).Msg("thumbnail generated")
	return thumbFilename, nil
}
