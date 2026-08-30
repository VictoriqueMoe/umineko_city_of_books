package text

import "unicode/utf8"

func ClampBytes(s string, max int) string {
	if max <= 0 {
		return ""
	}

	if len(s) <= max {
		return s
	}

	end := max
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}

	return s[:end]
}

func ClampRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}

	count := 0
	for i := range s {
		if count == max {
			return s[:i]
		}
		count++
	}

	return s
}
