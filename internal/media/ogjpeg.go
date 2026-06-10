package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"

	"golang.org/x/image/webp"
)

func WebPToJPEG(ctx context.Context, inputPath string) ([]byte, error) {
	img, err := decodeWebPFile(inputPath)
	if err != nil {
		img, err = decodeFirstFrame(ctx, inputPath)
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

func decodeWebPFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open webp: %w", err)
	}
	defer f.Close()

	return webp.Decode(f)
}

func decodeFirstFrame(ctx context.Context, inputPath string) (image.Image, error) {
	tmp, err := os.CreateTemp("", "ogframe-*.webp")
	if err != nil {
		return nil, fmt.Errorf("create temp frame: %w", err)
	}
	framePath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(framePath)

	cmd := exec.CommandContext(ctx, "webpmux", "-get", "frame", "1", inputPath, "-o", framePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("webpmux get frame: %w: %s", err, string(out))
	}

	return decodeWebPFile(framePath)
}
