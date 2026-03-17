# agit Tutorial: Git-Native State for AI Agents

> **Complete step‑by‑step guide** with real‑world scenarios, terminal examples, and troubleshooting

---

## 📋 Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Project Setup](#project-setup)
- [Real‑World Scenario: Go REST API](#realworld-scenario-go-rest-api)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Next Steps](#next-steps)

---

## 🎯 Overview

agit is a **Git‑native state persistence system** for AI coding agents. It solves the "stateless agent" problem by storing task context, test history, and checkpoints directly in your Git repository (`.git/agit.db`).

**What this tutorial covers:**
1. ✅ Installing agit from source
2. ✅ Setting up a new project with agit
3. ✅ Building a **real Go REST API** using agit + OpenCode
4. ✅ Recovering context across multiple AI sessions
5. ✅ Using checkpoints, task decomposition, and test tracing

**Time estimate:** 30‑45 minutes for the complete tutorial

---

## 📦 Prerequisites

### System Requirements
- **Go 1.21+** – [Install](https://go.dev/dl/)
- **Git** – Any recent version
- **SQLite** (automatically via `go‑sqlite3`)
- **Terminal** with bash/zsh

### AI Agent (Choose one)
- **OpenCode** with MCP support (recommended)
- **Claude Code** with MCP support
- **Cursor** with MCP plugin
- **OpenClaw** (CLI‑based, no MCP needed)
- Any MCP‑compatible AI coding assistant

**Note:** This tutorial uses **OpenCode** as the example agent, but all MCP‑compatible agents work similarly.

---

## 🔧 Installation

### 1. Clone and Build agit

```bash
# Clone the repository
git clone https://github.com/jsoprych/elko-agit.git
cd elko-agit

# Build both binaries
go build ./cmd/agit      # CLI interface
go build ./cmd/agit-mcp  # MCP server for AI agents

# Verify binaries work
./agit --version
# agit version 0.3.0

./agit-mcp --help
# agit-mcp: Go MCP server for agit
```

### 2. Optional: Install to PATH

```bash
# Copy to a directory in your PATH
sudo cp agit agit-mcp /usr/local/bin/

# Or add to ~/bin (if in PATH)
mkdir -p ~/bin
cp agit agit-mcp ~/bin/
```

**✅ Checkpoint:** You should have two working binaries: `agit` (CLI) and `agit‑mcp` (MCP server).

---

## 🚀 Project Setup

### 1. Create a New Go Project

```bash
# Create project directory
mkdir go-rest-api-tutorial
cd go-rest-api-tutorial

# Initialize Go module
go mod init example.com/api

# Create basic structure
mkdir -p internal/{handlers,models,db}
touch main.go internal/handlers/user.go
```

### 2. Initialize agit in Your Project

```bash
# Run from inside your project directory
/path/to/agit init

# Output:
# Initializing agit database at /path/to/go-rest-api-tutorial/.git/agit.db
# agit database initialized successfully

# Install Git hook (recommended)
/path/to/agit hook install

# Output:
# Installing post-checkout hook at /path/to/go-rest-api-tutorial/.git/hooks/post-checkout
# Hook installed successfully
```

**What happened:**
- Created `.git/agit.db` SQLite database
- Installed Git hook for automatic workspace switching
- Ready to track agent state

### 3. Configure Your AI Agent

**For OpenCode:**
```bash
mkdir -p .opencode/plugins

cat > .opencode/plugins/agit.json <<EOF
{
  "name": "agit",
  "mcp": {
    "server": {
      "command": "/absolute/path/to/agit-mcp",
      "args": ["--stdio"]
    }
  }
}
EOF
```

**For Claude Code:**
```bash
mkdir -p .claude

cat > .claude/mcp.json <<EOF
{
  "mcpServers": {
    "agit": {
      "command": "/absolute/path/to/agit-mcp",
      "args": ["--stdio"]
    }
  }
}
EOF
```

**Note:** Use absolute paths to avoid PATH issues.

### 4. Verify Configuration

```bash
# Check Git repository status
git status

# Check agit workspace status
/path/to/agit workspace-status

# Expected output (JSON):
# {
#   "branch": "main",
#   "current_task_id": null,
#   "agent_state": {},
#   "todo_list": "",
#   "last_commit": "..."
# }
```

**✅ Checkpoint:** Your project is now agit‑enabled with MCP configured.

---

## 🏗️ Real‑World Scenario: Go REST API

Let’s build a simple REST API with user authentication, using agit to manage the AI workflow.

### Scenario Goals:
1. **Session 1:** Project setup and basic structure
2. **Session 2:** User authentication implementation  
3. **Session 3:** REST endpoints and testing
4. **Session 4:** Documentation and cleanup

---

### Session 1: Project Setup

**Start fresh AI session** → OpenCode automatically calls `workspace_status`:

```json
{
  "branch": "main",
  "current_task_id": null,
  "todo_list": "",
  "agent_state": {}
}
```

**Create first task via MCP:**

```json
{
  "tool": "task_new",
  "parameters": {
    "description": "Set up Go REST API project structure",
    "test_command": "go mod tidy && go test ./... 2>&1 | head -20"
  }
}
```

**Response:**
```json
{
  "task_id": 1,
  "status": "pending"
}
```

**Start the task:**
```json
{
  "tool": "task_start",
  "parameters": {
    "task_id": 1
  }
}
```

**Agent creates project files:**

```bash
# Terminal output from agent
Creating main.go...
Creating internal/handlers/user.go...
Creating go.mod dependencies...
Writing initial tests...
```

**Checkpoint before risky changes:**
```json
{
  "tool": "checkpoint",
  "parameters": {
    "message": "Project skeleton complete"
  }
}
```

**Complete task (runs tests):**
```json
{
  "tool": "task_done",
  "parameters": {
    "task_id": 1
  }
}
```

**Output:**
```
Running test command: go mod tidy && go test ./... 2>&1 | head -20
✓ Tests passed
Task #1 marked as done
```

---

### Session 2: User Authentication

**Next day, new AI session starts** → `workspace_status` shows:

```json
{
  "branch": "main",
  "current_task_id": null,
  "todo_list": "- [ ] Implement user authentication\n- [ ] Add JWT support",
  "agent_state": {
    "last_action": "Created project skeleton",
    "context": "Ready for authentication implementation"
  }
}
```

**Create authentication task:**

```bash
# Using CLI this time (agent or human)
/path/to/agit task new \
  --test "go test ./internal/auth" \
  "Implement user authentication with JWT"
```

**Output:**
```
Task #2 created
```

**Break down into subtasks:**

```bash
/path/to/agit task breakdown \
  --parent 2 \
  --subtasks '[
    {"description": "Create User model and database schema", "test_command": "go test ./internal/models"},
    {"description": "Implement password hashing", "test_command": "go test ./internal/auth -run TestPassword"},
    {"description": "Add JWT token generation/validation", "test_command": "go test ./internal/auth -run TestJWT"}
  ]'
```

**Output:**
```
Created 3 subtasks under task #2:
  Task #3: Create User model and database schema
  Task #4: Implement password hashing  
  Task #5: Add JWT token generation/validation
```

**Work on subtasks:**

```bash
# Start first subtask
/path/to/agit task start 3

# ... agent implements User model ...

# Complete subtask
/path/to/agit task done 3

# Check test history
/path/to/agit trace --task 3 --limit 1
```

**Task tree view:**
```bash
/path/to/agit task tree
```

**Output:**
```
Task Tree (workspace: main)
├── #1: Set up Go REST API project structure ✓
└── #2: Implement user authentication with JWT
    ├── #3: Create User model and database schema ✓
    ├── #4: Implement password hashing (active)
    └── #5: Add JWT token generation/validation
```

---

### Session 3: REST Endpoints

**Create endpoint tasks:**

```bash
/path/to/agit task new \
  --test "go test ./internal/handlers -run TestUser" \
  "Implement user registration/login endpoints"

/path/to/agit task new \
  --test "go test ./internal/handlers -run TestProtected" \
  "Add protected endpoints with auth middleware"
```

**Check workspace status:**

```bash
/path/to/agit workspace-status | jq '.todo_list'
```

**Output:**
```
"- [ ] Implement user authentication\n- [ ] Add JWT support\n- [ ] User registration endpoint\n- [ ] Login endpoint\n- [ ] Protected routes"
```

**Create checkpoint before major refactor:**

```bash
/path/to/agit checkpoint "Before middleware refactor"
```

**If something breaks, restore:**
```bash
/path/to/agit restore
# Output: Restored to checkpoint "Before middleware refactor"
```

---

### Session 4: Documentation & Cleanup

**View all tasks:**

```bash
/path/to/agit task list
```

**Output:**
```
ID  Status  Description
1   done    Set up Go REST API project structure
2   active  Implement user authentication with JWT
3   done    Create User model and database schema
4   done    Implement password hashing
5   active  Add JWT token generation/validation
6   pending Implement user registration/login endpoints
7   pending Add protected endpoints with auth middleware
```

**Complete remaining tasks:**

```bash
/path/to/agit task done 5
/path/to/agit task start 6
/path/to/agit task done 6
/path/to/agit task start 7  
/path/to/agit task done 7
/path/to/agit task done 2  # Parent task
```

**Final checkpoint:**

```bash
/path/to/agit checkpoint "REST API v1.0 complete"
```

---

## 🎨 Advanced Usage

### 1. Dashboard for Human Oversight

```bash
# Start the web dashboard
/path/to/agit dashboard --port 8080

# Open browser to:
# http://localhost:8080
```

**Dashboard features:**
- 📊 Workspace overview
- 🌳 Task hierarchy visualization  
- 📈 Test history browser
- 📝 API documentation (auto‑generated)
- 🛠️ Command execution forms

### 2. Multi‑Agent Collaboration

**Agent A (OpenCode):**
```bash
agit task new "Implement rate limiting"
agit task start 8
agit checkpoint "Rate limiting draft"
```

**Agent B (Claude Code) later:**
- `workspace_status` shows task #8 active
- Continues implementation
- `task_done 8` runs tests

### 3. CI/CD Integration

**.github/workflows/test.yml:**
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup agit
        run: |
          go build ./cmd/agit
          ./agit init --force
          ./agit task new --test "go test ./..." "CI test run"
          ./agit task start 1
          ./agit task done 1
```

### 4. Custom Manifest (Future Feature)

**`.agit‑manifest.json` (conceptual):**
```json
{
  "version": "1.0.0",
  "name": "go-rest-api",
  "tasks": [
    {
      "id": 1,
      "description": "Project setup",
      "test": "go mod tidy"
    }
  ],
  "agents": {
    "preferred": "opencode"
  }
}
```

---

## 🔧 Troubleshooting

### Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| **"command not found: agit‑mcp"** | Use absolute path in MCP config |
| **"Not a git repository"** | Run `agit init` inside Git repo |
| **MCP connection errors** | Verify `agit‑mcp` builds without errors |
| **Database locked** | Only one agent per workspace; use `agit init --force` to reset |
| **Test command fails** | Check command syntax; use `sh -c` compatible commands |
| **Workspace not found** | `agit init` creates database; check Git repo |
| **Checkpoint fails** | Ensure Git is configured (user.email, user.name) |

### Debugging Commands

```bash
# Test MCP server directly
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | agit-mcp

# Examine database
sqlite3 .git/agit.db ".tables"
sqlite3 .git/agit.db "SELECT * FROM workspaces LIMIT 1;"

# Check Git hooks
ls -la .git/hooks/post-checkout
cat .git/hooks/post-checkout

# Reset agit (careful!)
rm -f .git/agit.db
agit init --force
```

### Agent‑Specific Troubleshooting

**OpenCode:**
- Check `.opencode/plugins/agit.json` syntax
- Restart OpenCode after config changes
- Verify MCP logs in OpenCode interface

**Claude Code:**
- Ensure `.claude/mcp.json` exists in project root
- Claude Desktop may need restart
- Check Claude’s MCP server list

**Cursor:**
- MCP via extension may need separate setup
- Check extension documentation
- May require global MCP config

---

## 📚 Next Steps

### 1. Explore Dashboard Features
```bash
agit dashboard
# Visit: http://localhost:8080/api-docs
# Try the interactive command executor
```

### 2. Integrate with Your Workflow
- Add agit tasks to your existing projects
- Use checkpoints before risky changes
- Implement CI/CD with agit test tracking

### 3. Contribute to agit
- Report issues on GitHub
- Suggest new features
- Submit pull requests

### 4. Learn More
- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [OpenCode Documentation](https://opencode.ai/docs)
- [Claude Code MCP Guide](https://docs.anthropic.com/claude/docs/mcp)

---

## 📝 Summary

You’ve now completed a **real‑world scenario** building a Go REST API with agit:

1. ✅ **Installed agit** and configured MCP for your AI agent
2. ✅ **Initialized agit** in a new project with Git hooks
3. ✅ **Created and managed tasks** with test commands
4. ✅ **Used checkpoints** for safe experimentation
5. ✅ **Decomposed complex tasks** into subtasks
6. ✅ **Tracked test history** across sessions
7. ✅ **Recovered context** between AI sessions

agit transforms stateless AI agents into **stateful collaborators** that remember what they were doing, track test results, and integrate seamlessly with Git workflows.

**Remember:** The `.git/agit.db` file contains all your project’s agent state. Commit it to share state across team members or machines.

---

*Need help?*  
- Check the [agit GitHub repository](https://github.com/jsoprych/elko-agit)  
- Read the [complete documentation](./README.md)  
- Open an issue for bugs or feature requests