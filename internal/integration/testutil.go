//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildAgitBinary builds the agit binary and returns its path.
func buildAgitBinary(t *testing.T) string {
	t.Helper()

	// Create temporary directory for binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agit")
	if strings.Contains(binaryPath, " ") {
		// Avoid spaces in path for easier command execution
		tmpDir = t.TempDir()
		binaryPath = filepath.Join(tmpDir, "agit")
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agit")
	cmd.Dir = projectRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build agit binary: %v\nOutput: %s", err, output)
	}

	// Ensure binary is executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		t.Fatalf("Failed to make binary executable: %v", err)
	}

	return binaryPath
}

// projectRoot returns the absolute path to the project root.
func projectRoot() string {
	// Get the path of this source file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	dir := filepath.Dir(filename)
	// internal/integration -> internal -> .
	return filepath.Join(dir, "..", "..")
}

// runCommand runs a command in the given working directory and returns stdout, stderr, and error.
func runCommand(t *testing.T, workdir, name string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runAgit runs the agit binary with given arguments in the specified directory.
func runAgit(t *testing.T, binaryPath, workdir string, args ...string) (string, string, error) {
	return runCommand(t, workdir, binaryPath, args...)
}

// initGitRepo initializes a Git repository in the given directory.
func initGitRepo(t *testing.T, dir string) {
	_, _, err := runCommand(t, dir, "git", "init")
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	// Set user identity for commits
	runCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runCommand(t, dir, "git", "config", "user.name", "Test User")
	// Make an initial commit
	runCommand(t, dir, "git", "commit", "--allow-empty", "-m", "Initial commit")
}

// cleanup removes the binary if needed (optional).
func cleanupBinary(t *testing.T, binaryPath string) {
	t.Helper()
	os.Remove(binaryPath)
}
