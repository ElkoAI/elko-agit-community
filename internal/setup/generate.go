package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agit/internal/manifest"
)

// GenerateManifest creates a default manifest for the detected tech stack.
// If stack is nil, a generic manifest is returned.
func GenerateManifest(stack *TechStack) *manifest.Manifest {
	tasks := []manifest.ManifestTask{
		{
			ID:           1,
			Description:  "Initialize agit and verify setup",
			Test:         suggestTestCommand(stack),
			Dependencies: []int64{},
		},
		{
			ID:           2,
			Description:  "Run project tests to ensure everything works",
			Test:         suggestTestCommand(stack),
			Dependencies: []int64{1},
		},
	}

	return &manifest.Manifest{
		Version:     "1.0.0",
		Name:        filepath.Base(GetCurrentDir()),
		Description: fmt.Sprintf("%s project managed by agit", getLanguage(stack)),
		Tasks:       tasks,
		Agents: manifest.AgentsConfig{
			Preferred:    "claude-code",
			Integrations: []string{"opencode", "cursor"},
		},
	}
}

// GenerateManifestCustom creates a manifest with custom project name, language, and test command.
func GenerateManifestCustom(projectName, language, testCommand string) *manifest.Manifest {
	tasks := []manifest.ManifestTask{
		{
			ID:           1,
			Description:  "Initialize agit and verify setup",
			Test:         testCommand,
			Dependencies: []int64{},
		},
		{
			ID:           2,
			Description:  "Run project tests to ensure everything works",
			Test:         testCommand,
			Dependencies: []int64{1},
		},
	}

	desc := fmt.Sprintf("%s project managed by agit", language)
	if language == "" {
		desc = "Project managed by agit"
	}

	return &manifest.Manifest{
		Version:     "1.0.0",
		Name:        projectName,
		Description: desc,
		Tasks:       tasks,
		Agents: manifest.AgentsConfig{
			Preferred:    "claude-code",
			Integrations: []string{"opencode", "cursor"},
		},
	}
}

// WriteManifest writes the manifest to .agit-manifest.json in the current directory.
// If the file already exists, it returns an error unless overwrite is true.
func WriteManifest(m *manifest.Manifest, overwrite bool) error {
	filename := ".agit-manifest.json"
	if _, err := os.Stat(filename); err == nil && !overwrite {
		return fmt.Errorf("%s already exists", filename)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}
	return nil
}

// suggestTestCommand returns a test command based on the stack.
func suggestTestCommand(stack *TechStack) string {
	if stack != nil {
		return stack.TestCommand
	}
	return "echo 'No tests configured'"
}

// getLanguage returns the language name or "Unknown".
func getLanguage(stack *TechStack) string {
	if stack != nil {
		return stack.Language
	}
	return "Unknown"
}

// GetCurrentDir returns the base name of the current working directory.
func GetCurrentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "project"
	}
	return filepath.Base(wd)
}
