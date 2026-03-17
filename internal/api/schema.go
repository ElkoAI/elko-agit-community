// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.

package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Tool represents a CLI/MCP tool definition.
type Tool struct {
	// Name is the canonical identifier (e.g., "task_new").
	Name string `json:"name"`
	// Description explains what the tool does.
	Description string `json:"description"`
	// CLICommand is the command template with {param} placeholders.
	// Example: "agit task new --test '{test_command}' '{description}'"
	CLICommand string `json:"cli_command,omitempty"`
	// MCPTool is the corresponding MCP tool name (if any).
	MCPTool string `json:"mcp_tool,omitempty"`
	// Parameters list of expected inputs.
	Parameters []Parameter `json:"parameters,omitempty"`
	// Output describes the expected output format.
	Output Output `json:"output,omitempty"`
}

// Parameter defines a single input parameter.
type Parameter struct {
	// Name is the parameter key (e.g., "description").
	Name string `json:"name"`
	// Type is the data type: "string", "integer", "boolean", "array".
	Type string `json:"type"`
	// Required indicates whether the parameter must be provided.
	Required bool `json:"required,omitempty"`
	// CLIFlag is the flag used in CLI (e.g., "--test").
	// If empty, the parameter is positional.
	CLIFlag string `json:"cli_flag,omitempty"`
	// Positional is the 1‑based index for positional arguments.
	// Zero means not positional (flag‑based).
	Positional int `json:"positional,omitempty"`
	// UILabel is a human‑readable label for UI forms.
	UILabel string `json:"ui_label,omitempty"`
	// UIHint is a short hint for UI forms.
	UIHint string `json:"ui_hint,omitempty"`
	// DefaultValue is a default value as JSON string.
	DefaultValue string `json:"default_value,omitempty"`
	// Enum lists allowed values (for string/integer types).
	Enum []string `json:"enum,omitempty"`
}

// Output describes the tool's output format.
type Output struct {
	// Format is "json", "text", "table", or "mixed".
	Format string `json:"format,omitempty"`
	// SuccessMessage is a template shown on success.
	SuccessMessage string `json:"success_message,omitempty"`
}

// Schema is the root container for all tool definitions.
type Schema struct {
	// Version of the schema format.
	Version string `json:"version"`
	// Tools list of all available tools.
	Tools []Tool `json:"tools"`
}

// LoadSchema reads a JSON schema file from the given path.
func LoadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema file: %w", err)
	}
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("decode schema JSON: %w", err)
	}
	return &schema, nil
}

// LoadEmbeddedSchema reads the default schema embedded in the binary.
// For now returns nil; we can embed later.
func LoadEmbeddedSchema() (*Schema, error) {
	return nil, fmt.Errorf("embedded schema not yet implemented")
}

// FindTool returns the tool with the given name, or nil if not found.
func (s *Schema) FindTool(name string) *Tool {
	for _, t := range s.Tools {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// SchemaPath returns the absolute path to the schema directory.
// It looks for schema/ relative to the executable.
func SchemaPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot locate executable: %w", err)
	}
	dir := filepath.Dir(exe)
	// If we're in a development environment, try project root.
	candidate := filepath.Join(dir, "schema")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	// Fall back to current working directory.
	cwd, _ := os.Getwd()
	candidate = filepath.Join(cwd, "schema")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("schema directory not found")
}
