package media

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/logger"
)

func getVideoDuration(ctx context.Context, videoPath string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

func GenerateThumbnail(videoPath string, outputDir string, filename string) (string, error) {
	thumbFilename := "thumb_" + replaceExt(filename, ".webp")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create thumbnail dir: %w", err)
	}

	destPath := filepath.Join(outputDir, thumbFilename)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := "1"
	duration, err := getVideoDuration(ctx, videoPath)
	if err == nil && duration > 1 {
		ts = fmt.Sprintf("%.2f", rand.Float64()*duration)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", ts,
		"-i", videoPath,
		"-frames:v", "1",
		"-vcodec", "libwebp",
		"-vf", "scale=-1:200",
		"-q:v", "80",
		"-y",
		destPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg thumbnail: %w: %s", err, string(output))
	}

	logger.Log.Debug().Str("video", videoPath).Str("thumb", destPath).Msg("thumbnail generated")
	return thumbFilename, nil
}
