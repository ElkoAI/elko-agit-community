package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// lockFile returns the path to the lock file for a given database path.
func lockFile(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), "agit.lock")
}

// gitLockFile returns the path to the Git operation lock file.
func gitLockFile(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), "agit.git.lock")
}

// lockFileWithRetry attempts to acquire a lock on the given lock path.
func lockFileWithRetry(lockPath string) (func(), error) {
	maxAttempts := 5
	baseDelay := 10 * time.Millisecond

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Try to create the lock file exclusively
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			// Write current PID to allow detection of stale locks
			pid := os.Getpid()
			_, err = f.Write([]byte(strconv.Itoa(pid)))
			if err != nil {
				f.Close()
				os.Remove(lockPath)
				return nil, fmt.Errorf("failed to write lock file: %w", err)
			}
			f.Close()

			// Successfully acquired lock
			unlock := func() {
				os.Remove(lockPath)
			}
			return unlock, nil
		}

		// If lock exists, check if it's stale (the owning process no longer alive)
		if _, err := os.Stat(lockPath); err == nil {
			content, err := os.ReadFile(lockPath)
			if err == nil {
				pidStr := strings.TrimSpace(string(content))
				if pid, err := strconv.Atoi(pidStr); err == nil && !processExists(pid) {
					// Stale lock – remove it and retry
					os.Remove(lockPath)
					continue
				}
			}
		}

		// Wait before retrying
		delay := baseDelay * (1 << uint(attempt)) // exponential backoff
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("could not acquire lock after %d attempts", maxAttempts)
}

// Lock acquires an exclusive file‑based lock for the database.
// It returns a cleanup function that releases the lock, or an error.
// The lock is implemented via a PID‑stamped lock file with O_EXCL.
// If the lock is already held, it retries up to 5 times with exponential backoff.
func Lock(dbPath string) (func(), error) {
	return lockFileWithRetry(lockFile(dbPath))
}

// GitLock acquires an exclusive file‑based lock for Git operations.
// This should be used by CLI commands that modify Git state (commit, reset, hooks).
// It returns a cleanup function that releases the lock, or an error.
func GitLock(dbPath string) (func(), error) {
	return lockFileWithRetry(gitLockFile(dbPath))
}

// processExists checks whether a process with the given PID is alive on Unix.
// Returns true if the process exists, false otherwise.
func processExists(pid int) bool {
	// On Unix, sending signal 0 to a process checks if it exists without disturbing it.
	// If the call returns an error, the process does not exist.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(os.Signal(nil)) // nil signal is a no-op check
	return err == nil
}
