package mcp

import (
	"context"
	"encoding/json"

	"agit/internal/db"
	"agit/internal/git"

	"github.com/mark3labs/mcp-go/mcp"
)

// CheckpointTool creates a checkpoint (git commit + agent state snapshot).
// Parameters:
// - message (string, required): Commit message for the checkpoint
func CheckpointTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// Parse required message
	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter 'message'"), nil
	}

	// Acquire Git lock before any repository modification
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to acquire lock: %v", err), nil
	}
	defer unlock()

	// Commit all changes
	commitHash, err := git.CommitAll(message)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to commit changes: %v", err), nil
	}

	// Get workspace snapshot as agent state
	agentState, err := db.GetWorkspaceSnapshot(dbPath, branch)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get workspace snapshot: %v", err), nil
	}

	// Create checkpoint record
	checkpointID, err := db.CreateCheckpoint(dbPath, branch, commitHash, message, agentState)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to create checkpoint: %v", err), nil
	}

	result := map[string]interface{}{
		"checkpoint_id": checkpointID,
		"branch":        branch,
		"commit_hash":   commitHash,
		"message":       message,
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

// RestoreTool restores the workspace to the latest checkpoint.
// No parameters.
func RestoreTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// Get latest checkpoint
	cp, err := db.GetLatestCheckpoint(dbPath, branch)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get latest checkpoint: %v", err), nil
	}
	if cp == nil {
		return mcp.NewToolResultError("no checkpoint found for this workspace"), nil
	}

	// Acquire Git lock before any repository modification
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to acquire lock: %v", err), nil
	}
	defer unlock()

	// Reset Git to the checkpoint commit
	if err := git.ResetToCommit(cp.CommitHash); err != nil {
		return mcp.NewToolResultErrorf("failed to reset to commit %s: %v", cp.CommitHash, err), nil
	}

	// Restore workspace state from the checkpoint's agent state snapshot
	if len(cp.AgentState) > 0 {
		if err := db.RestoreWorkspaceFromSnapshot(dbPath, branch, cp.AgentState); err != nil {
			return mcp.NewToolResultErrorf("failed to restore workspace state: %v", err), nil
		}
	}

	result := map[string]interface{}{
		"checkpoint_id": cp.ID,
		"branch":        branch,
		"commit_hash":   cp.CommitHash,
		"message":       cp.Message,
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
