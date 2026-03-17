# OpenClaw Integration with agit

OpenClaw is a personal AI assistant platform that primarily uses CLI tools rather than MCP (Model Context Protocol). agit integrates seamlessly with OpenClaw via its command-line interface.

## 📋 Integration Methods

### 1. **Direct CLI Usage (Recommended)**
OpenClaw can execute `agit` commands directly in its agent sessions:

```bash
# Example OpenClaw agent workflow
agit init
agit task new --test "npm test" "Implement authentication middleware"
agit task start 1
# ... write code ...
agit task done 1
agit checkpoint "Authentication complete"
```

### 2. **Shell Function Wrappers**
Create custom functions in OpenClaw's tool system:

```bash
function agit_task_new() {
    local description="$1"
    local test_cmd="$2"
    agit task new --test "$test_cmd" "$description"
}

function agit_workspace_status() {
    agit workspace-status | jq .
}
```

### 3. **OpenClaw Plugin (Future MCP Support)**
When OpenClaw adds MCP support, use this configuration:

```json
// .openclaw/mcp.json (future)
{
  "mcpServers": {
    "agit": {
      "command": "agit-mcp",
      "args": ["--stdio"]
    }
  }
}
```

## 🚀 Quick Start

1. **Install agit**:
   ```bash
   go install ./cmd/agit
   ```

2. **Initialize in your project**:
   ```bash
   cd /your/project
   agit init
   ```

3. **Use in OpenClaw sessions**:
   ```bash
   # In OpenClaw agent or automation scripts
   agit workspace-status
   agit task list
   ```

## 🔄 Typical Workflow

```bash
# 1. Start session (OpenClaw agent)
openclaw agent --message "Work on project X"

# 2. Check agit status
agit workspace-status

# 3. Create task with test
agit task new --test "go test ./..." "Fix memory leak"

# 4. Start working
agit task start 1

# 5. Write code...

# 6. Complete task (runs tests)
agit task done 1

# 7. Create checkpoint
agit checkpoint "Memory leak fixed"
```

## ⚙️ Configuration

No special configuration needed. Ensure:
- `agit` binary is in `$PATH`
- Project has been initialized with `agit init`
- Git repository exists (agit requires `.git/`)

## 📚 Resources

- [agit CLI Reference](../cli.md)
- [Full Integration Guide](../INTEGRATION.md)
- [OpenClaw Documentation](https://docs.openclaw.ai)

## 🤝 Compatibility

| Feature | OpenClaw + agit |
|---------|-----------------|
| Workspace state | ✅ Yes (via CLI) |
| Task management | ✅ Yes (via CLI) |
| Test execution | ✅ Yes (via `task done`) |
| Checkpoints | ✅ Yes (via CLI) |
| Real‑time MCP | ❌ Not yet (CLI only) |

## 🔧 Troubleshooting

**"agit: command not found"**
```bash
export PATH="$HOME/go/bin:$PATH"
# Or use absolute path
/absolute/path/to/agit
```

**"Not a git repository"**
```bash
git init
agit init
```

**Permission errors**
```bash
chmod +x /path/to/agit
```

## 🤖 OpenClaw Integration Script

For convenient use within OpenClaw agent sessions, agit provides a ready‑to‑source Bash script with wrapper functions.

### Installation

1. **Copy the script** to your OpenClaw environment:

   ```bash
   cp .openclaw/tools/agit.sh ~/.openclaw/tools/
   ```

2. **Source it** in your shell or OpenClaw agent configuration:

   ```bash
   source ~/.openclaw/tools/agit.sh
   ```

3. **Optionally set `AGIT_BIN`** if agit is not in PATH:

   ```bash
   export AGIT_BIN="$HOME/go/bin/agit"
   ```

### Available Functions

The script provides these Bash functions:

| Function | Description | Example |
|----------|-------------|---------|
| `agit_workspace_status` | Get current workspace state (JSON) | `agit_workspace_status` |
| `agit_task_new` | Create new task with optional test command | `agit_task_new "Fix bug" "go test"` |
| `agit_task_list` | List tasks (optionally filtered by status) | `agit_task_list pending` |
| `agit_task_start` | Mark a task as active | `agit_task_start 1` |
| `agit_task_done` | Complete task and run its test command | `agit_task_done 1` |
| `agit_task_breakdown` | Create subtasks under a parent task | `agit_task_breakdown 1 '[{"description":"subtask"}]'` |
| `agit_task_tree` | Show task hierarchy (JSON or tree) | `agit_task_tree --format tree` |
| `agit_checkpoint` | Create checkpoint | `agit_checkpoint "Before refactor"` |
| `agit_restore` | Restore to last checkpoint | `agit_restore` |
| `agit_trace` | Show test history for a task | `agit_trace 1 --limit 5` |
| `agit_init` | Initialize agit in current repository | `agit_init` |
| `agit_hook_install` | Install git hooks | `agit_hook_install` |

### Tool Configuration Example

If your OpenClaw environment supports JSON tool definitions, you can use the example configuration file `.openclaw/tools/agit‑tools.json`. This file defines agit tools with parameters and descriptions that OpenClaw can import.

```json
[
  {
    "name": "agit_workspace_status",
    "description": "Get current workspace state",
    "command": "agit workspace-status"
  },
  ...
]
```

See the full file in `.openclaw/tools/agit‑tools.json`.

### Example OpenClaw Agent Session

```bash
# Start OpenClaw agent
openclaw agent --message "Implement authentication"

# In the agent session, use agit functions:
agit_init
agit_task_new --test "npm test" "Add JWT middleware"
agit_task_start 1
# ... write code ...
agit_task_done 1
agit_checkpoint "JWT middleware complete"
```

### JSON Output

Most agit commands output JSON by default (e.g., `workspace‑status`, `task tree --format json`). The wrapper functions preserve this output. If you have `jq` installed, you can pipe any agit command to `jq` for pretty‑printing:

```bash
agit_workspace_status | jq .
agit_task_tree --format json | jq .
```

---

*OpenClaw + agit = CLI‑powered state persistence for personal AI assistants*