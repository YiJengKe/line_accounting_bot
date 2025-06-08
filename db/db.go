package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"accountingbot/config"
	"accountingbot/logger"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// Init initializes the database connection
func Init(ctx context.Context) {
	// Start tracing span
	ctx, span := logger.StartSpan(ctx, "db.Init")
	defer span.End()

	// Get database connection settings
	cfg := config.Get()
	logger.Info(ctx, "Connecting to database")

	// Create database connection
	var err error
	DB, err = sql.Open("postgres", cfg.Db.PsqlUrl)
	if err != nil {
		logger.Fatal(ctx, "Failed to create database connection", "error", err.Error())
	}

	// Set connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Try to connect
	retries := 5
	for i := range retries {
		err = DB.PingContext(ctx)
		if err == nil {
			break
		}

		logger.Warn(ctx, "Database connection test failed, retrying later",
			"attempt", i+1,
			"max_attempts", retries,
			"error", err.Error(),
		)

		time.Sleep(3 * time.Second)
	}

	if err != nil {
		logger.Fatal(ctx, "Failed to connect to database", "error", err.Error())
	}

	logger.Info(ctx, "Database connection successful")
	createTables(ctx)
}

// generateTestDbName generates a unique database name using timestamp and random suffix
func generateTestDbName(dbName string) string {
	randomSuffix := rand.Intn(1_000_000_000_000)

	return fmt.Sprintf("%s_%010d", dbName, randomSuffix)
}

// Init 初始化資料庫連線
func SetupTestDB(ctx context.Context) string {
	// Start tracing span
	ctx, span := logger.StartSpan(ctx, "db.SetupTestDB")
	defer span.End()

	logger.Info(ctx, "Connecting to database")

	// Create database connection
	testConnStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	var err error
	DB, err = sql.Open("postgres", testConnStr)
	if err != nil {
		logger.Fatal(ctx, "Failed to create database connection", "error", err.Error())
	}

	testDbName := generateTestDbName("accounting")
	_, err = DB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDbName))
	if err != nil {
		logger.Fatal(ctx, "Failed to create test database", "error", err.Error())
	}

	DB.Close()

	logger.Info(ctx, "Connecting to test database")

	testDBUrl := fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", testDbName)
	DB, err = sql.Open("postgres", testDBUrl)
	if err != nil {
		logger.Fatal(ctx, "Failed to create database connection", "error", err.Error())
	}

	// Set connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Try to connect
	retries := 5
	for i := range retries {
		err = DB.PingContext(ctx)
		if err == nil {
			break
		}

		logger.Warn(ctx, "Database connection test failed, retrying later",
			"attempt", i+1,
			"max_attempts", retries,
			"error", err.Error(),
		)

		time.Sleep(3 * time.Second)
	}

	if err != nil {
		logger.Fatal(ctx, "Failed to connect to database", "error", err.Error())
	}

	logger.Info(ctx, "Database connection successful")
	createTables(ctx)

	return testDbName
}

// CleanupTestDB drops the test database and closes the connection
func CleanupTestDB(ctx context.Context, testDbName string) error {
	// Start tracing span
	ctx, span := logger.StartSpan(ctx, "db.CleanupTestDB")
	defer span.End()

	// Close current test DB connection
	if DB != nil {
		DB.Close()
	}

	// Connect to the default postgres database to drop the test database
	connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	adminDB, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Error(ctx, "Failed to connect to admin database for cleanup", "error", err.Error())
		return err
	}
	defer adminDB.Close()

	// Terminate all connections to the test database before dropping (PostgreSQL requirement)
	_, _ = adminDB.ExecContext(ctx, fmt.Sprintf(
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s';`, testDbName,
	))

	// Drop the test database
	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDbName))
	if err != nil {
		logger.Error(ctx, "Failed to drop test database", "test_db", testDbName, "error", err.Error())
		return err
	}

	logger.Info(ctx, "Test database dropped successfully", "test_db", testDbName)
	return nil
}

// createTables creates the required tables
func createTables(ctx context.Context) {
	ctx, span := logger.StartSpan(ctx, "db.createTables")
	defer span.End()

	logger.Info(ctx, "Checking and creating tables")

	query := `
        CREATE TABLE IF NOT EXISTS categories (
            id SERIAL PRIMARY KEY,
            user_id TEXT NOT NULL,
            name TEXT NOT NULL,
            type TEXT NOT NULL,
            UNIQUE(user_id, name)
        );

        CREATE TABLE IF NOT EXISTS transactions (
            id SERIAL PRIMARY KEY,
            user_id TEXT NOT NULL,
            type TEXT NOT NULL,
            amount INTEGER NOT NULL,
            category_id INTEGER NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_category_id
			    FOREIGN KEY (category_id)
			    REFERENCES categories(id)
			    ON DELETE CASCADE
        );
    `

	_, err := DB.ExecContext(ctx, query)
	if err != nil {
		logger.Fatal(ctx, "Failed to create tables", "error", err.Error())
	}

	logger.Info(ctx, "Tables checked/created")
}

// QueryContext executes a query and returns rows
func QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := logger.StartSpan(ctx, "db.query")
	defer span.End()

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Error(ctx, "Query failed", "query", query, "error", err.Error())
	}
	return rows, err
}

// ExecContext executes a command and returns the result
func ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := logger.StartSpan(ctx, "db.exec")
	defer span.End()

	result, err := DB.ExecContext(ctx, query, args...)
	if err != nil {
		logger.Error(ctx, "Execution failed", "query", query, "error", err.Error())
	}
	return result, err
}

// QueryRowContext executes a query and returns a single row
func QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, span := logger.StartSpan(ctx, "db.queryRow")
	defer span.End()

	return DB.QueryRowContext(ctx, query, args...)
}
