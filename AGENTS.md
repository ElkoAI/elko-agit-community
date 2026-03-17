# AGENTS.md — agit Community Edition

This file provides guidance for AI coding agents (Claude Code, OpenCode, Cursor) working on **agit**.

## Quick Start for AI Agents

1. Run `agit init` to initialize state tracking
2. Run `agit setup` to create a project plan via the wizard
3. Use `agit task new <description> --test "<cmd>"` to create tasks
4. Use `agit task start <id>` / `agit task done <id>` to track progress
5. Use `agit checkpoint "<msg>"` before risky changes

## Project Overview

**agit** is a Go project with two binaries:
- `cmd/agit` — CLI interface
- `cmd/agit-mcp` — MCP server (stdio transport)

## Build & Test

```bash
go build ./...
go test ./...
go vet ./...
```

## Code Conventions

- Standard Go project layout (`cmd/`, `internal/`)
- SQLite via `github.com/mattn/go-sqlite3`
- MCP via `github.com/mark3labs/mcp-go`
- All database operations in `internal/db/`
- All MCP tool handlers in `internal/mcp/`

## Contributing

1. Fork the repo
2. Create a feature branch
3. Run `agit setup` to initialize your workspace
4. Submit a pull request with tests

See [TUTORIAL.md](./TUTORIAL.md) for full workflow examples.
