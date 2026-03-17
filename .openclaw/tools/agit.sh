#!/usr/bin/env bash
# agit OpenClaw integration script
# Source this script in your OpenClaw environment to get convenient wrapper functions
# for agit commands with JSON output support.

set -euo pipefail

# Ensure agit is in PATH or provide AGIT_BIN
AGIT_BIN="${AGIT_BIN:-agit}"

# Helper to run agit and capture output
_agit_run() {
    if ! command -v "$AGIT_BIN" >/dev/null 2>&1; then
        echo "Error: agit binary not found. Set AGIT_BIN or ensure agit is in PATH." >&2
        return 1
    fi
    "$AGIT_BIN" "$@"
}

# Helper to output JSON (if jq is available, pretty-print; otherwise raw)
_agit_json() {
    local raw_json="$1"
    if command -v jq >/dev/null 2>&1; then
        jq . <<< "$raw_json"
    else
        echo "$raw_json"
    fi
}

# workspace-status: get current workspace state as JSON
agit_workspace_status() {
    _agit_run workspace-status
}

# task new: create a new task
# Usage: agit_task_new "description" "test_command"
agit_task_new() {
    local description="$1"
    local test_cmd="${2:-}"
    if [[ -z "$test_cmd" ]]; then
        _agit_run task new "$description"
    else
        _agit_run task new --test "$test_cmd" "$description"
    fi
}

# task list: list tasks optionally filtered by status
# Usage: agit_task_list [status]
agit_task_list() {
    local status="${1:-}"
    if [[ -n "$status" ]]; then
        _agit_run task list --status "$status"
    else
        _agit_run task list
    fi
}

# task start: mark a task as active
# Usage: agit_task_start <task_id>
agit_task_start() {
    local task_id="$1"
    _agit_run task start "$task_id"
}

# task done: mark a task as done and run its test command
# Usage: agit_task_done <task_id>
agit_task_done() {
    local task_id="$1"
    _agit_run task done "$task_id"
}

# task breakdown: create subtasks under a parent task
# Usage: agit_task_breakdown <parent_task_id> '<json_subtasks_array>'
# Example: agit_task_breakdown 1 '[{"description":"subtask1","test_command":"go test"}]'
agit_task_breakdown() {
    local parent_id="$1"
    local subtasks_json="$2"
    _agit_run task breakdown "$parent_id" "$subtasks_json"
}

# task tree: show task hierarchy (JSON or tree format)
# Usage: agit_task_tree [--format json|tree]
agit_task_tree() {
    _agit_run task tree "$@"
}

# checkpoint: create a checkpoint (git commit + agent state snapshot)
# Usage: agit_checkpoint "message"
agit_checkpoint() {
    local message="$1"
    _agit_run checkpoint "$message"
}

# restore: restore to the last checkpoint
# Usage: agit_restore
agit_restore() {
    _agit_run restore
}

# trace: show test run history for a task
# Usage: agit_trace <task_id> [--limit N] [--failed-only]
agit_trace() {
    local task_id="$1"
    shift
    _agit_run trace --task "$task_id" "$@"
}

# init: initialize agit in current repository
# Usage: agit_init
agit_init() {
    _agit_run init
}

# hook install: install git hooks for automatic workspace switching
# Usage: agit_hook_install
agit_hook_install() {
    _agit_run hook install
}

# Main dispatcher for sourcing convenience
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # If script is executed directly, show usage
    echo "agit OpenClaw integration script"
    echo "Source this script with: source $(basename "$0")"
    echo "Available functions:"
    declare -F | grep 'agit_' | cut -d' ' -f3 | sort
fi