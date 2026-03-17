package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Task represents a task row in the database.
type Task struct {
	ID              int64           `json:"id"`
	WorkspaceBranch string          `json:"workspace_branch"`
	Description     string          `json:"description"`
	Status          string          `json:"status"` // pending, active, done, failed
	TestCommand     sql.NullString  `json:"test_command"`
	TestResult      json.RawMessage `json:"test_result"`
	ParentTaskID    sql.NullInt64   `json:"parent_task_id"`
	CreatedAt       time.Time       `json:"created_at"`
	CompletedAt     sql.NullTime    `json:"completed_at"`
}

// CreateTask inserts a new task for the given workspace branch.
// Returns the newly created task ID.
func CreateTask(dbPath, workspaceBranch, description, testCommand string, parentTaskID sql.NullInt64) (int64, error) {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

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

	// Validate parent task if specified
	if parentTaskID.Valid {
		var parentWorkspace string
		err = db.QueryRow("SELECT workspace_branch FROM tasks WHERE id = ?", parentTaskID.Int64).Scan(&parentWorkspace)
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("parent task %d does not exist", parentTaskID.Int64)
		} else if err != nil {
			return 0, fmt.Errorf("failed to check parent task: %w", err)
		}
		if parentWorkspace != workspaceBranch {
			return 0, fmt.Errorf("parent task %d belongs to workspace %q, not %q", parentTaskID.Int64, parentWorkspace, workspaceBranch)
		}
	}

	// Insert task with pending status
	var testCmd sql.NullString
	if testCommand != "" {
		testCmd = sql.NullString{String: testCommand, Valid: true}
	}
	res, err := db.Exec(`
		INSERT INTO tasks (workspace_branch, description, status, test_command, parent_task_id, created_at)
		VALUES (?, ?, 'pending', ?, ?, ?)
	`, workspaceBranch, description, testCmd, parentTaskID, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("insert task failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return id, nil
}

// ListTasks returns tasks for a given workspace branch, optionally filtered by status and search term.
// If statusFilter is empty, returns all tasks regardless of status.
// If search is not empty, filters tasks where description contains the search term (case-insensitive).
func ListTasks(dbPath, workspaceBranch, statusFilter, search string) ([]Task, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Build dynamic WHERE clause
	query := `
		SELECT id, workspace_branch, description, status, test_command, test_result, parent_task_id, created_at, completed_at
		FROM tasks
		WHERE workspace_branch = ?`
	args := []interface{}{workspaceBranch}

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}

	if search != "" {
		query += " AND description LIKE ?"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks failed: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var testCmd, testResult sql.NullString
		var parentTaskID sql.NullInt64
		var completedAt sql.NullTime
		err := rows.Scan(&t.ID, &t.WorkspaceBranch, &t.Description, &t.Status, &testCmd, &testResult, &parentTaskID, &t.CreatedAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("scan task row failed: %w", err)
		}
		if testCmd.Valid {
			t.TestCommand = sql.NullString{String: testCmd.String, Valid: true}
		}
		if testResult.Valid {
			t.TestResult = json.RawMessage(testResult.String)
		}
		if parentTaskID.Valid {
			t.ParentTaskID = sql.NullInt64{Int64: parentTaskID.Int64, Valid: true}
		}
		if completedAt.Valid {
			t.CompletedAt = sql.NullTime{Time: completedAt.Time, Valid: true}
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return tasks, nil
}

// StartTask marks a task as active and updates the workspace's current_task_id.
func StartTask(dbPath, workspaceBranch string, taskID int64) error {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify task belongs to workspace
	var count int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM tasks
		WHERE id = ? AND workspace_branch = ?
	`, taskID, workspaceBranch).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify task ownership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task %d not found in workspace %q", taskID, workspaceBranch)
	}

	// Update task status
	_, err = tx.Exec(`UPDATE tasks SET status = 'active' WHERE id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Update workspace current_task_id
	_, err = tx.Exec(`UPDATE workspaces SET current_task_id = ?, updated_at = ? WHERE branch = ?`,
		taskID, time.Now().UTC(), workspaceBranch)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// DoneTask marks a task as done, clears workspace current_task_id, and sets completed_at.
func DoneTask(dbPath, workspaceBranch string, taskID int64) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify task belongs to workspace
	var count int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM tasks
		WHERE id = ? AND workspace_branch = ?
	`, taskID, workspaceBranch).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify task ownership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task %d not found in workspace %q", taskID, workspaceBranch)
	}

	// Update task status and completed_at
	_, err = tx.Exec(`
		UPDATE tasks
		SET status = 'done', completed_at = ?
		WHERE id = ?
	`, time.Now().UTC(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Clear workspace current_task_id (set to NULL)
	_, err = tx.Exec(`UPDATE workspaces SET current_task_id = NULL, updated_at = ? WHERE branch = ?`,
		time.Now().UTC(), workspaceBranch)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetTask retrieves a task by its ID.
// Returns nil if the task does not exist.
func GetTask(dbPath string, taskID int64) (*Task, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var t Task
	var testCmd, testResult sql.NullString
	var parentTaskID sql.NullInt64
	var completedAt sql.NullTime
	err = db.QueryRow(`
		SELECT id, workspace_branch, description, status, test_command, test_result, parent_task_id, created_at, completed_at
		FROM tasks
		WHERE id = ?
	`, taskID).Scan(&t.ID, &t.WorkspaceBranch, &t.Description, &t.Status, &testCmd, &testResult, &parentTaskID, &t.CreatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("query task failed: %w", err)
	}
	if testCmd.Valid {
		t.TestCommand = sql.NullString{String: testCmd.String, Valid: true}
	}
	if testResult.Valid {
		t.TestResult = json.RawMessage(testResult.String)
	}
	if parentTaskID.Valid {
		t.ParentTaskID = sql.NullInt64{Int64: parentTaskID.Int64, Valid: true}
	}
	if completedAt.Valid {
		t.CompletedAt = sql.NullTime{Time: completedAt.Time, Valid: true}
	}
	return &t, nil
}

// GetSubtasks returns all child tasks for a given parent task ID.
func GetSubtasks(dbPath string, parentTaskID int64) ([]Task, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, workspace_branch, description, status, test_command, test_result, parent_task_id, created_at, completed_at
		FROM tasks
		WHERE parent_task_id = ?
		ORDER BY created_at DESC
	`, parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("query subtasks failed: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var testCmd, testResult sql.NullString
		var parentID sql.NullInt64
		var completedAt sql.NullTime
		err := rows.Scan(&t.ID, &t.WorkspaceBranch, &t.Description, &t.Status, &testCmd, &testResult, &parentID, &t.CreatedAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("scan subtask row failed: %w", err)
		}
		if testCmd.Valid {
			t.TestCommand = sql.NullString{String: testCmd.String, Valid: true}
		}
		if testResult.Valid {
			t.TestResult = json.RawMessage(testResult.String)
		}
		if parentID.Valid {
			t.ParentTaskID = sql.NullInt64{Int64: parentID.Int64, Valid: true}
		}
		if completedAt.Valid {
			t.CompletedAt = sql.NullTime{Time: completedAt.Time, Valid: true}
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return tasks, nil
}

// UpdateTaskStatus updates a task's status and clears workspace current_task_id if the task was active.
// Valid status values: "pending", "active", "done", "failed".
func UpdateTaskStatus(dbPath, workspaceBranch string, taskID int64, status string) error {
	// Validate status
	validStatuses := map[string]bool{"pending": true, "active": true, "done": true, "failed": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status %q", status)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify task belongs to workspace
	var count int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM tasks
		WHERE id = ? AND workspace_branch = ?
	`, taskID, workspaceBranch).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify task ownership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task %d not found in workspace %q", taskID, workspaceBranch)
	}

	// Update task status and optionally completed_at
	var query string
	if status == "done" || status == "failed" {
		query = `UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?`
		_, err = tx.Exec(query, status, time.Now().UTC(), taskID)
	} else {
		query = `UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?`
		_, err = tx.Exec(query, status, taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Clear workspace current_task_id if this task is the current task.
	_, err = tx.Exec(`UPDATE workspaces SET current_task_id = NULL, updated_at = ? WHERE branch = ? AND current_task_id = ?`,
		time.Now().UTC(), workspaceBranch, taskID)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// UpdateTaskTestResult updates the test_result JSON column for a task.
func UpdateTaskTestResult(dbPath string, taskID int64, result json.RawMessage) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Validate task exists
	var exists bool
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task %d does not exist", taskID)
	} else if err != nil {
		return fmt.Errorf("failed to check task: %w", err)
	}

	// Update test_result
	_, err = db.Exec(`UPDATE tasks SET test_result = ? WHERE id = ?`, result, taskID)
	if err != nil {
		return fmt.Errorf("failed to update test_result: %w", err)
	}
	return nil
}

// TaskTreeNode represents a task with its child tasks.
type TaskTreeNode struct {
	Task
	Children []*TaskTreeNode `json:"children"`
}

// GetTaskTree returns a hierarchical tree of tasks for the given workspace branch.
// Returns a forest of root nodes (tasks with no parent).
func GetTaskTree(dbPath, workspaceBranch string) ([]*TaskTreeNode, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Fetch all tasks for this workspace
	rows, err := db.Query(`
		SELECT id, workspace_branch, description, status, test_command, test_result, parent_task_id, created_at, completed_at
		FROM tasks
		WHERE workspace_branch = ?
		ORDER BY created_at DESC
	`, workspaceBranch)
	if err != nil {
		return nil, fmt.Errorf("query tasks failed: %w", err)
	}
	defer rows.Close()

	// Build flat list of tasks and map ID -> node
	nodes := make(map[int64]*TaskTreeNode)
	var roots []*TaskTreeNode

	for rows.Next() {
		var t Task
		var testCmd, testResult sql.NullString
		var parentTaskID sql.NullInt64
		var completedAt sql.NullTime
		err := rows.Scan(&t.ID, &t.WorkspaceBranch, &t.Description, &t.Status, &testCmd, &testResult, &parentTaskID, &t.CreatedAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("scan task row failed: %w", err)
		}
		if testCmd.Valid {
			t.TestCommand = sql.NullString{String: testCmd.String, Valid: true}
		}
		if testResult.Valid {
			t.TestResult = json.RawMessage(testResult.String)
		}
		if parentTaskID.Valid {
			t.ParentTaskID = sql.NullInt64{Int64: parentTaskID.Int64, Valid: true}
		}
		if completedAt.Valid {
			t.CompletedAt = sql.NullTime{Time: completedAt.Time, Valid: true}
		}

		node := &TaskTreeNode{Task: t}
		nodes[t.ID] = node

		if parentTaskID.Valid {
			// Will be attached later after all nodes are created
		} else {
			roots = append(roots, node)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Attach children to parent nodes
	for _, node := range nodes {
		if node.ParentTaskID.Valid {
			parent := nodes[node.ParentTaskID.Int64]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
			// If parent not found (shouldn't happen due to foreign key), treat as root
			// but we'll ignore this edge case
		}
	}

	return roots, nil
}
