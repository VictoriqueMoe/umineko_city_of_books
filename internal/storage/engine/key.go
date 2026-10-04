package engine

import (
	"fmt"
	"path"
	"strings"
)

func ValidateKey(key string) error {
	if key == "" || strings.ContainsAny(key, "\\:\x00") || path.Clean(key) != key || strings.HasPrefix(key, "/") {
		return fmt.Errorf("%w: %q", ErrInvalidKey, key)
	}

	for segment := range strings.SplitSeq(key, "/") {
		if strings.HasPrefix(segment, ".") {
			return fmt.Errorf("%w: %q", ErrInvalidKey, key)
		}
	}

	return nil
}
