package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImageName = "postgres:latest"
	postgresUser      = "umineko"
	postgresPassword  = "umineko_test"
	postgresAdminDB   = "postgres"
)

var (
	containerOnce sync.Once
	containerErr  error

	adminDSN string

	dbCounter   int
	dbCounterMu sync.Mutex
)

func ensureContainer() {
	containerOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		c, err := tcpostgres.Run(ctx,
			postgresImageName,
			tcpostgres.WithDatabase(postgresAdminDB),
			tcpostgres.WithUsername(postgresUser),
			tcpostgres.WithPassword(postgresPassword),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
			),
		)
		if err != nil {
			containerErr = fmt.Errorf("start postgres container: %w", err)
			return
		}

		dsn, err := c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			containerErr = fmt.Errorf("get admin connection string: %w", err)
			return
		}
		adminDSN = dsn
	})
}

func AdminDSN(t *testing.T) string {
	t.Helper()
	ensureContainer()
	require.NoError(t, containerErr)
	return adminDSN
}

func nextDBName() string {
	dbCounterMu.Lock()
	defer dbCounterMu.Unlock()
	dbCounter++
	return fmt.Sprintf("test_%d_%d", os.Getpid(), dbCounter)
}

func openAdmin(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", AdminDSN(t))
	require.NoError(t, err)
	return db
}

func DSNForDatabase(t *testing.T, dbName string) string {
	t.Helper()
	dsn := AdminDSN(t)
	idx := strings.LastIndex(dsn, "/")
	if idx < 0 {
		return dsn
	}
	prefix := dsn[:idx+1]
	rest := dsn[idx+1:]
	q := strings.Index(rest, "?")
	if q < 0 {
		return prefix + dbName
	}
	return prefix + dbName + rest[q:]
}

func NewEmptyDatabase(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbName := nextDBName()

	admin := openAdmin(t)
	defer admin.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := admin.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName))
	require.NoError(t, err)

	db, err := sql.Open("pgx", DSNForDatabase(t, dbName))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		dropDB, err := sql.Open("pgx", AdminDSN(t))
		if err != nil {
			return
		}
		defer dropDB.Close()
		_, _ = dropDB.ExecContext(dropCtx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, dbName))
	})

	return db, dbName
}

func NewDatabaseFromTemplate(t *testing.T, templateName string) *sql.DB {
	t.Helper()
	dbName := nextDBName()

	admin := openAdmin(t)
	defer admin.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := admin.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s`, dbName, templateName))
	require.NoError(t, err)

	db, err := sql.Open("pgx", DSNForDatabase(t, dbName))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		dropDB, err := sql.Open("pgx", AdminDSN(t))
		if err != nil {
			return
		}
		defer dropDB.Close()
		_, _ = dropDB.ExecContext(dropCtx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, dbName))
	})

	return db
}

func MarkAsTemplate(ctx context.Context, dbName string) error {
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return fmt.Errorf("open admin: %w", err)
	}
	defer admin.Close()
	if _, err := admin.ExecContext(ctx, fmt.Sprintf(`UPDATE pg_database SET datistemplate = TRUE WHERE datname = '%s'`, dbName)); err != nil {
		return fmt.Errorf("mark template: %w", err)
	}
	return nil
}

func CreateDatabase(ctx context.Context, dbName string) error {
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return fmt.Errorf("open admin: %w", err)
	}
	defer admin.Close()
	if _, err := admin.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		return fmt.Errorf("create db: %w", err)
	}
	return nil
}

func DSNFor(dbName string) string {
	idx := strings.LastIndex(adminDSN, "/")
	if idx < 0 {
		return adminDSN
	}
	prefix := adminDSN[:idx+1]
	rest := adminDSN[idx+1:]
	q := strings.Index(rest, "?")
	if q < 0 {
		return prefix + dbName
	}
	return prefix + dbName + rest[q:]
}

func EnsureRunning() error {
	ensureContainer()
	return containerErr
}
