package db

import (
	"database/sql"
	"sync"
	"testing"
)

func TestConcurrentCreateTask(t *testing.T) {
	// Setup temporary database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/agit.db"
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	branch := "test-branch"
	commit := "abc123"
	// Create workspace first
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, branch, commit)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Number of concurrent goroutines
	const workers = 10
	var wg sync.WaitGroup
	wg.Add(workers)

	// Channel to collect task IDs
	taskIDs := make(chan int64, workers)
	errors := make(chan error, workers)

	// Launch workers
	for i := 0; i < workers; i++ {
		go func(idx int) {
			defer wg.Done()
			desc := "concurrent task " + string(rune('A'+idx))
			taskID, err := CreateTask(dbPath, ws.Branch, desc, "", sql.NullInt64{})
			if err != nil {
				errors <- err
				return
			}
			taskIDs <- taskID
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()
	close(taskIDs)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("unexpected error: %v", err)
	}

	// Collect unique task IDs
	seen := make(map[int64]bool)
	for id := range taskIDs {
		if seen[id] {
			t.Errorf("duplicate task ID %d", id)
		}
		seen[id] = true
	}

	// Verify that exactly workers tasks were created
	if len(seen) != workers {
		t.Errorf("expected %d tasks, got %d", workers, len(seen))
	}

	// Verify in database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE workspace_branch = ?", ws.Branch).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count tasks: %v", err)
	}
	if count != workers {
		t.Errorf("database count mismatch: expected %d, got %d", workers, count)
	}
}
