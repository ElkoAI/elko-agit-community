// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"agit/internal/db"
	"agit/internal/git"
	"agit/internal/manifest"
	"agit/internal/setup"
)

const version = "0.3.0"

// getDBPath returns the absolute path to .git/agit.db in the current directory.
// It exits with an error if not in a git repository.
func getDBPath() string {
	// Find .git directory
	gitDir := ".git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		log.Fatal("Not a git repository (or any of the parent directories): .git not found")
	}
	dbPath := filepath.Join(gitDir, "agit.db")
	return dbPath
}

const hookScript = `#!/bin/bash
# agit post-checkout hook
# Automatically switches workspace when git checkout changes branch

branch=$(git rev-parse --abbrev-ref HEAD)
commit=$(git rev-parse HEAD)
echo "{\"branch\":\"$branch\",\"commit\":\"$commit\"}" | \
  agit-mcp tool workspace_switch 2>/dev/null || true
`

func main() {
	flag.CommandLine.SetInterspersed(true)
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "hook":
		cmdHook(os.Args[2:])
	case "workspace-status":
		cmdWorkspaceStatus(os.Args[2:])
	case "task":
		cmdTask(os.Args[2:])
	case "checkpoint":
		cmdCheckpoint(os.Args[2:])
	case "restore":
		cmdRestore(os.Args[2:])
	case "trace":
		cmdTrace(os.Args[2:])
	case "dashboard":
		cmdDashboard(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
	case "manifest":
		cmdManifest(os.Args[2:])
	case "setup":
		cmdSetup(os.Args[2:])
	case "version":
		fmt.Printf("agit version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	fs.SetInterspersed(true)
	force := fs.Bool("force", false, "Force reinitialization even if database exists")
	manifestFlag := fs.Bool("manifest", false, "Load manifest (.agit-manifest.json) after initialization")
	help := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit init - Initialize agit database in current repository

Usage:
  agit init [options]

Options:
  --force     Force reinitialization even if database exists
  --manifest  Load manifest (.agit-manifest.json) after initialization
  --help      Show this help

Examples:
  agit init
  agit init --force
  agit init --manifest
`)
	}

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *help {
		fs.Usage()
		return
	}

	// Determine database path
	dbPath := ".git/agit.db"
	if cwd, err := os.Getwd(); err == nil {
		dbPath = filepath.Join(cwd, ".git", "agit.db")
	}

	fmt.Printf("Initializing agit database at %s\n", dbPath)

	// Acquire Git lock to prevent concurrent repository modification
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer unlock()

	// If force flag is set, remove existing database first
	if *force {
		if _, err := os.Stat(dbPath); err == nil {
			if err := os.Remove(dbPath); err != nil {
				log.Fatalf("Failed to remove existing database: %v", err)
			}
			fmt.Println("Removed existing database")
		}
	}

	// Initialize database
	if err := db.InitDatabase(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("agit database initialized successfully")

	// Load manifest if requested
	if *manifestFlag {
		fmt.Println("Loading manifest...")
		if err := manifest.ApplyDefault(dbPath); err != nil {
			// Don't fail init if manifest load fails (maybe file not found)
			fmt.Printf("Warning: failed to load manifest: %v\n", err)
		} else {
			fmt.Println("Manifest applied successfully")
		}
	}
}

func cmdHook(args []string) {
	if len(args) < 1 {
		fmt.Print(`agit hook - Manage Git hooks for agit

Usage:
  agit hook <command>

Commands:
  install    Install post-checkout hook for workspace switching
  uninstall  Remove agit post-checkout hook

Examples:
  agit hook install    # Install the hook
  agit hook uninstall  # Remove the hook
`)
		os.Exit(1)
	}

	switch args[0] {
	case "install":
		installHook()
	case "uninstall":
		uninstallHook()
	default:
		fmt.Printf("Unknown hook command: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdWorkspaceStatus(args []string) {
	dbPath := getDBPath()
	ws, err := db.GetOrCreateWorkspace(dbPath)
	if err != nil {
		log.Fatalf("Failed to get workspace status: %v", err)
	}
	// Pretty print as JSON for now
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal workspace: %v", err)
	}
	fmt.Println(string(data))
}

func cmdTask(args []string) {
	if len(args) < 1 {
		fmt.Print(`agit task - Manage tasks in current workspace

Usage:
  agit task <subcommand> [options]

Subcommands:
  new     Create a new task
  list    List tasks (optional: --status pending|active|done|failed)
  start   Mark a task as active
  done    Mark a task as done and run test command
  breakdown Create subtasks under a parent task
  tree    Show task tree for current workspace

Examples:
  agit task new --test "go test ./..." "Implement feature X"
  agit task list --status pending
`)
		os.Exit(1)
	}

	dbPath := getDBPath()
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatalf("Failed to get current branch: %v", err)
	}

	switch args[0] {
	case "new":
		cmdTaskNew(dbPath, branch, args[1:])
	case "list":
		cmdTaskList(dbPath, branch, args[1:])
	case "start":
		cmdTaskStart(dbPath, branch, args[1:])
	case "done":
		cmdTaskDone(dbPath, branch, args[1:])
	case "breakdown":
		cmdTaskBreakdown(dbPath, branch, args[1:])
	case "tree":
		cmdTaskTree(dbPath, branch, args[1:])
	default:
		fmt.Printf("Unknown task subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdTaskNew(dbPath, branch string, args []string) {
	fs := flag.NewFlagSet("task-new", flag.ExitOnError)
	fs.SetInterspersed(true)
	testFlag := fs.String("test", "", "Test command to run when task is marked done")
	parentFlag := fs.Int64("parent", 0, "Parent task ID (optional)")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit task new - Create a new task

Usage:
  agit task new [options] "description"

Options:
  --test <cmd>   Test command to run when task is marked done
  --parent <id>  Parent task ID (optional)

Examples:
  agit task new --test "go test ./..." "Implement feature X"
  agit task new --parent 3 "Subtask"
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag || fs.NArg() == 0 {
		fs.Usage()
		return
	}
	description := fs.Arg(0)
	var parentID sql.NullInt64
	if *parentFlag > 0 {
		parentID = sql.NullInt64{Int64: *parentFlag, Valid: true}
	}
	taskID, err := db.CreateTask(dbPath, branch, description, *testFlag, parentID)
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}
	fmt.Printf("Created task #%d: %s\n", taskID, description)
}

func cmdTaskList(dbPath, branch string, args []string) {
	fs := flag.NewFlagSet("task-list", flag.ExitOnError)
	fs.SetInterspersed(true)
	statusFlag := fs.String("status", "", "Filter by status (pending, active, done, failed)")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit task list - List tasks in current workspace

Usage:
  agit task list [options]

Options:
  --status <status>  Filter by status (pending, active, done, failed)

Examples:
  agit task list
  agit task list --status pending
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}
	tasks, err := db.ListTasks(dbPath, branch, *statusFlag, "")
	if err != nil {
		log.Fatalf("Failed to list tasks: %v", err)
	}
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}
	// Simple table output
	fmt.Printf("%-4s %-10s %-40s %-8s\n", "ID", "Status", "Description", "Created")
	for _, t := range tasks {
		desc := t.Description
		if len(desc) > 37 {
			desc = desc[:34] + "..."
		}
		fmt.Printf("%-4d %-10s %-40s %-8s\n", t.ID, t.Status, desc, t.CreatedAt.Format("2006-01-02"))
	}
}

func cmdTaskStart(dbPath, branch string, args []string) {
	fs := flag.NewFlagSet("task-start", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit task start - Mark a task as active and set as current workspace task

Usage:
  agit task start [options] <task_id>

Options:
  --help  Show this help

Examples:
  agit task start 1
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag || fs.NArg() == 0 {
		fs.Usage()
		return
	}
	taskIDStr := fs.Arg(0)
	var taskID int64
	if _, err := fmt.Sscanf(taskIDStr, "%d", &taskID); err != nil || taskID <= 0 {
		log.Fatalf("Invalid task ID: %s (must be positive integer)", taskIDStr)
	}
	err := db.StartTask(dbPath, branch, taskID)
	if err != nil {
		log.Fatalf("Failed to start task: %v", err)
	}
	fmt.Printf("Task #%d marked as active (current task for workspace)\n", taskID)
}

func cmdTaskDone(dbPath, branch string, args []string) {
	fs := flag.NewFlagSet("task-done", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit task done - Mark a task as done, run its test command, and record trace

Usage:
  agit task done [options] <task_id>

Options:
  --help  Show this help

Examples:
  agit task done 1
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag || fs.NArg() == 0 {
		fs.Usage()
		return
	}
	taskIDStr := fs.Arg(0)
	var taskID int64
	if _, err := fmt.Sscanf(taskIDStr, "%d", &taskID); err != nil || taskID <= 0 {
		log.Fatalf("Invalid task ID: %s (must be positive integer)", taskIDStr)
	}

	// Retrieve task details
	task, err := db.GetTask(dbPath, taskID)
	if err != nil {
		log.Fatalf("Failed to retrieve task: %v", err)
	}
	if task == nil {
		log.Fatalf("Task %d not found", taskID)
	}

	var exitCode *int
	var output string
	var durationMs *int64
	var testResultJSON json.RawMessage

	// Run test command if present
	if task.TestCommand.Valid && task.TestCommand.String != "" {
		start := time.Now()
		cmd := exec.Command("sh", "-c", task.TestCommand.String)
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		elapsed := time.Since(start)
		elapsedMs := elapsed.Milliseconds()
		durationMs = &elapsedMs

		combinedOutput := stdoutBuf.String() + stderrBuf.String()
		output = combinedOutput

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				ec := exitErr.ExitCode()
				exitCode = &ec
			} else {
				// command failed to start or was interrupted
				ec := -1
				exitCode = &ec
				output = err.Error() + "\n" + output
			}
		} else {
			ec := 0
			exitCode = &ec
		}

		// Create trace
		_, err = db.CreateTrace(dbPath, taskID, exitCode, output, durationMs)
		if err != nil {
			log.Fatalf("Failed to create trace: %v", err)
		}

		// Prepare test result JSON
		testResult := map[string]interface{}{
			"exit_code":   exitCode,
			"output":      output,
			"duration_ms": durationMs,
			"run_at":      time.Now().UTC().Format(time.RFC3339),
		}
		resultBytes, err := json.Marshal(testResult)
		if err != nil {
			log.Fatalf("Failed to marshal test result: %v", err)
		}
		testResultJSON = json.RawMessage(resultBytes)
	}

	// Determine status based on exit code (if test command was run)
	status := "done"
	if exitCode != nil && *exitCode != 0 {
		status = "failed"
	}

	// Update task status (and clear workspace current task)
	err = db.UpdateTaskStatus(dbPath, branch, taskID, status)
	if err != nil {
		log.Fatalf("Failed to update task status: %v", err)
	}

	// Update test_result column if we have a test result
	if testResultJSON != nil {
		err = db.UpdateTaskTestResult(dbPath, taskID, testResultJSON)
		if err != nil {
			log.Fatalf("Failed to update test result: %v", err)
		}
	}

	// Output result
	if exitCode != nil {
		fmt.Printf("Task #%d marked as %s (exit code: %d, duration: %d ms)\n", taskID, status, *exitCode, *durationMs)
		if *exitCode != 0 && output != "" {
			fmt.Println("Output:")
			fmt.Println(output)
		}
	} else {
		fmt.Printf("Task #%d marked as %s (no test command)\n", taskID, status)
	}
}

func cmdTaskBreakdown(dbPath, branch string, args []string) {
	fs := flag.NewFlagSet("task-breakdown", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit task breakdown - Create subtasks under a parent task

Usage:
  agit task breakdown <parent_task_id> '<json_subtasks_array>'

The JSON array must contain objects with "description" and optional "test_command".
Example:
  agit task breakdown 1 '[{"description": "Subtask 1", "test_command": "go test ./..."}]'

Options:
  --help  Show this help
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag || fs.NArg() < 2 {
		fs.Usage()
		return
	}
	parentIDStr := fs.Arg(0)
	var parentID int64
	if _, err := fmt.Sscanf(parentIDStr, "%d", &parentID); err != nil || parentID <= 0 {
		log.Fatalf("Invalid parent task ID: %s (must be positive integer)", parentIDStr)
	}
	subtasksJSON := fs.Arg(1)

	// Parse subtasks array
	var subtasksSlice []interface{}
	if err := json.Unmarshal([]byte(subtasksJSON), &subtasksSlice); err != nil {
		log.Fatalf("Failed to parse subtasks JSON: %v", err)
	}
	if len(subtasksSlice) == 0 {
		log.Fatal("subtasks array cannot be empty")
	}
	// Parse each subtask
	type subtask struct {
		Description string `json:"description"`
		TestCommand string `json:"test_command,omitempty"`
	}
	subtasks := make([]subtask, 0, len(subtasksSlice))
	for i, raw := range subtasksSlice {
		item, ok := raw.(map[string]interface{})
		if !ok {
			log.Fatalf("subtask %d must be an object", i+1)
		}
		// description (required)
		descRaw, ok := item["description"]
		if !ok {
			log.Fatalf("subtask %d missing description", i+1)
		}
		desc, ok := descRaw.(string)
		if !ok || desc == "" {
			log.Fatalf("subtask %d description must be a non-empty string", i+1)
		}
		// test_command (optional)
		testCmd := ""
		if testRaw, ok := item["test_command"]; ok {
			if testStr, ok := testRaw.(string); ok {
				testCmd = testStr
			}
			// ignore non‑string values
		}
		subtasks = append(subtasks, subtask{
			Description: desc,
			TestCommand: testCmd,
		})
	}

	// Verify parent task exists and belongs to current workspace
	parent, err := db.GetTask(dbPath, parentID)
	if err != nil {
		log.Fatalf("Failed to retrieve parent task: %v", err)
	}
	if parent == nil {
		log.Fatalf("Parent task %d not found", parentID)
	}
	if parent.WorkspaceBranch != branch {
		log.Fatalf("Parent task %d belongs to workspace %q, not current workspace %q",
			parentID, parent.WorkspaceBranch, branch)
	}

	// Create subtasks
	subtaskIDs := make([]int64, 0, len(subtasks))
	for _, st := range subtasks {
		if st.Description == "" {
			log.Fatal("subtask description is required")
		}
		parentID := sql.NullInt64{Int64: parentID, Valid: true}
		subtaskID, err := db.CreateTask(dbPath, branch, st.Description, st.TestCommand, parentID)
		if err != nil {
			log.Fatalf("Failed to create subtask %q: %v", st.Description, err)
		}
		subtaskIDs = append(subtaskIDs, subtaskID)
	}

	fmt.Printf("Created %d subtask(s) under task #%d:\n", len(subtaskIDs), parentID)
	for i, id := range subtaskIDs {
		fmt.Printf("  #%d: %s\n", id, subtasks[i].Description)
	}
}

func cmdTaskTree(dbPath, branch string, args []string) {
	fs := flag.NewFlagSet("task-tree", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	formatFlag := fs.String("format", "json", "Output format: json or tree")
	fs.Usage = func() {
		fmt.Print(`agit task tree - Show hierarchical tree of tasks for current workspace

Usage:
  agit task tree [options]

Options:
  --help         Show this help
  --format <fmt> Output format: json or tree (default json)

Examples:
  agit task tree
  agit task tree --format tree
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}
	tree, err := db.GetTaskTree(dbPath, branch)
	if err != nil {
		log.Fatalf("Failed to retrieve task tree: %v", err)
	}
	switch *formatFlag {
	case "tree":
		printTaskTree(tree, "")
	case "json":
		fallthrough
	default:
		data, err := json.MarshalIndent(tree, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal task tree: %v", err)
		}
		fmt.Println(string(data))
	}
}

// printTaskTree recursively prints a task tree with ASCII indentation.
func printTaskTree(nodes []*db.TaskTreeNode, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		// Current node marker
		marker := "├── "
		if isLast {
			marker = "└── "
		}
		fmt.Printf("%s%s#%d %s [%s]\n", prefix, marker, node.ID, node.Description, node.Status)
		// Child prefix extends with spaces or │
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
		printTaskTree(node.Children, childPrefix)
	}
}

func cmdCheckpoint(args []string) {
	fs := flag.NewFlagSet("checkpoint", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit checkpoint - Create a checkpoint (git commit + workspace snapshot)

Usage:
  agit checkpoint "message"

Examples:
  agit checkpoint "Before risky refactor"
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag || fs.NArg() == 0 {
		fs.Usage()
		return
	}
	message := fs.Arg(0)
	dbPath := getDBPath()
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatalf("Failed to get current branch: %v", err)
	}
	// Acquire Git lock before any repository modification
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer unlock()
	// Commit all changes
	commitHash, err := git.CommitAll(message)
	if err != nil {
		log.Fatalf("Failed to commit changes: %v", err)
	}
	// Get workspace snapshot
	agentState, err := db.GetWorkspaceSnapshot(dbPath, branch)
	if err != nil {
		log.Fatalf("Failed to get workspace snapshot: %v", err)
	}
	// Create checkpoint record
	checkpointID, err := db.CreateCheckpoint(dbPath, branch, commitHash, message, agentState)
	if err != nil {
		log.Fatalf("Failed to create checkpoint: %v", err)
	}
	fmt.Printf("Checkpoint #%d created at commit %s: %s\n", checkpointID, commitHash[:8], message)
}

func cmdRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit restore - Restore workspace to latest checkpoint

Usage:
  agit restore

This will reset Git to the last checkpoint commit and restore workspace state.
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}
	dbPath := getDBPath()
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatalf("Failed to get current branch: %v", err)
	}
	// Get latest checkpoint
	cp, err := db.GetLatestCheckpoint(dbPath, branch)
	if err != nil {
		log.Fatalf("Failed to get latest checkpoint: %v", err)
	}
	if cp == nil {
		log.Fatal("No checkpoint found for this workspace.")
	}
	// Acquire Git lock before any repository modification
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer unlock()
	// Reset Git to the checkpoint commit
	if err := git.ResetToCommit(cp.CommitHash); err != nil {
		log.Fatalf("Failed to reset to commit %s: %v", cp.CommitHash, err)
	}
	// Restore workspace state from snapshot
	if len(cp.AgentState) > 0 {
		if err := db.RestoreWorkspaceFromSnapshot(dbPath, branch, cp.AgentState); err != nil {
			log.Fatalf("Failed to restore workspace state: %v", err)
		}
	}
	fmt.Printf("Restored to checkpoint #%d (%s): %s\n", cp.ID, cp.CommitHash[:8], cp.Message)
}

func cmdTrace(args []string) {
	fs := flag.NewFlagSet("trace", flag.ExitOnError)
	fs.SetInterspersed(true)
	taskIDFlag := fs.Int64("task", 0, "Task ID (required)")
	limitFlag := fs.Int("limit", 10, "Maximum number of traces to show")
	failedOnlyFlag := fs.Bool("failed-only", false, "Show only failed test runs")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit trace - Show test run history for a task

Usage:
  agit trace --task <task_id> [options]

Options:
  --task <id>         Task ID (required)
  --limit <n>         Maximum number of traces to show (default 10)
  --failed-only       Show only failed test runs (exit code != 0)
  --help              Show this help

Examples:
  agit trace --task 1
  agit trace --task 1 --limit 5 --failed-only
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag || *taskIDFlag == 0 {
		fs.Usage()
		return
	}
	dbPath := getDBPath()
	var traces []db.Trace
	var err error
	if *failedOnlyFlag {
		traces, err = db.GetTracesFailed(dbPath, *taskIDFlag, *limitFlag)
	} else {
		traces, err = db.GetTraces(dbPath, *taskIDFlag, *limitFlag)
	}
	if err != nil {
		log.Fatalf("Failed to retrieve traces: %v", err)
	}
	if len(traces) == 0 {
		fmt.Println("No traces found.")
		return
	}
	// Table header
	fmt.Printf("%-4s %-10s %-12s %-10s %s\n", "ID", "Exit Code", "Duration(ms)", "Run At", "Output Preview")
	for _, t := range traces {
		exitCode := "-"
		if t.ExitCode != nil {
			exitCode = fmt.Sprintf("%d", *t.ExitCode)
		}
		duration := "-"
		if t.DurationMs != nil {
			duration = fmt.Sprintf("%d", *t.DurationMs)
		}
		runAt := t.RunAt.Format("2006-01-02 15:04:05")
		preview := t.Output
		if len(preview) > 40 {
			preview = preview[:37] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", "\\n")
		fmt.Printf("%-4d %-10s %-12s %-10s %s\n", t.ID, exitCode, duration, runAt, preview)
	}
}

func cmdDashboard(args []string) {
	// Try to find agit-dashboard binary
	dashboardBin := "agit-dashboard"
	// First, look in PATH
	if path, err := exec.LookPath(dashboardBin); err == nil {
		dashboardBin = path
	} else {
		// Not in PATH, try same directory as current executable
		exe, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exe)
			candidate := filepath.Join(exeDir, "agit-dashboard")
			if _, err := os.Stat(candidate); err == nil {
				dashboardBin = candidate
			} else {
				log.Fatalf("agit-dashboard binary not found in PATH or next to agit executable.\nPlease build it with: go build ./cmd/agit-dashboard")
			}
		} else {
			log.Fatalf("Failed to locate agit executable: %v", err)
		}
	}

	// Prepare command
	cmd := exec.Command(dashboardBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run and wait
	if err := cmd.Run(); err != nil {
		// Exit with same exit code if possible
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatal(err)
	}
}

func cmdManifest(args []string) {
	if len(args) < 1 {
		fmt.Print(`agit manifest - Manage agit project manifest

Usage:
  agit manifest <subcommand> [options]

Subcommands:
  load        Load and apply manifest from default location (.agit-manifest.json)
  init        Create a sample manifest file in current directory

Examples:
  agit manifest load
  agit manifest init
`)
		os.Exit(1)
	}

	dbPath := getDBPath()

	switch args[0] {
	case "load":
		cmdManifestLoad(dbPath, args[1:])
	case "init":
		cmdManifestInit(args[1:])
	default:
		fmt.Printf("Unknown manifest subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdManifestLoad(dbPath string, args []string) {
	fs := flag.NewFlagSet("manifest-load", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit manifest load - Load and apply manifest from default location

Usage:
  agit manifest load [options]

Options:
  --help  Show this help

Loads .agit-manifest.json (or agit-manifest.json) from current directory
and creates tasks in the current workspace.
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}

	fmt.Println("Loading manifest...")
	if err := manifest.ApplyDefault(dbPath); err != nil {
		log.Fatalf("Failed to load manifest: %v", err)
	}
	fmt.Println("Manifest applied successfully")
}

func cmdManifestInit(args []string) {
	fs := flag.NewFlagSet("manifest-init", flag.ExitOnError)
	fs.SetInterspersed(true)
	helpFlag := fs.Bool("help", false, "Show help")
	forceFlag := fs.Bool("force", false, "Overwrite existing manifest file")
	fs.Usage = func() {
		fmt.Print(`agit manifest init - Create a sample manifest file

Usage:
  agit manifest init [options]

Options:
  --force  Overwrite existing manifest file
  --help   Show this help

Creates .agit-manifest.json in current directory with example tasks.
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}

	// Choose filename
	filename := ".agit-manifest.json"
	if _, err := os.Stat(filename); err == nil && !*forceFlag {
		log.Fatalf("%s already exists. Use --force to overwrite.", filename)
	}

	sample := `{
  "version": "1.0.0",
  "name": "agit",
  "description": "Agent Git MCP Server",
  "tasks": [
    {
      "id": 1,
      "description": "Implement workspace_status tool",
      "test": "go test ./internal/mcp -run TestWorkspaceStatus",
      "dependencies": []
    },
    {
      "id": 2,
      "description": "Implement task_new and task_list tools",
      "test": "go test ./internal/mcp -run TestTask",
      "dependencies": [1]
    }
  ],
  "agents": {
    "preferred": "claude-code",
    "integrations": ["opencode", "cursor"]
  }
}
`
	if err := os.WriteFile(filename, []byte(sample), 0644); err != nil {
		log.Fatalf("Failed to write manifest file: %v", err)
	}
	fmt.Printf("Created sample manifest at %s\n", filename)
}

func installHook() {
	// Find .git directory
	gitDir := ".git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		log.Fatal("Not a git repository (or any of the parent directories): .git not found")
	}

	dbPath := getDBPath()
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer unlock()

	hookPath := ".git/hooks/post-checkout"

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		// Read first line to see if it's an agit hook
		content, err := os.ReadFile(hookPath)
		if err == nil && len(content) > 0 {
			firstLine := string(content)
			if len(firstLine) >= 100 {
				firstLine = firstLine[:100]
			}
			if strings.Contains(firstLine, "agit") {
				fmt.Println("agit hook already installed")
				return
			} else {
				fmt.Printf("Warning: %s already exists and is not an agit hook\n", hookPath)
				fmt.Println("Use --force to overwrite (not yet implemented)")
				return
			}
		}
	}

	// Write hook script
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		log.Fatalf("Failed to write hook script: %v", err)
	}

	fmt.Printf("agit post-checkout hook installed at %s\n", hookPath)
	fmt.Println("Note: agit-mcp server must be built and in PATH for hook to work")
}

func uninstallHook() {
	hookPath := ".git/hooks/post-checkout"
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		fmt.Println("agit hook not found")
		return
	}

	// Acquire Git lock before modifying repository
	dbPath := getDBPath()
	unlock, err := db.GitLock(dbPath)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer unlock()

	// Read file to confirm it's an agit hook
	content, err := os.ReadFile(hookPath)
	if err != nil {
		log.Fatalf("Failed to read hook file: %v", err)
	}

	if strings.Contains(string(content), "agit") {
		if err := os.Remove(hookPath); err != nil {
			log.Fatalf("Failed to remove hook: %v", err)
		}
		fmt.Println("agit hook removed")
	} else {
		fmt.Printf("File %s exists but doesn't appear to be an agit hook\n", hookPath)
		fmt.Println("Use manual removal if desired")
	}
}

func cmdSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	fs.SetInterspersed(true)
	autoFlag := fs.Bool("auto", false, "Auto-detect tech stack and generate manifest without prompts")
	interactiveFlag := fs.Bool("interactive", true, "Interactive wizard (default)")
	manifestFlag := fs.Bool("manifest", true, "Generate .agit-manifest.json")
	overwriteFlag := fs.Bool("overwrite", false, "Overwrite existing manifest file")
	hooksFlag := fs.Bool("hooks", false, "Install Git hooks")
	agentConfigsFlag := fs.Bool("agent-configs", false, "Generate agent configs")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit setup - Bootstrap agit for current project with project planning wizard

Usage:
  agit setup [options]

Options:
  --auto          Auto-detect tech stack and generate manifest without prompts
  --interactive   Interactive project planning wizard (default)
  --manifest      Generate .agit-manifest.json (default true)
  --overwrite     Overwrite existing manifest file
  --hooks         Install Git hooks
  --agent-configs Generate agent configs for Claude Code, OpenCode, Cursor
  --help          Show this help

The interactive project planning wizard will:
1. Conduct a detailed interview about your project (type, tech stack, features, architecture)
2. Create a comprehensive project plan with task breakdowns
3. Generate .agit-manifest.json with initial tasks tailored to your project
4. Install Git hooks (optional)
5. Generate agent configs for Claude Code, OpenCode, Cursor (optional)
6. Validate existing agit configuration

Auto mode (--auto) will:
1. Detect tech stack (Go, Node.js, Python, etc.)
2. Suggest appropriate test commands
3. Generate .agit-manifest.json with generic initial tasks
4. Install Git hooks (optional)
5. Generate agent configs (optional)

Examples:
  agit setup
  agit setup --auto
  agit setup --manifest --hooks
  agit setup --manifest --overwrite
`)
	}

	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}

	// Determine mode
	interactive := *interactiveFlag && !*autoFlag
	auto := *autoFlag

	var cfg *setup.WizardConfig
	var stack *setup.TechStack
	var plan *setup.ProjectPlan

	if interactive {
		// Run project planning wizard
		plan = setup.RunProjectPlanningWizard()
		stack = plan.TechStack
		// Override hooks/agent configs with command line flags if set
		if *hooksFlag {
			plan.InstallHooks = true
		}
		if *agentConfigsFlag {
			plan.GenerateAgentCfgs = true
		}
		// Create a minimal WizardConfig for compatibility with later code
		language := stack.Language
		customTestCmd := ""
		if language == "Custom" {
			customTestCmd = stack.TestCommand
			language = "Other"
		}
		cfg = &setup.WizardConfig{
			TechStack:         stack,
			Language:          language,
			CustomTestCmd:     customTestCmd,
			InstallHooks:      plan.InstallHooks,
			GenerateManifest:  true, // always generate manifest from plan
			GenerateAgentCfgs: plan.GenerateAgentCfgs,
			ProjectName:       plan.Name,
		}
	} else if auto {
		// Auto‑detect mode: use flag values, no prompts
		stack = setup.Detect()
		if stack == nil {
			fmt.Println("No known tech stack detected.")
			fmt.Println("Suggested test command:", setup.SuggestTestCommand())
		} else {
			fmt.Printf("Detected %s project (%s)\n", stack.Language, stack.BuildTool)
			fmt.Printf("Suggested test command: %s\n", stack.TestCommand)
			fmt.Printf("Detected files: %v\n", stack.DetectedFiles)
		}
		// Create a minimal config based on flags
		cfg = &setup.WizardConfig{
			TechStack:         stack,
			Language:          "Unknown",
			CustomTestCmd:     "",
			InstallHooks:      *hooksFlag,
			GenerateManifest:  *manifestFlag,
			GenerateAgentCfgs: *agentConfigsFlag,
			ProjectName:       setup.GetCurrentDir(),
		}
		if stack != nil {
			cfg.Language = stack.Language
		}
	} else {
		// Should not happen (interactive defaults to true)
		cfg = &setup.WizardConfig{}
	}

	// Generate manifest if requested (interactive mode uses cfg.GenerateManifest,
	// auto mode uses *manifestFlag)
	generateManifest := cfg.GenerateManifest
	if !interactive && !auto {
		generateManifest = *manifestFlag
	}
	if generateManifest {
		fmt.Println("\nGenerating manifest...")
		var m *manifest.Manifest
		if interactive && plan != nil {
			// Generate from project plan
			m = setup.ConvertProjectPlanToManifest(plan)
		} else if cfg.Language == "Other" && cfg.CustomTestCmd != "" {
			// Custom test command
			m = setup.GenerateManifestCustom(cfg.ProjectName, cfg.Language, cfg.CustomTestCmd)
		} else if stack != nil {
			// Use detected stack
			m = setup.GenerateManifest(stack)
		} else {
			// Generic manifest
			m = setup.GenerateManifest(nil)
		}
		err := setup.WriteManifest(m, *overwriteFlag)
		if err != nil {
			log.Printf("Warning: failed to write manifest: %v", err)
		} else {
			fmt.Println("Manifest written to .agit-manifest.json")
		}
	}

	// Generate requirements file if we have a project plan with requirements
	if interactive && plan != nil && plan.Requirements != nil {
		requirementsFile := ".agit-requirements.json"
		err := setup.WriteRequirements(plan.Requirements, requirementsFile, *overwriteFlag)
		if err != nil {
			log.Printf("Warning: failed to write requirements: %v", err)
		} else {
			fmt.Println("Requirements written to .agit-requirements.json")
		}
	}

	// Install hooks if requested
	installHooks := cfg.InstallHooks
	if !interactive && !auto {
		installHooks = *hooksFlag
	}
	if installHooks {
		fmt.Println("\nInstalling Git hooks...")
		installHook()
	}

	// Generate agent configs if requested
	generateAgentCfgs := cfg.GenerateAgentCfgs
	if !interactive && !auto {
		generateAgentCfgs = *agentConfigsFlag
	}
	if generateAgentCfgs {
		fmt.Println("\nGenerating agent configs...")
		err := setup.GenerateAgentConfigs(*overwriteFlag)
		if err != nil {
			log.Printf("Warning: failed to generate agent configs: %v", err)
		} else {
			fmt.Println("Agent configs generated.")
		}
	}

	fmt.Println("\nSetup wizard completed.")
}

func cmdDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	fs.SetInterspersed(true)
	fixFlag := fs.Bool("fix", false, "Attempt to fix issues automatically")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.Usage = func() {
		fmt.Print(`agit doctor - Validate and repair agit configuration

Usage:
  agit doctor [options]

Options:
  --fix   Attempt to fix issues automatically
  --help  Show this help

Examples:
  agit doctor
  agit doctor --fix
`)
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	if *helpFlag {
		fs.Usage()
		return
	}

	fmt.Println("agit doctor - validating configuration...")
	issues := 0

	// Check 1: agit database
	dbPath := ".git/agit.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("  ✗ agit database not found at %s\n", dbPath)
		issues++
		if *fixFlag {
			fmt.Println("    Running 'agit init'...")
			// We could call cmdInit directly, but for simplicity run exec
			cmd := exec.Command("agit", "init")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("    Failed to initialize: %v\n", err)
			} else {
				fmt.Println("    Database initialized.")
			}
		}
	} else {
		fmt.Println("  ✓ agit database exists")
	}

	// Check 2: Git hook
	hookPath := ".git/hooks/post-checkout"
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		fmt.Printf("  ✗ agit Git hook not found at %s\n", hookPath)
		issues++
		if *fixFlag {
			fmt.Println("    Installing hook...")
			installHook()
		}
	} else {
		// Check if hook contains agit
		content, err := os.ReadFile(hookPath)
		if err == nil && strings.Contains(string(content), "agit") {
			fmt.Println("  ✓ agit Git hook installed")
		} else {
			fmt.Printf("  ✗ Git hook exists but does not appear to be an agit hook\n")
			issues++
			if *fixFlag {
				fmt.Println("    Replacing hook...")
				installHook()
			}
		}
	}

	// Check 3: manifest file
	manifestPath := ".agit-manifest.json"
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		fmt.Printf("  ✗ manifest file not found at %s\n", manifestPath)
		issues++
		if *fixFlag {
			fmt.Println("    Generating manifest...")
			stack := setup.Detect()
			m := setup.GenerateManifest(stack)
			if err := setup.WriteManifest(m, true); err != nil {
				fmt.Printf("    Failed to write manifest: %v\n", err)
			} else {
				fmt.Println("    Manifest generated.")
			}
		}
	} else {
		fmt.Println("  ✓ manifest file exists")
	}

	// Check 4: requirements file (optional)
	requirementsPath := ".agit-requirements.json"
	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		fmt.Printf("  ⚠ requirements file not found at %s (optional)\n", requirementsPath)
		// Not counting as issue
	} else {
		fmt.Println("  ✓ requirements file exists")
		// Validate requirements
		req, err := setup.LoadRequirements(requirementsPath)
		if err != nil {
			fmt.Printf("  ✗ failed to load requirements: %v\n", err)
			issues++
		} else {
			validationIssues := req.Validate(".")
			for _, issue := range validationIssues {
				fmt.Printf("  ✗ %s\n", issue)
				issues++
				if *fixFlag {
					// Attempt to fix structure issues
					if strings.HasPrefix(issue, "missing required directory:") {
						dir := strings.TrimPrefix(issue, "missing required directory: ")
						os.MkdirAll(dir, 0755)
						fmt.Printf("    Created directory: %s\n", dir)
					} else if strings.HasPrefix(issue, "missing required file:") {
						file := strings.TrimPrefix(issue, "missing required file: ")
						// Create empty file
						os.WriteFile(file, []byte{}, 0644)
						fmt.Printf("    Created empty file: %s\n", file)
					}
				}
			}
		}
	}

	// Check 5: agent configs (optional)
	configs := []string{
		".claude/mcp.json",
		".opencode/plugins/agit.json",
		".cursor/mcp.json",
	}
	for _, cfg := range configs {
		if _, err := os.Stat(cfg); os.IsNotExist(err) {
			fmt.Printf("  ⚠ agent config missing: %s (optional)\n", cfg)
			// Not counting as issue
			if *fixFlag {
				fmt.Printf("    Generating %s...\n", cfg)
				// Use setup.GenerateAgentConfigs with overwrite false
				setup.GenerateAgentConfigs(false)
			}
		} else {
			fmt.Printf("  ✓ agent config exists: %s\n", cfg)
		}
	}

	// Summary
	if issues == 0 {
		fmt.Println("\nAll checks passed. agit is properly configured.")
	} else {
		fmt.Printf("\nFound %d issue(s). Run 'agit doctor --fix' to attempt repairs.\n", issues)
	}
}

func printUsage() {
	fmt.Print(`agit - Agent Git MCP Server for Stateful Agent Workflows

Usage:
  agit <command> [options]

Commands:
  init            Initialize agit database in current repository
  hook            Manage Git hooks for workspace switching
  workspace-status Show current workspace status
  task            Manage tasks (new, list, start, done, breakdown, tree)
  checkpoint      Create a checkpoint (git commit + workspace snapshot)
  restore         Restore workspace to latest checkpoint
  trace           Show test run history for a task
  dashboard       Start web dashboard (read‑only)
  doctor           Validate and repair agit configuration
  setup            Bootstrap agit for current project
  version         Show version information
  help            Show this help message

Examples:
  agit init               # Initialize agit database
  agit init --force       # Force reinitialization
  agit hook install       # Install post-checkout hook
  agit hook uninstall     # Remove post-checkout hook
  agit workspace-status   # Show current workspace status
  agit task new --test "go test ./..." "Implement feature"
  agit checkpoint "Before risky refactor"
  agit restore            # Restore to last checkpoint
  agit dashboard          # Start web dashboard on localhost:8080
  agit dashboard --port 8081 --host 0.0.0.0
  agit setup               # Bootstrap agit for current project
  agit version            # Show version
`)
}
