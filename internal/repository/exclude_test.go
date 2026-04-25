package repository_test

import (
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExcludeClause_Empty(t *testing.T) {
	var ids []uuid.UUID

	clause, args := repository.ExcludeClause("user_id", ids, 1)

	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClause_Nil(t *testing.T) {
	clause, args := repository.ExcludeClause("user_id", nil, 1)

	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClause_SingleID(t *testing.T) {
	id := uuid.New()
	ids := []uuid.UUID{id}

	clause, args := repository.ExcludeClause("user_id", ids, 1)

	assert.Equal(t, " AND user_id NOT IN ($1)", clause)
	assert.Equal(t, []interface{}{id}, args)
}

func TestExcludeClause_MultipleIDs(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	ids := []uuid.UUID{a, b, c}

	clause, args := repository.ExcludeClause("author_id", ids, 1)

	assert.Equal(t, " AND author_id NOT IN ($1,$2,$3)", clause)
	assert.Equal(t, []interface{}{a, b, c}, args)
}

func TestExcludeClause_ColumnNameInterpolation(t *testing.T) {
	ids := []uuid.UUID{uuid.New()}

	clause, _ := repository.ExcludeClause("p.posted_by", ids, 5)

	assert.Contains(t, clause, "p.posted_by NOT IN")
	assert.Equal(t, " AND p.posted_by NOT IN ($5)", clause)
}
