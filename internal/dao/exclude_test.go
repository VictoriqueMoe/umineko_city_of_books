package dao_test

import (
	"testing"

	"umineko_city_of_books/internal/dao"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExcludeClause_Empty(t *testing.T) {
	var ids []uuid.UUID

	clause, args := dao.ExcludeClause("user_id", ids, 1)

	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClause_Nil(t *testing.T) {
	clause, args := dao.ExcludeClause("user_id", nil, 1)

	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClause_SingleID(t *testing.T) {
	id := uuid.New()
	ids := []uuid.UUID{id}

	clause, args := dao.ExcludeClause("user_id", ids, 1)

	assert.Equal(t, " AND user_id NOT IN ($1)", clause)
	assert.Equal(t, []any{id}, args)
}

func TestExcludeClause_MultipleIDs(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	ids := []uuid.UUID{a, b, c}

	clause, args := dao.ExcludeClause("author_id", ids, 1)

	assert.Equal(t, " AND author_id NOT IN ($1,$2,$3)", clause)
	assert.Equal(t, []any{a, b, c}, args)
}

func TestExcludeClauseNullable_Empty(t *testing.T) {
	clause, args := dao.ExcludeClauseNullable("cr.created_by", nil, 1)

	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClauseNullable_KeepsNullRows(t *testing.T) {
	a := uuid.New()
	b := uuid.New()

	clause, args := dao.ExcludeClauseNullable("cr.created_by", []uuid.UUID{a, b}, 3)

	assert.Equal(t, " AND (cr.created_by IS NULL OR cr.created_by NOT IN ($3,$4))", clause)
	assert.Equal(t, []any{a, b}, args)
}

func TestExcludeClause_ColumnNameInterpolation(t *testing.T) {
	ids := []uuid.UUID{uuid.New()}

	clause, _ := dao.ExcludeClause("p.posted_by", ids, 5)

	assert.Contains(t, clause, "p.posted_by NOT IN")
	assert.Equal(t, " AND p.posted_by NOT IN ($5)", clause)
}
