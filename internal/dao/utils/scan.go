package utils

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func ScanIDs(rows *sql.Rows, what string) ([]uuid.UUID, error) {
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan %s: %w", what, err)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func ScanStrings(rows *sql.Rows, what string) ([]string, error) {
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan %s: %w", what, err)
		}

		values = append(values, value)
	}

	return values, rows.Err()
}

func ScanGroups[K comparable, V any](rows *sql.Rows, what string) (map[K][]V, error) {
	defer rows.Close()

	groups := make(map[K][]V)
	for rows.Next() {
		var (
			key   K
			value V
		)
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan %s: %w", what, err)
		}

		groups[key] = append(groups[key], value)
	}

	return groups, rows.Err()
}

func ScanMap[K comparable, V any](rows *sql.Rows, what string) (map[K]V, error) {
	defer rows.Close()

	values := make(map[K]V)
	for rows.Next() {
		var (
			key   K
			value V
		)
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan %s: %w", what, err)
		}

		values[key] = value
	}

	return values, rows.Err()
}
