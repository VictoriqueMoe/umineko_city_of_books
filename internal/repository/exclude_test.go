package repository_test

import (
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExcludeClause_Empty(t *testing.T) {
	// given
	var ids []uuid.UUID

	// when
	clause, args := repository.ExcludeClause("user_id", ids)

	// then
	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClause_Nil(t *testing.T) {
	// given

	// when
	clause, args := repository.ExcludeClause("user_id", nil)

	// then
	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestExcludeClause_SingleID(t *testing.T) {
	// given
	id := uuid.New()
	ids := []uuid.UUID{id}

	// when
	clause, args := repository.ExcludeClause("user_id", ids)

	// then
	assert.Equal(t, " AND user_id NOT IN (?)", clause)
	assert.Equal(t, []interface{}{id}, args)
}

func TestExcludeClause_MultipleIDs(t *testing.T) {
	// given
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	ids := []uuid.UUID{a, b, c}

	// when
	clause, args := repository.ExcludeClause("author_id", ids)

	// then
	assert.Equal(t, " AND author_id NOT IN (?,?,?)", clause)
	assert.Equal(t, []interface{}{a, b, c}, args)
}

func TestExcludeClause_ColumnNameInterpolation(t *testing.T) {
	// given
	ids := []uuid.UUID{uuid.New()}

	// when
	clause, _ := repository.ExcludeClause("p.posted_by", ids)

	// then
	assert.Contains(t, clause, "p.posted_by NOT IN")
	assert.Equal(t, " AND p.posted_by NOT IN (?)", clause)
}
