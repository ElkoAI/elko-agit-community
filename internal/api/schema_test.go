// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.

package api

import (
	"testing"
)

func TestLoadSchema(t *testing.T) {
	schema, err := LoadSchema("../../schema/agit-api.json")
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}
	if schema.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", schema.Version)
	}
	if len(schema.Tools) == 0 {
		t.Error("Expected at least one tool definition")
	}
	// Verify a few known tools exist
	expectedTools := []string{"workspace_status", "task_new", "task_list", "task_start", "task_done", "checkpoint", "restore", "trace"}
	for _, name := range expectedTools {
		tool := schema.FindTool(name)
		if tool == nil {
			t.Errorf("Expected tool %q not found in schema", name)
		}
	}
	// Verify task_new parameters
	taskNew := schema.FindTool("task_new")
	if taskNew == nil {
		t.Fatal("task_new tool missing")
	}
	if len(taskNew.Parameters) != 3 {
		t.Errorf("task_new expected 3 parameters, got %d", len(taskNew.Parameters))
	}
	// Check description parameter
	descParam := taskNew.Parameters[0]
	if descParam.Name != "description" {
		t.Errorf("first parameter should be 'description', got %q", descParam.Name)
	}
	if !descParam.Required {
		t.Error("description parameter should be required")
	}
	if descParam.Positional != 1 {
		t.Errorf("description positional should be 1, got %d", descParam.Positional)
	}
}

func TestSchemaPath(t *testing.T) {
	path, err := SchemaPath()
	if err != nil {
		t.Skipf("SchemaPath failed (maybe not in project root): %v", err)
	}
	t.Logf("Schema directory: %s", path)
}
