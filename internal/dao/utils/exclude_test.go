package utils_test

import (
	"testing"

	"umineko_city_of_books/internal/dao/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExcludeClauseNullable_Empty(t *testing.T) {
	clause, args := utils.ExcludeClauseNullable("cr.created_by", nil, 1)

	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClauseNullable_KeepsNullRows(t *testing.T) {
	a := uuid.New()
	b := uuid.New()

	clause, args := utils.ExcludeClauseNullable("cr.created_by", []uuid.UUID{a, b}, 3)

	assert.Equal(t, " AND (cr.created_by IS NULL OR cr.created_by NOT IN ($3,$4))", clause)
	assert.Equal(t, []any{a, b}, args)
}

func TestExcludeClauseNullable_ColumnNameInterpolation(t *testing.T) {
	ids := []uuid.UUID{uuid.New()}

	clause, _ := utils.ExcludeClauseNullable("p.posted_by", ids, 5)

	assert.Equal(t, " AND (p.posted_by IS NULL OR p.posted_by NOT IN ($5))", clause)
}
