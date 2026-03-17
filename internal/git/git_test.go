package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}
	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git user email: %v", err)
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git user name: %v", err)
	}
	// Create a dummy file and commit
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}
	return repoDir
}

func TestGitRoot(t *testing.T) {
	repoDir := setupGitRepo(t)
	// Change to subdirectory
	subDir := filepath.Join(repoDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to change to subdir: %v", err)
	}
	root, err := GitRoot()
	if err != nil {
		t.Fatalf("GitRoot failed: %v", err)
	}
	if root != repoDir {
		t.Errorf("GitRoot expected %q, got %q", repoDir, root)
	}
}

func TestIsGitRepository(t *testing.T) {
	repoDir := setupGitRepo(t)
	// Inside repo
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change to repo: %v", err)
	}
	if !IsGitRepository() {
		t.Error("IsGitRepository should return true inside git repo")
	}
	// Outside repo (temp directory)
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	if IsGitRepository() {
		t.Error("IsGitRepository should return false outside git repo")
	}
}

func TestCurrentBranch(t *testing.T) {
	repoDir := setupGitRepo(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change to repo: %v", err)
	}
	branch, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch failed: %v", err)
	}
	if branch != "master" && branch != "main" {
		t.Errorf("Expected branch master or main, got %q", branch)
	}
	// Create and switch to new branch
	cmd := exec.Command("git", "checkout", "-b", "test-branch")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	branch, err = CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch after checkout failed: %v", err)
	}
	if branch != "test-branch" {
		t.Errorf("Expected branch test-branch, got %q", branch)
	}
}

func TestCurrentCommit(t *testing.T) {
	repoDir := setupGitRepo(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change to repo: %v", err)
	}
	commit, err := CurrentCommit()
	if err != nil {
		t.Fatalf("CurrentCommit failed: %v", err)
	}
	if len(commit) != 40 {
		t.Errorf("Expected 40-char commit hash, got %q (len %d)", commit, len(commit))
	}
}

func TestAgitDBPath(t *testing.T) {
	repoDir := setupGitRepo(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change to repo: %v", err)
	}
	dbPath, err := AgitDBPath()
	if err != nil {
		t.Fatalf("AgitDBPath failed: %v", err)
	}
	expected := filepath.Join(repoDir, ".git", "agit.db")
	if dbPath != expected {
		t.Errorf("AgitDBPath expected %q, got %q", expected, dbPath)
	}
}

func TestCommitAll(t *testing.T) {
	repoDir := setupGitRepo(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change to repo: %v", err)
	}
	// Create a new file
	newFile := filepath.Join(repoDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	// Commit all changes
	commitHash, err := CommitAll("test commit")
	if err != nil {
		t.Fatalf("CommitAll failed: %v", err)
	}
	if len(commitHash) != 40 {
		t.Errorf("Expected 40-char commit hash, got %q", commitHash)
	}
	// Verify the commit exists
	cmd := exec.Command("git", "cat-file", "-t", commitHash)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Errorf("commit hash %s not valid: %v", commitHash, err)
	}
}

func TestResetToCommit(t *testing.T) {
	repoDir := setupGitRepo(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change to repo: %v", err)
	}
	// Get current commit
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	initialCommit := strings.TrimSpace(string(output))
	// Make a change and commit
	newFile := filepath.Join(repoDir, "change.txt")
	if err := os.WriteFile(newFile, []byte("change"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", "change.txt")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "new commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	// Reset to initial commit
	if err := ResetToCommit(initialCommit); err != nil {
		t.Fatalf("ResetToCommit failed: %v", err)
	}
	// Verify change.txt is gone
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Error("ResetToCommit did not remove file")
	}
}
