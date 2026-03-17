package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agit/internal/db"
)

// Manifest represents the agit project configuration file.
type Manifest struct {
	Version     string         `json:"version"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Tasks       []ManifestTask `json:"tasks"`
	Agents      AgentsConfig   `json:"agents"`
}

// ManifestTask defines a task in the manifest.
type ManifestTask struct {
	ID           int64   `json:"id"`
	Description  string  `json:"description"`
	Test         string  `json:"test"`         // test command
	Dependencies []int64 `json:"dependencies"` // IDs of tasks that must be completed before this one
}

// AgentsConfig defines preferred agent integrations.
type AgentsConfig struct {
	Preferred    string   `json:"preferred"`    // e.g., "claude-code", "opencode"
	Integrations []string `json:"integrations"` // e.g., ["opencode", "cursor"]
}

// Load reads a manifest file from the given path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if m.Version == "" {
		return nil, fmt.Errorf("manifest version is required")
	}
	if m.Name == "" {
		return nil, fmt.Errorf("manifest name is required")
	}
	// Validate tasks
	for i, task := range m.Tasks {
		if task.Description == "" {
			return nil, fmt.Errorf("task %d: description is required", task.ID)
		}
		if task.ID <= 0 {
			return nil, fmt.Errorf("task %d: ID must be positive", i+1)
		}
		// Ensure IDs are unique (though map validation could be added)
	}
	return &m, nil
}

// LoadFromDefault searches for a manifest file in the current directory.
// It looks for `.agit-manifest.json` (preferred) and `agit-manifest.json`.
func LoadFromDefault() (*Manifest, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	preferred := filepath.Join(cwd, ".agit-manifest.json")
	alternate := filepath.Join(cwd, "agit-manifest.json")

	var path string
	if _, err := os.Stat(preferred); err == nil {
		path = preferred
	} else if _, err := os.Stat(alternate); err == nil {
		path = alternate
	} else {
		return nil, fmt.Errorf("no manifest file found (looked for .agit-manifest.json and agit-manifest.json)")
	}
	return Load(path)
}

// Apply loads the manifest and creates its tasks in the database for the current workspace.
// It maps manifest task IDs to database task IDs using a provided mapping function.
// If a mapping function is nil, it will create tasks without preserving manifest IDs
// (database auto‑increment will assign new IDs).
func Apply(dbPath string, manifest *Manifest, mapID func(manifestID int64) sql.NullInt64) error {
	// Get current workspace (branch)
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get current workspace: %w", err)
	}

	// Build mapping from manifest task ID to database task ID
	idMap := make(map[int64]int64)

	// Create tasks in order they appear in manifest (assume topological order)
	for _, mt := range manifest.Tasks {
		// Determine parent task ID (if dependencies include a single parent)
		var parentID sql.NullInt64
		if len(mt.Dependencies) > 0 {
			// Look up mapped database ID for the first dependency
			if dbID, ok := idMap[mt.Dependencies[0]]; ok {
				parentID = sql.NullInt64{Int64: dbID, Valid: true}
			}
			// If not yet mapped, we assume manifest tasks are topologically sorted,
			// so parent should have been created earlier.
		}

		// Create task in database
		dbID, err := db.CreateTask(dbPath, ws.Branch, mt.Description, mt.Test, parentID)
		if err != nil {
			return fmt.Errorf("failed to create task %q (manifest ID %d): %w", mt.Description, mt.ID, err)
		}
		idMap[mt.ID] = dbID
	}

	return nil
}

// ApplyDefault loads the manifest from the default location and applies it to the current workspace.
func ApplyDefault(dbPath string) error {
	manifest, err := LoadFromDefault()
	if err != nil {
		return err
	}
	return Apply(dbPath, manifest, nil)
}
