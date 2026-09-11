package dao

import (
	"database/sql"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/db"
)

func txOrDB(handle *sql.DB, tx []*sql.Tx) sqlcgen.DBTX {
	return db.TxOrDB(handle, tx)
}

func genQueries(handle *sql.DB, tx []*sql.Tx) *sqlcgen.Queries {
	return sqlcgen.New(txOrDB(handle, tx))
}
