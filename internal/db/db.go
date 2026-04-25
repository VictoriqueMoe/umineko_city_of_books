package db

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	//go:embed migrations/*.sql
	migrationsFS embed.FS
)

func Open(dsn string) (*sql.DB, error) {
	pgConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	connector := stdlib.GetConnector(*pgConfig)

	db := otelsql.OpenDB(connector,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			DisableErrSkip:  true,
			OmitConnPrepare: true,
			OmitRows:        true,
		}),
	)

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func Migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
