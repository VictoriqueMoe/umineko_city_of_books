package dao

import "time"

func timePtrToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return new(t.UTC().Format(time.RFC3339))
}
