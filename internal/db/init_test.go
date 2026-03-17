package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitDatabaseCreatesValidDatabase(t *testing.T) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Initialize database
	err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Database file was not created")
	}

	// Verify database is valid using DatabaseExists
	exists, err := DatabaseExists(dbPath)
	if err != nil {
		t.Fatalf("DatabaseExists failed: %v", err)
	}
	if !exists {
		t.Fatal("DatabaseExists returned false for newly created database")
	}
}

func TestInitDatabaseOverwritesExistingDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Create a dummy file at database path
	dummyContent := []byte("not a sqlite database")
	if err := os.WriteFile(dbPath, dummyContent, 0644); err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	// Initialize database - should overwrite the dummy file
	err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Verify database is now valid
	exists, err := DatabaseExists(dbPath)
	if err != nil {
		t.Fatalf("DatabaseExists failed: %v", err)
	}
	if !exists {
		t.Fatal("DatabaseExists returned false after overwriting dummy file")
	}
}

func TestDatabaseExistsReturnsFalseForMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nonexistent.db")

	exists, err := DatabaseExists(dbPath)
	if err != nil {
		t.Fatalf("DatabaseExists failed: %v", err)
	}
	if exists {
		t.Fatal("DatabaseExists returned true for missing file")
	}
}

func TestDatabaseExistsReturnsFalseForInvalidSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "not-a-db.db")

	// Create a file that's not a valid SQLite database
	dummyContent := []byte("not a sqlite database")
	if err := os.WriteFile(dbPath, dummyContent, 0644); err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	exists, err := DatabaseExists(dbPath)
	if err != nil {
		t.Fatalf("DatabaseExists failed: %v", err)
	}
	if exists {
		t.Fatal("DatabaseExists returned true for invalid SQLite file")
	}
}

func TestDatabaseExistsReturnsFalseForMissingTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "incomplete.db")

	// Create a valid SQLite database but without required tables
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}
	defer db.Close()

	// Create a table that's not one of the required tables
	_, err = db.Exec("CREATE TABLE unrelated (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create unrelated table: %v", err)
	}

	exists, err := DatabaseExists(dbPath)
	if err != nil {
		t.Fatalf("DatabaseExists failed: %v", err)
	}
	if exists {
		t.Fatal("DatabaseExists returned true for database missing required tables")
	}
}
