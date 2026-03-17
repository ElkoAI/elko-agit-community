package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateTask(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Initialize database
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Create a workspace first
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/test", "abc123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Create task without parent
	taskID, err := CreateTask(dbPath, "feat/test", "Implement feature X", "go test ./...", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if taskID <= 0 {
		t.Errorf("Expected positive task ID, got %d", taskID)
	}

	// Verify task exists and has correct fields
	var description, status, testCommand sql.NullString
	var parentTaskID sql.NullInt64
	var createdAt time.Time
	err = db.QueryRow(`SELECT description, status, test_command, parent_task_id, created_at FROM tasks WHERE id = ?`, taskID).
		Scan(&description, &status, &testCommand, &parentTaskID, &createdAt)
	if err != nil {
		t.Fatalf("Failed to query created task: %v", err)
	}
	if description.String != "Implement feature X" {
		t.Errorf("Expected description 'Implement feature X', got %q", description.String)
	}
	if status.String != "pending" {
		t.Errorf("Expected status 'pending', got %q", status.String)
	}
	if !testCommand.Valid || testCommand.String != "go test ./..." {
		t.Errorf("Expected test_command 'go test ./...', got %v", testCommand)
	}
	if parentTaskID.Valid {
		t.Errorf("Expected null parent_task_id, got %v", parentTaskID)
	}
	if time.Since(createdAt) > 5*time.Second {
		t.Errorf("CreatedAt is too old: %v", createdAt)
	}
}

func TestCreateTaskWithParent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/parent", "def456", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Create parent task
	parentID, err := CreateTask(dbPath, "feat/parent", "Parent task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask for parent failed: %v", err)
	}

	// Create child task referencing parent
	childID, err := CreateTask(dbPath, "feat/parent", "Child task", "npm test", sql.NullInt64{Int64: parentID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask for child failed: %v", err)
	}

	// Verify child's parent_task_id
	var parentTaskID sql.NullInt64
	err = db.QueryRow(`SELECT parent_task_id FROM tasks WHERE id = ?`, childID).Scan(&parentTaskID)
	if err != nil {
		t.Fatalf("Failed to query child task: %v", err)
	}
	if !parentTaskID.Valid || parentTaskID.Int64 != parentID {
		t.Errorf("Expected parent_task_id %d, got %v", parentID, parentTaskID)
	}
}
func TestCreateTaskParentValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create two workspaces
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/ws1", "abc123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace 1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/ws2", "def456", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace 2: %v", err)
	}

	// Create parent task in workspace 1
	parentID, err := CreateTask(dbPath, "feat/ws1", "Parent task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask for parent failed: %v", err)
	}

	// Test 1: child with valid parent (same workspace) should succeed
	childID, err := CreateTask(dbPath, "feat/ws1", "Child task", "npm test", sql.NullInt64{Int64: parentID, Valid: true})
	if err != nil {
		t.Errorf("Creating child with valid parent should succeed, got error: %v", err)
	}
	if childID <= 0 {
		t.Error("Expected positive child task ID")
	}

	// Test 2: child with non-existent parent should fail
	_, err = CreateTask(dbPath, "feat/ws1", "Child task", "", sql.NullInt64{Int64: 9999, Valid: true})
	if err == nil {
		t.Error("Expected error when parent task does not exist")
	}

	// Test 3: child with parent from different workspace should fail
	_, err = CreateTask(dbPath, "feat/ws2", "Child task", "", sql.NullInt64{Int64: parentID, Valid: true})
	if err == nil {
		t.Error("Expected error when parent belongs to different workspace")
	}

	// Test 4: child with null parent should succeed (already tested in TestCreateTask)
}
func TestGetSubtasks(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create workspace
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/subtasks", "abc123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Create parent task
	parentID, err := CreateTask(dbPath, "feat/subtasks", "Parent task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask for parent failed: %v", err)
	}

	// Create child tasks
	var childIDs []int64
	for i := 1; i <= 3; i++ {
		id, err := CreateTask(dbPath, "feat/subtasks", fmt.Sprintf("Child %d", i), "", sql.NullInt64{Int64: parentID, Valid: true})
		if err != nil {
			t.Fatalf("CreateTask for child %d failed: %v", i, err)
		}
		childIDs = append(childIDs, id)
	}

	// Also create a task with no parent (should not appear in subtasks)
	unrelatedID, err := CreateTask(dbPath, "feat/subtasks", "Unrelated", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask for unrelated failed: %v", err)
	}

	// Get subtasks
	subtasks, err := GetSubtasks(dbPath, parentID)
	if err != nil {
		t.Fatalf("GetSubtasks failed: %v", err)
	}

	if len(subtasks) != 3 {
		t.Errorf("Expected 3 subtasks, got %d", len(subtasks))
	}

	// Verify each subtask
	found := make(map[int64]bool)
	for _, st := range subtasks {
		if st.WorkspaceBranch != "feat/subtasks" {
			t.Errorf("Subtask workspace branch mismatch: %s", st.WorkspaceBranch)
		}
		if !st.ParentTaskID.Valid || st.ParentTaskID.Int64 != parentID {
			t.Errorf("Subtask parent_task_id mismatch: %v", st.ParentTaskID)
		}
		found[st.ID] = true
	}
	for _, cid := range childIDs {
		if !found[cid] {
			t.Errorf("Child task %d not found in subtasks", cid)
		}
	}
	if found[unrelatedID] {
		t.Errorf("Unrelated task %d incorrectly included in subtasks", unrelatedID)
	}

	// Get subtasks for non-existent parent returns empty slice
	empty, err := GetSubtasks(dbPath, 9999)
	if err != nil {
		t.Fatalf("GetSubtasks for non-existent parent should not error, got: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Expected empty slice for non-existent parent, got %d tasks", len(empty))
	}
}

func TestCreateTaskFailsForMissingWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	_, err := CreateTask(dbPath, "nonexistent", "Some task", "", sql.NullInt64{})
	if err == nil {
		t.Fatal("Expected error when workspace does not exist")
	}
}

func TestListTasks(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/list", "xyz789", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Insert tasks directly via SQL to control statuses
	_, err = db.Exec(`
		INSERT INTO tasks (workspace_branch, description, status, test_command, created_at)
		VALUES 
			('feat/list', 'Task 1', 'pending', 'go test 1', ?),
			('feat/list', 'Task 2', 'active', 'go test 2', ?),
			('feat/list', 'Task 3', 'done', 'go test 3', ?),
			('feat/list', 'Task 4', 'pending', 'go test 4', ?)
	`, time.Now(), time.Now(), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test tasks: %v", err)
	}

	// List all tasks
	tasks, err := ListTasks(dbPath, "feat/list", "", "")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(tasks))
	}
	// Should be ordered by created_at DESC (newest first)
	if tasks[0].Description != "Task 4" {
		t.Errorf("Expected newest task first, got %q", tasks[0].Description)
	}

	// List only pending tasks
	pending, err := ListTasks(dbPath, "feat/list", "pending", "")
	if err != nil {
		t.Fatalf("ListTasks with filter failed: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(pending))
	}
	for _, tsk := range pending {
		if tsk.Status != "pending" {
			t.Errorf("Expected status 'pending', got %q", tsk.Status)
		}
	}

	// List for non-existent workspace returns empty slice
	empty, err := ListTasks(dbPath, "missing", "", "")
	if err != nil {
		t.Fatalf("ListTasks for missing workspace failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Expected empty slice for missing workspace, got %d tasks", len(empty))
	}
}

func TestStartTask(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create workspace with initial current_task_id NULL
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/start", "abc123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Create a pending task
	taskID, err := CreateTask(dbPath, "feat/start", "Test task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Start the task
	err = StartTask(dbPath, "feat/start", taskID)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Verify task status updated
	var status string
	err = db.QueryRow(`SELECT status FROM tasks WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query task: %v", err)
	}
	if status != "active" {
		t.Errorf("Expected task status 'active', got %q", status)
	}

	// Verify workspace current_task_id updated
	var currentTaskID sql.NullInt64
	err = db.QueryRow(`SELECT current_task_id FROM workspaces WHERE branch = ?`, "feat/start").Scan(&currentTaskID)
	if err != nil {
		t.Fatalf("Failed to query workspace: %v", err)
	}
	if !currentTaskID.Valid || currentTaskID.Int64 != taskID {
		t.Errorf("Expected workspace current_task_id = %d, got %v", taskID, currentTaskID)
	}

	// Starting a non-existent task should fail
	err = StartTask(dbPath, "feat/start", 9999)
	if err == nil {
		t.Error("Expected error when starting non-existent task")
	}

	// Starting a task from wrong workspace should fail
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"other", "def456", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert other workspace: %v", err)
	}
	otherTaskID, err := CreateTask(dbPath, "other", "Other task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask for other workspace failed: %v", err)
	}
	err = StartTask(dbPath, "feat/start", otherTaskID)
	if err == nil {
		t.Error("Expected error when starting task belonging to different workspace")
	}
}

func TestDoneTask(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create workspace with a current_task_id set
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at, current_task_id) VALUES (?, ?, ?, ?, ?)`,
		"feat/done", "abc123", time.Now(), time.Now(), 42) // dummy current_task_id
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Create an active task
	taskID, err := CreateTask(dbPath, "feat/done", "Test task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	_, err = db.Exec(`UPDATE tasks SET status = 'active' WHERE id = ?`, taskID)
	if err != nil {
		t.Fatalf("Failed to set task active: %v", err)
	}

	// Mark task as done
	err = DoneTask(dbPath, "feat/done", taskID)
	if err != nil {
		t.Fatalf("DoneTask failed: %v", err)
	}

	// Verify task status and completed_at
	var status string
	var completedAt sql.NullTime
	err = db.QueryRow(`SELECT status, completed_at FROM tasks WHERE id = ?`, taskID).
		Scan(&status, &completedAt)
	if err != nil {
		t.Fatalf("Failed to query task: %v", err)
	}
	if status != "done" {
		t.Errorf("Expected task status 'done', got %q", status)
	}
	if !completedAt.Valid {
		t.Error("Expected completed_at to be set")
	}

	// Verify workspace current_task_id cleared
	var currentTaskID sql.NullInt64
	err = db.QueryRow(`SELECT current_task_id FROM workspaces WHERE branch = ?`, "feat/done").Scan(&currentTaskID)
	if err != nil {
		t.Fatalf("Failed to query workspace: %v", err)
	}
	if currentTaskID.Valid {
		t.Errorf("Expected workspace current_task_id NULL after done, got %v", currentTaskID)
	}

	// Done on non-existent task should fail
	err = DoneTask(dbPath, "feat/done", 9999)
	if err == nil {
		t.Error("Expected error when marking non-existent task done")
	}

	// Done on task from wrong workspace should fail
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"other", "def456", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert other workspace: %v", err)
	}
	otherTaskID, err := CreateTask(dbPath, "other", "Other task", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask for other workspace failed: %v", err)
	}
	err = DoneTask(dbPath, "feat/done", otherTaskID)
	if err == nil {
		t.Error("Expected error when marking task belonging to different workspace done")
	}
}

func TestGetTaskTree(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create workspace
	_, err = db.Exec(`INSERT INTO workspaces (branch, last_commit, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"feat/tree", "abc123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Create tasks with parent-child relationships
	// root1 (no parent)
	root1ID, err := CreateTask(dbPath, "feat/tree", "Root task 1", "", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask root1 failed: %v", err)
	}
	// root2 (no parent)
	root2ID, err := CreateTask(dbPath, "feat/tree", "Root task 2", "go test ./...", sql.NullInt64{})
	if err != nil {
		t.Fatalf("CreateTask root2 failed: %v", err)
	}
	// child1 of root1
	child1ID, err := CreateTask(dbPath, "feat/tree", "Child task 1", "", sql.NullInt64{Int64: root1ID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask child1 failed: %v", err)
	}
	// child2 of root1
	child2ID, err := CreateTask(dbPath, "feat/tree", "Child task 2", "echo test", sql.NullInt64{Int64: root1ID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask child2 failed: %v", err)
	}
	// grandchild of child1
	grandchildID, err := CreateTask(dbPath, "feat/tree", "Grandchild task", "", sql.NullInt64{Int64: child1ID, Valid: true})
	if err != nil {
		t.Fatalf("CreateTask grandchild failed: %v", err)
	}

	// Get task tree
	tree, err := GetTaskTree(dbPath, "feat/tree")
	if err != nil {
		t.Fatalf("GetTaskTree failed: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("Expected 2 root nodes, got %d", len(tree))
	}

	// Find root1 and root2
	var root1, root2 *TaskTreeNode
	for _, root := range tree {
		if root.ID == root1ID {
			root1 = root
		} else if root.ID == root2ID {
			root2 = root
		}
	}
	if root1 == nil {
		t.Fatal("root1 not found in tree")
	}
	if root2 == nil {
		t.Fatal("root2 not found in tree")
	}

	// Verify root1 fields
	if root1.Description != "Root task 1" {
		t.Errorf("root1 description mismatch: expected %q, got %q", "Root task 1", root1.Description)
	}
	if root1.Status != "pending" {
		t.Errorf("root1 status mismatch: expected 'pending', got %q", root1.Status)
	}
	if root1.TestCommand.Valid {
		t.Error("root1 test_command should be empty")
	}
	if root1.ParentTaskID.Valid {
		t.Error("root1 parent_task_id should be invalid")
	}
	// Verify root1 children
	if len(root1.Children) != 2 {
		t.Fatalf("root1 expected 2 children, got %d", len(root1.Children))
	}
	var child1, child2 *TaskTreeNode
	for _, child := range root1.Children {
		if child.ID == child1ID {
			child1 = child
		} else if child.ID == child2ID {
			child2 = child
		}
	}
	if child1 == nil {
		t.Fatal("child1 not found under root1")
	}
	if child2 == nil {
		t.Fatal("child2 not found under root1")
	}
	if child1.Description != "Child task 1" {
		t.Errorf("child1 description mismatch")
	}
	if child2.Description != "Child task 2" {
		t.Errorf("child2 description mismatch")
	}
	if child2.TestCommand.String != "echo test" {
		t.Errorf("child2 test_command mismatch: expected %q, got %q", "echo test", child2.TestCommand.String)
	}
	// Verify child1 has grandchild
	if len(child1.Children) != 1 {
		t.Fatalf("child1 expected 1 grandchild, got %d", len(child1.Children))
	}
	if child1.Children[0].ID != grandchildID {
		t.Errorf("grandchild ID mismatch")
	}
	if child1.Children[0].Description != "Grandchild task" {
		t.Errorf("grandchild description mismatch")
	}

	// Verify root2 has no children
	if len(root2.Children) != 0 {
		t.Errorf("root2 expected no children, got %d", len(root2.Children))
	}
	if root2.TestCommand.String != "go test ./..." {
		t.Errorf("root2 test_command mismatch")
	}

	// Test empty tree for non-existent workspace
	empty, err := GetTaskTree(dbPath, "missing")
	if err != nil {
		t.Fatalf("GetTaskTree for missing workspace failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Expected empty tree for missing workspace, got %d roots", len(empty))
	}
}

func TestGetTask(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/agit.db"
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	branch := "test-branch"
	commit := "abc123"
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, branch, commit)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create a task
	taskID, err := CreateTask(dbPath, ws.Branch, "Test task", "go test ./...", sql.NullInt64{})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Retrieve the task
	task, err := GetTask(dbPath, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.ID != taskID {
		t.Errorf("task ID mismatch: expected %d, got %d", taskID, task.ID)
	}
	if task.Description != "Test task" {
		t.Errorf("description mismatch: expected %q, got %q", "Test task", task.Description)
	}
	if task.Status != "pending" {
		t.Errorf("status mismatch: expected %q, got %q", "pending", task.Status)
	}
	if !task.TestCommand.Valid || task.TestCommand.String != "go test ./..." {
		t.Errorf("test_command mismatch: expected %q, got %v", "go test ./...", task.TestCommand)
	}

	// Test non-existent task
	nonexistent, err := GetTask(dbPath, 9999)
	if err != nil {
		t.Fatalf("GetTask for non-existent task failed: %v", err)
	}
	if nonexistent != nil {
		t.Errorf("expected nil for non-existent task, got %+v", nonexistent)
	}
}

func TestUpdateTaskStatusAndResult(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/agit.db"
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	branch := "test-branch"
	commit := "abc123"
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, branch, commit)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create a task
	taskID, err := CreateTask(dbPath, ws.Branch, "Update test task", "echo test", sql.NullInt64{})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Mark task as active
	err = UpdateTaskStatus(dbPath, ws.Branch, taskID, "active")
	if err != nil {
		t.Fatalf("UpdateTaskStatus to active failed: %v", err)
	}
	task, _ := GetTask(dbPath, taskID)
	if task.Status != "active" {
		t.Errorf("expected status 'active', got %q", task.Status)
	}

	// Mark task as done
	err = UpdateTaskStatus(dbPath, ws.Branch, taskID, "done")
	if err != nil {
		t.Fatalf("UpdateTaskStatus to done failed: %v", err)
	}
	task, _ = GetTask(dbPath, taskID)
	if task.Status != "done" {
		t.Errorf("expected status 'done', got %q", task.Status)
	}
	if !task.CompletedAt.Valid {
		t.Error("completed_at should be set for done status")
	}

	// Update test result
	testResult := json.RawMessage(`{"exit_code": 0, "output": "test passed"}`)
	err = UpdateTaskTestResult(dbPath, taskID, testResult)
	if err != nil {
		t.Fatalf("UpdateTaskTestResult failed: %v", err)
	}
	task, _ = GetTask(dbPath, taskID)
	if string(task.TestResult) != string(testResult) {
		t.Errorf("test result mismatch: expected %s, got %s", testResult, task.TestResult)
	}

	// Test invalid status
	err = UpdateTaskStatus(dbPath, ws.Branch, taskID, "invalid")
	if err == nil {
		t.Error("expected error for invalid status")
	}

	// Test non-existent task
	err = UpdateTaskStatus(dbPath, ws.Branch, 9999, "pending")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestListTasksFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/agit.db"
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	branch := "test-branch"
	commit := "abc123"
	ws, err := GetOrCreateWorkspaceForBranch(dbPath, branch, commit)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create tasks with different statuses
	task1, _ := CreateTask(dbPath, ws.Branch, "Pending task", "", sql.NullInt64{})
	task2, _ := CreateTask(dbPath, ws.Branch, "Active task", "", sql.NullInt64{})
	task3, _ := CreateTask(dbPath, ws.Branch, "Done task", "", sql.NullInt64{})
	task4, _ := CreateTask(dbPath, ws.Branch, "Failed task", "", sql.NullInt64{})

	// Update statuses
	UpdateTaskStatus(dbPath, ws.Branch, task2, "active")
	UpdateTaskStatus(dbPath, ws.Branch, task3, "done")
	UpdateTaskStatus(dbPath, ws.Branch, task4, "failed")

	// List all tasks
	all, err := ListTasks(dbPath, ws.Branch, "", "")
	if err != nil {
		t.Fatalf("ListTasks all failed: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(all))
	}

	// List pending tasks
	pending, err := ListTasks(dbPath, ws.Branch, "pending", "")
	if err != nil {
		t.Fatalf("ListTasks pending failed: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending task, got %d", len(pending))
	}
	if pending[0].ID != task1 {
		t.Errorf("pending task ID mismatch: expected %d, got %d", task1, pending[0].ID)
	}

	// List active tasks
	active, err := ListTasks(dbPath, ws.Branch, "active", "")
	if err != nil {
		t.Fatalf("ListTasks active failed: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active task, got %d", len(active))
	}
	if active[0].ID != task2 {
		t.Errorf("active task ID mismatch: expected %d, got %d", task2, active[0].ID)
	}

	// List done tasks
	done, err := ListTasks(dbPath, ws.Branch, "done", "")
	if err != nil {
		t.Fatalf("ListTasks done failed: %v", err)
	}
	if len(done) != 1 {
		t.Errorf("expected 1 done task, got %d", len(done))
	}
	if done[0].ID != task3 {
		t.Errorf("done task ID mismatch: expected %d, got %d", task3, done[0].ID)
	}

	// List failed tasks
	failed, err := ListTasks(dbPath, ws.Branch, "failed", "")
	if err != nil {
		t.Fatalf("ListTasks failed failed: %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("expected 1 failed task, got %d", len(failed))
	}
	if failed[0].ID != task4 {
		t.Errorf("failed task ID mismatch: expected %d, got %d", task4, failed[0].ID)
	}

	// Non-existent status filter returns empty
	none, err := ListTasks(dbPath, ws.Branch, "nonexistent", "")
	if err != nil {
		t.Fatalf("ListTasks with non-existent status failed: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected empty list for non-existent status, got %d", len(none))
	}
}
