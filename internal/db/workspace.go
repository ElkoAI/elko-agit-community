package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agit/internal/git"
)

// Workspace represents a workspace row in the database.
type Workspace struct {
	Branch        string          `json:"branch"`
	CurrentTaskID sql.NullInt64   `json:"current_task_id"`
	AgentState    json.RawMessage `json:"agent_state"`
	TodoList      string          `json:"todo_list"`
	LastCommit    string          `json:"last_commit"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// GetOrCreateWorkspaceForBranch returns the workspace for the given branch and commit.
// If the workspace does not exist, it will be created with the provided commit.
// The database must already be initialized and accessible.
func GetOrCreateWorkspaceForBranch(dbPath, branch, commit string) (*Workspace, error) {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Try to fetch existing workspace
	var ws Workspace
	var currentTaskID sql.NullInt64
	var agentState, todoList sql.NullString
	err = db.QueryRow(`
		SELECT branch, current_task_id, agent_state, todo_list, last_commit, created_at, updated_at
		FROM workspaces
		WHERE branch = ?
	`, branch).Scan(&ws.Branch, &currentTaskID, &agentState, &todoList, &ws.LastCommit, &ws.CreatedAt, &ws.UpdatedAt)
	if err == sql.ErrNoRows {
		// Workspace does not exist, create it
		ws, err = createWorkspace(db, branch, commit)
		if err != nil {
			return nil, fmt.Errorf("failed to create workspace: %w", err)
		}
		return &ws, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query workspace: %w", err)
	}

	// Convert nullable fields
	ws.CurrentTaskID = currentTaskID
	if agentState.Valid {
		ws.AgentState = json.RawMessage(agentState.String)
	}
	if todoList.Valid {
		ws.TodoList = todoList.String
	}
	return &ws, nil
}

// GetOrCreateWorkspace returns the workspace for the current Git branch.
// If the workspace does not exist, it will be created with the current commit.
// The database must already be initialized and accessible.
func GetOrCreateWorkspace(dbPath string) (*Workspace, error) {
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	commit, err := git.CurrentCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %w", err)
	}
	return GetOrCreateWorkspaceForBranch(dbPath, branch, commit)
}

// createWorkspace inserts a new workspace row.
func createWorkspace(db *sql.DB, branch, commit string) (Workspace, error) {
	now := time.Now().UTC()
	res, err := db.Exec(`
		INSERT INTO workspaces (branch, current_task_id, agent_state, todo_list, last_commit, created_at, updated_at)
		VALUES (?, NULL, NULL, '', ?, ?, ?)
	`, branch, commit, now, now)
	if err != nil {
		return Workspace{}, fmt.Errorf("insert workspace failed: %w", err)
	}
	// Verify insertion
	rows, err := res.RowsAffected()
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows != 1 {
		return Workspace{}, fmt.Errorf("expected 1 row affected, got %d", rows)
	}
	return Workspace{
		Branch:     branch,
		TodoList:   "",
		LastCommit: commit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
