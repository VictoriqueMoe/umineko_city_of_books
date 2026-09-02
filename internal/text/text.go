package text

import "unicode/utf8"

func ClampBytes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}

	if len(s) <= limit {
		return s
	}

	end := limit
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}

	return s[:end]
}

func ClampRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}

	count := 0
	for i := range s {
		if count == limit {
			return s[:i]
		}
		count++
	}

	return s
}
