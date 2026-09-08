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
		db  *sql.DB
		gen *sqlcgen.Queries
	}
)

func newSQLCSource(db *sql.DB) sqlcSource {
	return sqlcSource{db: db, gen: sqlcgen.New(db)}
}

func (s sqlcSource) queries(tx []*sql.Tx) *sqlcgen.Queries {
	if len(tx) > 0 && tx[0] != nil {
		return s.gen.WithTx(tx[0])
	}

	return s.gen
}

func genQueries(db *sql.DB, tx []*sql.Tx) *sqlcgen.Queries {
	if len(tx) > 0 && tx[0] != nil {
		return sqlcgen.New(tx[0])
	}

	return sqlcgen.New(db)
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
