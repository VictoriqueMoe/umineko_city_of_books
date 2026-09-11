package utils

import (
	"strings"

	"github.com/google/uuid"
)

func ExcludeClauseNullable(column string, ids []uuid.UUID, startIndex int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := PlaceholderArgs(ids, startIndex)

	return " AND (" + column + " IS NULL OR " + column + " NOT IN (" + strings.Join(placeholders, ",") + "))", args
}
