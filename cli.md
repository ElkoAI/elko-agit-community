# agit CLI Reference

Complete command‑line interface reference for agit. All MCP tools have equivalent CLI commands.

## 📋 Command Overview

| Command | Description | Equivalent MCP Tool |
|---------|-------------|-------------------|
| `agit init` | Initialize agit database | – |
| `agit hook` | Manage Git hooks | – |
| `agit workspace-status` | Show current workspace status | `workspace_status` |
| `agit task new` | Create new task | `task_new` |
| `agit task list` | List tasks | `task_list` |
| `agit task start` | Mark task as active | `task_start` |
| `agit task done` | Mark task as done + run tests | `task_done` |
| `agit task breakdown` | Create subtasks | `task_breakdown` |
| `agit task tree` | Show hierarchical task tree | `task_tree` |
| `agit checkpoint` | Create checkpoint | `checkpoint` |
| `agit restore` | Restore to last checkpoint | `restore` |
| `agit trace` | Show test run history | `trace` |
| `agit version` | Show version | – |
| `agit help` | Show help | – |

## 🚀 Getting Started

### Initialize agit

```bash
# Initialize agit database in current repository
agit init

# Force reinitialization (overwrites existing database)
agit init --force
```

### Install Git Hooks

```bash
# Install post-checkout hook for automatic workspace switching
agit hook install

# Remove agit hooks
agit hook uninstall
```

## 🛠️ Core Commands

### `agit workspace-status`

Show current workspace status as JSON.

```bash
agit workspace-status
```

**Example Output:**
```json
{
  "branch": "feat/217-impl-check",
  "current_task_id": 5,
  "agent_state": {
    "last_action": "implemented method set comparison",
    "context": "Working on analyzer refactor"
  },
  "todo_list": "- [ ] Fix edge case\n- [ ] Add tests",
  "last_commit": "a1b2c3d4e5f678901234567890abcdef12345678",
  "created_at": "2026-02-18T10:30:00Z",
  "updated_at": "2026-02-19T14:45:00Z"
}
```

### Task Management

#### `agit task new`

Create a new task.

```bash
# Basic task
agit task new "Implement feature X"

# With test command
agit task new --test "go test ./..." "Implement feature X"

# As subtask of parent
agit task new --parent 3 "Subtask"
```

**Options:**
- `--test <cmd>`: Test command to run when task is done
- `--parent <id>`: Parent task ID (optional)

#### `agit task list`

List tasks with optional status filter.

```bash
# List all tasks
agit task list

# Filter by status
agit task list --status pending
agit task list --status active
agit task list --status done
agit task list --status failed
```

**Example Output:**
```
ID   Status     Description                              Created
1    pending    Implement method set comparison         2026-02-19
2    active     Add TypeScript support                  2026-02-19
```

#### `agit task start`

Mark a task as active and set as current workspace task.

```bash
agit task start 1
```

#### `agit task done`

Mark a task as done, run its test command, and record trace.

```bash
agit task done 1
```

**Behavior:**
1. Runs test command via `sh -c` (if task has one)
2. Records trace with exit code, output, and duration
3. Updates task status (`done` if exit code 0, `failed` otherwise)
4. Clears workspace's current task

**Example Output (success):**
```
Task #1 marked as done (exit code: 0, duration: 420 ms)
```

**Example Output (failure):**
```
Task #1 marked as failed (exit code: 1, duration: 320 ms)
Output:
FAIL
Error: expected 10 but got 11
```

#### `agit task breakdown`

Create subtasks under a parent task.

```bash
agit task breakdown 1 '[{"description": "Subtask 1", "test_command": "go test ./..."}]'
```

**JSON Format:**
```json
[
  {
    "description": "Subtask description",
    "test_command": "optional test command"
  }
]
```

**Example Output:**
```
Created 2 subtask(s) under task #1:
  #8: Parse TypeScript interface
  #9: Handle generic types
```

#### `agit task tree`

Show hierarchical tree of tasks as JSON.

```bash
agit task tree
```

**Example Output:**
```json
[
  {
    "id": 1,
    "workspace_branch": "feat/217-impl-check",
    "description": "Implement analyzer",
    "status": "active",
    "test_command": "go test ./internal/analyzer",
    "parent_task_id": null,
    "created_at": "2026-02-19T10:30:00Z",
    "completed_at": null,
    "children": [
      {
        "id": 2,
        "workspace_branch": "feat/217-impl-check",
        "description": "Parse TypeScript interface",
        "status": "pending",
        "test_command": "npm test -- --grep 'interface parsing'",
        "parent_task_id": 1,
        "created_at": "2026-02-19T11:00:00Z",
        "completed_at": null,
        "children": []
      }
    ]
  }
]
```

### Checkpoint System

#### `agit checkpoint`

Create a checkpoint (git commit + workspace snapshot).

```bash
agit checkpoint "Before risky refactor"
```

**Behavior:**
1. Commits all changes with `git commit -am "<message>"`
2. Creates workspace state snapshot
3. Stores checkpoint record with commit hash

**Example Output:**
```
Checkpoint #5 created at commit a1b2c3d4: Before risky refactor
```

#### `agit restore`

Restore workspace to latest checkpoint.

```bash
agit restore
```

**Behavior:**
1. Resets Git to checkpoint commit
2. Restores workspace state from snapshot

**Example Output:**
```
Restored to checkpoint #5 (a1b2c3d4): Before risky refactor
```

### Test Tracing

#### `agit trace`

Show test run history for a task.

```bash
# Basic usage
agit trace --task 1

# Limit results
agit trace --task 1 --limit 5

# Show only failed runs
agit trace --task 1 --failed-only

# Combined options
agit trace --task 1 --limit 3 --failed-only
```

**Options:**
- `--task <id>`: Task ID (required)
- `--limit <n>`: Maximum number of traces to show (default: 10)
- `--failed-only`: Show only failed test runs (exit code ≠ 0)

**Example Output:**
```
ID   Exit Code  Duration(ms) Run At       Output Preview
3    1          450          2026-02-19   FAIL\nExpected 10 but got 11
2    0          320          2026-02-19   PASS\nAll tests passed
1    1          5000         2026-02-19   FAIL\nTest timeout
```

### Utility Commands

#### `agit version`

Show version information.

```bash
agit version
```

#### `agit help`

Show help message.

```bash
agit help
agit --help
agit -h
```

## 🔄 Complete Workflow Example

```bash
# 1. Initialize agit in your project
agit init
agit hook install

# 2. Create a task with test command
agit task new --test "go test ./internal/auth" "Implement authentication middleware"

# 3. Start working on it
agit task start 1

# 4. Write some code...

# 5. Create checkpoint before risky change
agit checkpoint "Middleware skeleton complete"

# 6. Break down into subtasks
agit task breakdown 1 '[{"description": "Add JWT verification"}, {"description": "Implement rate limiting"}]'

# 7. Work on subtask 2
agit task start 2

# 8. Write code for JWT verification...

# 9. Mark subtask as done (runs tests)
agit task done 2

# 10. View test history
agit trace --task 2

# 11. Check overall progress
agit task list
agit task tree

# 12. Restore if something went wrong
agit restore
```

## 🐛 Error Messages

Common errors and solutions:

**`Not a git repository`**
```bash
Error: Not a git repository (or any of the parent directories): .git not found
```
→ Run in a Git repository root directory.

**`workspace not found`**
```bash
Error: workspace "main" does not exist
```
→ Run `agit init` first.

**`Invalid task ID`**
```bash
Error: Invalid task ID: abc (must be positive integer)
```
→ Use numeric task IDs.

**`parent task not found`**
```bash
Error: parent task 999 does not exist
```
→ Check parent task exists with `agit task list`.

**`No checkpoint found`**
```bash
Error: No checkpoint found for this workspace.
```
→ Create a checkpoint first with `agit checkpoint`.

## ⚙️ Configuration Files

agit stores its database at `.git/agit.db`. No additional configuration files are required.

### Git Hook Location
- `.git/hooks/post-checkout` (created by `agit hook install`)

### Agent Configuration
See [MCP Interface](./mcp.md) for agent‑specific configuration files.

## 📚 Environment Variables

No environment variables are required. agit uses:
- Current Git repository (from `.git/` directory)
- SQLite database at `.git/agit.db`

## 🔧 Debugging

### Verbose Output
Add `--help` to any command to see usage information.

### Database Inspection
The SQLite database can be inspected directly:
```bash
sqlite3 .git/agit.db ".tables"
sqlite3 .git/agit.db "SELECT * FROM workspaces"
```

### Test Execution
Test commands are executed via `sh -c`. For complex commands, consider wrapping in a script.

## 📝 Tips & Best Practices

1. **Always initialize first**: Run `agit init` in each repository
2. **Install hooks**: `agit hook install` for automatic workspace switching
3. **Use test commands**: Add `--test` to tasks for automatic verification
4. **Checkpoint frequently**: Create checkpoints before risky changes
5. **Break down large tasks**: Use `agit task breakdown` for complex work
6. **Check status often**: Use `agit task list` and `agit workspace-status`
7. **Review test history**: Use `agit trace` to debug failing tests

## 📚 See Also

- [MCP Interface](./mcp.md) - AI agent integration
- [README](./README.md) - Project overview
- [Design Spec](./agent/design-spec.md) - Architecture and data model