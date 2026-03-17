package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

// InitDatabase creates a new agit database at the given path.
// If a valid agit database already exists, it will be left untouched.
// If the file exists but is not a valid agit database, it will be replaced.
func InitDatabase(dbPath string) error {
	// Check if a valid agit database already exists
	if exists, err := DatabaseExists(dbPath); err != nil {
		return fmt.Errorf("failed to check existing database: %w", err)
	} else if exists {
		return nil // Database already exists and is valid
	}

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Remove existing file if it exists (and is not a valid agit database)
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("failed to remove invalid database file: %w", err)
		}
	}

	// Open database connection with foreign keys enabled
	db, err := sql.Open("sqlite3", dbPath+"?foreign_keys=on")
	if err != nil {
		return fmt.Errorf("failed to open database %s: %w", dbPath, err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database %s: %w", dbPath, err)
	}

	// Apply embedded schema
	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to apply schema to database %s: %w", dbPath, err)
	}

	return nil
}

// DatabaseExists checks if an agit database exists at the given path.
func DatabaseExists(dbPath string) (bool, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	// Try to open and verify it's a valid SQLite database
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return false, nil // Not a valid SQLite database
	}
	defer db.Close()

	// Check if required tables exist
	requiredTables := []string{"workspaces", "tasks", "checkpoints", "traces", "templates"}
	for _, table := range requiredTables {
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
		var name string
		if err := row.Scan(&name); err != nil {
			return false, nil // Missing required table
		}
	}

	return true, nil
}
