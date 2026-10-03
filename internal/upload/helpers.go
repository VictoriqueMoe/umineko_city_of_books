package upload

import (
	"strings"
)

const (
	urlPrefix = "/uploads/"
)

func keyFromURL(urlPath string) (string, bool) {
	key, ok := strings.CutPrefix(urlPath, urlPrefix)

	return key, ok && key != ""
}
