package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lostfound/internal/config"
	"lostfound/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")
}

func runMigrations(db *database.DB) error {
	ctx := context.Background()

	// Create migrations table if it doesn't exist
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			filename VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		)`

	if err := db.Exec(ctx, createMigrationsTable); err != nil {
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
	appliedMigrations, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Run pending migrations
	for _, filename := range migrationFiles {
		if !appliedMigrations[filename] {
			log.Printf("Running migration: %s", filename)
			
			if err := runMigration(ctx, db, filename); err != nil {
				return fmt.Errorf("failed to run migration %s: %w", filename, err)
			}

			// Record migration as applied
			if err := recordMigration(ctx, db, filename); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", filename, err)
			}

			log.Printf("Migration %s completed", filename)
		} else {
			log.Printf("Migration %s already applied, skipping", filename)
		}
	}

	return nil
}

func getAppliedMigrations(ctx context.Context, db *database.DB) (map[string]bool, error) {
	query := `SELECT filename FROM migrations ORDER BY applied_at`
	rows, err := db.Query(ctx, query)
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

	return applied, nil
}

func runMigration(ctx context.Context, db *database.DB, filename string) error {
	filepath := filepath.Join("migrations", filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Split content by semicolon to handle multiple statements
	statements := strings.Split(string(content), ";")

	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(statement, "--") {
			continue
		}

		if err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}

func recordMigration(ctx context.Context, db *database.DB, filename string) error {
	query := `INSERT INTO migrations (filename) VALUES ($1)`
	return db.Exec(ctx, query, filename)
} 