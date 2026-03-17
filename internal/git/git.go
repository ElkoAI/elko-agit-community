package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// CurrentBranch returns the current Git branch name.
// Returns empty string if not in a Git repository or error occurs.
func CurrentBranch() (string, error) {
	root, err := GitRoot()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// CurrentCommit returns the current Git commit hash.
// Returns empty string if not in a Git repository or error occurs.
func CurrentCommit() (string, error) {
	root, err := GitRoot()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GitRoot returns the root directory of the Git repository.
// Returns error if not in a Git repository or cannot determine root.
func GitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// IsGitRepository checks if the current directory is inside a Git repository.
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

// AgitDBPath returns the path to the agit database file (.git/agit.db)
// relative to the Git repository root.
// Returns error if not in a Git repository.
func AgitDBPath() (string, error) {
	root, err := GitRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".git", "agit.db"), nil
}

// CommitAll commits all changes with the given message, allowing empty commits.
// Returns the new commit hash, or error if commit fails.
func CommitAll(message string) (string, error) {
	root, err := GitRoot()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "commit", "--allow-empty", "-a", "-m", message)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}
	// Get the new commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ResetToCommit resets the working directory and index to the given commit hash (hard reset).
func ResetToCommit(commitHash string) error {
	root, err := GitRoot()
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "reset", "--hard", commitHash)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git reset --hard %s failed: %w", commitHash, err)
	}
	return nil
}
