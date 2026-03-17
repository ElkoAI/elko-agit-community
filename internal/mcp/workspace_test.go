package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"agit/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestWorkspaceStatusTool(t *testing.T) {
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
	// Set user config for commits
	if err := runGit(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set user email: %v", err)
	}
	if err := runGit(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set user name: %v", err)
	}
	// Create an initial commit
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

	// Change to repo directory and set up agit database
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

	// Call WorkspaceStatusTool with a dummy request
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "workspace_status",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := WorkspaceStatusTool(context.Background(), req)
	if err != nil {
		t.Fatalf("WorkspaceStatusTool failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}
	// Extract text content
	if len(result.Content) == 0 {
		t.Fatal("Expected at least one content item")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	t.Logf("text: %q", text)
	if text == "" {
		t.Fatal("Expected non-empty text content")
	}
	// Parse JSON response
	var ws db.Workspace
	if err := json.Unmarshal([]byte(text), &ws); err != nil {
		t.Fatalf("Failed to unmarshal workspace JSON: %v", err)
	}
	// Verify branch is non-empty (default could be "master" or "main")
	if ws.Branch == "" {
		t.Error("Expected non-empty branch")
	}
	// Verify last_commit is a SHA (40 chars)
	if len(ws.LastCommit) != 40 {
		t.Errorf("Expected commit hash length 40, got %d (%q)", len(ws.LastCommit), ws.LastCommit)
	}
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestWorkspaceStatusToolNoGitRepo(t *testing.T) {
	// Create a temporary directory without a Git repository
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize agit database in a subdirectory (no .git folder)
	dbPath := filepath.Join(tmpDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("Failed to init agit database: %v", err)
	}

	// Call WorkspaceStatusTool - should fail because git commands fail
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "workspace_status",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := WorkspaceStatusTool(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error when git repository not found")
	}
	if result != nil {
		t.Error("Expected nil result when error returned")
	}
}
