package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/XSAM/otelsql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq" // Import the PostgreSQL driver for side effects
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func setupDBWithTracing(ctx context.Context) (*sql.DB, error) {
	db, err := otelsql.Open("postgres",
		"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		otelsql.WithAttributes(
			semconv.DBSystemPostgreSQL,
			attribute.String("db.name", "postgres"),
			attribute.String("db.user", "postgres"),
		),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			DisableQuery:   false,
			DisableErrSkip: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection works
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	if err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	)); err != nil {
		return nil, fmt.Errorf("failed to register DB metrics: %w", err)
	}

	if err := doMigration(db, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to apply migration: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func doMigration(db *sql.DB, migration string) error {
	// Using golang-migrate library for migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Check if migration directory exists
	if _, err := os.Stat(migration); os.IsNotExist(err) {
		return fmt.Errorf("migration directory does not exist: %s", migration)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migration),
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Apply the migration
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	return nil
}
