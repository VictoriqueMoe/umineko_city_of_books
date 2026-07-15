package bounds

import (
	"errors"
	"fmt"
)

const FallbackMaxImagePixels = 100000000

var (
	ErrImageBounds = errors.New("image outside allowed bounds")
)

func ImagePixels(width, height, maxPixels int) error {
	if maxPixels <= 0 {
		maxPixels = FallbackMaxImagePixels
	}

	if width <= 0 || height <= 0 {
		return fmt.Errorf("%w: invalid dimensions %dx%d", ErrImageBounds, width, height)
	}

	if width > maxPixels/height {
		return fmt.Errorf("%w: %dx%d exceeds maximum of %d pixels", ErrImageBounds, width, height, maxPixels)
	}

	return nil
}
