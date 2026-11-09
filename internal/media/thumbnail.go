package media

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/logger"
)

func getVideoDuration(ctx context.Context, videoPath string, limits []string) (float64, error) {
	args := slices.Concat(limits, []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	})

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

func generateThumbnail(job Job, threads int) (string, error) {
	videoPath := job.InputPath
	destPath := filepath.Join(filepath.Dir(videoPath), "thumb_"+replaceExt(filepath.Base(videoPath), ".webp"))
	limits := decodeLimits(job.MaxPixels, threads)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := "1"
	duration, err := getVideoDuration(ctx, videoPath, limits)
	if err == nil && duration > 1 {
		centiseconds, randErr := rand.Int(rand.Reader, big.NewInt(int64(duration*100)))
		if randErr != nil {
			return "", fmt.Errorf("pick thumbnail frame: %w", randErr)
		}

		ts = fmt.Sprintf("%.2f", float64(centiseconds.Int64())/100)
	}

	args := slices.Concat([]string{"-ss", ts}, limits, []string{
		"-i", videoPath,
		"-frames:v", "1",
		"-vcodec", "libwebp",
		"-vf", "scale=-1:200",
		"-q:v", "80",
		"-y",
		destPath,
	})

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg thumbnail: %w: %s", err, string(output))
	}

	logger.Ctx(ctx).Debug().Str("video", videoPath).Str("thumb", destPath).Msg("thumbnail generated")
	return destPath, nil
}
