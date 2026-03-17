package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AgentConfig represents the configuration for an AI agent integration.
type AgentConfig struct {
	Name    string   `json:"name,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// GenerateAgentConfigs creates default MCP configuration files for
// Claude Code, OpenCode, and Cursor (if they don't already exist).
func GenerateAgentConfigs(overwrite bool) error {
	configs := []struct {
		dir      string
		filename string
		config   interface{}
	}{
		{
			dir:      ".claude",
			filename: "mcp.json",
			config: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"agit": map[string]interface{}{
						"command": "agit-mcp",
						"args":    []string{"--stdio"},
					},
				},
			},
		},
		{
			dir:      ".opencode/plugins",
			filename: "agit.json",
			config: map[string]interface{}{
				"name": "agit",
				"mcp": map[string]interface{}{
					"server": map[string]interface{}{
						"command": "agit-mcp",
						"args":    []string{"--stdio"},
					},
				},
			},
		},
		{
			dir:      ".cursor",
			filename: "mcp.json",
			config: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"agit": map[string]interface{}{
						"command": "agit-mcp",
						"args":    []string{"--stdio"},
					},
				},
			},
		},
	}

	for _, cfg := range configs {
		if err := writeConfigFile(cfg.dir, cfg.filename, cfg.config, overwrite); err != nil {
			return fmt.Errorf("failed to write %s/%s: %w", cfg.dir, cfg.filename, err)
		}
	}
	return nil
}

// writeConfigFile writes a JSON config file, creating parent directories if needed.
func writeConfigFile(dir, filename string, config interface{}, overwrite bool) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil && !overwrite {
		// File exists and we shouldn't overwrite
		return nil
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}
