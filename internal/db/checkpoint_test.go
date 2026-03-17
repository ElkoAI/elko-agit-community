package db

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Create a workspace first
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

	// Create checkpoint with minimal agent state
	agentState := json.RawMessage(`{"todo": "test"}`)
	checkpointID, err := CreateCheckpoint(dbPath, "feat/test", "def456", "Test checkpoint", agentState)
	if err != nil {
		t.Fatalf("CreateCheckpoint failed: %v", err)
	}
	if checkpointID <= 0 {
		t.Errorf("Expected positive checkpoint ID, got %d", checkpointID)
	}

	// Verify checkpoint exists and has correct fields
	var commitHash, message string
	var storedAgentState sql.NullString
	var createdAt time.Time
	err = db.QueryRow(`SELECT commit_hash, agent_state, message, created_at FROM checkpoints WHERE id = ?`, checkpointID).
		Scan(&commitHash, &storedAgentState, &message, &createdAt)
	if err != nil {
		t.Fatalf("Failed to query created checkpoint: %v", err)
	}
	if commitHash != "def456" {
		t.Errorf("Expected commit_hash 'def456', got %q", commitHash)
	}
	if message != "Test checkpoint" {
		t.Errorf("Expected message 'Test checkpoint', got %q", message)
	}
	if !storedAgentState.Valid {
		t.Error("Expected agent_state to be valid")
	} else {
		var stored, expected map[string]interface{}
		if err := json.Unmarshal([]byte(storedAgentState.String), &stored); err != nil {
			t.Errorf("Failed to unmarshal stored agent_state: %v", err)
		}
		if err := json.Unmarshal(agentState, &expected); err != nil {
			t.Errorf("Failed to unmarshal expected agent_state: %v", err)
		}
		if len(stored) != len(expected) || stored["todo"] != expected["todo"] {
			t.Errorf("Agent state mismatch: got %v, want %v", stored, expected)
		}
	}
	if time.Since(createdAt) > 5*time.Second {
		t.Errorf("CreatedAt is too old: %v", createdAt)
	}
}

func TestCreateCheckpointFailsForMissingWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	agentState := json.RawMessage(`{}`)
	_, err := CreateCheckpoint(dbPath, "nonexistent", "abc123", "Test", agentState)
	if err == nil {
		t.Fatal("Expected error when workspace does not exist")
	}
}

func TestGetLatestCheckpoint(t *testing.T) {
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
		"feat/latest", "xyz789", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Insert two checkpoints with different timestamps
	_, err = db.Exec(`
		INSERT INTO checkpoints (workspace_branch, commit_hash, agent_state, message, created_at)
		VALUES 
			('feat/latest', 'commit1', '{"state":"old"}', 'First', ?),
			('feat/latest', 'commit2', '{"state":"new"}', 'Second', ?)
	`, time.Now().Add(-2*time.Hour), time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert test checkpoints: %v", err)
	}

	// Get latest checkpoint
	cp, err := GetLatestCheckpoint(dbPath, "feat/latest")
	if err != nil {
		t.Fatalf("GetLatestCheckpoint failed: %v", err)
	}
	if cp == nil {
		t.Fatal("Expected non-nil checkpoint")
	}
	if cp.CommitHash != "commit2" {
		t.Errorf("Expected latest commit_hash 'commit2', got %q", cp.CommitHash)
	}
	if cp.Message != "Second" {
		t.Errorf("Expected message 'Second', got %q", cp.Message)
	}
	if string(cp.AgentState) != `{"state":"new"}` {
		t.Errorf("Expected agent_state %q, got %q", `{"state":"new"}`, string(cp.AgentState))
	}

	// Get latest for workspace with no checkpoints returns nil, nil
	cp2, err := GetLatestCheckpoint(dbPath, "missing")
	if err != nil {
		t.Fatalf("GetLatestCheckpoint for missing workspace should not error, got: %v", err)
	}
	if cp2 != nil {
		t.Errorf("Expected nil checkpoint for missing workspace, got %v", cp2)
	}
}

func TestGetWorkspaceSnapshot(t *testing.T) {
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
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at, current_task_id, todo_list) VALUES (?, ?, ?, ?, ?, ?)`,
		"feat/snap", "abc123", time.Now(), time.Now(), 42, "## Todo\n- item")
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	snapshot, err := GetWorkspaceSnapshot(dbPath, "feat/snap")
	if err != nil {
		t.Fatalf("GetWorkspaceSnapshot failed: %v", err)
	}
	if len(snapshot) == 0 {
		t.Error("Expected non-empty snapshot JSON")
	}
	// Parse to verify structure
	var ws Workspace
	if err := json.Unmarshal(snapshot, &ws); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}
	if ws.Branch != "feat/snap" {
		t.Errorf("Expected branch 'feat/snap', got %q", ws.Branch)
	}
	if !ws.CurrentTaskID.Valid || ws.CurrentTaskID.Int64 != 42 {
		t.Errorf("Expected current_task_id 42, got %v", ws.CurrentTaskID)
	}
	if ws.TodoList != "## Todo\n- item" {
		t.Errorf("Expected todo_list, got %q", ws.TodoList)
	}
	if ws.LastCommit != "abc123" {
		t.Errorf("Expected last_commit 'abc123', got %q", ws.LastCommit)
	}

	// Snapshot for missing workspace should error
	_, err = GetWorkspaceSnapshot(dbPath, "missing")
	if err == nil {
		t.Error("Expected error for missing workspace")
	}
}

func TestRestoreWorkspaceFromSnapshot(t *testing.T) {
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
	// Create workspace with some initial state
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, current_task_id, todo_list, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"feat/restore", "initial", 5, "## Initial\n- task", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Take a snapshot of the workspace
	snapshot, err := GetWorkspaceSnapshot(dbPath, "feat/restore")
	if err != nil {
		t.Fatalf("GetWorkspaceSnapshot failed: %v", err)
	}

	// Modify the workspace (simulate work after snapshot)
	_, err = db.Exec(`UPDATE workspaces SET current_task_id = ?, todo_list = ?, last_commit = ?, updated_at = ? WHERE branch = ?`,
		10, "## Modified\n- new task", "modified", time.Now(), "feat/restore")
	if err != nil {
		t.Fatalf("Failed to modify workspace: %v", err)
	}

	// Restore from snapshot
	if err := RestoreWorkspaceFromSnapshot(dbPath, "feat/restore", snapshot); err != nil {
		t.Fatalf("RestoreWorkspaceFromSnapshot failed: %v", err)
	}

	// Verify workspace fields match original snapshot
	var currentTaskID sql.NullInt64
	var todoList, lastCommit string
	err = db.QueryRow(`SELECT current_task_id, todo_list, last_commit FROM workspaces WHERE branch = ?`, "feat/restore").
		Scan(&currentTaskID, &todoList, &lastCommit)
	if err != nil {
		t.Fatalf("Failed to query restored workspace: %v", err)
	}
	if !currentTaskID.Valid || currentTaskID.Int64 != 5 {
		t.Errorf("Expected current_task_id 5 after restore, got %v", currentTaskID)
	}
	if todoList != "## Initial\n- task" {
		t.Errorf("Expected todo_list '## Initial\\n- task', got %q", todoList)
	}
	if lastCommit != "initial" {
		t.Errorf("Expected last_commit 'initial', got %q", lastCommit)
	}
}
