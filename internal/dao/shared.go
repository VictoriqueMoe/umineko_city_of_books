package dao

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
)

func joinUUIDs(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return ""
	}

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, id.String())
	}

	return strings.Join(parts, ",")
}

func timePtrToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return new(t.UTC().Format(time.RFC3339))
}

func nullTimeToStringPtr(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}

	return new(nt.Time.UTC().Format(time.RFC3339))
}
