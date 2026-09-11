package dynamicsql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/db"
)

type (
	Upload struct {
		db *sql.DB
	}
)

var (
	safeIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func NewUpload(db *sql.DB) *Upload {
	return &Upload{db: db}
}

func (r *Upload) GetAllReferencedFiles(tx ...*sql.Tx) ([]string, error) {
	ctx := context.Background()

	query, err := r.buildUnionQuery(ctx, tx...)
	if err != nil {
		return nil, err
	}

	if query == "" {
		return nil, nil
	}

	rows, err := db.TxOrDB(r.db, tx).QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query referenced files: %w", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			continue
		}

		url = strings.TrimSpace(url)
		if url != "" {
			results = append(results, url)
		}
	}

	return results, rows.Err()
}

func (r *Upload) buildUnionQuery(ctx context.Context, tx ...*sql.Tx) (string, error) {
	columns, err := sqlcgen.New(db.TxOrDB(r.db, tx)).ListUploadTextColumns(ctx)
	if err != nil {
		return "", fmt.Errorf("list text columns: %w", err)
	}

	var parts []string
	for _, column := range columns {
		if !safeIdentifier.MatchString(column.TableName) {
			continue
		}

		if !safeIdentifier.MatchString(column.ColumnName) {
			continue
		}

		parts = append(parts, fmt.Sprintf(`SELECT DISTINCT "%s" FROM "%s" WHERE "%s" LIKE '/uploads/%%'`, column.ColumnName, column.TableName, column.ColumnName))
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, " UNION "), nil
}
