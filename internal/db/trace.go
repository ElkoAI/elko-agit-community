package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Trace represents a test run trace row in the database.
type Trace struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	ExitCode   *int      `json:"exit_code,omitempty"` // pointer to allow null in JSON
	Output     string    `json:"output"`
	DurationMs *int64    `json:"duration_ms,omitempty"` // pointer to allow null
	RunAt      time.Time `json:"run_at"`
}

// CreateTrace inserts a new test run trace for a task.
// Returns the newly created trace ID.
func CreateTrace(dbPath string, taskID int64, exitCode *int, output string, durationMs *int64) (int64, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Validate task exists
	var exists bool
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("task %d does not exist", taskID)
	} else if err != nil {
		return 0, fmt.Errorf("failed to check task: %w", err)
	}

	// Prepare nullable values for SQL
	var exitCodeNull sql.NullInt64
	if exitCode != nil {
		exitCodeNull = sql.NullInt64{Int64: int64(*exitCode), Valid: true}
	}
	var durationNull sql.NullInt64
	if durationMs != nil {
		durationNull = sql.NullInt64{Int64: *durationMs, Valid: true}
	}
	outputNull := sql.NullString{String: output, Valid: output != ""}

	// Insert trace
	res, err := db.Exec(`
		INSERT INTO traces (task_id, exit_code, output, duration_ms, run_at)
		VALUES (?, ?, ?, ?, ?)
	`, taskID, exitCodeNull, outputNull, durationNull, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("insert trace failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return id, nil
}

// GetTraces returns traces for a task, optionally limited by count.
// Returns empty slice if no traces exist.
func GetTraces(dbPath string, taskID int64, limit int) ([]Trace, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT id, task_id, exit_code, output, duration_ms, run_at
		FROM traces
		WHERE task_id = ?
		ORDER BY run_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, taskID)
	if err != nil {
		return nil, fmt.Errorf("query traces failed: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		var exitCodeNull sql.NullInt64
		var outputNull sql.NullString
		var durationNull sql.NullInt64
		err := rows.Scan(&t.ID, &t.TaskID, &exitCodeNull, &outputNull, &durationNull, &t.RunAt)
		if err != nil {
			return nil, fmt.Errorf("scan trace row failed: %w", err)
		}
		if exitCodeNull.Valid {
			exitCode := int(exitCodeNull.Int64)
			t.ExitCode = &exitCode
		}
		if outputNull.Valid {
			t.Output = outputNull.String
		}
		if durationNull.Valid {
			duration := durationNull.Int64
			t.DurationMs = &duration
		}
		traces = append(traces, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return traces, nil
}

// GetLastTrace returns the most recent trace for a task, or nil if none exist.
func GetLastTrace(dbPath string, taskID int64) (*Trace, error) {
	traces, err := GetTraces(dbPath, taskID, 1)
	if err != nil {
		return nil, err
	}
	if len(traces) == 0 {
		return nil, nil
	}
	return &traces[0], nil
}

// GetTracesFailed returns traces for a task where exit_code != 0 (failed test runs).
// Returns empty slice if no failed traces exist.
func GetTracesFailed(dbPath string, taskID int64, limit int) ([]Trace, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT id, task_id, exit_code, output, duration_ms, run_at
		FROM traces
		WHERE task_id = ? AND exit_code IS NOT NULL AND exit_code != 0
		ORDER BY run_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, taskID)
	if err != nil {
		return nil, fmt.Errorf("query traces failed: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		var exitCodeNull sql.NullInt64
		var outputNull sql.NullString
		var durationNull sql.NullInt64
		err := rows.Scan(&t.ID, &t.TaskID, &exitCodeNull, &outputNull, &durationNull, &t.RunAt)
		if err != nil {
			return nil, fmt.Errorf("scan trace row failed: %w", err)
		}
		if exitCodeNull.Valid {
			exitCode := int(exitCodeNull.Int64)
			t.ExitCode = &exitCode
		}
		if outputNull.Valid {
			t.Output = outputNull.String
		}
		if durationNull.Valid {
			duration := durationNull.Int64
			t.DurationMs = &duration
		}
		traces = append(traces, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return traces, nil
}

// ListTracesForWorkspace returns traces for all tasks in a workspace, ordered by run_at DESC.
// exitCodeFilter can be "success" (exit_code = 0), "failure" (exit_code != 0), or "" for any.
// search filters by task description (partial match).
func ListTracesForWorkspace(dbPath, workspaceBranch, exitCodeFilter, search string, limit int) ([]Trace, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT t.id, t.task_id, t.exit_code, t.output, t.duration_ms, t.run_at
		FROM traces t
		INNER JOIN tasks k ON t.task_id = k.id
		WHERE k.workspace_branch = ?
	`
	args := []interface{}{workspaceBranch}

	if exitCodeFilter == "success" {
		query += ` AND t.exit_code = 0`
	} else if exitCodeFilter == "failure" {
		query += ` AND t.exit_code IS NOT NULL AND t.exit_code != 0`
	}

	if search != "" {
		query += ` AND k.description LIKE ?`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY t.run_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query traces failed: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		var exitCodeNull sql.NullInt64
		var outputNull sql.NullString
		var durationNull sql.NullInt64
		err := rows.Scan(&t.ID, &t.TaskID, &exitCodeNull, &outputNull, &durationNull, &t.RunAt)
		if err != nil {
			return nil, fmt.Errorf("scan trace row failed: %w", err)
		}
		if exitCodeNull.Valid {
			exitCode := int(exitCodeNull.Int64)
			t.ExitCode = &exitCode
		}
		if outputNull.Valid {
			t.Output = outputNull.String
		}
		if durationNull.Valid {
			duration := durationNull.Int64
			t.DurationMs = &duration
		}
		traces = append(traces, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return traces, nil
}
