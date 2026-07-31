package media

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"umineko_city_of_books/internal/bounds"

	_ "golang.org/x/image/webp"
)

func CheckImageFileBounds(path string, maxPixels int) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil
	}

	return bounds.ImagePixels(cfg.Width, cfg.Height, maxPixels)
}

func DecodeDimensions(r io.Reader) (int, int) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0
	}

	return cfg.Width, cfg.Height
}
