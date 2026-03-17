// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.
package main

import (
	"log"

	"agit/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"agit",
		"0.3.0",
	)

	// Register tools
	mcp.RegisterTools(s)

	// Start server with stdio transport
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}
