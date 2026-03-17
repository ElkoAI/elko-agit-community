// Copyright (c) 2026 John Soprych <johnsoprych@gmail.com>
// with assistance from Elko.AI's "TheMachine" Dark Software Factory
// Licensed under the MIT License.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"agit/internal/api"
	"agit/internal/db"
	"agit/internal/git"
	agitmcp "agit/internal/mcp"

	"github.com/mark3labs/mcp-go/mcp"
)

const version = "0.3.0"

//go:embed templates/*.html static/*
var content embed.FS

type Dashboard struct {
	dbPath string
	tmpl   *template.Template
}

func main() {
	var port int
	var host string
	var help bool
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.StringVar(&host, "host", "localhost", "Host to bind to")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()

	if help {
		fmt.Printf("agit-dashboard v%s\n", version)
		fmt.Println("Usage: agit-dashboard [--host localhost] [--port 8080]")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Ensure we're in a git repository
	if !git.IsGitRepository() {
		log.Fatal("agit-dashboard must be run inside a Git repository")
	}

	// Get agit database path
	dbPath, err := git.AgitDBPath()
	if err != nil {
		log.Fatalf("Failed to get agit database path: %v", err)
	}

	// Ensure database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatal("agit database not found. Run 'agit init' first.")
	}

	// Initialize dashboard
	dashboard, err := NewDashboard(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize dashboard: %v", err)
	}

	// Setup routes
	http.HandleFunc("/", dashboard.handleIndex)
	http.HandleFunc("/workspace", dashboard.handleWorkspace)
	http.HandleFunc("/tasks", dashboard.handleTasks)
	http.HandleFunc("/tasks/", dashboard.handleTaskDetail)
	http.HandleFunc("/checkpoints", dashboard.handleCheckpoints)
	http.HandleFunc("/traces", dashboard.handleTraces)
	http.HandleFunc("/api-docs", dashboard.handleAPIDocs)
	http.HandleFunc("/api/schema", dashboard.handleAPISchema)
	http.HandleFunc("/api/execute", dashboard.handleAPIExecute)
	http.HandleFunc("/execute", dashboard.handleExecute)
	http.HandleFunc("/static/", dashboard.handleStatic)

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("agit dashboard starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func NewDashboard(dbPath string) (*Dashboard, error) {
	// Parse templates
	tmpl, err := template.ParseFS(content, "templates/*.html")
	if err != nil {
		// If templates aren't embedded yet, create a minimal fallback
		log.Printf("Warning: could not parse templates: %v", err)
		tmpl = template.New("fallback")
	}

	return &Dashboard{
		dbPath: dbPath,
		tmpl:   tmpl,
	}, nil
}

// loadSchema loads the agit API schema from known locations.
func loadSchema() (*api.Schema, error) {
	// First try relative path from current working directory
	schemaPath := filepath.Join(".", "schema", "agit-api.json")
	if _, statErr := os.Stat(schemaPath); statErr == nil {
		return api.LoadSchema(schemaPath)
	}
	// Try relative to executable
	exe, exeErr := os.Executable()
	if exeErr != nil {
		return nil, exeErr
	}
	exeDir := filepath.Dir(exe)
	schemaPath = filepath.Join(exeDir, "schema", "agit-api.json")
	return api.LoadSchema(schemaPath)
}

// callMCPTool executes an MCP tool directly via its handler.
func callMCPTool(toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	handler, ok := agitmcp.ToolHandlers[toolName]
	if !ok {
		return nil, fmt.Errorf("no MCP handler for tool %q", toolName)
	}
	// Marshal arguments to JSON
	argumentsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %v", err)
	}
	var argumentsMap map[string]interface{}
	if err := json.Unmarshal(argumentsJSON, &argumentsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %v", err)
	}
	// Create CallToolRequest
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: argumentsMap,
		},
	}
	ctx := context.Background()
	result, err := handler(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("MCP tool execution failed: %v", err)
	}
	// Convert result to map
	output := ""
	if len(result.Content) > 0 {
		// Try to extract text from TextContent
		for _, content := range result.Content {
			if tc, ok := mcp.AsTextContent(content); ok {
				output = tc.Text
				break
			}
		}
	}
	success := !result.IsError
	return map[string]interface{}{
		"output":  output,
		"success": success,
	}, nil
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// Redirect to workspace page for now
	http.Redirect(w, r, "/workspace", http.StatusFound)
}

func (d *Dashboard) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, err := db.GetOrCreateWorkspace(d.dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get workspace: %v", err), http.StatusInternalServerError)
		return
	}

	tasks, err := db.ListTasks(d.dbPath, ws.Branch, "", "")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list tasks: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Workspace": ws,
		"Tasks":     tasks,
		"Title":     "Workspace Overview",
		"Active":    "workspace",
	}

	if err := d.renderTemplate(w, "workspace.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) handleTasks(w http.ResponseWriter, r *http.Request) {
	branch, err := git.CurrentBranch()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get current branch: %v", err), http.StatusInternalServerError)
		return
	}

	// Get filter parameters from query string
	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")

	tasks, err := db.ListTasks(d.dbPath, branch, statusFilter, search)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list tasks: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Tasks":        tasks,
		"Title":        "All Tasks",
		"Active":       "tasks",
		"StatusFilter": statusFilter,
		"Search":       search,
	}

	if err := d.renderTemplate(w, "tasks.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from URL path (format: /tasks/{id})
	path := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if path == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}
	taskID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := db.GetTask(d.dbPath, taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get task: %v", err), http.StatusInternalServerError)
		return
	}
	if task == nil {
		http.NotFound(w, r)
		return
	}

	// Get traces for this task
	traces, err := db.GetTraces(d.dbPath, taskID, 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get traces: %v", err), http.StatusInternalServerError)
		return
	}

	// Get subtasks (tasks where parent_task_id = taskID)
	// We can reuse ListTasks with workspace branch filter and parent filter
	// For simplicity, we'll just list all tasks and filter client-side? Let's implement a simple query.
	// We'll add a function later; for now, skip subtasks.

	data := map[string]interface{}{
		"Task":   task,
		"Traces": traces,
		"Title":  fmt.Sprintf("Task #%d", taskID),
		"Active": "tasks",
	}

	if err := d.renderTemplate(w, "task.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	branch, err := git.CurrentBranch()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get current branch: %v", err), http.StatusInternalServerError)
		return
	}

	// Get search parameter from query string
	search := r.URL.Query().Get("q")

	checkpoints, err := db.ListCheckpoints(d.dbPath, branch, search)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list checkpoints: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Checkpoints": checkpoints,
		"Title":       "Checkpoints",
		"Active":      "checkpoints",
		"Search":      search,
	}

	if err := d.renderTemplate(w, "checkpoints.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) handleTraces(w http.ResponseWriter, r *http.Request) {
	branch, err := git.CurrentBranch()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get current branch: %v", err), http.StatusInternalServerError)
		return
	}

	// Get filter parameters from query string
	exitFilter := r.URL.Query().Get("exit")
	search := r.URL.Query().Get("q")

	traces, err := db.ListTracesForWorkspace(d.dbPath, branch, exitFilter, search, 50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list traces: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Traces":     traces,
		"Title":      "Test Traces",
		"Active":     "traces",
		"ExitFilter": exitFilter,
		"Search":     search,
	}

	if err := d.renderTemplate(w, "traces.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
func (d *Dashboard) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	// Try to load schema from known locations
	var schema *api.Schema
	var err error

	// First try relative path from current working directory
	schemaPath := filepath.Join(".", "schema", "agit-api.json")
	if _, statErr := os.Stat(schemaPath); statErr == nil {
		schema, err = api.LoadSchema(schemaPath)
	} else {
		// Try relative to executable
		exe, exeErr := os.Executable()
		if exeErr == nil {
			exeDir := filepath.Dir(exe)
			schemaPath = filepath.Join(exeDir, "schema", "agit-api.json")
			schema, err = api.LoadSchema(schemaPath)
		} else {
			err = exeErr
		}
	}

	if err != nil {
		log.Printf("Failed to load API schema: %v", err)
		http.Error(w, fmt.Sprintf("API schema not found: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Schema": schema,
		"Title":  "agit API Documentation",
		"Active": "api-docs",
	}

	if err := d.renderTemplate(w, "api-docs.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
func (d *Dashboard) handleAPISchema(w http.ResponseWriter, r *http.Request) {
	// Try to load schema from known locations
	var schema *api.Schema
	var err error

	// First try relative path from current working directory
	schemaPath := filepath.Join(".", "schema", "agit-api.json")
	if _, statErr := os.Stat(schemaPath); statErr == nil {
		schema, err = api.LoadSchema(schemaPath)
	} else {
		// Try relative to executable
		exe, exeErr := os.Executable()
		if exeErr == nil {
			exeDir := filepath.Dir(exe)
			schemaPath = filepath.Join(exeDir, "schema", "agit-api.json")
			schema, err = api.LoadSchema(schemaPath)
		} else {
			err = exeErr
		}
	}

	if err != nil {
		log.Printf("Failed to load API schema: %v", err)
		http.Error(w, fmt.Sprintf("API schema not found: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	if err := json.NewEncoder(w).Encode(schema); err != nil {
		log.Printf("Failed to encode schema JSON: %v", err)
	}
}
func (d *Dashboard) handleExecute(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "Execute agit Commands",
		"Active": "execute",
	}

	if err := d.renderTemplate(w, "execute.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
func (d *Dashboard) handleAPIExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Tool       string                 `json:"tool"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON request: %v", err), http.StatusBadRequest)
		return
	}

	// Load schema to get tool definition
	schema, err := loadSchema()
	if err != nil {
		log.Printf("Failed to load schema for execution: %v", err)
		http.Error(w, fmt.Sprintf("Cannot load API schema: %v", err), http.StatusInternalServerError)
		return
	}

	tool := schema.FindTool(req.Tool)
	if tool == nil {
		http.Error(w, fmt.Sprintf("Tool %q not found", req.Tool), http.StatusNotFound)
		return
	}

	// Try MCP handler first if available
	if _, ok := agitmcp.ToolHandlers[req.Tool]; ok {
		// Call MCP handler directly
		result, err := callMCPTool(req.Tool, req.Parameters)
		if err != nil {
			http.Error(w, fmt.Sprintf("MCP tool execution failed: %v", err), http.StatusInternalServerError)
			return
		}
		// Ensure result contains success field
		response := map[string]interface{}{
			"tool":    req.Tool,
			"command": req.Tool, // MCP tool name
			"output":  result["output"],
			"success": result["success"],
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode execution response: %v", err)
		}
		return
	}

	// Fall back to CLI execution
	// Build CLI command from template
	if tool.CLICommand == "" {
		http.Error(w, fmt.Sprintf("Tool %q has no CLI command defined", req.Tool), http.StatusNotImplemented)
		return
	}

	cmdStr := tool.CLICommand
	// Replace parameters in command string
	for _, param := range tool.Parameters {
		value, ok := req.Parameters[param.Name]
		if !ok && param.Required {
			http.Error(w, fmt.Sprintf("Missing required parameter: %s", param.Name), http.StatusBadRequest)
			return
		}
		if ok {
			// Convert value to string based on type
			var strVal string
			switch v := value.(type) {
			case string:
				strVal = v
			case int, int64, float64:
				strVal = fmt.Sprintf("%v", v)
			case bool:
				if v {
					strVal = "true"
				} else {
					strVal = "false"
				}
			default:
				// For arrays or objects, serialize as JSON
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					http.Error(w, fmt.Sprintf("Invalid parameter %s: %v", param.Name, err), http.StatusBadRequest)
					return
				}
				strVal = string(jsonBytes)
			}

			// Replace placeholder {param.name} in command string
			placeholder := "{" + param.Name + "}"
			cmdStr = strings.ReplaceAll(cmdStr, placeholder, strVal)
		}
	}

	// Execute command
	log.Printf("Executing CLI command: %s", cmdStr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir, _ = os.Getwd() // Run in current working directory
	output, err := cmd.CombinedOutput()

	response := map[string]interface{}{
		"tool":    req.Tool,
		"command": cmdStr,
		"output":  string(output),
		"success": err == nil,
	}
	if err != nil {
		response["error"] = err.Error()
		response["exit_code"] = cmd.ProcessState.ExitCode()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode execution response: %v", err)
	}
}

func (d *Dashboard) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files from embedded filesystem
	fs := http.FS(content)
	http.FileServer(fs).ServeHTTP(w, r)
}

func (d *Dashboard) renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	if d.tmpl == nil {
		// Fallback plain text response
		fmt.Fprintf(w, "Template not available. Data: %v", data)
		return nil
	}
	return d.tmpl.ExecuteTemplate(w, name, data)
}
