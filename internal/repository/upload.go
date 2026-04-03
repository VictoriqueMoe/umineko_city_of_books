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
	tables, err := r.db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'goose_%'`)
	if err != nil {
		return "", fmt.Errorf("list tables: %w", err)
	}
	defer tables.Close()

	var parts []string
	for tables.Next() {
		var table string
		if err := tables.Scan(&table); err != nil {
			continue
		}
		if !safeIdentifier.MatchString(table) {
			continue
		}

		cols, err := r.db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, table))
		if err != nil {
			continue
		}

		for cols.Next() {
			var (
				cid           int
				name, colType string
				notNull, pk   int
				dflt          *string
			)
			if err := cols.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
				continue
			}
			if !safeIdentifier.MatchString(name) {
				continue
			}
			if strings.EqualFold(colType, "TEXT") {
				parts = append(parts, fmt.Sprintf(`SELECT DISTINCT "%s" FROM "%s" WHERE "%s" LIKE '/uploads/%%'`, name, table, name))
			}
		}
		_ = cols.Close()
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, " UNION "), nil
}
