package dao

import (
	"strings"

	"umineko_city_of_books/internal/dao/utils"

	"github.com/google/uuid"
)

func ExcludeClause(column string, ids []uuid.UUID, startIndex int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := utils.PlaceholderArgs(ids, startIndex)

	return " AND " + column + " NOT IN (" + strings.Join(placeholders, ",") + ")", args
}

func ExcludeClauseNullable(column string, ids []uuid.UUID, startIndex int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := utils.PlaceholderArgs(ids, startIndex)

	return " AND (" + column + " IS NULL OR " + column + " NOT IN (" + strings.Join(placeholders, ",") + "))", args
}
