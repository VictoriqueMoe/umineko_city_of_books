package repository

import (
	"strings"

	"github.com/google/uuid"
)

func ExcludeClause(column string, ids []uuid.UUID) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	return " AND " + column + " NOT IN (" + strings.Join(placeholders, ",") + ")", args
}
