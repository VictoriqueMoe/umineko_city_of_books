package media

import "strings"

const (
	maxFilenameRunes = 120
)

func SafeFilename(name string) string {
	name = strings.TrimSpace(name)

	if cut := strings.LastIndexAny(name, `/\`); cut >= 0 {
		name = name[cut+1:]
	}

	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}

		return r
	}, name)

	name = strings.TrimSpace(name)

	if runes := []rune(name); len(runes) > maxFilenameRunes {
		name = string(runes[:maxFilenameRunes])
	}

	return name
}
