package manifest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"agit/internal/db"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	// Set default branch to main
	runCommand(t, dir, "git", "config", "init.defaultBranch", "main")
	_, _, err := runCommand(t, dir, "git", "init")
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	// Set user identity for commits
	runCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runCommand(t, dir, "git", "config", "user.name", "Test User")
	// Make an initial commit
	runCommand(t, dir, "git", "commit", "--allow-empty", "-m", "Initial commit")
}

func runCommand(t *testing.T, workdir, name string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestLoadValidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, ".agit-manifest.json")
	manifestContent := `{
		"version": "1.0.0",
		"name": "test",
		"description": "Test manifest",
		"tasks": [
			{
				"id": 1,
				"description": "Task 1",
				"test": "echo test",
				"dependencies": []
			}
		],
		"agents": {
			"preferred": "claude-code",
			"integrations": ["opencode"]
		}
	}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if m.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.Version)
	}
	if m.Name != "test" {
		t.Errorf("expected name test, got %s", m.Name)
	}
	if len(m.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(m.Tasks))
	}
	if m.Tasks[0].ID != 1 {
		t.Errorf("expected task ID 1, got %d", m.Tasks[0].ID)
	}
	if m.Tasks[0].Description != "Task 1" {
		t.Errorf("expected description 'Task 1', got %s", m.Tasks[0].Description)
	}
	if m.Tasks[0].Test != "echo test" {
		t.Errorf("expected test 'echo test', got %s", m.Tasks[0].Test)
	}
	if m.Agents.Preferred != "claude-code" {
		t.Errorf("expected preferred agent claude-code, got %s", m.Agents.Preferred)
	}
	if len(m.Agents.Integrations) != 1 || m.Agents.Integrations[0] != "opencode" {
		t.Errorf("expected integrations [opencode], got %v", m.Agents.Integrations)
	}
}

func TestLoadInvalidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, ".agit-manifest.json")
	// Missing version
	invalidContent := `{
		"name": "test",
		"tasks": []
	}`
	if err := os.WriteFile(manifestPath, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(manifestPath)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestLoadFromDefault(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create .agit-manifest.json
	manifestContent := `{
		"version": "1.0.0",
		"name": "default",
		"description": "Default manifest",
		"tasks": [],
		"agents": {}
	}`
	if err := os.WriteFile(".agit-manifest.json", []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadFromDefault()
	if err != nil {
		t.Fatalf("LoadFromDefault failed: %v", err)
	}
	if m.Name != "default" {
		t.Errorf("expected name default, got %s", m.Name)
	}
}

func TestLoadFromDefaultNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	_, err = LoadFromDefault()
	if err == nil {
		t.Fatal("expected error when no manifest file exists")
	}
}

func TestApplyManifest(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("failed to get workspace: %v", err)
	}
	t.Logf("workspace branch: %s", ws.Branch)
	manifestContent := `{
		"version": "1.0.0",
		"name": "apply-test",
		"description": "Apply test",
		"tasks": [
			{
				"id": 1,
				"description": "First task",
				"test": "echo first",
				"dependencies": []
			},
			{
				"id": 2,
				"description": "Second task",
				"test": "echo second",
				"dependencies": [1]
			}
		],
		"agents": {}
	}`
	manifestPath := filepath.Join(tmpDir, ".agit-manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}
	if err := Apply(dbPath, m, nil); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	tasks, err := db.ListTasks(dbPath, ws.Branch, "", "")
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	t.Logf("found %d tasks", len(tasks))
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
	desc1 := tasks[0].Description
	desc2 := tasks[1].Description
	if !((desc1 == "First task" && desc2 == "Second task") || (desc1 == "Second task" && desc2 == "First task")) {
		t.Errorf("unexpected task descriptions: %s, %s", desc1, desc2)
	}
}

func TestApplyDefault(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, ".git", "agit.db")
	if err := db.InitDatabase(dbPath); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		t.Fatalf("failed to get workspace: %v", err)
	}
	t.Logf("workspace branch: %s", ws.Branch)
	manifestContent := `{
		"version": "1.0.0",
		"name": "default-test",
		"description": "Default test",
		"tasks": [
			{
				"id": 1,
				"description": "Default task",
				"test": "echo default",
				"dependencies": []
			}
		],
		"agents": {}
	}`
	if err := os.WriteFile(".agit-manifest.json", []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyDefault(dbPath); err != nil {
		t.Fatalf("ApplyDefault failed: %v", err)
	}
	tasks, err := db.ListTasks(dbPath, ws.Branch, "", "")
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	t.Logf("found %d tasks", len(tasks))
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Description != "Default task" {
		t.Errorf("expected description 'Default task', got %s", tasks[0].Description)
	}
}

// Test JSON marshalling/unmarshalling
func TestManifestJSONRoundtrip(t *testing.T) {
	original := Manifest{
		Version:     "1.0.0",
		Name:        "roundtrip",
		Description: "Roundtrip test",
		Tasks: []ManifestTask{
			{
				ID:           1,
				Description:  "Task",
				Test:         "go test",
				Dependencies: []int64{},
			},
		},
		Agents: AgentsConfig{
			Preferred:    "claude-code",
			Integrations: []string{"opencode"},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Version != original.Version {
		t.Errorf("version mismatch: %s != %s", decoded.Version, original.Version)
	}
	if decoded.Name != original.Name {
		t.Errorf("name mismatch: %s != %s", decoded.Name, original.Name)
	}
	if len(decoded.Tasks) != len(original.Tasks) {
		t.Errorf("tasks length mismatch: %d != %d", len(decoded.Tasks), len(original.Tasks))
	}
	if decoded.Tasks[0].ID != original.Tasks[0].ID {
		t.Errorf("task ID mismatch: %d != %d", decoded.Tasks[0].ID, original.Tasks[0].ID)
	}
	if decoded.Agents.Preferred != original.Agents.Preferred {
		t.Errorf("preferred agent mismatch: %s != %s", decoded.Agents.Preferred, original.Agents.Preferred)
	}
}
