package mcp

import (
	"context"
	"encoding/json"

	"agit/internal/git"
	"agit/internal/manifest"

	"github.com/mark3labs/mcp-go/mcp"
)

// ManifestTool loads and applies the agit manifest from default location.
func ManifestTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		return nil, err
	}

	// Load and apply manifest
	if err := manifest.ApplyDefault(dbPath); err != nil {
		return nil, err
	}

	// Return success message
	result := map[string]string{
		"status":  "manifest_applied",
		"message": "Manifest loaded and tasks created successfully",
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}
