package dao_test

import (
	"testing"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/db/dbtest"

	"github.com/stretchr/testify/require"
)

const lastMigrationBeforeSchemaHardening = 20260726205627

func TestMigrations_DownAndUpRoundTrip(t *testing.T) {
	// given
	conn, _ := dbtest.NewEmptyDatabase(t)
	require.NoError(t, db.Migrate(conn))

	// when
	require.NoError(t, db.MigrateDownTo(conn, lastMigrationBeforeSchemaHardening))

	// then
	require.NoError(t, db.Migrate(conn))
}
