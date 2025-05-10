package storage

import (
	"context"
	"database/sql"
	"fmt"

	"wapp/constant"
	"wapp/tractx"

	"go.opentelemetry.io/otel/attribute"
)

type Storage struct {
	db *sql.DB
}

func New(ctx context.Context) (*Storage, error) {
	db, err := setupDBWithTracing(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	if err := doMigration(db, "migrations/postgres"); err != nil {
		return nil, fmt.Errorf("failed to apply migration: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) GetSurname(ctx tractx.Context, name string) (string, error) {
	// Создаем новый спан для операции получения фамилии
	ctxWithSpan, span, stop := ctx.TracerStart("get-surname")
	defer stop()

	span.SetAttributes(attribute.String("nameInStorage", name))

	// Используем QueryContext с контекстом спана
	rows, err := s.db.QueryContext(ctxWithSpan, "SELECT surname FROM users WHERE name = $1", name)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	// Check if we have results
	if !rows.Next() {
		err := fmt.Errorf("%w: no user found with name %s", constant.NotFound, name)
		span.RecordError(err)
		return "", err
	}

	var result string
	err = rows.Scan(&result)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to scan database result: %w", err)
	}

	span.SetAttributes(attribute.String("result.surname", result))
	return result, nil
}
