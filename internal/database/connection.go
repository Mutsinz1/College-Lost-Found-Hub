package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"lostfound/internal/config"
)

type DB struct {
	pool *pgxpool.Pool
}

// NewConnection creates a new database connection pool
func NewConnection(cfg *config.Config) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to database successfully")

	return &DB{pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// GetPool returns the underlying connection pool
func (db *DB) GetPool() *pgxpool.Pool {
	return db.pool
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// Exec executes a query without returning rows
func (db *DB) Exec(ctx context.Context, sql string, arguments ...interface{}) error {
	_, err := db.pool.Exec(ctx, sql, arguments...)
	return err
}

// ExecRows executes a query and returns the number of rows affected
func (db *DB) ExecRows(ctx context.Context, sql string, arguments ...interface{}) (int64, error) {
	tag, err := db.pool.Exec(ctx, sql, arguments...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Query executes a query and returns rows
func (db *DB) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, arguments...)
}

// QueryRow executes a query and returns a single row
func (db *DB) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, arguments...)
}
