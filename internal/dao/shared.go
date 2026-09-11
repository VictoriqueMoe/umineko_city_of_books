package dao

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
)

type (
	sqlcSource struct {
		db *sql.DB
	}
)

func newSQLCSource(db *sql.DB) sqlcSource {
	return sqlcSource{db: db}
}

func (s sqlcSource) queries(tx []*sql.Tx) *sqlcgen.Queries {
	return genQueries(s.db, tx)
}

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

func nullTimeToStringPtr(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}

	return new(nt.Time.UTC().Format(time.RFC3339))
}
