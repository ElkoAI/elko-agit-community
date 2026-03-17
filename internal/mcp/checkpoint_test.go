package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agit/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestCheckpointTool(t *testing.T) {
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

	// Create a change to commit
	changeFile := filepath.Join(repoDir, "change.txt")
	if err := os.WriteFile(changeFile, []byte("some change"), 0644); err != nil {
		t.Fatalf("Failed to write change file: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "change.txt"); err != nil {
		t.Fatalf("Failed to git add change: %v", err)
	}

	// Call CheckpointTool with a message
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "checkpoint",
			Arguments: map[string]interface{}{"message": "Test checkpoint"},
		},
	}
	result, err := CheckpointTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CheckpointTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["checkpoint_id"] == nil {
		t.Fatal("Expected checkpoint_id in response")
	}
	if resp["branch"] != ws.Branch {
		t.Errorf("Expected branch %q, got %v", ws.Branch, resp["branch"])
	}
	if resp["message"] != "Test checkpoint" {
		t.Errorf("Expected message 'Test checkpoint', got %v", resp["message"])
	}
	commitHash, ok := resp["commit_hash"].(string)
	if !ok || commitHash == "" {
		t.Error("Expected non-empty commit_hash")
	}

	// Verify checkpoint exists in database
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	var cpID int64
	var cpCommitHash, cpMessage string
	err = sqlDB.QueryRow(`SELECT id, commit_hash, message FROM checkpoints WHERE workspace_branch = ?`, ws.Branch).
		Scan(&cpID, &cpCommitHash, &cpMessage)
	if err != nil {
		t.Fatalf("Failed to query checkpoint: %v", err)
	}
	if cpCommitHash != commitHash {
		t.Errorf("Database commit_hash %q doesn't match response %q", cpCommitHash, commitHash)
	}
	if cpMessage != "Test checkpoint" {
		t.Errorf("Database message %q, expected %q", cpMessage, "Test checkpoint")
	}

	// Verify the commit exists in Git
	if err := runGitCmd(repoDir, "log", "--oneline", "-1"); err != nil {
		t.Fatalf("Failed to verify commit: %v", err)
	}
}

func TestCheckpointToolMissingMessage(t *testing.T) {
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
			Name:      "checkpoint",
			Arguments: map[string]interface{}{}, // missing message
		},
	}
	result, err := CheckpointTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CheckpointTool should handle missing param gracefully, got error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected IsError true for missing message")
	}
}

func TestRestoreTool(t *testing.T) {
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

	// Create a checkpoint with a real commit
	// First, make a change and commit it via CheckpointTool
	changeFile := filepath.Join(repoDir, "change.txt")
	if err := os.WriteFile(changeFile, []byte("version 1"), 0644); err != nil {
		t.Fatalf("Failed to write change file: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "change.txt"); err != nil {
		t.Fatalf("Failed to git add change: %v", err)
	}
	reqCheckpoint := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "checkpoint",
			Arguments: map[string]interface{}{"message": "Checkpoint with change"},
		},
	}
	resultCheckpoint, err := CheckpointTool(context.Background(), reqCheckpoint)
	if err != nil {
		t.Fatalf("CheckpointTool failed: %v", err)
	}
	textCheckpoint := mcp.GetTextFromContent(resultCheckpoint.Content[0])
	var cpResp map[string]interface{}
	if err := json.Unmarshal([]byte(textCheckpoint), &cpResp); err != nil {
		t.Fatalf("Failed to unmarshal checkpoint response: %v", err)
	}
	commitHash, ok := cpResp["commit_hash"].(string)
	if !ok || commitHash == "" {
		t.Fatal("Expected non-empty commit_hash from checkpoint")
	}

	// Now modify the file (simulating work after checkpoint)
	if err := os.WriteFile(changeFile, []byte("version 2"), 0644); err != nil {
		t.Fatalf("Failed to modify change file: %v", err)
	}
	if err := runGitCmd(repoDir, "add", "change.txt"); err != nil {
		t.Fatalf("Failed to git add modified change: %v", err)
	}
	if err := runGitCmd(repoDir, "commit", "-m", "later change"); err != nil {
		t.Fatalf("Failed to commit later change: %v", err)
	}
	// Verify file contains version 2
	content, err := os.ReadFile(changeFile)
	if err != nil {
		t.Fatalf("Failed to read change file: %v", err)
	}
	if string(content) != "version 2" {
		t.Fatalf("Expected file content 'version 2', got %q", string(content))
	}

	// Call RestoreTool to go back to checkpoint
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "restore",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := RestoreTool(context.Background(), req)
	if err != nil {
		t.Fatalf("RestoreTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["checkpoint_id"] == nil {
		t.Fatal("Expected checkpoint_id in response")
	}
	if resp["branch"] != ws.Branch {
		t.Errorf("Expected branch %q, got %v", ws.Branch, resp["branch"])
	}
	if resp["commit_hash"] != commitHash {
		t.Errorf("Expected commit_hash %q, got %v", commitHash, resp["commit_hash"])
	}

	// Verify the file reverted to version 1 (git reset hard)
	contentAfter, err := os.ReadFile(changeFile)
	if err != nil {
		t.Fatalf("Failed to read change file after restore: %v", err)
	}
	if string(contentAfter) != "version 1" {
		t.Errorf("Expected file content 'version 1' after restore, got %q", string(contentAfter))
	}
}

func TestRestoreToolNoCheckpoint(t *testing.T) {
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
			Name:      "restore",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := RestoreTool(context.Background(), req)
	if err != nil {
		t.Fatalf("RestoreTool should handle missing checkpoint gracefully, got error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected IsError true when no checkpoint exists")
	}
}

func TestCheckpointToolNoStagedChanges(t *testing.T) {
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

	// No staged changes, git commit will fail
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "checkpoint",
			Arguments: map[string]interface{}{"message": "empty commit"},
		},
	}
	result, err := CheckpointTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CheckpointTool failed: %v", err)
	}
	if result.IsError {
		t.Error("Expected IsError false when git commit succeeds with --allow-empty")
	}
	// Verify checkpoint created
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp["checkpoint_id"] == nil {
		t.Fatal("Expected checkpoint_id in response")
	}
	if resp["branch"] == nil {
		t.Error("Expected branch in response")
	}
	if resp["commit_hash"] == nil {
		t.Error("Expected commit_hash in response")
	}
	if resp["message"] != "empty commit" {
		t.Errorf("Expected message 'empty commit', got %v", resp["message"])
	}
}
