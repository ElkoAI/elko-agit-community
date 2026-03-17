package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"agit/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestTaskNewTool(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize Git repository
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

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
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	// Create workspace entry
	wsDB, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if wsDB.Branch == "" {
		t.Fatal("Expected non-empty branch")
	}

	// Call TaskNewTool with required description only
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_new",
			Arguments: map[string]interface{}{"description": "Implement feature X"},
		},
	}
	result, err := TaskNewTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskNewTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["task_id"] == nil {
		t.Fatal("Expected task_id in response")
	}
	if resp["description"] != "Implement feature X" {
		t.Errorf("Expected description 'Implement feature X', got %v", resp["description"])
	}
	if resp["status"] != "pending" {
		t.Errorf("Expected status 'pending', got %v", resp["status"])
	}
	if resp["branch"] != wsDB.Branch {
		t.Errorf("Expected branch %q, got %v", wsDB.Branch, resp["branch"])
	}

	// Call TaskNewTool with test_command and parent_task_id
	req2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_new",
			Arguments: map[string]interface{}{
				"description":    "Write tests",
				"test_command":   "go test ./internal/mcp -run TestTaskNewTool",
				"parent_task_id": resp["task_id"],
			},
		},
	}
	result2, err := TaskNewTool(context.Background(), req2)
	if err != nil {
		t.Fatalf("TaskNewTool with optional params failed: %v", err)
	}
	text2 := mcp.GetTextFromContent(result2.Content[0])
	var resp2 map[string]interface{}
	if err := json.Unmarshal([]byte(text2), &resp2); err != nil {
		t.Fatalf("Failed to unmarshal response2: %v", err)
	}
	if resp2["test_command"] != "go test ./internal/mcp -run TestTaskNewTool" {
		t.Errorf("Expected test_command, got %v", resp2["test_command"])
	}
	if resp2["parent_task_id"] != resp["task_id"] {
		t.Errorf("Expected parent_task_id %v, got %v", resp["task_id"], resp2["parent_task_id"])
	}
}

func TestTaskNewToolMissingDescription(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}
	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	db.GetOrCreateWorkspace(dbPath) // ensure workspace exists

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_new",
			Arguments: map[string]interface{}{}, // missing description
		},
	}
	result, err := TaskNewTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskNewTool should handle missing param gracefully, got error: %v", err)
	}
	// Should return error result with isError true
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !result.IsError {
		t.Error("Expected IsError true for missing description")
	}
}

func TestTaskListTool(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Insert tasks directly via database to control statuses
	// Use raw SQL for simplicity
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`
		INSERT INTO tasks (workspace_branch, description, status, test_command, created_at)
		VALUES (?, ?, 'pending', 'go test 1', ?),
		       (?, ?, 'active', 'go test 2', ?),
		       (?, ?, 'done', 'go test 3', ?)
	`, ws.Branch, "Task 1", ws.CreatedAt,
		ws.Branch, "Task 2", ws.CreatedAt,
		ws.Branch, "Task 3", ws.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to insert test tasks: %v", err)
	}

	// List all tasks
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_list",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := TaskListTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskListTool failed: %v", err)
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var tasks []db.Task
	if err := json.Unmarshal([]byte(text), &tasks); err != nil {
		t.Fatalf("Failed to unmarshal tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	// List only pending tasks
	req2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_list",
			Arguments: map[string]interface{}{"status": "pending"},
		},
	}
	result2, err := TaskListTool(context.Background(), req2)
	if err != nil {
		t.Fatalf("TaskListTool with filter failed: %v", err)
	}
	text2 := mcp.GetTextFromContent(result2.Content[0])
	var pending []db.Task
	if err := json.Unmarshal([]byte(text2), &pending); err != nil {
		t.Fatalf("Failed to unmarshal pending tasks: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending task, got %d", len(pending))
	}
	if pending[0].Status != "pending" {
		t.Errorf("Expected status 'pending', got %q", pending[0].Status)
	}
}

func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestTaskStartTool(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create a pending task via database
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	res, err := sqlDB.Exec(`
		INSERT INTO tasks (workspace_branch, description, status, created_at)
		VALUES (?, ?, 'pending', ?)
	`, ws.Branch, "Test task", ws.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	taskID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert id: %v", err)
	}

	// Start the task
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_start",
			Arguments: map[string]interface{}{"task_id": float64(taskID)},
		},
	}
	result, err := TaskStartTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskStartTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["task_id"].(float64) != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, resp["task_id"])
	}
	if resp["status"] != "active" {
		t.Errorf("Expected status 'active', got %v", resp["status"])
	}

	// Verify task status updated in database
	var status string
	err = sqlDB.QueryRow(`SELECT status FROM tasks WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query task: %v", err)
	}
	if status != "active" {
		t.Errorf("Expected database status 'active', got %q", status)
	}

	// Verify workspace current_task_id updated
	var currentTaskID sql.NullInt64
	err = sqlDB.QueryRow(`SELECT current_task_id FROM workspaces WHERE branch = ?`, ws.Branch).Scan(&currentTaskID)
	if err != nil {
		t.Fatalf("Failed to query workspace: %v", err)
	}
	if !currentTaskID.Valid || currentTaskID.Int64 != taskID {
		t.Errorf("Expected workspace current_task_id = %d, got %v", taskID, currentTaskID)
	}

	// Missing task_id should return error
	reqMissing := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_start",
			Arguments: map[string]interface{}{},
		},
	}
	resultMissing, err := TaskStartTool(context.Background(), reqMissing)
	if err != nil {
		t.Fatalf("TaskStartTool should handle missing param gracefully, got error: %v", err)
	}
	if !resultMissing.IsError {
		t.Error("Expected IsError true for missing task_id")
	}
}

func TestTaskDoneTool(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create an active task with workspace current_task_id set
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	res, err := sqlDB.Exec(`
		INSERT INTO tasks (workspace_branch, description, status, created_at)
		VALUES (?, ?, 'active', ?)
	`, ws.Branch, "Test task", ws.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	taskID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert id: %v", err)
	}
	_, err = sqlDB.Exec(`UPDATE workspaces SET current_task_id = ? WHERE branch = ?`, taskID, ws.Branch)
	if err != nil {
		t.Fatalf("Failed to set workspace current_task_id: %v", err)
	}

	// Mark task as done
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_done",
			Arguments: map[string]interface{}{"task_id": float64(taskID)},
		},
	}
	result, err := TaskDoneTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskDoneTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["task_id"].(float64) != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, resp["task_id"])
	}
	if resp["status"] != "done" {
		t.Errorf("Expected status 'done', got %v", resp["status"])
	}

	// Verify task status and completed_at updated in database
	var status string
	var completedAt sql.NullTime
	err = sqlDB.QueryRow(`SELECT status, completed_at FROM tasks WHERE id = ?`, taskID).
		Scan(&status, &completedAt)
	if err != nil {
		t.Fatalf("Failed to query task: %v", err)
	}
	if status != "done" {
		t.Errorf("Expected database status 'done', got %q", status)
	}
	if !completedAt.Valid {
		t.Error("Expected completed_at to be set")
	}

	// Verify workspace current_task_id cleared
	var currentTaskID sql.NullInt64
	err = sqlDB.QueryRow(`SELECT current_task_id FROM workspaces WHERE branch = ?`, ws.Branch).Scan(&currentTaskID)
	if err != nil {
		t.Fatalf("Failed to query workspace: %v", err)
	}
	if currentTaskID.Valid {
		t.Errorf("Expected workspace current_task_id NULL, got %v", currentTaskID)
	}

	// Missing task_id should return error
	reqMissing := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_done",
			Arguments: map[string]interface{}{},
		},
	}
	resultMissing, err := TaskDoneTool(context.Background(), reqMissing)
	if err != nil {
		t.Fatalf("TaskDoneTool should handle missing param gracefully, got error: %v", err)
	}
	if !resultMissing.IsError {
		t.Error("Expected IsError true for missing task_id")
	}
}

func TestTaskDoneToolWithTestCommand(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create a task with test command
	taskID, err := db.CreateTask(dbPath, ws.Branch, "Test task with echo", "echo hello", sql.NullInt64{})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Mark task as active
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`UPDATE workspaces SET current_task_id = ? WHERE branch = ?`, taskID, ws.Branch)
	if err != nil {
		t.Fatalf("Failed to set workspace current_task_id: %v", err)
	}

	// Call TaskDoneTool
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_done",
			Arguments: map[string]interface{}{"task_id": float64(taskID)},
		},
	}
	result, err := TaskDoneTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskDoneTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["task_id"].(float64) != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, resp["task_id"])
	}
	if resp["status"] != "done" {
		t.Errorf("Expected status 'done', got %v", resp["status"])
	}
	if resp["exit_code"].(float64) != 0 {
		t.Errorf("Expected exit_code 0, got %v", resp["exit_code"])
	}
	if resp["output"].(string) != "hello\n" {
		t.Errorf("Expected output 'hello\\n', got %q", resp["output"])
	}
	if resp["duration_ms"].(float64) < 0 {
		t.Errorf("Expected non-negative duration_ms, got %v", resp["duration_ms"])
	}

	// Verify trace created
	var traceCount int
	err = sqlDB.QueryRow(`SELECT COUNT(*) FROM traces WHERE task_id = ?`, taskID).Scan(&traceCount)
	if err != nil {
		t.Fatalf("Failed to query traces: %v", err)
	}
	if traceCount != 1 {
		t.Errorf("Expected 1 trace, got %d", traceCount)
	}

	// Verify test_result column updated
	var testResult sql.NullString
	err = sqlDB.QueryRow(`SELECT test_result FROM tasks WHERE id = ?`, taskID).Scan(&testResult)
	if err != nil {
		t.Fatalf("Failed to query test_result: %v", err)
	}
	if !testResult.Valid {
		t.Error("Expected test_result to be set")
	}
}

func TestTaskBreakdownTool(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create a parent task
	parentID, err := db.CreateTask(dbPath, ws.Branch, "Parent task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("Failed to create parent task: %v", err)
	}

	// Call TaskBreakdownTool with subtasks
	subtasks := []map[string]interface{}{
		{"description": "Subtask 1"},
		{"description": "Subtask 2", "test_command": "go test ./internal/mcp"},
	}
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_breakdown",
			Arguments: map[string]interface{}{
				"parent_task_id": float64(parentID),
				"subtasks":       subtasks,
			},
		},
	}
	result, err := TaskBreakdownTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskBreakdownTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["parent_task_id"].(float64) != float64(parentID) {
		t.Errorf("Expected parent_task_id %d, got %v", parentID, resp["parent_task_id"])
	}
	subtaskIDsRaw, ok := resp["subtask_ids"].([]interface{})
	if !ok {
		t.Fatalf("Expected subtask_ids array, got %T", resp["subtask_ids"])
	}
	if len(subtaskIDsRaw) != 2 {
		t.Errorf("Expected 2 subtask IDs, got %d", len(subtaskIDsRaw))
	}
	subtaskIDs := make([]int64, 2)
	for i, id := range subtaskIDsRaw {
		subtaskIDs[i] = int64(id.(float64))
	}

	// Verify subtasks created in database
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	for i, subtaskID := range subtaskIDs {
		var description, testCommand sql.NullString
		var parentIDDB sql.NullInt64
		var status string
		err := sqlDB.QueryRow(`
			SELECT description, test_command, parent_task_id, status
			FROM tasks WHERE id = ?`, subtaskID).
			Scan(&description, &testCommand, &parentIDDB, &status)
		if err != nil {
			t.Fatalf("Failed to query subtask %d: %v", i+1, err)
		}
		if description.String != subtasks[i]["description"] {
			t.Errorf("Subtask %d description mismatch: expected %q, got %q",
				i+1, subtasks[i]["description"], description.String)
		}
		if parentIDDB.Int64 != parentID {
			t.Errorf("Subtask %d parent_task_id mismatch: expected %d, got %d",
				i+1, parentID, parentIDDB.Int64)
		}
		if status != "pending" {
			t.Errorf("Subtask %d status mismatch: expected 'pending', got %q", i+1, status)
		}
		// optional test_command
		if tc, ok := subtasks[i]["test_command"]; ok {
			if testCommand.String != tc {
				t.Errorf("Subtask %d test_command mismatch: expected %q, got %q",
					i+1, tc, testCommand.String)
			}
		}

	}

	// Error case: missing parent_task_id
	reqMissingParent := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_breakdown",
			Arguments: map[string]interface{}{"subtasks": subtasks},
		},
	}
	resultMissingParent, err := TaskBreakdownTool(context.Background(), reqMissingParent)
	if err != nil {
		t.Fatalf("TaskBreakdownTool should handle missing parent_task_id gracefully, got error: %v", err)
	}
	if !resultMissingParent.IsError {
		t.Error("Expected IsError true for missing parent_task_id")
	}

	// Error case: missing subtasks
	reqMissingSubtasks := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_breakdown",
			Arguments: map[string]interface{}{"parent_task_id": float64(parentID)},
		},
	}
	resultMissingSubtasks, err := TaskBreakdownTool(context.Background(), reqMissingSubtasks)
	if err != nil {
		t.Fatalf("TaskBreakdownTool should handle missing subtasks gracefully, got error: %v", err)
	}
	if !resultMissingSubtasks.IsError {
		t.Error("Expected IsError true for missing subtasks")
	}

	// Error case: empty subtasks array
	reqEmptySubtasks := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_breakdown",
			Arguments: map[string]interface{}{
				"parent_task_id": float64(parentID),
				"subtasks":       []map[string]interface{}{},
			},
		},
	}
	resultEmptySubtasks, err := TaskBreakdownTool(context.Background(), reqEmptySubtasks)
	if err != nil {
		t.Fatalf("TaskBreakdownTool should handle empty subtasks gracefully, got error: %v", err)
	}
	if !resultEmptySubtasks.IsError {
		t.Error("Expected IsError true for empty subtasks")
	}

	// Error case: subtask missing description
	reqBadSubtasks := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_breakdown",
			Arguments: map[string]interface{}{
				"parent_task_id": float64(parentID),
				"subtasks": []map[string]interface{}{
					{"test_command": "go test"},
				},
			},
		},
	}
	resultBadSubtasks, err := TaskBreakdownTool(context.Background(), reqBadSubtasks)
	if err != nil {
		t.Fatalf("TaskBreakdownTool should handle missing description gracefully, got error: %v", err)
	}
	if !resultBadSubtasks.IsError {
		t.Error("Expected IsError true for subtask missing description")
	}

	// Error case: parent task not found
	reqParentNotFound := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_breakdown",
			Arguments: map[string]interface{}{
				"parent_task_id": float64(9999),
				"subtasks":       subtasks,
			},
		},
	}
	resultParentNotFound, err := TaskBreakdownTool(context.Background(), reqParentNotFound)
	if err != nil {
		t.Fatalf("TaskBreakdownTool should handle nonexistent parent gracefully, got error: %v", err)
	}
	if !resultParentNotFound.IsError {
		t.Error("Expected IsError true for nonexistent parent")
	}
}

func TestTaskTreeTool(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create tasks with parent-child relationships
	// root1 (no parent)
	root1ID, err := db.CreateTask(dbPath, ws.Branch, "Root task 1", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask root1 failed: %v", err)
	}
	// root2 (no parent)
	root2ID, err := db.CreateTask(dbPath, ws.Branch, "Root task 2", "go test ./...", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask root2 failed: %v", err)
	}
	// child1 of root1
	child1ID, err := db.CreateTask(dbPath, ws.Branch, "Child task 1", "", sql.NullInt64{Int64: root1ID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask child1 failed: %v", err)
	}
	// child2 of root1
	child2ID, err := db.CreateTask(dbPath, ws.Branch, "Child task 2", "echo test", sql.NullInt64{Int64: root1ID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask child2 failed: %v", err)
	}
	// grandchild of child1
	grandchildID, err := db.CreateTask(dbPath, ws.Branch, "Grandchild task", "", sql.NullInt64{Int64: child1ID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask grandchild failed: %v", err)
	}

	// Call TaskTreeTool
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_tree",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := TaskTreeTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskTreeTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var tree []*db.TaskTreeNode
	if err := json.Unmarshal([]byte(text), &tree); err != nil {
		t.Fatalf("Failed to unmarshal tree: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("Expected 2 root nodes, got %d", len(tree))
	}

	// Find root1 and root2
	var root1, root2 *db.TaskTreeNode
	for i := range tree {
		if tree[i].ID == root1ID {
			root1 = tree[i]
		} else if tree[i].ID == root2ID {
			root2 = tree[i]
		}
	}
	if root1 == nil {
		t.Fatal("root1 not found in tree")
	}
	if root2 == nil {
		t.Fatal("root2 not found in tree")
	}

	// Verify root1 fields
	if root1.Description != "Root task 1" {
		t.Errorf("root1 description mismatch: expected %q, got %q", "Root task 1", root1.Description)
	}
	if root1.Status != "pending" {
		t.Errorf("root1 status mismatch: expected 'pending', got %q", root1.Status)
	}
	if root1.TestCommand.Valid {
		t.Error("root1 test_command should be empty")
	}
	if root1.ParentTaskID.Valid {
		t.Error("root1 parent_task_id should be invalid")
	}
	// Verify root1 children
	if len(root1.Children) != 2 {
		t.Fatalf("root1 expected 2 children, got %d", len(root1.Children))
	}
	var child1, child2 *db.TaskTreeNode
	for i := range root1.Children {
		if root1.Children[i].ID == child1ID {
			child1 = root1.Children[i]
		} else if root1.Children[i].ID == child2ID {
			child2 = root1.Children[i]
		}
	}
	if child1 == nil {
		t.Fatal("child1 not found under root1")
	}
	if child2 == nil {
		t.Fatal("child2 not found under root1")
	}
	if child1.Description != "Child task 1" {
		t.Errorf("child1 description mismatch")
	}
	if child2.Description != "Child task 2" {
		t.Errorf("child2 description mismatch")
	}
	if child2.TestCommand.String != "echo test" {
		t.Errorf("child2 test_command mismatch: expected %q, got %q", "echo test", child2.TestCommand.String)
	}
	// Verify child1 has grandchild
	if len(child1.Children) != 1 {
		t.Fatalf("child1 expected 1 grandchild, got %d", len(child1.Children))
	}
	if child1.Children[0].ID != grandchildID {
		t.Errorf("grandchild ID mismatch")
	}
	if child1.Children[0].Description != "Grandchild task" {
		t.Errorf("grandchild description mismatch")
	}

	// Verify root2 has no children
	if len(root2.Children) != 0 {
		t.Errorf("root2 expected no children, got %d", len(root2.Children))
	}
	if root2.TestCommand.String != "go test ./..." {
		t.Errorf("root2 test_command mismatch")
	}
}

func TestTaskDoneToolWithFailingTestCommand(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}
	if err := runGitCmd(repoDir, "init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGitCmd(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create a task with failing test command
	taskID, err := db.CreateTask(dbPath, ws.Branch, "Test task with false", "false", sql.NullInt64{})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Mark task as active
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`UPDATE workspaces SET current_task_id = ? WHERE branch = ?`, taskID, ws.Branch)
	if err != nil {
		t.Fatalf("Failed to set workspace current_task_id: %v", err)
	}

	// Call TaskDoneTool
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task_done",
			Arguments: map[string]interface{}{"task_id": float64(taskID)},
		},
	}
	result, err := TaskDoneTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskDoneTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["task_id"].(float64) != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, resp["task_id"])
	}
	if resp["status"] != "failed" {
		t.Errorf("Expected status 'failed', got %v", resp["status"])
	}
	// exit_code should be non-zero (false returns 1)
	if resp["exit_code"].(float64) == 0 {
		t.Errorf("Expected non-zero exit_code, got %v", resp["exit_code"])
	}
	// output may be empty
	if resp["duration_ms"].(float64) < 0 {
		t.Errorf("Expected non-negative duration_ms, got %v", resp["duration_ms"])
	}

	// Verify trace created
	var traceCount int
	err = sqlDB.QueryRow(`SELECT COUNT(*) FROM traces WHERE task_id = ?`, taskID).Scan(&traceCount)
	if err != nil {
		t.Fatalf("Failed to query traces: %v", err)
	}
	if traceCount != 1 {
		t.Errorf("Expected 1 trace, got %d", traceCount)
	}

	// Verify test_result column updated
	var testResult sql.NullString
	err = sqlDB.QueryRow(`SELECT test_result FROM tasks WHERE id = ?`, taskID).Scan(&testResult)
	if err != nil {
		t.Fatalf("Failed to query test_result: %v", err)
	}
	if !testResult.Valid {
		t.Error("Expected test_result to be set")
	}
}
