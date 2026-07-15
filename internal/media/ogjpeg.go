package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strconv"

	"umineko_city_of_books/internal/bounds"

	"golang.org/x/image/webp"
)

const ogMaxWidth = 1200

func WebPToJPEG(ctx context.Context, inputPath string, maxPixels int) ([]byte, error) {
	img, err := decodeForOG(ctx, inputPath, maxPixels)
	if err != nil {
		if errors.Is(err, bounds.ErrImageBounds) {
			return nil, err
		}

		framePath, frameErr := extractFirstFrame(ctx, inputPath)
		if frameErr != nil {
			return nil, err
		}
		defer os.Remove(framePath)

		img, err = decodeForOG(ctx, framePath, maxPixels)
		if err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}

	return buf.Bytes(), nil
}

func decodeForOG(ctx context.Context, path string, maxPixels int) (image.Image, error) {
	cfg, err := webpConfig(path)
	if err != nil {
		return nil, err
	}

	if err := bounds.ImagePixels(cfg.Width, cfg.Height, maxPixels); err != nil {
		return nil, err
	}

	if cfg.Width <= ogMaxWidth {
		return decodeWebPFile(path)
	}

	return decodeScaledWebP(ctx, path)
}

func webpConfig(path string) (image.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, fmt.Errorf("open webp: %w", err)
	}
	defer f.Close()

	cfg, err := webp.DecodeConfig(f)
	if err != nil {
		return image.Config{}, fmt.Errorf("decode webp config: %w", err)
	}

	return cfg, nil
}

func decodeWebPFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open webp: %w", err)
	}
	defer f.Close()

	return webp.Decode(f)
}

func decodeScaledWebP(ctx context.Context, path string) (image.Image, error) {
	tmp, err := os.CreateTemp("", "ogscaled-*.png")
	if err != nil {
		return nil, fmt.Errorf("create temp scaled: %w", err)
	}
	outPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(outPath)

	cmd := exec.CommandContext(ctx, "dwebp", path, "-resize", strconv.Itoa(ogMaxWidth), "0", "-o", outPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("dwebp resize: %w: %s", err, string(out))
	}

	f, err := os.Open(outPath)
	if err != nil {
		return nil, fmt.Errorf("open scaled png: %w", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode scaled png: %w", err)
	}

	return img, nil
}

func extractFirstFrame(ctx context.Context, inputPath string) (string, error) {
	tmp, err := os.CreateTemp("", "ogframe-*.webp")
	if err != nil {
		return "", fmt.Errorf("create temp frame: %w", err)
	}
	framePath := tmp.Name()
	_ = tmp.Close()

	cmd := exec.CommandContext(ctx, "webpmux", "-get", "frame", "1", inputPath, "-o", framePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(framePath)
		return "", fmt.Errorf("webpmux get frame: %w: %s", err, string(out))
	}

	return framePath, nil
}
