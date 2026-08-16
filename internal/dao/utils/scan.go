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
