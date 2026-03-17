package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agit/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestTraceTool(t *testing.T) {
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

	// Create a task
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'done', ?)`,
		ws.Branch, "Task with traces", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = sqlDB.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", ws.Branch).Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Insert three test traces
	_, err = sqlDB.Exec(`
		INSERT INTO traces (task_id, exit_code, output, duration_ms, run_at)
		VALUES 
			(?, 0, 'first run', 100, ?),
			(?, 1, 'second run', 200, ?),
			(?, 0, 'third run', 300, ?)
	`, taskID, time.Now().Add(-2*time.Hour), taskID, time.Now().Add(-1*time.Hour), taskID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test traces: %v", err)
	}

	// Call TraceTool with limit 2
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "trace",
			Arguments: map[string]interface{}{
				"task_id": int(taskID),
				"limit":   2,
			},
		},
	}
	result, err := TraceTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TraceTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if result.IsError {
		t.Fatalf("TraceTool returned error: %s", text)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, text: %q", err, text)
	}
	if resp["task_id"] != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, resp["task_id"])
	}
	if resp["limit"] != float64(2) {
		t.Errorf("Expected limit 2, got %v", resp["limit"])
	}
	if resp["count"] != float64(2) {
		t.Errorf("Expected count 2, got %v", resp["count"])
	}
	tracesRaw, ok := resp["traces"].([]interface{})
	if !ok || len(tracesRaw) != 2 {
		t.Fatalf("Expected 2 traces in response, got %v", tracesRaw)
	}
	// Verify first trace is most recent (third run)
	firstTrace := tracesRaw[0].(map[string]interface{})
	if firstTrace["output"] != "third run" {
		t.Errorf("Expected first trace output 'third run', got %v", firstTrace["output"])
	}
	if firstTrace["exit_code"] != float64(0) {
		t.Errorf("Expected first trace exit_code 0, got %v", firstTrace["exit_code"])
	}
	// Verify second trace is second run
	secondTrace := tracesRaw[1].(map[string]interface{})
	if secondTrace["output"] != "second run" {
		t.Errorf("Expected second trace output 'second run', got %v", secondTrace["output"])
	}
	if secondTrace["exit_code"] != float64(1) {
		t.Errorf("Expected second trace exit_code 1, got %v", secondTrace["exit_code"])
	}
}

func TestTraceToolWithoutLimit(t *testing.T) {
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
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'active', ?)`,
		ws.Branch, "Another task", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = sqlDB.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", ws.Branch).Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}
	// Insert one trace
	_, err = sqlDB.Exec(`INSERT INTO traces (task_id, exit_code, output, run_at) VALUES (?, 0, 'only run', ?)`,
		taskID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert trace: %v", err)
	}

	// Call TraceTool without limit (default 0 = all)
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "trace",
			Arguments: map[string]interface{}{
				"task_id": int(taskID),
			},
		},
	}
	result, err := TraceTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TraceTool failed: %v", err)
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if result.IsError {
		t.Fatalf("TraceTool returned error: %s", text)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, text: %q", err, text)
	}
	if resp["limit"] != float64(0) {
		t.Errorf("Expected limit 0 (default), got %v", resp["limit"])
	}
	if resp["count"] != float64(1) {
		t.Errorf("Expected count 1, got %v", resp["count"])
	}
	tracesRaw := resp["traces"].([]interface{})
	if len(tracesRaw) != 1 {
		t.Fatalf("Expected 1 trace, got %d", len(tracesRaw))
	}
	trace := tracesRaw[0].(map[string]interface{})
	if trace["output"] != "only run" {
		t.Errorf("Expected trace output 'only run', got %v", trace["output"])
	}
}

func TestTraceToolMissingTaskID(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "trace",
			Arguments: map[string]interface{}{}, // missing task_id
		},
	}
	result, err := TraceTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TraceTool should handle missing param gracefully, got error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected IsError true for missing task_id")
	}
}

func TestTraceToolEmptyTraces(t *testing.T) {
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
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'pending', ?)`,
		ws.Branch, "No traces task", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = sqlDB.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", ws.Branch).Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Call TraceTool for task with no traces
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "trace",
			Arguments: map[string]interface{}{
				"task_id": int(taskID),
			},
		},
	}
	result, err := TraceTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TraceTool failed: %v", err)
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if result.IsError {
		t.Fatalf("TraceTool returned error: %s", text)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, text: %q", err, text)
	}
	if resp["count"] != float64(0) {
		t.Errorf("Expected count 0, got %v", resp["count"])
	}
	tracesRaw := resp["traces"].([]interface{})
	if len(tracesRaw) != 0 {
		t.Errorf("Expected empty traces slice, got %d", len(tracesRaw))
	}
}

func TestTraceToolFailedOnly(t *testing.T) {
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

	// Create a task
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`INSERT INTO tasks (workspace_branch, description, status, created_at) VALUES (?, ?, 'done', ?)`,
		ws.Branch, "Task with mixed traces", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}
	var taskID int64
	err = sqlDB.QueryRow("SELECT id FROM tasks WHERE workspace_branch = ?", ws.Branch).Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}

	// Insert three test traces: exit_code 0 (success), 1 (fail), 2 (fail)
	_, err = sqlDB.Exec(`
		INSERT INTO traces (task_id, exit_code, output, duration_ms, run_at)
		VALUES 
			(?, 0, 'success run', 100, ?),
			(?, 1, 'fail run 1', 200, ?),
			(?, 2, 'fail run 2', 300, ?)
	`, taskID, time.Now().Add(-2*time.Hour), taskID, time.Now().Add(-1*time.Hour), taskID, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test traces: %v", err)
	}

	// Call TraceTool with failed_only true
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "trace",
			Arguments: map[string]interface{}{
				"task_id":     int(taskID),
				"failed_only": true,
			},
		},
	}
	result, err := TraceTool(context.Background(), req)
	if err != nil {
		t.Fatalf("TraceTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if result.IsError {
		t.Fatalf("TraceTool returned error: %s", text)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, text: %q", err, text)
	}
	if resp["task_id"] != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, resp["task_id"])
	}
	if resp["failed_only"] != true {
		t.Errorf("Expected failed_only true, got %v", resp["failed_only"])
	}
	if resp["count"] != float64(2) {
		t.Errorf("Expected count 2, got %v", resp["count"])
	}
	tracesRaw, ok := resp["traces"].([]interface{})
	if !ok || len(tracesRaw) != 2 {
		t.Fatalf("Expected 2 traces in response, got %v", tracesRaw)
	}
	// Verify both traces have exit_code != 0
	for i, tr := range tracesRaw {
		trace := tr.(map[string]interface{})
		if trace["exit_code"] == float64(0) {
			t.Errorf("Trace %d has exit_code 0, expected non-zero", i)
		}
	}
}
