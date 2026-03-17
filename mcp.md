# agit MCP Interface Reference

Complete reference for agit's Model Context Protocol (MCP) tools. These tools enable AI coding agents to interact with agit's Git‑native state persistence system.

## 📋 Overview

agit exposes 10 MCP tools that map to its core concepts:

| Tool | Purpose | Parameters |
|------|---------|------------|
| `workspace_status` | Get current workspace state | None |
| `task_new` | Create new task | `description`, `test_command` (optional), `parent_task_id` (optional) |
| `task_list` | List tasks | `status` (optional) |
| `task_start` | Mark task as active | `task_id` |
| `task_done` | Mark task as done + run tests | `task_id` |
| `task_breakdown` | Create subtasks | `parent_task_id`, `subtasks` array |
| `task_tree` | Get hierarchical task tree | None |
| `checkpoint` | Create checkpoint (commit + snapshot) | `message` |
| `restore` | Restore to last checkpoint | None |
| `trace` | Show test run history | `task_id`, `limit` (optional) |

All tools operate on the **current workspace** (determined by Git branch).

## 🛠️ Tool Reference

### 1. `workspace_status`

Get current workspace state including branch, current task, agent state, todo list, and last commit.

**Parameters:** None

**Example Request:**
```json
{}
```

**Example Response:**
```json
{
  "branch": "feat/217-impl-check",
  "current_task_id": 5,
  "agent_state": {
    "last_action": "implemented method set comparison",
    "context": "Working on analyzer refactor"
  },
  "todo_list": "- [ ] Fix edge case in TypeScript parser\n- [ ] Add tests for recursive types",
  "last_commit": "a1b2c3d4e5f678901234567890abcdef12345678",
  "created_at": "2026-02-18T10:30:00Z",
  "updated_at": "2026-02-19T14:45:00Z"
}
```

### 2. `task_new`

Create a new task in the current workspace.

**Parameters:**
- `description` (string, required): Task description
- `test_command` (string, optional): Command to run when task is marked done
- `parent_task_id` (integer, optional): ID of parent task for decomposition

**Example Request:**
```json
{
  "description": "Implement method set comparison",
  "test_command": "go test ./internal/analyzer",
  "parent_task_id": 3
}
```

**Example Response:**
```json
{
  "task_id": 7,
  "description": "Implement method set comparison",
  "branch": "feat/217-impl-check",
  "status": "pending",
  "test_command": "go test ./internal/analyzer",
  "parent_task_id": 3
}
```

### 3. `task_list`

List tasks for the current workspace, optionally filtered by status.

**Parameters:**
- `status` (string, optional): Filter by status (`pending`, `active`, `done`, `failed`)

**Example Request:**
```json
{
  "status": "pending"
}
```

**Example Response:**
```json
[
  {
    "id": 1,
    "workspace_branch": "feat/217-impl-check",
    "description": "Implement method set comparison",
    "status": "pending",
    "test_command": "go test ./internal/analyzer",
    "parent_task_id": null,
    "created_at": "2026-02-19T10:30:00Z",
    "completed_at": null
  },
  {
    "id": 2,
    "workspace_branch": "feat/217-impl-check",
    "description": "Add TypeScript support",
    "status": "active",
    "test_command": "npm test",
    "parent_task_id": null,
    "created_at": "2026-02-19T11:15:00Z",
    "completed_at": null
  }
]
```

### 4. `task_start`

Mark a task as active and set it as the workspace's current task.

**Parameters:**
- `task_id` (integer, required): Task ID to start

**Example Request:**
```json
{
  "task_id": 1
}
```

**Example Response:**
```json
{
  "task_id": 1,
  "branch": "feat/217-impl-check",
  "status": "active"
}
```

### 5. `task_done`

Mark a task as done, clear workspace current task, and run test command if present.

**Parameters:**
- `task_id` (integer, required): Task ID to mark as done

**Behavior:**
1. If task has a `test_command`, executes it via `sh -c`
2. Records trace with exit code, output, and duration
3. Updates task status (`done` if tests pass, `failed` if non‑zero exit)
4. Clears workspace's `current_task_id`

**Example Request:**
```json
{
  "task_id": 1
}
```

**Example Response (test passed):**
```json
{
  "task_id": 1,
  "branch": "feat/217-impl-check",
  "status": "done",
  "exit_code": 0,
  "output": "PASS\nok  \tagit/internal/analyzer\t0.42s",
  "duration_ms": 420
}
```

**Example Response (test failed):**
```json
{
  "task_id": 1,
  "branch": "feat/217-impl-check",
  "status": "failed",
  "exit_code": 1,
  "output": "FAIL\nError: expected 10 but got 11",
  "duration_ms": 320
}
```

### 6. `task_breakdown`

Create subtasks under a parent task.

**Parameters:**
- `parent_task_id` (integer, required): ID of parent task
- `subtasks` (array of objects, required): Array of subtask objects:
  - `description` (string, required): Subtask description
  - `test_command` (string, optional): Test command for this subtask

**Example Request:**
```json
{
  "parent_task_id": 3,
  "subtasks": [
    {
      "description": "Parse TypeScript interface",
      "test_command": "npm test -- --grep 'interface parsing'"
    },
    {
      "description": "Handle generic types",
      "test_command": "npm test -- --grep 'generics'"
    }
  ]
}
```

**Example Response:**
```json
{
  "parent_task_id": 3,
  "branch": "feat/217-impl-check",
  "subtask_ids": [8, 9]
}
```

### 7. `task_tree`

Return hierarchical tree of tasks for the current workspace.

**Parameters:** None

**Example Response:**
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

### 8. `checkpoint`

Create a checkpoint (git commit + workspace state snapshot).

**Parameters:**
- `message` (string, required): Checkpoint message (used as commit message)

**Behavior:**
1. Commits all uncommitted changes with `git commit -am "<message>"`
2. Creates workspace state snapshot (agent state, todo list, current task)
3. Stores checkpoint record with commit hash

**Example Request:**
```json
{
  "message": "Before risky refactor"
}
```

**Example Response:**
```json
{
  "checkpoint_id": 5,
  "branch": "feat/217-impl-check",
  "commit_hash": "a1b2c3d4e5f678901234567890abcdef12345678",
  "message": "Before risky refactor",
  "created_at": "2026-02-19T15:30:00Z"
}
```

### 9. `restore`

Restore workspace to latest checkpoint.

**Parameters:** None

**Behavior:**
1. Resets Git to checkpoint commit (`git reset --hard <commit>`)
2. Restores workspace state from snapshot
3. Updates workspace record

**Example Request:**
```json
{}
```

**Example Response:**
```json
{
  "checkpoint_id": 5,
  "branch": "feat/217-impl-check",
  "commit_hash": "a1b2c3d4e5f678901234567890abcdef12345678",
  "message": "Before risky refactor",
  "restored_at": "2026-02-19T16:45:00Z"
}
```

### 10. `trace`

Show test run history for a task.

**Parameters:**
- `task_id` (integer, required): Task ID
- `limit` (integer, optional): Maximum number of traces to return (default: all)

**Example Request:**
```json
{
  "task_id": 1,
  "limit": 3
}
```

**Example Response:**
```json
[
  {
    "id": 3,
    "task_id": 1,
    "exit_code": 1,
    "output": "FAIL\nExpected 10 but got 11",
    "duration_ms": 450,
    "run_at": "2026-02-19T14:30:00Z"
  },
  {
    "id": 2,
    "task_id": 1,
    "exit_code": 0,
    "output": "PASS\nAll tests passed",
    "duration_ms": 320,
    "run_at": "2026-02-19T13:15:00Z"
  },
  {
    "id": 1,
    "task_id": 1,
    "exit_code": 1,
    "output": "FAIL\nTest timeout",
    "duration_ms": 5000,
    "run_at": "2026-02-19T12:00:00Z"
  }
]
```

## 🔄 Workflow Examples

### Complete Development Session

```json
// 1. Check current workspace
{}

// 2. Create main task
{
  "description": "Implement new authentication middleware",
  "test_command": "go test ./internal/auth"
}

// Response: {"task_id": 1, ...}

// 3. Start working
{
  "task_id": 1
}

// 4. Create subtasks
{
  "parent_task_id": 1,
  "subtasks": [
    {"description": "Add JWT verification"},
    {"description": "Implement rate limiting"}
  ]
}

// 5. Checkpoint before risky change
{
  "message": "Middleware skeleton complete"
}

// 6. Mark subtask as done (runs tests)
{
  "task_id": 2
}

// 7. View test history
{
  "task_id": 2,
  "limit": 5
}

// 8. Restore if tests fail
{}
```

### Error Recovery Flow

```json
// 1. Create task with test
{
  "description": "Fix memory leak",
  "test_command": "npm test -- --grep 'memory'"
}

// 2. Start task
{"task_id": 1}

// 3. Write code, test fails
{"task_id": 1}
// Response: {"status": "failed", "exit_code": 1, ...}

// 4. Checkpoint to save broken state
{
  "message": "Memory leak investigation (tests failing)"
}

// 5. Fix code
// 6. Retry test
{"task_id": 1}
// Response: {"status": "done", "exit_code": 0, ...}

// 7. Checkpoint successful fix
{
  "message": "Memory leak fixed"
}
```

## ⚙️ Configuration

### MCP Server Setup

The agit MCP server is configured via standard MCP configuration:

**Claude Code** (`.claude/mcp.json`):
```json
{
  "mcpServers": {
    "agit": {
      "command": "agit-mcp",
      "args": ["--stdio"]
    }
  }
}
```

**OpenCode** (`.opencode/plugins/agit.json`):
```json
{
  "name": "agit",
  "mcp": {
    "server": {
      "command": "agit-mcp",
      "args": ["--stdio"]
    }
  }
}
```

### Git Hooks

Install hooks for automatic workspace switching:
```bash
agit hook install
```

This creates a `.git/hooks/post-checkout` hook that automatically calls `workspace_switch` when you change branches.

## 🐛 Error Handling

agit tools return structured errors:

```json
{
  "error": "Task 999 not found in workspace feat/217-impl-check"
}
```

Common errors:
- `workspace not found`: Run `agit init` first
- `task not found`: Invalid task ID
- `parent task belongs to different workspace`: Can't attach subtask across branches
- `not a git repository`: Run in Git repository root

## 📚 See Also

- [CLI Reference](./cli.md) - Command‑line interface
- [README](./README.md) - Project overview
- [Design Spec](./agent/design-spec.md) - Architecture and data model