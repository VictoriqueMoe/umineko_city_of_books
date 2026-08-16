package gameroom

import (
	"fmt"
	"time"
)

func DurationSeconds(createdAt, finishedAt string) int {
	if createdAt == "" {
		return 0
	}

	start, err := parseDBTime(createdAt)
	if err != nil {
		return 0
	}

	end := time.Now().UTC()
	if finishedAt != "" {
		parsed, perr := parseDBTime(finishedAt)
		if perr != nil {
			return 0
		}
		end = parsed
	}

	d := end.Sub(start)
	if d < 0 {
		return 0
	}

	return int(d.Seconds())
}

func parseDBTime(s string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognised time format: %s", s)
}
