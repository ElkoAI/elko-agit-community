<p align="center">
  <img src="docs/images/logo.png" alt="agit logo" width="120"/>
</p>

# agit - The AI Workflow Machine

**Git‑native state persistence + workflow orchestration for AI coding agents**  
*The missing link between Git's version control and agentic workflows*  
*Built by **[Elko.AI](https://elko.ai)'s Dark Software Factory***

[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Built by Elko.AI](https://img.shields.io/badge/Built_by-Elko.AI-black)](https://elko.ai)

**agit** is the **AI workflow machine**—a Go MCP server that provides Git‑native state persistence **and workflow orchestration** for AI coding agents. It lives in `.git/agit.db` and acts as the conductor between Git's version control and agentic workflows, keeping AI agents on track with test‑driven tasks, contextual checkpoints, and project‑aware requirements.

## 🎯 The AI Workflow Machine

AI coding agents (OpenCode, Claude Code, Cursor) are stateless—brilliant but forgetful. **agit** is the **workflow machine** that keeps them on track, transforming vague prompts into structured, test‑driven development workflows:

- **🧠 Context Recovery**: Persistent state across AI sessions, no more starting from scratch
- **🏗️ Workflow Orchestration**: Project‑aware task planning with architecture‑guided decomposition  
- **🧪 Test‑Driven Discipline**: Automatic test execution with historical traceability
- **🔀 Git‑Integrated Checkpoints**: Commits + agent state snapshots for risk‑free experimentation
- **🌿 Workspace Awareness**: Branch‑specific context switching for parallel feature development
- **🤖 MCP + CLI Interfaces**: Full control via AI agents or command‑line

## ✨ Features

- **🧠 State Persistence**: Store agent context, todo lists, and task state in `.git/agit.db`
- **🤖 MCP Server**: 11 tools for AI agents to interact with agit state
- **⌨️ CLI Interface**: Mirror of all MCP tools for manual/script use
- **🧪 Test Integration**: Automatic test execution and history tracking
- **🔀 Checkpoint System**: Git commits + agent state snapshots
- **🌳 Task Decomposition**: Parent/child tasks with hierarchical views
- **📋 Manifest System**: Project configuration with `.agit-manifest.json` for auto‑seeding tasks
- **🎯 Project Planning Wizard**: Interactive interview that creates detailed project plans with architecture‑aware task breakdowns
- **📊 Minimum Requirements System**: `.agit-requirements.json` defines structure, quality, workflow, and security standards
- **🔧 Git Hooks**: Automatic workspace switching on branch changes
- **⚡ Minimal Dependencies**: SQLite + Go standard library

## 🚀 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/yourusername/agit.git
cd agit
go build ./cmd/agit
go build ./cmd/agit-mcp

# Install binaries (optional)
go install ./cmd/agit
go install ./cmd/agit-mcp
```

### Initialize agit in Your Project

```bash
cd /your/project
agit init
agit hook install
```

### Configure Your AI Agent

**Claude Code**: Create `.claude/mcp.json`:
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

**OpenCode**: Create `.opencode/plugins/agit.json`:
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

## 📋 Core Concepts

| Term | Definition | Example |
|------|------------|---------|
| **Workspace** | Current Git branch + agent context | `feat/217-impl-check` |
| **Task** | Atomic unit of work with test command | `Task: implement method set comparison` |
| **Checkpoint** | Git commit + agent state snapshot | `agit checkpoint "before risky refactor"` |
| **Trace** | Test run history for a task | Exit codes, output, duration |
| **Todo List** | Markdown checklist for current workspace | Agent's plan of action |

## 🌱 The Input Requirement: Seeding the Workflow Machine

agit is a **workflow machine** that needs **sufficiently complete input** to drive effective agentic workflows. Think of it as fuel for the machine—without structured tasks, context, and tests upfront, AI agents may spin wheels on vague goals.

### The Challenge
- agit requires **initial tasks** with **test commands** to track progress
- **Technology stack** must be defined (Go, Node, Python, etc.) for appropriate tooling
- **Project structure** should be established before agents can work efficiently

### Solutions

1. **Manifest System** (`.agit-manifest.json`) – Static project configuration that seeds initial tasks
2. **Bootstrap Wizard** (`agit setup`) – Interactive setup that guides users through project initialization, detects tech stack, and generates optimal agit configuration
3. **Import from Existing Tools** – Trello, Jira, GitHub Issues, or Markdown import capabilities (future)

**Phase 18 Complete**: `agit setup` is now available – a wizard that ensures your project has the complete input agit needs to start orchestrating AI‑driven development.

*Read the [Input Requirement Discussion](./docs/ISSUE-INPUT-REQUIREMENT.md) for detailed analysis.*

## 🚀 Project Planning Wizard (`agit setup`)

The `agit setup` command now features a **conversational project planning wizard** that conducts a detailed interview to create comprehensive project plans—giving agit the complete input needed to orchestrate AI‑driven development from day one.

```bash
agit setup
```

### 🎯 **What the Wizard Captures:**

1. **Project Type & Architecture** – Web‑app, CLI tool, library, API, data‑pipeline, or custom
2. **Tech Stack & Test Strategy** – Language‑specific tooling with smart defaults
3. **Key Features** – User‑defined requirements turned into actionable tasks  
4. **Minimum Requirements** – Structure, code quality, workflow, and security standards
5. **Task Breakdown** – Architecture‑aware decomposition with dependencies and priorities

### ✨ **Enhanced Features:**

- **🗣️ Conversational Interview** – Guides you through project planning with contextual prompts
- **🏗️ Architecture‑Aware Tasks** – Generates task breakdowns specific to your project type
- **📊 Minimum Requirements System** – Creates `.agit-requirements.json` with structure, quality, workflow, and security standards
- **🧪 Language‑Smart Defaults** – Test commands, linters, and formatters tailored to your stack
- **🔧 Auto‑Detect Mode** – `--auto` flag for quick, non‑interactive setup
- **🏥 Validation & Repair** – `agit doctor` validates against requirements and can fix issues

### 📋 **Example Wizard Session:**

```bash
$ agit setup
==================================================
agit Project Planning Wizard
I'll help you create a detailed project plan that agit can use to
orchestrate AI-driven development from day one.
==================================================

Project name [my‑project]: my‑api
Brief project description (what are you building)?: A REST API for task management

What type of project are you building?
  1) Web Application (frontend + backend)
  2) CLI Tool / Utility
  3) Library / Package
  4) API / Microservice
  5) Data Pipeline / ETL
  6) Other
Choice (1‑6): 4

What's the primary technology stack?
  1) Go
  2) Node.js (JavaScript/TypeScript)
  3) Python
  4) Rust
  5) Other (specify test command)
Choice (1‑5): 1
...
```

### ⚡ **Quick Setup Options:**

```bash
# Full interactive project planning
agit setup

# Auto‑detect tech stack (non‑interactive)
agit setup --auto --hooks --agent-configs

# Validate your project against requirements
agit doctor

# Fix missing requirements
agit doctor --fix
```

*See [Bootstrap Documentation](./docs/BOOTSTRAP.md) for full details.*

## 🛠️ Usage Examples

### Basic Workflow

```bash
# Initialize agit
agit init
agit hook install

# Create a task with test command
agit task new "Implement feature X" --test "go test ./..."

# Start working on the task
agit task start 1

# ... write code ...

# Mark task as done (runs tests automatically)
agit task done 1

# Create a checkpoint
agit checkpoint "Feature X complete"

# View task tree
agit task tree

# See test history
agit trace --task 1
```

### MCP Workflow (AI Agent)

```json
// 1. Create task via MCP
{
  "description": "Implement method set comparison",
  "test_command": "go test ./internal/analyzer"
}

// 2. Start task
{
  "task_id": 1
}

// 3. Checkpoint before risky change
{
  "message": "Before risky refactor"
}

// 4. Restore if needed
{}
```

## 📖 Documentation

- **[MCP Interface](./mcp.md)** - Complete MCP tool reference with examples
- **[CLI Reference](./cli.md)** - Command‑line interface guide  
- **[Integration Guide](./INTEGRATION.md)** - How to use with AI agents (Claude Code, OpenCode, Cursor, Zed)
- **[Tutorial](./TUTORIAL.md)** - Step‑by‑step guide with real‑world scenarios
- **[Architecture](./agent/design-spec.md)** - Technical design and data model
- **[Agent Workflow](./agent/AGENT_WORKFLOW.md)** - Development workflow for AI agents

## 🏗️ Architecture

```
agit (Go binary)
├── cmd/
│   ├── agit                 # CLI interface
│   └── agit-mcp            # MCP server (stdio)
├── internal/
│   ├── db/                 # SQLite schema/operations
│   ├── git/                # Git porcelain/plumbing
│   ├── mcp/                # MCP tool handlers
│   └── workspace/          # Workspace state machine
└── .git/agit.db            # Single source of truth
```

### Database Schema

agit uses SQLite with these core tables:
- `workspaces` - Branch‑based agent state
- `tasks` - Work units with test commands
- `checkpoints` - Git commit + state snapshots
- `traces` - Test run history

See [design spec](./agent/design-spec.md) for full schema.

## 🔧 Development

### Build Commands

```bash
# Build all binaries
go build ./...

# Build MCP server
go build -o agit-mcp ./cmd/agit-mcp

# Build CLI
go build -o agit ./cmd/agit

# Run tests
go test ./...

# Format code
go fmt ./...

# Static analysis
go vet ./...
```

### Project Structure

- `cmd/` - CLI and MCP server entry points
- `internal/db/` - SQLite database layer
- `internal/git/` - Git operations
- `internal/mcp/` - MCP tool implementations
- `agent/` - Design specs and workflow documentation
- `.git/agit.db` - Database file (created by `agit init`)

## 👥 Authors & Sponsors

**Primary Development & AI Assistance**  
- **[Elko.AI](https://elko.ai) "TheMachine" Dark Software Factory** – Core architecture, AI‑driven development, and production engineering  
- **John Soprych** <johnsoprych@gmail.com> – Creator, maintainer, and project lead

*agit is proudly built and supported by Elko.AI's agentic software development platform.*

## 📄 License

MIT License - see [LICENSE](./LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Follow the [agent workflow](./agent/AGENT_WORKFLOW.md)
4. Submit a pull request

## 🚧 Roadmap

- [x] **Phase 0**: Core Infrastructure (init, hooks, DB schema)
- [x] **Phase 1**: MCP Server Basics (workspace, tasks)
- [x] **Phase 2**: Checkpoint System (commit + state snapshots)
- [x] **Phase 3**: Test Tracing (automatic test execution)
- [x] **Phase 4**: Agent Integrations (MCP configs)
- [x] **Phase 5**: Task Decomposition (parent/child tasks)
- [x] **Phase 6**: CLI Parity (mirror all MCP tools)
- [x] **Phase 7‑15**: Enhancements (security, testing, dashboard, API definitions)
- [x] **Phase 16**: Tutorial documentation (TUTORIAL.md)
- [x] **Phase 17**: Manifest System (`.agit‑manifest.json` for project seeding)
- [x] **Phase 18**: Project Planning Wizard (`agit setup` with conversational interview & requirements system)
- [ ] **Future**: Plugin system, extended imports, enterprise features

## 📚 Related Projects

- [MCP (Model Context Protocol)](https://spec.modelcontextprotocol.io/) - Standard for AI tool integration
- [OpenCode](https://opencode.ai/) - AI coding assistant with MCP support
- [Claude Code](https://claude.ai/code) - Claude's coding environment

---

*agit: The missing link between Git's version control and agentic workflows.*

---

## ☕ Support

If these tools are useful to you, consider supporting Elko.AI's open source work:

[![Support on Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/elkoai)

*Built by [Elko.AI](https://elko.ai) × [DarkFabrik.AI](https://darkfabrik.ai) — open source AI dev tooling from the Dark Software Factory.* 🖤
