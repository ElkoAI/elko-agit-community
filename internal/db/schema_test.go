package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSchemaLoadsSuccessfully(t *testing.T) {
	// Read schema file
	schemaPath := filepath.Join("schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	schema := string(schemaBytes)
	if strings.TrimSpace(schema) == "" {
		t.Fatal("Schema file is empty")
	}

	// Verify schema contains expected tables
	expectedTables := []string{"workspaces", "tasks", "checkpoints", "traces"}
	for _, table := range expectedTables {
		if !strings.Contains(strings.ToLower(schema), strings.ToLower(table)) {
			t.Errorf("Schema missing table: %s", table)
		}
	}
}

func TestSchemaCanBeApplied(t *testing.T) {
	// Create in-memory SQLite database with foreign keys enabled
	db, err := sql.Open("sqlite3", ":memory:?foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Read schema file
	schemaPath := filepath.Join("schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	// Execute entire schema (SQLite can handle multiple statements)
	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		t.Fatalf("Failed to execute schema: %v", err)
	}

	// Verify tables were created
	expectedTables := []string{"workspaces", "tasks", "checkpoints", "traces"}
	for _, table := range expectedTables {
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
		var name string
		if err := row.Scan(&name); err != nil {
			t.Errorf("Table %s was not created: %v", table, err)
		}
	}

	// Verify indexes were created
	expectedIndexes := []string{
		"idx_workspaces_branch",
		"idx_tasks_workspace",
		"idx_tasks_status",
		"idx_checkpoints_workspace",
		"idx_checkpoints_commit",
		"idx_traces_task",
		"idx_traces_run_at",
	}
	for _, idx := range expectedIndexes {
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx)
		var name string
		if err := row.Scan(&name); err != nil {
			t.Errorf("Index %s was not created: %v", idx, err)
		}
	}
}
