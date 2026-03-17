package mcp

import (
	"context"
	"encoding/json"

	"agit/internal/db"
	"agit/internal/git"

	"github.com/mark3labs/mcp-go/mcp"
)

// TraceTool shows test run history for a task.
// Parameters:
// - task_id (int, required): ID of the task to show traces for
// - limit (int, optional): Maximum number of traces to return (default: all)
// - failed_only (bool, optional): If true, return only traces where exit_code != 0 (failed tests)
func TraceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Parse required task ID
	taskID, err := request.RequireInt("task_id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'task_id'"), nil
	}

	// Optional limit (0 means all)
	limit := request.GetInt("limit", 0)
	// Optional failed_only filter (default false)
	failedOnly := request.GetBool("failed_only", false)

	var traces []db.Trace
	if failedOnly {
		traces, err = db.GetTracesFailed(dbPath, int64(taskID), limit)
	} else {
		traces, err = db.GetTraces(dbPath, int64(taskID), limit)
	}
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get traces: %v", err), nil
	}

	// If no traces, return empty array
	if traces == nil {
		traces = []db.Trace{}
	}

	result := map[string]interface{}{
		"task_id":     taskID,
		"limit":       limit,
		"failed_only": failedOnly,
		"traces":      traces,
		"count":       len(traces),
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
