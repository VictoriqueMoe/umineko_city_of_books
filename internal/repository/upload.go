package repository

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var safeIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type (
	UploadRepository interface {
		GetAllReferencedFiles() ([]string, error)
	}

	uploadRepository struct {
		db *sql.DB
	}
)

func (r *uploadRepository) GetAllReferencedFiles() ([]string, error) {
	query, err := r.buildUnionQuery()
	if err != nil {
		return nil, err
	}
	if query == "" {
		return nil, nil
	}

	rows, err := r.db.Query(query)
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

func (r *uploadRepository) buildUnionQuery() (string, error) {
	rows, err := r.db.Query(
		`SELECT table_name, column_name
		 FROM information_schema.columns
		 WHERE table_schema = 'public'
		   AND data_type IN ('text', 'character varying', 'citext')
		   AND table_name NOT LIKE 'goose_%'`,
	)
	if err != nil {
		return "", fmt.Errorf("list text columns: %w", err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			continue
		}
		if !safeIdentifier.MatchString(table) {
			continue
		}
		if !safeIdentifier.MatchString(column) {
			continue
		}
		parts = append(parts, fmt.Sprintf(`SELECT DISTINCT "%s" FROM "%s" WHERE "%s" LIKE '/uploads/%%'`, column, table, column))
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate text columns: %w", err)
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, " UNION "), nil
}
