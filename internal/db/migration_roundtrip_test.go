package db

import (
	"testing"

	"umineko_city_of_books/internal/db/dbtest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

const lastMigrationBeforeSchemaHardening = 20260726205627

func TestMigrations_DownAndUpRoundTrip(t *testing.T) {
	// given
	conn, _ := dbtest.NewEmptyDatabase(t)
	require.NoError(t, Migrate(conn))

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("postgres"))

	// when
	require.NoError(t, goose.DownTo(conn, "migrations", lastMigrationBeforeSchemaHardening))

	// then
	require.NoError(t, goose.Up(conn, "migrations"))
}
