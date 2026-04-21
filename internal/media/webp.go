package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"umineko_city_of_books/internal/logger"

	"github.com/disintegration/imaging"
)

const (
	AvatarMaxWidth  = 96
	AvatarMaxHeight = 0
	AvatarQuality   = 60
	BannerMaxWidth  = 1600
	BannerMaxHeight = 0
	BannerQuality   = 72
	DefaultQuality  = 80
)

type (
	WebPOptions struct {
		MaxWidth   int
		MaxHeight  int
		Quality    int
		SquareCrop bool
	}
)

func EncodeWebP(ctx context.Context, inputPath string, opts WebPOptions) (string, error) {
	if opts.Quality <= 0 {
		opts.Quality = DefaultQuality
	}
	lower := strings.ToLower(inputPath)

	if strings.HasSuffix(lower, ".webp") {
		return reencodeWebPInPlace(ctx, inputPath, opts)
	}

	outputPath := replaceExt(inputPath, ".webp")

	if strings.HasSuffix(lower, ".gif") {
		if err := encodeAnimatedWebP(ctx, inputPath, outputPath, opts); err != nil {
			return "", err
		}
		_ = os.Remove(inputPath)
		return outputPath, nil
	}

	cwebpInput := inputPath
	var tempFiles []string
	if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") {
		oriented, err := applyExifOrientation(inputPath)
		if err != nil {
			logger.Log.Warn().Err(err).Str("input", inputPath).Msg("exif auto-orient failed, using original")
		} else if oriented != "" {
			cwebpInput = oriented
			tempFiles = append(tempFiles, oriented)
		}
	}

	if opts.SquareCrop {
		cropped, err := centerCropSquare(cwebpInput)
		if err != nil {
			logger.Log.Warn().Err(err).Str("input", cwebpInput).Msg("square crop failed, using original aspect")
		} else if cropped != "" {
			cwebpInput = cropped
			tempFiles = append(tempFiles, cropped)
		}
	}

	if err := runCwebp(ctx, cwebpInput, outputPath, opts); err != nil {
		for _, p := range tempFiles {
			_ = os.Remove(p)
		}
		return "", err
	}
	for _, p := range tempFiles {
		_ = os.Remove(p)
	}
	_ = os.Remove(inputPath)
	return outputPath, nil
}

func centerCropSquare(inputPath string) (string, error) {
	img, err := imaging.Open(inputPath, imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	size := img.Bounds().Dx()
	if img.Bounds().Dy() < size {
		size = img.Bounds().Dy()
	}
	if size <= 0 {
		return "", fmt.Errorf("invalid image dimensions")
	}
	cropped := imaging.CropCenter(img, size, size)
	tmpPath := replaceExt(inputPath, ".square.jpg")
	if err := imaging.Save(cropped, tmpPath, imaging.JPEGQuality(95)); err != nil {
		return "", fmt.Errorf("save cropped: %w", err)
	}
	return tmpPath, nil
}

func reencodeWebPInPlace(ctx context.Context, inputPath string, opts WebPOptions) (string, error) {
	tmpPath := replaceExt(inputPath, ".tmp.webp")
	_ = os.Remove(tmpPath)

	cwebpInput := inputPath
	var tempFiles []string
	if opts.SquareCrop {
		cropped, err := centerCropSquare(inputPath)
		if err != nil {
			logger.Log.Warn().Err(err).Str("input", inputPath).Msg("square crop failed, using original aspect")
		} else if cropped != "" {
			cwebpInput = cropped
			tempFiles = append(tempFiles, cropped)
		}
	}

	if err := runCwebp(ctx, cwebpInput, tmpPath, opts); err != nil {
		_ = os.Remove(tmpPath)
		for _, p := range tempFiles {
			_ = os.Remove(p)
		}
		if isAnimatedWebPError(err) {
			logger.Log.Debug().Str("path", inputPath).Msg("skipping animated webp re-encode")
			return inputPath, nil
		}
		return "", err
	}
	for _, p := range tempFiles {
		_ = os.Remove(p)
	}

	if err := os.Remove(inputPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, inputPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return inputPath, nil
}

func isAnimatedWebPError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "animated WebP") || strings.Contains(msg, "Decoding of an animated")
}

func runCwebp(ctx context.Context, inputPath, outputPath string, opts WebPOptions) error {
	args := []string{"-q", strconv.Itoa(opts.Quality)}
	if opts.MaxWidth > 0 || opts.MaxHeight > 0 {
		args = append(args, "-resize", strconv.Itoa(opts.MaxWidth), strconv.Itoa(opts.MaxHeight))
	}
	args = append(args, inputPath, "-o", outputPath)

	cmd := exec.CommandContext(ctx, "cwebp", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cwebp: %w: %s", err, string(out))
	}
	return nil
}

func encodeAnimatedWebP(ctx context.Context, inputPath, outputPath string, opts WebPOptions) error {
	args := []string{"-i", inputPath}

	if opts.MaxWidth > 0 || opts.MaxHeight > 0 {
		widthExpr := "iw"
		heightExpr := "ih"
		if opts.MaxWidth > 0 {
			widthExpr = fmt.Sprintf("min(iw\\,%d)", opts.MaxWidth)
		}
		if opts.MaxHeight > 0 {
			heightExpr = fmt.Sprintf("min(ih\\,%d)", opts.MaxHeight)
		}
		args = append(args, "-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease", widthExpr, heightExpr))
	}

	args = append(args,
		"-an",
		"-c:v", "libwebp_anim",
		"-quality", strconv.Itoa(opts.Quality),
		"-loop", "0",
		"-y",
		outputPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg animated webp: %w: %s", err, string(out))
	}
	return nil
}

func applyExifOrientation(inputPath string) (string, error) {
	img, err := imaging.Open(inputPath, imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}

	tmpPath := replaceExt(inputPath, ".oriented.jpg")
	if err := imaging.Save(img, tmpPath, imaging.JPEGQuality(95)); err != nil {
		return "", fmt.Errorf("save oriented image: %w", err)
	}
	return tmpPath, nil
}

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)] + newExt
}
