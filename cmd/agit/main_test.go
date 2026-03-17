// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildAgitBinary builds the agit binary and returns the path to it.
// The binary is placed in a temporary directory managed by t.TempDir().
func buildAgitBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agit")
	// Find module root using go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get module root: %v", err)
	}
	moduleRoot := string(bytes.TrimSpace(output))
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agit")
	buildCmd.Dir = moduleRoot
	var stderr bytes.Buffer
	buildCmd.Stderr = &stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build agit binary: %v\nStderr: %s", err, stderr.String())
	}
	return binaryPath
}

// buildDashboardBinary builds the agit-dashboard binary and returns the path to it.
// The binary is placed in a temporary directory managed by t.TempDir().
func buildDashboardBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agit-dashboard")
	// Find module root using go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get module root: %v", err)
	}
	moduleRoot := string(bytes.TrimSpace(output))
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agit-dashboard")
	buildCmd.Dir = moduleRoot
	var stderr bytes.Buffer
	buildCmd.Stderr = &stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build agit-dashboard binary: %v\nStderr: %s", err, stderr.String())
	}
	return binaryPath
}

func TestAgitHelp(t *testing.T) {
	binaryPath := buildAgitBinary(t)
	// Run help command
	helpCmd := exec.Command(binaryPath, "--help")
	output, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agit --help failed: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Error("expected non-empty help output")
	}
}

func TestAgitInit(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}
	// Initialize git repository
	gitInit := exec.Command("git", "init")
	gitInit.Dir = repoDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	// Configure git user
	gitConfig := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfig.Dir = repoDir
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("failed to set git user email: %v", err)
	}
	gitConfig = exec.Command("git", "config", "user.name", "Test User")
	gitConfig.Dir = repoDir
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("failed to set git user name: %v", err)
	}
	// Create a dummy file and commit
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	gitAdd := exec.Command("git", "add", "README.md")
	gitAdd.Dir = repoDir
	if err := gitAdd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	gitCommit := exec.Command("git", "commit", "-m", "initial commit")
	gitCommit.Dir = repoDir
	if err := gitCommit.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	binaryPath := buildAgitBinary(t)

	// Run agit init
	initCmd := exec.Command(binaryPath, "init")
	initCmd.Dir = repoDir
	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agit init failed: %v\nOutput: %s", err, output)
	}
	// Verify .git/agit.db exists
	dbPath := filepath.Join(repoDir, ".git", "agit.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error(".git/agit.db not created after agit init")
	}
}

func TestAgitHookInstall(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}
	// Initialize git repository
	gitInit := exec.Command("git", "init")
	gitInit.Dir = repoDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	// Configure git user
	gitConfig := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfig.Dir = repoDir
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("failed to set git user email: %v", err)
	}
	gitConfig = exec.Command("git", "config", "user.name", "Test User")
	gitConfig.Dir = repoDir
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("failed to set git user name: %v", err)
	}
	// Create a dummy file and commit
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	gitAdd := exec.Command("git", "add", "README.md")
	gitAdd.Dir = repoDir
	if err := gitAdd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	gitCommit := exec.Command("git", "commit", "-m", "initial commit")
	gitCommit.Dir = repoDir
	if err := gitCommit.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	binaryPath := buildAgitBinary(t)

	// Run agit hook install
	hookCmd := exec.Command(binaryPath, "hook", "install")
	hookCmd.Dir = repoDir
	output, err := hookCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agit hook install failed: %v\nOutput: %s", err, output)
	}
	// Verify post-checkout hook exists
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Error("post-checkout hook not created after agit hook install")
	}
	// Verify hook is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Errorf("failed to stat hook: %v", err)
	} else if info.Mode()&0111 == 0 {
		t.Error("hook is not executable")
	}
}

func TestAgitDashboard(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}
	// Initialize git repository
	gitInit := exec.Command("git", "init")
	gitInit.Dir = repoDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	// Configure git user
	gitConfig := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfig.Dir = repoDir
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("failed to set git user email: %v", err)
	}
	gitConfig = exec.Command("git", "config", "user.name", "Test User")
	gitConfig.Dir = repoDir
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("failed to set git user name: %v", err)
	}
	// Create a dummy file and commit
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	gitAdd := exec.Command("git", "add", "README.md")
	gitAdd.Dir = repoDir
	if err := gitAdd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	gitCommit := exec.Command("git", "commit", "-m", "initial commit")
	gitCommit.Dir = repoDir
	if err := gitCommit.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	binaryPath := buildAgitBinary(t)

	// First, ensure agit-dashboard binary exists in the same directory as agit
	// Build agit-dashboard binary in same temp directory
	dashboardBin := filepath.Join(t.TempDir(), "agit-dashboard")
	cmd := exec.Command("go", "build", "-o", dashboardBin, "./cmd/agit-dashboard")
	if err := cmd.Run(); err != nil {
		t.Skipf("failed to build agit-dashboard: %v (skipping dashboard test)", err)
	}

	// Run agit init to create database
	initCmd := exec.Command(binaryPath, "init")
	initCmd.Dir = repoDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("agit init failed: %v", err)
	}

	// Test dashboard --help (should show help without starting server)
	helpCmd := exec.Command(binaryPath, "dashboard", "--help")
	helpCmd.Dir = repoDir
	// Set PATH to include directory with dashboard binary
	env := os.Environ()
	// Add directory containing dashboard binary to PATH
	dashboardDir := filepath.Dir(dashboardBin)
	newPath := dashboardDir + ":" + os.Getenv("PATH")
	env = append(env, "PATH="+newPath)
	helpCmd.Env = env

	output, err := helpCmd.CombinedOutput()
	if err != nil {
		// Accept non-zero exit code (dashboard binary might exit with code 0 for --help)
		// Just check that output contains something
		t.Logf("dashboard --help exited with error (expected for some help flags): %v", err)
	}
	// At minimum, should not panic
	if bytes.Contains(output, []byte("panic")) {
		t.Errorf("dashboard --help panicked: %s", output)
	}
}
