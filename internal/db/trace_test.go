package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateTrace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Create workspace and task first
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/test", "abc123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'pending', ?)`,
		"feat/test", "Test task", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = db.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", "feat/test").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Create trace with all fields
	exitCode := 0
	durationMs := int64(150)
	traceID, err := CreateTrace(dbPath, taskID, &exitCode, "Test output", &durationMs)
	if err != nil {
		t.Fatalf("CreateTrace failed: %v", err)
	}
	if traceID <= 0 {
		t.Errorf("Expected positive trace ID, got %d", traceID)
	}

	// Verify trace exists and has correct fields
	var storedExitCode sql.NullInt64
	var storedOutput sql.NullString
	var storedDuration sql.NullInt64
	var runAt time.Time
	err = db.QueryRow(`SELECT exit_code, output, duration_ms, run_at FROM traces WHERE id = ?`, traceID).
		Scan(&storedExitCode, &storedOutput, &storedDuration, &runAt)
	if err != nil {
		t.Fatalf("Failed to query created trace: %v", err)
	}
	if !storedExitCode.Valid || storedExitCode.Int64 != 0 {
		t.Errorf("Expected exit_code 0, got %v", storedExitCode)
	}
	if !storedOutput.Valid || storedOutput.String != "Test output" {
		t.Errorf("Expected output 'Test output', got %v", storedOutput)
	}
	if !storedDuration.Valid || storedDuration.Int64 != 150 {
		t.Errorf("Expected duration_ms 150, got %v", storedDuration)
	}
	if time.Since(runAt) > 5*time.Second {
		t.Errorf("RunAt is too old: %v", runAt)
	}
}

func TestCreateTraceWithNullFields(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/null", "xyz789", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'active', ?)`,
		"feat/null", "Null test", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = db.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", "feat/null").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Create trace with nil exit_code and duration (output empty string)
	traceID, err := CreateTrace(dbPath, taskID, nil, "", nil)
	if err != nil {
		t.Fatalf("CreateTrace with null fields failed: %v", err)
	}

	var exitCodeNull sql.NullInt64
	var outputNull sql.NullString
	var durationNull sql.NullInt64
	err = db.QueryRow(`SELECT exit_code, output, duration_ms FROM traces WHERE id = ?`, traceID).
		Scan(&exitCodeNull, &outputNull, &durationNull)
	if err != nil {
		t.Fatalf("Failed to query trace: %v", err)
	}
	if exitCodeNull.Valid {
		t.Error("Expected exit_code to be NULL")
	}
	if outputNull.Valid {
		t.Error("Expected output to be NULL (empty string becomes NULL)")
	}
	if durationNull.Valid {
		t.Error("Expected duration_ms to be NULL")
	}
}

func TestCreateTraceFailsForMissingTask(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	exitCode := 1
	_, err := CreateTrace(dbPath, 999, &exitCode, "output", nil)
	if err == nil {
		t.Fatal("Expected error when task does not exist")
	}
}

func TestGetTraces(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/traces", "commit1", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'done', ?)`,
		"feat/traces", "Trace test", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = db.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", "feat/traces").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Insert three traces with different timestamps
	_, err = db.Exec(`
		INSERT INTO traces (task_id, exit_code, output, duration_ms, run_at)
		VALUES 
			(?, 0, 'first run', 100, ?),
			(?, 1, 'second run', 200, ?),
			(?, 0, 'third run', 300, ?)
	`, taskID, time.Now().Add(-2*time.Hour), taskID, time.Now().Add(-1*time.Hour), taskID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test traces: %v", err)
	}

	// Get all traces (no limit)
	traces, err := GetTraces(dbPath, taskID, 0)
	if err != nil {
		t.Fatalf("GetTraces failed: %v", err)
	}
	if len(traces) != 3 {
		t.Fatalf("Expected 3 traces, got %d", len(traces))
	}
	// Should be ordered by run_at DESC (most recent first)
	if *traces[0].ExitCode != 0 || traces[0].Output != "third run" || *traces[0].DurationMs != 300 {
		t.Errorf("First trace mismatch: %+v", traces[0])
	}
	if *traces[1].ExitCode != 1 || traces[1].Output != "second run" || *traces[1].DurationMs != 200 {
		t.Errorf("Second trace mismatch: %+v", traces[1])
	}
	if *traces[2].ExitCode != 0 || traces[2].Output != "first run" || *traces[2].DurationMs != 100 {
		t.Errorf("Third trace mismatch: %+v", traces[2])
	}

	// Get limited traces
	limited, err := GetTraces(dbPath, taskID, 2)
	if err != nil {
		t.Fatalf("GetTraces with limit failed: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("Expected 2 traces with limit, got %d", len(limited))
	}
	// Should still be most recent first
	if limited[0].Output != "third run" {
		t.Errorf("Limited first trace output: %s", limited[0].Output)
	}
	if limited[1].Output != "second run" {
		t.Errorf("Limited second trace output: %s", limited[1].Output)
	}

	// Get traces for non‑existent task returns empty slice
	empty, err := GetTraces(dbPath, 999, 0)
	if err != nil {
		t.Fatalf("GetTraces for missing task should not error, got: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Expected empty slice for missing task, got %d traces", len(empty))
	}
}

func TestGetLastTrace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/last", "commit2", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'failed', ?)`,
		"feat/last", "Last trace test", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = db.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", "feat/last").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Insert two traces
	_, err = db.Exec(`
		INSERT INTO traces (task_id, exit_code, output, run_at)
		VALUES (?, 0, 'older', ?), (?, 1, 'newer', ?)
	`, taskID, time.Now().Add(-1*time.Hour), taskID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test traces: %v", err)
	}

	last, err := GetLastTrace(dbPath, taskID)
	if err != nil {
		t.Fatalf("GetLastTrace failed: %v", err)
	}
	if last == nil {
		t.Fatal("Expected non‑nil last trace")
	}
	if last.Output != "newer" || *last.ExitCode != 1 {
		t.Errorf("Expected last trace with output 'newer' exit 1, got %+v", last)
	}

	// No traces returns nil
	lastMissing, err := GetLastTrace(dbPath, 999)
	if err != nil {
		t.Fatalf("GetLastTrace for missing task should not error, got: %v", err)
	}
	if lastMissing != nil {
		t.Errorf("Expected nil for missing task, got %v", lastMissing)
	}
}

func TestGetTracesFailed(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	branch := "feat/traces"
	commit := "abc123"
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, branch, commit)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	taskID, err := CreateTask(dbPath, ws.Branch, "Task with traces", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Create mixed traces: success (0), failure (1), success (0)
	exit0 := 0
	exit1 := 1
	_, err = CreateTrace(dbPath, taskID, &exit0, "success 1", nil)
	if err != nil {
		t.Fatalf("failed to create trace 1: %v", err)
	}
	_, err = CreateTrace(dbPath, taskID, &exit1, "failure 1", nil)
	if err != nil {
		t.Fatalf("failed to create trace 2: %v", err)
	}
	_, err = CreateTrace(dbPath, taskID, &exit0, "success 2", nil)
	if err != nil {
		t.Fatalf("failed to create trace 3: %v", err)
	}

	// Get failed traces only
	failed, err := GetTracesFailed(dbPath, taskID, 10)
	if err != nil {
		t.Fatalf("GetTracesFailed failed: %v", err)
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed trace, got %d", len(failed))
	}
	if failed[0].Output != "failure 1" {
		t.Errorf("expected output 'failure 1', got %q", failed[0].Output)
	}
	if failed[0].ExitCode == nil || *failed[0].ExitCode != 1 {
		t.Errorf("expected exit code 1, got %v", failed[0].ExitCode)
	}

	// Limit zero returns all failed
	failedAll, err := GetTracesFailed(dbPath, taskID, 0)
	if err != nil {
		t.Fatalf("GetTracesFailed with limit 0 failed: %v", err)
	}
	if len(failedAll) != 1 {
		t.Errorf("limit 0 should return all failed, got %d", len(failedAll))
	}

	// No failed traces for another task
	otherID, _ := CreateTask(dbPath, ws.Branch, "Other task", "", sql.NullInt64{})
	none, err := GetTracesFailed(dbPath, otherID, 10)
	if err != nil {
		t.Fatalf("GetTracesFailed for task with no traces failed: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected empty slice, got %d", len(none))
	}
}
