package db

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetOrCreateWorkspaceForBranchCreatesNewWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Initialize database
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Call GetOrCreateWorkspaceForBranch with a new branch
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, "feat/test", "abc123")
	if err != nil {
		t.Fatalf("GetOrCreateWorkspaceForBranch failed: %v", err)
	}

	// Verify workspace fields
	if ws.Branch != "feat/test" {
		t.Errorf("Expected branch 'feat/test', got %q", ws.Branch)
	}
	if ws.LastCommit != "abc123" {
		t.Errorf("Expected last_commit 'abc123', got %q", ws.LastCommit)
	}
	if ws.TodoList != "" {
		t.Errorf("Expected empty todo_list, got %q", ws.TodoList)
	}
	if ws.AgentState != nil {
		t.Errorf("Expected nil agent_state, got %v", ws.AgentState)
	}
	if !ws.CurrentTaskID.Valid {
		// Should be null
	}
	// CreatedAt and UpdatedAt should be recent
	if time.Since(ws.CreatedAt) > 5*time.Second {
		t.Errorf("CreatedAt is too old: %v", ws.CreatedAt)
	}
	if time.Since(ws.UpdatedAt) > 5*time.Second {
		t.Errorf("UpdatedAt is too old: %v", ws.UpdatedAt)
	}
}

func TestGetOrCreateWorkspaceForBranchRetrievesExistingWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Initialize database
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// First call creates the workspace
	ws1, err := GetOrCreateWorkspaceForBranch(dbPath, "main", "commit1")
	if err != nil {
		t.Fatalf("GetOrCreateWorkspaceForBranch failed: %v", err)
	}

	// Second call should retrieve the same workspace
	ws2, err := GetOrCreateWorkspaceForBranch(dbPath, "main", "commit2")
	if err != nil {
		t.Fatalf("GetOrCreateWorkspaceForBranch failed: %v", err)
	}

	// Branch should match
	if ws2.Branch != "main" {
		t.Errorf("Expected branch 'main', got %q", ws2.Branch)
	}
	// Last commit should still be the original (not updated)
	if ws2.LastCommit != "commit1" {
		t.Errorf("Expected last_commit 'commit1', got %q", ws2.LastCommit)
	}
	// CreatedAt should be identical
	if !ws1.CreatedAt.Equal(ws2.CreatedAt) {
		t.Errorf("CreatedAt mismatch: %v vs %v", ws1.CreatedAt, ws2.CreatedAt)
	}
	// UpdatedAt should be identical (no updates)
	if !ws1.UpdatedAt.Equal(ws2.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: %v vs %v", ws1.UpdatedAt, ws2.UpdatedAt)
	}
}

func TestGetOrCreateWorkspaceForBranchUpdatesNullableFields(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Initialize database
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Insert a workspace with agent_state and todo_list
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	agentState := `{"todo": ["item1"]}`
	todoList := "- [ ] item1\n- [x] item2"
	_, err = db.Exec(`
		INSERT INTO workspaces (branch, current_task_id, agent_state, todo_list, last_commit, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "feat/with-data", 42, agentState, todoList, "def456", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Retrieve workspace
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, "feat/with-data", "newcommit")
	if err != nil {
		t.Fatalf("GetOrCreateWorkspaceForBranch failed: %v", err)
	}

	// Verify fields
	if ws.Branch != "feat/with-data" {
		t.Errorf("Expected branch 'feat/with-data', got %q", ws.Branch)
	}
	if ws.LastCommit != "def456" {
		t.Errorf("Expected last_commit 'def456', got %q", ws.LastCommit)
	}
	if !ws.CurrentTaskID.Valid || ws.CurrentTaskID.Int64 != 42 {
		t.Errorf("Expected current_task_id 42, got %v", ws.CurrentTaskID)
	}
	if ws.TodoList != todoList {
		t.Errorf("Expected todo_list %q, got %q", todoList, ws.TodoList)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(ws.AgentState, &decoded); err != nil {
		t.Errorf("Failed to decode agent_state: %v", err)
	}
	if decoded["todo"] == nil {
		t.Errorf("Expected agent_state.todo field")
	}
}

func TestGetOrCreateWorkspaceForBranchHandlesMissingDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "missing.db")

	// Do not initialize database; call GetOrCreateWorkspaceForBranch directly
	_, err := GetOrCreateWorkspaceForBranch(dbPath, "main", "abc")
	if err == nil {
		t.Fatal("Expected error when database does not exist")
	}
	// Should get SQLite error about missing table
}

func TestGetOrCreateWorkspaceForBranchHandlesEmptyBranch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Empty branch is allowed at database level (primary key can be empty string)
	// but git will never return empty branch, so this is a boundary case.
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, "", "abc")
	if err != nil {
		t.Fatalf("GetOrCreateWorkspaceForBranch failed: %v", err)
	}
	if ws.Branch != "" {
		t.Errorf("Expected empty branch, got %q", ws.Branch)
	}
	if ws.LastCommit != "abc" {
		t.Errorf("Expected last_commit 'abc', got %q", ws.LastCommit)
	}
}

func TestGetOrCreateWorkspaceForBranchHandlesEmptyCommit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Empty commit is allowed (could be a new repository with no commits)
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, "main", "")
	if err != nil {
		t.Fatalf("GetOrCreateWorkspaceForBranch failed: %v", err)
	}
	if ws.LastCommit != "" {
		t.Errorf("Expected empty last_commit, got %q", ws.LastCommit)
	}
}

func TestGetOrCreateWorkspace(t *testing.T) {
	// Create a temporary directory and initialize a Git repository
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize Git repository
	if err := runGit(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGit(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGit(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGit(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGit(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Change to repo directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	// Initialize agit database
	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}

	// Call GetOrCreateWorkspace
	ws, err := GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("GetOrCreateWorkspace failed: %v", err)
	}

	// Verify workspace fields
	if ws.Branch == "" {
		t.Error("Expected non-empty branch")
	}
	if ws.LastCommit == "" {
		t.Error("Expected non-empty last_commit")
	}
	if len(ws.LastCommit) != 40 {
		t.Errorf("Expected commit hash length 40, got %d (%q)", len(ws.LastCommit), ws.LastCommit)
	}
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}
