-- agit SQLite Schema
-- Version: 1.0.0
-- Stores agent state in .git/agit.db

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;
-- Enable Write-Ahead Logging for better concurrency
PRAGMA journal_mode = WAL;

-- Workspace = branch + agent context
CREATE TABLE IF NOT EXISTS workspaces (
    branch TEXT PRIMARY KEY,           -- git rev-parse --abbrev-ref HEAD
    current_task_id INTEGER,           
    agent_state JSON,                  -- Serialized agent context
    todo_list TEXT,                    -- Markdown checklist
    last_commit TEXT,                  -- git rev-parse HEAD
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tasks = atomic units of work
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_branch TEXT NOT NULL,
    description TEXT NOT NULL,         -- "Implement X"
    status TEXT CHECK(status IN ('pending','active','done','failed')),
    test_command TEXT,                 -- "go test ./..." 
    test_result JSON,                  -- Last run output
    parent_task_id INTEGER,            -- For decomposition
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY(workspace_branch) REFERENCES workspaces(branch) ON DELETE CASCADE,
    FOREIGN KEY(parent_task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Checkpoints = git commits + agent state snapshots
CREATE TABLE IF NOT EXISTS checkpoints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_branch TEXT NOT NULL,
    commit_hash TEXT NOT NULL,         -- git commit hash
    agent_state JSON,                  -- Snapshot at commit time
    message TEXT,                      -- User-provided message
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(workspace_branch) REFERENCES workspaces(branch) ON DELETE CASCADE
);

-- Traces = test run history
CREATE TABLE IF NOT EXISTS traces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    exit_code INTEGER,
    output TEXT,
    duration_ms INTEGER,
    run_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Dashboard templates = customizable HTML/CSS/JS for web dashboard
CREATE TABLE IF NOT EXISTS templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,           -- Template identifier (e.g., "base.html", "dark.css")
    type TEXT CHECK(type IN ('html', 'css', 'js')), -- Template type
    content TEXT NOT NULL,               -- Template content
    is_active BOOLEAN DEFAULT FALSE,     -- Only one template per type can be active
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_workspaces_branch ON workspaces(branch);
CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace_branch);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_checkpoints_workspace ON checkpoints(workspace_branch);
CREATE INDEX IF NOT EXISTS idx_checkpoints_commit ON checkpoints(commit_hash);
CREATE INDEX IF NOT EXISTS idx_traces_task ON traces(task_id);
CREATE INDEX IF NOT EXISTS idx_traces_run_at ON traces(run_at);
CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name);
CREATE INDEX IF NOT EXISTS idx_templates_type ON templates(type);
CREATE INDEX IF NOT EXISTS idx_templates_active ON templates(is_active) WHERE is_active = TRUE;