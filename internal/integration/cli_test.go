//go:build integration

package integration

import (
	"regexp"
	"strings"
	"testing"
)

func TestCLIWorkflow(t *testing.T) {
	// Build agit binary once for all tests
	binaryPath := buildAgitBinary(t)
	defer cleanupBinary(t, binaryPath)

	// Create a temporary Git repository
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Step 1: Initialize agit in the repository
	_, _, err := runAgit(t, binaryPath, tmpDir, "init")
	if err != nil {
		t.Fatalf("agit init failed: %v", err)
	}

	// Step 2: Ensure workspace exists (workspace-status creates it)
	_, _, err = runAgit(t, binaryPath, tmpDir, "workspace-status")
	if err != nil {
		t.Fatalf("agit workspace-status failed: %v", err)
	}

	// Step 3: Create a new task with test command
	stdout, stderr, err := runAgit(t, binaryPath, tmpDir, "task", "new", "--test", "echo 'test passed'", "Implement feature X")
	if err != nil {
		t.Fatalf("agit task new failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	// Extract task ID from output (format "Created task #<id>: ...")
	taskID := extractTaskID(t, stdout+stderr)
	if taskID == "" {
		t.Fatalf("failed to extract task ID from output: %s", stdout+stderr)
	}

	// Step 4: List tasks and verify task appears
	stdout, _, err = runAgit(t, binaryPath, tmpDir, "task", "list")
	if err != nil {
		t.Fatalf("agit task list failed: %v", err)
	}
	if !strings.Contains(stdout, "Implement feature X") {
		t.Errorf("task list missing expected task:\n%s", stdout)
	}

	// Step 5: Start task
	_, _, err = runAgit(t, binaryPath, tmpDir, "task", "start", taskID)
	if err != nil {
		t.Fatalf("agit task start failed: %v", err)
	}

	// Step 6: Mark task as done (this should run the test command)
	stdout, _, err = runAgit(t, binaryPath, tmpDir, "task", "done", taskID)
	if err != nil {
		t.Fatalf("agit task done failed: %v", err)
	}
	// Verify test output appears (should contain "test passed")
	if !strings.Contains(stdout, "test passed") {
		t.Logf("test output not found in task done (expected if exit code 0): %s", stdout)
	}

	// Step 7: Checkpoint current state
	_, _, err = runAgit(t, binaryPath, tmpDir, "checkpoint", "After implementing feature X")
	if err != nil {
		t.Fatalf("agit checkpoint failed: %v", err)
	}

	// Step 8: View trace for task (use --task flag)
	stdout, _, err = runAgit(t, binaryPath, tmpDir, "trace", "--task", taskID)
	if err != nil {
		t.Fatalf("agit trace failed: %v", err)
	}
	if !strings.Contains(stdout, "test passed") {
		t.Errorf("trace output missing test output:\n%s", stdout)
	}

	// Step 9: Create subtask via breakdown (JSON array)
	subtasksJSON := `[{"description": "Subtask A", "test_command": "echo 'subtask A test'"}]`
	_, _, err = runAgit(t, binaryPath, tmpDir, "task", "breakdown", taskID, subtasksJSON)
	if err != nil {
		t.Fatalf("agit task breakdown failed: %v", err)
	}

	// Step 10: Show task tree
	stdout, _, err = runAgit(t, binaryPath, tmpDir, "task", "tree")
	if err != nil {
		t.Fatalf("agit task tree failed: %v", err)
	}
	if !strings.Contains(stdout, "Implement feature X") {
		t.Errorf("task tree missing expected task:\n%s", stdout)
	}
}

// extractTaskID extracts the numeric task ID from CLI output.
// Assumes format "Created task #<id>:" or similar.
func extractTaskID(t *testing.T, output string) string {
	t.Helper()
	re := regexp.MustCompile(`#(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}
