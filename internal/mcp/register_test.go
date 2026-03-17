package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestRegisterTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0")
	RegisterTools(s)
	// If registration fails, it would panic; we just verify no panic.
	// Could also check that tools are registered by inspecting server internals,
	// but that's not necessary for coverage.
}
