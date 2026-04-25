package repository

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func ExcludeClause(column string, ids []uuid.UUID, startIndex int) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
		args[i] = id
	}
	return " AND " + column + " NOT IN (" + strings.Join(placeholders, ",") + ")", args
}
