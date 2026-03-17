# Bootstrap Wizard: Solving agit's Input Requirement

## The Input Requirement Problem

agit is a stateful workflow engine for AI‑driven development, but it requires a **well‑formed seed** to start effectively. Without an initial set of tasks, test commands, and agent configurations, agit cannot orchestrate meaningful progress.

The problem is documented in [`docs/ISSUE-INPUT-REQUIREMENT.md`](./ISSUE-INPUT-REQUIREMENT.md). In short, users must manually perform six or more steps before agit can operate optimally:

1. Initialize the agit database (`agit init`)
2. Install Git hooks (`agit hook install`)
3. Detect the project’s tech stack and decide on appropriate test commands
4. Create a manifest with initial tasks (`.agit-manifest.json`)
5. Generate MCP configuration files for AI agents (Claude Code, OpenCode, Cursor, etc.)
6. Validate that everything is correctly set up

Missing any of these steps leads to a sub‑optimal experience.

## The Bootstrap Wizard (`agit setup`)

Phase 18 of agit introduces a **bootstrap wizard** that automates the entire onboarding process. The wizard:

- **Detects** the project’s tech stack (Go, Node.js, Python, Rust, etc.)
- **Suggests** appropriate test commands (`go test ./...`, `npm test`, `pytest`, etc.)
- **Generates** a tailored manifest with two starter tasks
- **Optionally installs** the Git post‑checkout hook for automatic workspace switching
- **Optionally creates** agent‑specific MCP configuration files
- **Validates** the resulting setup and can repair missing pieces (`agit doctor`)

### Usage

```bash
# Interactive wizard (default)
agit setup

# Auto‑detect and generate without prompts
agit setup --auto

# Generate manifest and install hooks
agit setup --manifest --hooks

# Overwrite existing manifest file
agit setup --manifest --overwrite

# Generate agent configs for all supported editors
agit setup --agent-configs

# Run validation and repair
agit doctor
agit doctor --fix
```

### How It Works

1. **Tech‑stack detection** (`internal/setup/detect.go`)  
   The wizard looks for characteristic files (`go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, etc.) and returns a `TechStack` struct with a suggested test command.

2. **Manifest generation** (`internal/setup/generate.go`)  
   Based on the detected stack, it creates a `.agit-manifest.json` with two initial tasks:
   - *Initialize agit and verify setup* – runs the project’s test command to confirm the environment works.
   - *Run project tests to ensure everything works* – depends on the first task, reinforcing the test‑first workflow.

3. **Hook installation**  
   Reuses the existing `agit hook install` command to add the post‑checkout hook that automatically switches agit workspaces on `git checkout`.

4. **Agent‑config generation** (`internal/setup/agentconfig.go`)  
   Creates (or skips if already present) the MCP configuration files for:
   - Claude Code (`.claude/mcp.json`)
   - OpenCode (`.opencode/plugins/agit.json`)
   - Cursor (`.cursor/mcp.json`)

5. **Validation & repair** (`agit doctor`)  
   The `doctor` command checks for:
   - agit database existence
   - Git hook presence and content
   - manifest file existence
   - agent config files (optional)  
   With the `--fix` flag it attempts to repair any missing component by calling the appropriate setup routines.

## Integration with LLM‑Assisted Workflows

The bootstrap wizard is designed to be extended with **LLM‑facilitated input generation**. Future versions can:

- Use an LLM (local or remote) to analyze the codebase and suggest more specific, context‑aware tasks.
- Parse existing issue trackers (GitHub Issues, Trello, Jira) and import them as agit tasks.
- Generate custom test commands that match the project’s actual test suite (e.g., `pytest tests/unit/` instead of just `pytest`).
- Recommend task dependencies based on code coupling.

The wizard already provides the scaffolding; the LLM layer can supply the **content** that turns a generic seed into a project‑specific roadmap.

## Design Philosophy

1. **Zero‑configuration defaults** – agit should work out‑of‑the‑box for the most common project layouts.
2. **Progressive disclosure** – advanced options (agent configs, hook installation) are optional and clearly documented.
3. **Non‑destructive** – the wizard never overwrites existing files unless explicitly asked (`--overwrite`).
4. **Validation‑first** – the `doctor` command lets users verify their setup at any time and repair it with a single flag.

## Example Walkthrough

```bash
$ cd my‑go‑project
$ agit setup --manifest --hooks --agent-configs
Detected Go project (go)
Suggested test command: go test ./...
Detected files: [go.mod]

Generating manifest...
Manifest written to .agit-manifest.json

Installing Git hooks...
agit post‑checkout hook installed at .git/hooks/post‑checkout
Note: agit‑mcp server must be built and in PATH for hook to work

Generating agent configs...
Agent configs generated.

Setup wizard completed.

$ agit doctor
agit doctor - validating configuration...
  ✓ agit database exists
  ✓ agit Git hook installed
  ✓ manifest file exists
  ✓ agent config exists: .claude/mcp.json
  ✓ agent config exists: .opencode/plugins/agit.json
  ✓ agent config exists: .cursor/mcp.json

All checks passed. agit is properly configured.
```

## Next Steps

- **LLM integration** – connect the wizard to an LLM API to generate richer task descriptions and test commands.
- **Import plugins** – add support for importing tasks from external systems (GitHub Issues, Trello, markdown todo lists).
- **Custom templates** – allow users to define their own manifest templates for different project archetypes.
- **Multi‑stack detection** – handle projects that use multiple languages/frameworks simultaneously.

## Conclusion

The bootstrap wizard eliminates the “input requirement” bottleneck that previously prevented users from getting immediate value from agit. By automating the initial configuration, agit can now be **ready‑to‑work** in seconds, while still offering the full power of its stateful, test‑driven, agent‑orchestrated workflow engine.

Built by [Elko.AI](https://elko.ai)’s “TheMachine” Dark Software Factory.