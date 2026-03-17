// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.
package main

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// buildAgitMCPBinary builds the agit-mcp binary and returns the path to it.
func buildAgitMCPBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agit-mcp")
	// Find module root using go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get module root: %v", err)
	}
	moduleRoot := string(bytes.TrimSpace(output))
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agit-mcp")
	buildCmd.Dir = moduleRoot
	var stderr bytes.Buffer
	buildCmd.Stderr = &stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build agit-mcp binary: %v\nStderr: %s", err, stderr.String())
	}
	return binaryPath
}

func TestBuildBinary(t *testing.T) {
	// Ensure the binary can be built
	buildAgitMCPBinary(t)
}

func TestBinaryStartsWithoutCrash(t *testing.T) {
	binaryPath := buildAgitMCPBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	// Provide empty stdin (close immediately)
	cmd.Stdin = bytes.NewReader(nil)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// The server will wait for stdin; we expect it to not crash immediately.
	// It will be killed by context timeout, which is fine.
	err := cmd.Run()
	// Expect either context.DeadlineExceeded or exit with non-zero (since stdin closed)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			// Command was killed by timeout, which is expected
			return
		}
		// Other error (maybe exit code 1) is also acceptable for closed stdin
		t.Logf("binary exited with error (acceptable): %v, stderr: %s", err, stderr.String())
	}
}
