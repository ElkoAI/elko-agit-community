package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Checkpoint represents a checkpoint row in the database.
type Checkpoint struct {
	ID              int64           `json:"id"`
	WorkspaceBranch string          `json:"workspace_branch"`
	CommitHash      string          `json:"commit_hash"`
	AgentState      json.RawMessage `json:"agent_state"`
	Message         string          `json:"message"`
	CreatedAt       time.Time       `json:"created_at"`
}

// CreateCheckpoint inserts a new checkpoint for the given workspace branch.
// The agentState should be a JSON snapshot of the workspace state at the time of commit.
// Returns the newly created checkpoint ID.
func CreateCheckpoint(dbPath, workspaceBranch, commitHash, message string, agentState json.RawMessage) (int64, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Validate workspace exists
	var exists bool
	err = db.QueryRow("SELECT 1 FROM workspaces WHERE branch = ?", workspaceBranch).Scan(&exists)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("workspace %q does not exist", workspaceBranch)
	} else if err != nil {
		return 0, fmt.Errorf("failed to check workspace: %w", err)
	}

	// Insert checkpoint
	var agentStateJSON sql.NullString
	if len(agentState) > 0 {
		agentStateJSON = sql.NullString{String: string(agentState), Valid: true}
	}
	res, err := db.Exec(`
		INSERT INTO checkpoints (workspace_branch, commit_hash, agent_state, message, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, workspaceBranch, commitHash, agentStateJSON, message, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("insert checkpoint failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return id, nil
}

// GetLatestCheckpoint returns the most recent checkpoint for a workspace branch.
// Returns nil, nil if no checkpoint exists.
func GetLatestCheckpoint(dbPath, workspaceBranch string) (*Checkpoint, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var cp Checkpoint
	var agentState sql.NullString
	err = db.QueryRow(`
		SELECT id, workspace_branch, commit_hash, agent_state, message, created_at
		FROM checkpoints
		WHERE workspace_branch = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, workspaceBranch).Scan(&cp.ID, &cp.WorkspaceBranch, &cp.CommitHash, &agentState, &cp.Message, &cp.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("query checkpoint failed: %w", err)
	}
	if agentState.Valid {
		cp.AgentState = json.RawMessage(agentState.String)
	}
	return &cp, nil
}

// GetWorkspaceSnapshot returns a JSON snapshot of the current workspace state.
// This includes the workspace row fields that should be preserved in a checkpoint.
func GetWorkspaceSnapshot(dbPath, workspaceBranch string) (json.RawMessage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var ws Workspace
	var currentTaskID sql.NullInt64
	var agentState, todoList sql.NullString
	err = db.QueryRow(`
		SELECT branch, current_task_id, agent_state, todo_list, last_commit, created_at, updated_at
		FROM workspaces
		WHERE branch = ?
	`, workspaceBranch).Scan(&ws.Branch, &currentTaskID, &agentState, &todoList, &ws.LastCommit, &ws.CreatedAt, &ws.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace %q does not exist", workspaceBranch)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query workspace: %w", err)
	}
	ws.CurrentTaskID = currentTaskID
	if agentState.Valid {
		ws.AgentState = json.RawMessage(agentState.String)
	}
	if todoList.Valid {
		ws.TodoList = todoList.String
	}
	// Serialize workspace to JSON
	data, err := json.Marshal(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace: %w", err)
	}
	return json.RawMessage(data), nil
}

// RestoreWorkspaceFromSnapshot restores a workspace from a previously saved snapshot.
// The agentState must be a JSON blob produced by GetWorkspaceSnapshot.
// It updates current_task_id, agent_state, todo_list, last_commit, and updated_at.
func RestoreWorkspaceFromSnapshot(dbPath, workspaceBranch string, agentState json.RawMessage) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var ws Workspace
	if err := json.Unmarshal(agentState, &ws); err != nil {
		return fmt.Errorf("failed to unmarshal agent state: %w", err)
	}
	// Ensure the snapshot branch matches the target workspace (should always be true)
	if ws.Branch != workspaceBranch {
		return fmt.Errorf("snapshot branch %q does not match workspace branch %q", ws.Branch, workspaceBranch)
	}
	// Prepare nullable fields for SQL update
	var currentTaskID sql.NullInt64
	if ws.CurrentTaskID.Valid {
		currentTaskID = ws.CurrentTaskID
	}
	var agentStateJSON sql.NullString
	if len(ws.AgentState) > 0 {
		agentStateJSON = sql.NullString{String: string(ws.AgentState), Valid: true}
	}
	var todoList sql.NullString
	if ws.TodoList != "" {
		todoList = sql.NullString{String: ws.TodoList, Valid: true}
	}
	// Update workspace row (branch is primary key, created_at stays unchanged)
	_, err = db.Exec(`
		UPDATE workspaces
		SET current_task_id = ?, agent_state = ?, todo_list = ?, last_commit = ?, updated_at = ?
		WHERE branch = ?
	`, currentTaskID, agentStateJSON, todoList, ws.LastCommit, time.Now().UTC(), workspaceBranch)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}
	return nil
}

// ListCheckpoints returns all checkpoints for a workspace branch, ordered by created_at DESC.
// If search is not empty, filters checkpoints where message contains the search term (case-insensitive).
func ListCheckpoints(dbPath, workspaceBranch, search string) ([]Checkpoint, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT id, workspace_branch, commit_hash, agent_state, message, created_at
		FROM checkpoints
		WHERE workspace_branch = ?`
	args := []interface{}{workspaceBranch}

	if search != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query checkpoints failed: %w", err)
	}
	defer rows.Close()

	var checkpoints []Checkpoint
	for rows.Next() {
		var cp Checkpoint
		var agentState sql.NullString
		err := rows.Scan(&cp.ID, &cp.WorkspaceBranch, &cp.CommitHash, &agentState, &cp.Message, &cp.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan checkpoint failed: %w", err)
		}
		if agentState.Valid {
			cp.AgentState = json.RawMessage(agentState.String)
		}
		checkpoints = append(checkpoints, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return checkpoints, nil
}
