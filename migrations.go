package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	sql, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("could not read migration file: %w", err)
	}

	_, err = db.Exec(ctx, string(sql))
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("✓ Migrations applied")
	return nil
}