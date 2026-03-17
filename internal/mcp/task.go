package mcp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os/exec"
	"time"

	"agit/internal/db"
	"agit/internal/git"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattn/go-shellwords"
)

// TaskNewTool creates a new task in the current workspace.
// Parameters:
// - description (string, required): Task description
// - test_command (string, optional): Test command to run for this task
// - parent_task_id (int, optional): ID of parent task for decomposition
func TaskNewTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get current workspace branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Parse required parameters
	description, err := request.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'description'"), nil
	}

	// Optional parameters
	testCommand := request.GetString("test_command", "")
	var parentTaskID sql.NullInt64
	if parentID := request.GetInt("parent_task_id", 0); parentID > 0 {
		parentTaskID = sql.NullInt64{Int64: int64(parentID), Valid: true}
	}

	// Create task
	taskID, err := db.CreateTask(dbPath, branch, description, testCommand, parentTaskID)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to create task: %v", err), nil
	}

	// Return success response
	result := map[string]interface{}{
		"task_id":     taskID,
		"description": description,
		"branch":      branch,
		"status":      "pending",
	}
	if testCommand != "" {
		result["test_command"] = testCommand
	}
	if parentTaskID.Valid {
		result["parent_task_id"] = parentTaskID.Int64
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to marshal response: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// TaskListTool lists tasks for the current workspace.
// Parameters:
// - status (string, optional): Filter by status (pending, active, done, failed)
func TaskListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get current workspace branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Optional status filter
	statusFilter := request.GetString("status", "")

	// List tasks
	tasks, err := db.ListTasks(dbPath, branch, statusFilter, "")
	if err != nil {
		return mcp.NewToolResultErrorf("failed to list tasks: %v", err), nil
	}

	// Convert to JSON
	tasksJSON, err := json.Marshal(tasks)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to marshal tasks: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(tasksJSON)),
		},
	}, nil
}

// TaskBreakdownTool creates subtasks under a parent task.
// Parameters:
// - parent_task_id (int, required): ID of the parent task
// - subtasks (array, required): Array of subtask objects, each with:
//   - description (string, required): Subtask description
//   - test_command (string, optional): Test command for this subtask
func TaskBreakdownTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get current workspace branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Parse parent_task_id
	parentTaskID, err := request.RequireInt("parent_task_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'parent_task_id'"), nil
	}
	if parentTaskID <= 0 {
		return mcp.NewToolResultError("parent_task_id must be positive"), nil
	}

	// Parse subtasks array
	subtasksRaw, ok := request.GetArguments()["subtasks"]
	if !ok {
		return mcp.NewToolResultError("missing required parameter 'subtasks'"), nil
	}
	var subtasksSlice []interface{}
	switch v := subtasksRaw.(type) {
	case []interface{}:
		subtasksSlice = v
	case []map[string]interface{}:
		// Convert to []interface{}
		subtasksSlice = make([]interface{}, len(v))
		for i, m := range v {
			subtasksSlice[i] = m
		}
	default:
		return mcp.NewToolResultError("subtasks must be an array"), nil
	}
	if len(subtasksSlice) == 0 {
		return mcp.NewToolResultError("subtasks array cannot be empty"), nil
	}
	// Parse each subtask
	type subtask struct {
		Description string `json:"description"`
		TestCommand string `json:"test_command,omitempty"`
	}
	subtasks := make([]subtask, 0, len(subtasksSlice))
	for i, raw := range subtasksSlice {
		item, ok := raw.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultErrorf("subtask %d must be an object", i+1), nil
		}
		// description (required)
		descRaw, ok := item["description"]
		if !ok {
			return mcp.NewToolResultErrorf("subtask %d missing description", i+1), nil
		}
		desc, ok := descRaw.(string)
		if !ok || desc == "" {
			return mcp.NewToolResultErrorf("subtask %d description must be a non-empty string", i+1), nil
		}
		// test_command (optional)
		testCmd := ""
		if testRaw, ok := item["test_command"]; ok {
			if testStr, ok := testRaw.(string); ok {
				testCmd = testStr
			}
			// ignore non‑string values
		}
		subtasks = append(subtasks, subtask{
			Description: desc,
			TestCommand: testCmd,
		})
	}

	// Verify parent task exists and belongs to current workspace
	parent, err := db.GetTask(dbPath, int64(parentTaskID))
	if err != nil {
		return mcp.NewToolResultErrorf("failed to retrieve parent task: %v", err), nil
	}
	if parent == nil {
		return mcp.NewToolResultErrorf("parent task %d not found", parentTaskID), nil
	}
	if parent.WorkspaceBranch != branch {
		return mcp.NewToolResultErrorf("parent task %d belongs to workspace %q, not current workspace %q",
			parentTaskID, parent.WorkspaceBranch, branch), nil
	}

	// Create subtasks
	subtaskIDs := make([]int64, 0, len(subtasks))
	for _, st := range subtasks {
		if st.Description == "" {
			return mcp.NewToolResultError("subtask description is required"), nil
		}
		parentID := sql.NullInt64{Int64: int64(parentTaskID), Valid: true}
		subtaskID, err := db.CreateTask(dbPath, branch, st.Description, st.TestCommand, parentID)
		if err != nil {
			return mcp.NewToolResultErrorf("failed to create subtask %q: %v", st.Description, err), nil
		}
		subtaskIDs = append(subtaskIDs, subtaskID)
	}

	// Build response
	result := map[string]interface{}{
		"parent_task_id": parentTaskID,
		"branch":         branch,
		"subtask_ids":    subtaskIDs,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to marshal response: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// TaskStartTool marks a task as active and sets it as the workspace's current task.
// Parameters:
// - task_id (int, required): ID of the task to start
func TaskStartTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get current workspace branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Parse required task ID
	taskID, err := request.RequireInt("task_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'task_id'"), nil
	}

	// Start the task
	err = db.StartTask(dbPath, branch, int64(taskID))
	if err != nil {
		return mcp.NewToolResultErrorf("failed to start task: %v", err), nil
	}

	result := map[string]interface{}{
		"task_id": taskID,
		"branch":  branch,
		"status":  "active",
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to marshal response: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// TaskDoneTool marks a task as done, clears workspace current task, and runs test command if present.
// Parameters:
// - task_id (int, required): ID of the task to mark as done
func TaskDoneTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get current workspace branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Parse required task ID
	taskID, err := request.RequireInt("task_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'task_id'"), nil
	}

	// Retrieve task details
	task, err := db.GetTask(dbPath, int64(taskID))
	if err != nil {
		return mcp.NewToolResultErrorf("failed to retrieve task: %v", err), nil
	}
	if task == nil {
		return mcp.NewToolResultErrorf("task %d not found", taskID), nil
	}

	var exitCode *int
	var output string
	var durationMs *int64
	var testResultJSON json.RawMessage

	// Run test command if present
	if task.TestCommand.Valid && task.TestCommand.String != "" {
		start := time.Now()
		var cmd *exec.Cmd
		// Try to parse as shell words; if successful and no shell operators, execute directly
		args, err := shellwords.Parse(task.TestCommand.String)
		if err != nil || len(args) == 0 {
			// Fallback to sh -c for shell operators or parse failure
			cmd = exec.CommandContext(ctx, "sh", "-c", task.TestCommand.String)
		} else {
			// Simple command without shell operators: execute directly
			cmd = exec.CommandContext(ctx, args[0], args[1:]...)
		}
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		err = cmd.Run()
		elapsed := time.Since(start)
		elapsedMs := elapsed.Milliseconds()
		durationMs = &elapsedMs

		combinedOutput := stdoutBuf.String() + stderrBuf.String()
		output = combinedOutput

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				ec := exitErr.ExitCode()
				exitCode = &ec
			} else {
				// command failed to start or was interrupted
				ec := -1
				exitCode = &ec
				output = err.Error() + "\n" + output
			}
		} else {
			ec := 0
			exitCode = &ec
		}

		// Create trace
		_, err = db.CreateTrace(dbPath, int64(taskID), exitCode, output, durationMs)
		if err != nil {
			return mcp.NewToolResultErrorf("failed to create trace: %v", err), nil
		}

		// Prepare test result JSON
		testResult := map[string]interface{}{
			"exit_code":   exitCode,
			"output":      output,
			"duration_ms": durationMs,
			"run_at":      time.Now().UTC().Format(time.RFC3339),
		}
		resultBytes, err := json.Marshal(testResult)
		if err != nil {
			return mcp.NewToolResultErrorf("failed to marshal test result: %v", err), nil
		}
		testResultJSON = json.RawMessage(resultBytes)
	}

	// Determine status based on exit code (if test command was run)
	status := "done"
	if exitCode != nil && *exitCode != 0 {
		status = "failed"
	}

	// Update task status (and clear workspace current task)
	err = db.UpdateTaskStatus(dbPath, branch, int64(taskID), status)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to update task status: %v", err), nil
	}

	// Update test_result column if we have a test result
	if testResultJSON != nil {
		err = db.UpdateTaskTestResult(dbPath, int64(taskID), testResultJSON)
		if err != nil {
			return mcp.NewToolResultErrorf("failed to update test result: %v", err), nil
		}
	}

	// Build response
	result := map[string]interface{}{
		"task_id": taskID,
		"branch":  branch,
		"status":  status,
	}
	if exitCode != nil {
		result["exit_code"] = *exitCode
		result["output"] = output
		result["duration_ms"] = *durationMs
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to marshal response: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// TaskTreeTool returns a hierarchical tree of tasks for the current workspace.
// Parameters: none.
func TaskTreeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get current workspace branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Retrieve task tree
	tree, err := db.GetTaskTree(dbPath, branch)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to retrieve task tree: %v", err), nil
	}

	// Marshal to JSON
	treeJSON, err := json.Marshal(tree)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to marshal task tree: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(treeJSON)),
		},
	}, nil
}
