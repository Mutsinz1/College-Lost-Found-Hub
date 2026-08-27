package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"lostfound/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	// Use the simple query protocol so a migration file can contain multiple
	// statements, functions and triggers (dollar-quoted bodies included).
	connCfg, err := pgx.ParseConfig(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}
	connCfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	conn, err := pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Run migrations
	if err := runMigrations(ctx, conn); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")
}

func runMigrations(ctx context.Context, conn *pgx.Conn) error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			filename VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		)`

	if _, err := conn.Exec(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	migrationsDir := "migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name
	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	// Get applied migrations
	appliedMigrations, err := getAppliedMigrations(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Run pending migrations
	for _, filename := range migrationFiles {
		if appliedMigrations[filename] {
			log.Printf("Migration %s already applied, skipping", filename)
			continue
		}

		log.Printf("Running migration: %s", filename)

		if err := runMigration(ctx, conn, filename); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", filename, err)
		}

		if err := recordMigration(ctx, conn, filename); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		log.Printf("Migration %s completed", filename)
	}

	return nil
}

func getAppliedMigrations(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, `SELECT filename FROM migrations ORDER BY applied_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		applied[filename] = true
	}

	return applied, rows.Err()
}

// runMigration executes a migration file in a single transaction. The whole
// file is sent as one script: no splitting on semicolons, so functions and
// triggers with dollar-quoted bodies work correctly.
func runMigration(ctx context.Context, conn *pgx.Conn, filename string) error {
	content, err := os.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return tx.Commit(ctx)
}

func recordMigration(ctx context.Context, conn *pgx.Conn, filename string) error {
	_, err := conn.Exec(ctx, `INSERT INTO migrations (filename) VALUES ($1)`, filename)
	return err
}
