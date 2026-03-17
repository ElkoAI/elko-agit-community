package mcp

import (
	"context"
	"encoding/json"

	"agit/internal/db"
	"agit/internal/git"

	"github.com/mark3labs/mcp-go/mcp"
)

// WorkspaceStatusTool returns the current workspace status.
func WorkspaceStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Get or create workspace
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		return nil, err
	}

	// Marshal workspace to JSON for response
	wsJSON, err := json.Marshal(ws)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(wsJSON)),
		},
	}, nil
}
