// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.
package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolHandlers maps tool names to their handler functions.
// This allows direct invocation of MCP tools without going through the MCP server.
var ToolHandlers = map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){}

// registerHandlers populates the ToolHandlers map.
// Called automatically by init() and also by RegisterTools.
func registerHandlers() {
	if len(ToolHandlers) > 0 {
		return // already registered
	}
	ToolHandlers["workspace_status"] = WorkspaceStatusTool
	ToolHandlers["task_new"] = TaskNewTool
	ToolHandlers["task_list"] = TaskListTool
	ToolHandlers["task_start"] = TaskStartTool
	ToolHandlers["task_done"] = TaskDoneTool
	ToolHandlers["checkpoint"] = CheckpointTool
	ToolHandlers["restore"] = RestoreTool
	ToolHandlers["trace"] = TraceTool
	ToolHandlers["task_breakdown"] = TaskBreakdownTool
	ToolHandlers["task_tree"] = TaskTreeTool
	ToolHandlers["manifest"] = ManifestTool
}

func init() {
	registerHandlers()
}

// RegisterTools registers all agit tools with the MCP server.
func RegisterTools(s *server.MCPServer) {
	// Ensure tool handlers are registered
	registerHandlers()

	// workspace_status tool
	tool := mcp.NewTool("workspace_status", mcp.WithDescription("Get current workspace status (branch, current task, agent state, todo list, last commit)."))
	s.AddTool(tool, WorkspaceStatusTool)

	// task_new tool
	type taskNewParams struct {
		Description  string `json:"description"`
		TestCommand  string `json:"test_command,omitempty"`
		ParentTaskID int64  `json:"parent_task_id,omitempty"`
	}
	taskNewTool := mcp.NewTool("task_new",
		mcp.WithDescription("Create a new task in the current workspace."),
		mcp.WithInputSchema[taskNewParams](),
	)
	s.AddTool(taskNewTool, TaskNewTool)

	// task_list tool
	type taskListParams struct {
		Status string `json:"status,omitempty"`
	}
	taskListTool := mcp.NewTool("task_list",
		mcp.WithDescription("List tasks for the current workspace, optionally filtered by status."),
		mcp.WithInputSchema[taskListParams](),
	)
	s.AddTool(taskListTool, TaskListTool)

	// task_start tool
	type taskStartParams struct {
		TaskID int64 `json:"task_id"`
	}
	taskStartTool := mcp.NewTool("task_start",
		mcp.WithDescription("Mark a task as active and set it as the workspace's current task."),
		mcp.WithInputSchema[taskStartParams](),
	)
	s.AddTool(taskStartTool, TaskStartTool)

	// task_done tool
	type taskDoneParams struct {
		TaskID int64 `json:"task_id"`
	}
	taskDoneTool := mcp.NewTool("task_done",
		mcp.WithDescription("Mark a task as done, clear workspace current task, and run test command."),
		mcp.WithInputSchema[taskDoneParams](),
	)
	s.AddTool(taskDoneTool, TaskDoneTool)

	// checkpoint tool
	type checkpointParams struct {
		Message string `json:"message"`
	}
	checkpointTool := mcp.NewTool("checkpoint",
		mcp.WithDescription("Save state + git commit (creates a checkpoint with workspace snapshot)."),
		mcp.WithInputSchema[checkpointParams](),
	)
	s.AddTool(checkpointTool, CheckpointTool)

	// restore tool
	restoreTool := mcp.NewTool("restore",
		mcp.WithDescription("Restore to last checkpoint (git reset + workspace state restore)."),
	)
	s.AddTool(restoreTool, RestoreTool)

	// trace tool
	type traceParams struct {
		TaskID int64 `json:"task_id"`
		Limit  int64 `json:"limit,omitempty"`
	}
	traceTool := mcp.NewTool("trace",
		mcp.WithDescription("Show test run history for a task."),
		mcp.WithInputSchema[traceParams](),
	)
	s.AddTool(traceTool, TraceTool)

	// task_breakdown tool
	type subtaskParam struct {
		Description string `json:"description"`
		TestCommand string `json:"test_command,omitempty"`
	}
	type taskBreakdownParams struct {
		ParentTaskID int64          `json:"parent_task_id"`
		Subtasks     []subtaskParam `json:"subtasks"`
	}
	taskBreakdownTool := mcp.NewTool("task_breakdown",
		mcp.WithDescription("Create subtasks under a parent task."),
		mcp.WithInputSchema[taskBreakdownParams](),
	)
	s.AddTool(taskBreakdownTool, TaskBreakdownTool)

	// task_tree tool
	taskTreeTool := mcp.NewTool("task_tree",
		mcp.WithDescription("Return hierarchical tree of tasks for the current workspace."),
	)
	s.AddTool(taskTreeTool, TaskTreeTool)

	// manifest tool
	manifestTool := mcp.NewTool("manifest",
		mcp.WithDescription("Load and apply agit manifest from default location (.agit-manifest.json)."),
	)
	s.AddTool(manifestTool, ManifestTool)
}
