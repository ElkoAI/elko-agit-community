# agit Integration Guide

Complete guide for integrating agit with AI coding agents and development environments.

## 📋 Overview

agit provides **Git‑native state persistence** for AI coding agents via two interfaces:

1. **MCP (Model Context Protocol) Server** – Standard protocol for AI tool integration
2. **CLI (Command‑Line Interface)** – Manual/script access to all features

All major AI coding agents support MCP, making agit a universal state‑tracking solution.

## 🛠️ MCP Integration

### What is MCP?

**Model Context Protocol (MCP)** is an open standard developed by Anthropic for AI tools. It enables:
- AI agents to discover and call tools
- Tools to expose capabilities via JSON‑RPC over stdio
- Standardized authentication, error handling, and schemas

agit implements MCP server `agit-mcp` that exposes 10 tools for workspace, task, checkpoint, and trace management.

### How Agents Use agit via MCP

When configured, AI agents can:

1. **Discover agit tools** at startup
2. **Call tools with JSON parameters** (e.g., `task_new`, `checkpoint`)
3. **Receive structured responses** with task IDs, status, test results
4. **Maintain state across sessions** via workspace persistence

### Universal MCP Configuration

All MCP‑compatible agents use similar configuration:

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

The `agit-mcp` binary must be built and in your PATH (or use absolute path).

## 🤖 Agent‑Specific Integration

### Claude Code

**Configuration Location:** `.claude/mcp.json`

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

**Workflow Example:**
```
Claude: "I need to implement authentication middleware."
→ Uses agit tool `task_new` with description and test command
→ Gets task ID 1
→ Uses `task_start` to mark as active
→ Writes code...
→ Uses `task_done` to run tests and mark complete
→ Uses `checkpoint` before risky changes
```

### OpenCode

**Configuration Location:** `.opencode/plugins/agit.json`

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

**Workflow Example:**
```
OpenCode session starts
→ Calls `workspace_status` to restore context
→ Sees current task is "Fix memory leak" (ID: 3)
→ Continues where last session left off
→ Uses `task_done` when fix complete (runs tests)
→ Creates new task for next feature
```

### Cursor / VS Code (with MCP)

**Configuration Location:** Depends on MCP client plugin

Cursor (as of 2026) supports MCP via configuration file:
```
~/.cursor/mcp.json
```

Or project‑local:
```
.cursor/mcp.json
```

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

**Note:** Cursor's built‑in agent uses different integration; MCP support may be via extension.

### Zed (Future)

Zed has announced MCP support coming. Expected configuration:
```
~/.zed/settings.json
```

```json
{
  "mcp_servers": {
    "agit": {
      "command": "agit-mcp",
      "args": ["--stdio"]
    }
  }
}
```

### GitHub Copilot / Codex

Currently, GitHub Copilot doesn't support MCP directly. However:

1. **Use agit CLI** in Copilot Chat or terminal
2. **Manual workflow** with `agit` commands
3. **Future**: MCP support may be added

### OpenClaw (CLI‑based)

OpenClaw is a personal AI assistant platform that primarily uses CLI tools rather than MCP. Integration is straightforward via agit's CLI:

**Usage Pattern:**
```bash
# OpenClaw agent can execute agit commands directly
openclaw agent --message "Create agit task for authentication"
# → Agent runs: agit task new --test "npm test" "Implement authentication"
# → Agent runs: agit task start 1
# → Agent writes code...
# → Agent runs: agit task done 1
```

**Configuration:** No MCP configuration needed. Ensure `agit` is in PATH or use absolute path.

**Workflow Example:**
```bash
# OpenClaw shell tool can wrap agit commands
function agit_task_new() {
    agit task new --test "$2" "$1"
    echo "Task created"
}

# Agent calls the function via OpenClaw's tool system
```

**Benefits:**
- **Direct CLI access** – No MCP overhead
- **Scriptable workflows** – Use in OpenClaw automation scripts
- **State persistence** – Same Git‑native storage as MCP

**See also:** `.openclaw/README.md` for detailed integration examples.

## 🔄 Typical Agent Workflow

### Session Start (Context Recovery)

```json
// Agent calls automatically:
{
  "tool": "workspace_status",
  "parameters": {}
}

// Response:
{
  "branch": "feat/217-auth",
  "current_task_id": 5,
  "agent_state": {
    "last_action": "Implemented JWT verification",
    "context": "Working on rate limiting"
  },
  "todo_list": "- [ ] Add Redis caching\n- [ ] Write integration tests"
}
```

Agent resumes work where it left off.

### Creating and Working on Tasks

```json
// 1. Create task
{
  "tool": "task_new",
  "parameters": {
    "description": "Add Redis caching",
    "test_command": "npm test -- --grep 'cache'"
  }
}

// 2. Start task
{
  "tool": "task_start",
  "parameters": {
    "task_id": 6
  }
}

// 3. Write code...

// 4. Complete task (runs tests)
{
  "tool": "task_done",
  "parameters": {
    "task_id": 6
  }
}
```

### Checkpoint System

```json
// Before risky refactor
{
  "tool": "checkpoint",
  "parameters": {
    "message": "Before Redis client abstraction"
  }
}

// If something breaks
{
  "tool": "restore",
  "parameters": {}
}
```

## 🛠️ CLI Integration (Fallback / Alternative)

When MCP isn't available or for manual control:

### Direct CLI Usage

```bash
# Initialize agit
agit init
agit hook install

# Create task (agent or human)
agit task new --test "go test ./internal/db" "Refactor database layer"

# Check status
agit workspace-status
agit task list

# Complete task (runs tests)
agit task done 1

# Create checkpoint
agit checkpoint "Database refactor complete"
```

### Script Integration

```bash
#!/bin/bash
# pre‑commit hook using agit
current_task=$(agit workspace-status | jq -r '.current_task_id')
if [ "$current_task" != "null" ]; then
  echo "Cannot commit: task $current_task is active"
  exit 1
fi
```

### CI/CD Integration

```yaml
# GitHub Actions example
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
      - name: Build agit
        run: go build ./cmd/agit
      - name: Initialize agit
        run: |
          agit init --force
          agit task new --test "go test ./..." "Run CI tests"
          agit task start 1
          agit task done 1
```

## 📊 State Persistence Across Agents

### Multi‑Agent Scenario

Different agents can work on the same project:

```
Day 1: OpenCode
  → Creates task #1 "Implement auth"
  → Starts task #1
  → Checkpoint "Auth skeleton"

Day 2: Claude Code  
  → workspace_status shows task #1 active
  → Continues auth implementation
  → Completes task #1 (runs tests)
  → Creates task #2 "Add tests"
```

### State Compatibility

agit ensures:
- **Branch‑based isolation**: `main` vs `feat/x` have separate states
- **Agent‑agnostic storage**: JSON state works with any MCP client
- **Git‑integrated**: Checkpoints are real commits

## ⚙️ Advanced Integration Patterns

### 1. Project‑Specific Workflows

Create `.agit-manifest.json` to seed project state:

```json
{
  "version": "1.0.0",
  "name": "my-project",
  "tasks": [
    {
      "id": 1,
      "description": "Set up CI/CD pipeline",
      "test": "bash scripts/test-ci.sh"
    }
  ],
  "agents": {
    "preferred": "opencode",
    "integrations": ["claude-code", "cursor"]
  }
}
```

### 2. Git Hooks for Automation

```bash
# .git/hooks/post-checkout (installed by agit hook install)
#!/bin/bash
branch=$(git rev-parse --abbrev-ref HEAD)
commit=$(git rev-parse HEAD)
echo "{\"branch\":\"$branch\",\"commit\":\"$commit\"}" | \
  agit-mcp tool workspace_switch 2>/dev/null || true
```

### 3. Custom Agent Instructions

Add agit‑specific instructions to agent prompts:

```
For this project, use agit for state management:

1. Check workspace_status at session start
2. Create tasks with test commands
3. Use checkpoints before risky changes
4. Run tests via task_done
5. Break down complex work with task_breakdown
```

### 4. Team Collaboration

```bash
# Team member A
agit task new "Implement API endpoint"
agit task start 1
# ... code ...
agit checkpoint "API endpoint draft"

# Team member B (pulls changes)
git pull
agit workspace-status  # Sees task #1 active
agit task list        # Can see all tasks
```

## 🔧 Troubleshooting

### Common Issues

**"command not found: agit-mcp"**
```bash
# Build and install
go build -o ~/bin/agit-mcp ./cmd/agit-mcp
# Or use absolute path in config
"command": "/path/to/agit-mcp"
```

**"Not a git repository"**
```bash
cd /your/project/root
agit init
```

**MCP connection errors**
- Verify `agit-mcp` builds without errors
- Check agent MCP configuration syntax
- Ensure no other process is using stdio

**"workspace not found"**
```bash
agit init  # Initialize database
agit hook install  # Optional but recommended
```

### Debugging

```bash
# Test MCP server directly
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | agit-mcp

# Check database
sqlite3 .git/agit.db ".tables"
sqlite3 .git/agit.db "SELECT * FROM workspaces"

# Verbose logging (if implemented)
AGIT_DEBUG=1 agit-mcp
```

## 📚 Best Practices

### For AI Agents

1. **Always start with `workspace_status`** – Recover context
2. **Create tasks with test commands** – Enable automatic verification
3. **Use checkpoints frequently** – Save recoverable states
4. **Break down complex work** – Use `task_breakdown` for large features
5. **Check test history** – Use `trace` to debug failures

### For Developers

1. **Initialize agit early** – `agit init` at project start
2. **Install hooks** – `agit hook install` for branch switching
3. **Use CLI for manual tasks** – Quick status checks, task creation
4. **Review agent work** – `agit task tree`, `agit trace`
5. **Integrate with CI** – Run agit tasks in automated tests

### For Teams

1. **Shared agit database** – `.git/agit.db` is in `.git/`, shared via Git
2. **Consistent workflows** – Agree on task/checkpoint patterns
3. **Cross‑agent compatibility** – Different agents can collaborate
4. **Documentation** – Add `.agit-manifest.json` for project setup

## 🚀 Getting Started Checklist

1. [ ] **Build agit**: `go build ./cmd/agit-mcp`
2. [ ] **Initialize project**: `cd /project && agit init`
3. [ ] **Install hooks**: `agit hook install` (optional)
4. [ ] **Configure agent**: Add MCP config for your AI assistant
5. [ ] **Test integration**: Agent should see `workspace_status`
6. [ ] **Create first task**: Via agent or `agit task new`
7. [ ] **Establish workflow**: Checkpoints, test commands, breakdowns

## 📖 See Also

- [MCP Interface](./mcp.md) – Complete tool reference
- [CLI Reference](./cli.md) – Command‑line guide  
- [README](./README.md) – Project overview
- [Design Spec](./agent/design-spec.md) – Architecture details

---

*agit bridges the gap between Git's version control and AI agent workflows, enabling stateful, test‑driven development across any MCP‑compatible assistant.*