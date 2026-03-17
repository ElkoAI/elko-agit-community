package setup

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"agit/internal/manifest"
)

// ProjectPlan captures detailed project information gathered through conversation.
type ProjectPlan struct {
	Name              string
	Description       string
	ProjectType       string // "web-app", "cli-tool", "library", "api", "data-pipeline", "other"
	TechStack         *TechStack
	Features          []string
	Architecture      string // Brief architecture description
	Tasks             []ProjectTask
	TestStrategy      string
	InstallHooks      bool
	GenerateAgentCfgs bool
	Requirements      *ProjectRequirements
}

// ProjectTask represents a task in the project plan (before conversion to manifest).
type ProjectTask struct {
	ID          int64
	Title       string
	Description string
	TestCommand string
	DependsOn   []int64 // IDs of tasks this depends on
	Priority    string  // "high", "medium", "low"
}

// RunProjectPlanningWizard conducts a conversational interview to create a detailed project plan.
func RunProjectPlanningWizard() *ProjectPlan {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("agit Project Planning Wizard")
	fmt.Println("I'll help you create a detailed project plan that agit can use to")
	fmt.Println("orchestrate AI-driven development from day one.")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	plan := &ProjectPlan{}

	// 1. Basic project information
	plan.Name = promptProjectName()
	plan.Description = promptWithDefault("Brief project description (what are you building)?", "")

	// 2. Project type
	plan.ProjectType = promptProjectType()

	// 3. Tech stack
	lang, customTestCmd := promptTechStackDetailed()
	if lang != "Other" {
		plan.TechStack = detectStackForLanguage(lang)
	} else {
		plan.TechStack = &TechStack{
			Language:    "Custom",
			BuildTool:   "custom",
			TestCommand: customTestCmd,
		}
		if customTestCmd == "" {
			plan.TechStack.TestCommand = "echo 'No tests configured'"
		}
	}

	// 4. Key features/requirements
	plan.Features = promptFeatures()

	// 5. Architecture overview
	plan.Architecture = promptWithDefault("Brief architecture overview (optional):", "")

	// 6. Generate initial task breakdown
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("Based on your project type and features, I'll suggest an initial task breakdown.")
	fmt.Println("We'll refine it together.")

	plan.Tasks = generateInitialTasks(plan)
	plan.Tasks = refineTasks(plan.Tasks)

	// 7. Test strategy
	plan.TestStrategy = promptTestStrategy(plan.TechStack)

	// 8. Project requirements
	plan.Requirements = DefaultRequirements(plan.ProjectType, plan.TechStack)
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("PROJECT REQUIREMENTS")
	fmt.Println("agit can define minimum requirements for your project that will be")
	fmt.Println("validated by 'agit doctor'. These include:")
	fmt.Println(plan.Requirements.Summary())
	includeRequirements := promptYesNo("\nInclude .agit-requirements.json with these requirements?", true)
	if !includeRequirements {
		plan.Requirements = nil
	}

	// 9. agit configuration
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("agit Configuration")
	plan.InstallHooks = promptYesNo("Install Git post‑checkout hook for automatic workspace switching?", true)
	plan.GenerateAgentCfgs = promptYesNo("Generate MCP configuration files for AI agents (Claude Code, OpenCode, Cursor)?", true)

	// 10. Summary and confirmation
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("PROJECT PLAN SUMMARY")
	fmt.Println(strings.Repeat("=", 50))
	printProjectSummary(plan)

	if !promptYesNo("\nCreate this project plan and initialize agit?", true) {
		fmt.Println("Setup cancelled.")
		os.Exit(0)
	}

	return plan
}

// promptProjectType asks about the type of project being built.
func promptProjectType() string {
	fmt.Println("\nWhat type of project are you building?")
	fmt.Println("  1) Web Application (frontend + backend)")
	fmt.Println("  2) CLI Tool / Utility")
	fmt.Println("  3) Library / Package")
	fmt.Println("  4) API / Microservice")
	fmt.Println("  5) Data Pipeline / ETL")
	fmt.Println("  6) Other")

	for {
		fmt.Print("Choice (1‑6): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		switch input {
		case "1":
			return "web-app"
		case "2":
			return "cli-tool"
		case "3":
			return "library"
		case "4":
			return "api"
		case "5":
			return "data-pipeline"
		case "6":
			return promptWithDefault("Specify project type:", "other")
		default:
			fmt.Println("Invalid choice. Please enter 1‑6.")
		}
	}
}

// promptTechStackDetailed asks about tech stack with more context.
func promptTechStackDetailed() (language, customTestCmd string) {
	fmt.Println("\nWhat's the primary technology stack?")
	fmt.Println("  1) Go")
	fmt.Println("  2) Node.js (JavaScript/TypeScript)")
	fmt.Println("  3) Python")
	fmt.Println("  4) Rust")
	fmt.Println("  5) Other (specify test command)")

	for {
		fmt.Print("Choice (1‑5): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		switch input {
		case "1":
			return "Go", ""
		case "2":
			return "Node.js", ""
		case "3":
			return "Python", ""
		case "4":
			return "Rust", ""
		case "5":
			fmt.Print("Enter custom test command (e.g., 'make test', './run-tests.sh'): ")
			cmd, _ := reader.ReadString('\n')
			cmd = strings.TrimSpace(cmd)
			return "Other", cmd
		default:
			fmt.Println("Invalid choice. Please enter 1‑5.")
		}
	}
}

// promptFeatures asks for key features/requirements.
func promptFeatures() []string {
	fmt.Println("\nWhat are the key features or requirements?")
	fmt.Println("(Enter one per line, empty line when done)")

	var features []string
	reader := bufio.NewReader(os.Stdin)

	for i := 1; ; i++ {
		fmt.Printf("Feature %d: ", i)
		feature, _ := reader.ReadString('\n')
		feature = strings.TrimSpace(feature)
		if feature == "" {
			if i == 1 {
				fmt.Println("Please enter at least one key feature.")
				continue
			}
			break
		}
		features = append(features, feature)
	}

	return features
}

// promptTestStrategy asks about testing approach.
func promptTestStrategy(stack *TechStack) string {
	defaultStrategy := ""
	switch stack.Language {
	case "Go":
		defaultStrategy = "Unit tests with go test, table‑driven tests for business logic"
	case "JavaScript":
		defaultStrategy = "Jest for unit tests, React Testing Library for UI components"
	case "Python":
		defaultStrategy = "pytest for unit tests, hypothesis for property‑based testing"
	case "Rust":
		defaultStrategy = "cargo test with unit tests in src/ and integration tests in tests/"
	default:
		defaultStrategy = "Unit tests for core logic, integration tests for APIs"
	}

	fmt.Printf("\nTesting strategy [%s]: ", defaultStrategy)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultStrategy
	}
	return input
}

// promptWithDefault prompts with a default value.
func promptWithDefault(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// generateInitialTasks creates an initial task breakdown based on project type.
func generateInitialTasks(plan *ProjectPlan) []ProjectTask {
	var tasks []ProjectTask

	// Common initial tasks for all projects
	tasks = append(tasks, ProjectTask{
		ID:          1,
		Title:       "Project setup and initialization",
		Description: "Initialize agit, set up development environment, configure tooling",
		TestCommand: plan.TechStack.TestCommand,
		DependsOn:   []int64{},
		Priority:    "high",
	})

	// Type-specific tasks
	switch plan.ProjectType {
	case "web-app":
		tasks = append(tasks, ProjectTask{
			ID:          2,
			Title:       "Basic project structure",
			Description: "Create directory structure, package.json/go.mod/Cargo.toml, basic config files",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{1},
			Priority:    "high",
		})
		tasks = append(tasks, ProjectTask{
			ID:          3,
			Title:       "Core application skeleton",
			Description: "Set up main application entry point, routing, basic components",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{2},
			Priority:    "high",
		})
		// Add feature-specific tasks
		for i, feature := range plan.Features {
			tasks = append(tasks, ProjectTask{
				ID:          int64(4 + i),
				Title:       fmt.Sprintf("Implement: %s", feature),
				Description: feature,
				TestCommand: plan.TechStack.TestCommand,
				DependsOn:   []int64{3},
				Priority:    "medium",
			})
		}

	case "cli-tool":
		tasks = append(tasks, ProjectTask{
			ID:          2,
			Title:       "Command-line interface",
			Description: "Implement argument parsing, help text, basic command structure",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{1},
			Priority:    "high",
		})
		tasks = append(tasks, ProjectTask{
			ID:          3,
			Title:       "Core functionality",
			Description: "Implement main business logic of the tool",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{2},
			Priority:    "high",
		})
		for i, feature := range plan.Features {
			tasks = append(tasks, ProjectTask{
				ID:          int64(4 + i),
				Title:       fmt.Sprintf("Feature: %s", feature),
				Description: feature,
				TestCommand: plan.TechStack.TestCommand,
				DependsOn:   []int64{3},
				Priority:    "medium",
			})
		}

	case "library":
		tasks = append(tasks, ProjectTask{
			ID:          2,
			Title:       "API design and exports",
			Description: "Define public API, export structure, versioning strategy",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{1},
			Priority:    "high",
		})
		tasks = append(tasks, ProjectTask{
			ID:          3,
			Title:       "Core implementation",
			Description: "Implement main library functionality",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{2},
			Priority:    "high",
		})
		for i, feature := range plan.Features {
			tasks = append(tasks, ProjectTask{
				ID:          int64(4 + i),
				Title:       fmt.Sprintf("Feature: %s", feature),
				Description: feature,
				TestCommand: plan.TechStack.TestCommand,
				DependsOn:   []int64{3},
				Priority:    "medium",
			})
		}

	case "api":
		tasks = append(tasks, ProjectTask{
			ID:          2,
			Title:       "API specification",
			Description: "Define endpoints, request/response schemas, authentication",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{1},
			Priority:    "high",
		})
		tasks = append(tasks, ProjectTask{
			ID:          3,
			Title:       "Server setup",
			Description: "Set up HTTP server, routing, middleware",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{2},
			Priority:    "high",
		})
		for i, feature := range plan.Features {
			tasks = append(tasks, ProjectTask{
				ID:          int64(4 + i),
				Title:       fmt.Sprintf("Endpoint: %s", feature),
				Description: feature,
				TestCommand: plan.TechStack.TestCommand,
				DependsOn:   []int64{3},
				Priority:    "medium",
			})
		}

	default:
		// Generic project structure
		tasks = append(tasks, ProjectTask{
			ID:          2,
			Title:       "Project scaffolding",
			Description: "Set up basic project structure and configuration",
			TestCommand: plan.TechStack.TestCommand,
			DependsOn:   []int64{1},
			Priority:    "high",
		})
		for i, feature := range plan.Features {
			tasks = append(tasks, ProjectTask{
				ID:          int64(3 + i),
				Title:       fmt.Sprintf("Implement: %s", feature),
				Description: feature,
				TestCommand: plan.TechStack.TestCommand,
				DependsOn:   []int64{2},
				Priority:    "medium",
			})
		}
	}

	return tasks
}

// refineTasks allows the user to modify the suggested task breakdown.
func refineTasks(tasks []ProjectTask) []ProjectTask {
	for {
		fmt.Println("\nCurrent task breakdown:")
		printTasks(tasks)

		fmt.Println("\nOptions:")
		fmt.Println("  1) Edit a task")
		fmt.Println("  2) Add a new task")
		fmt.Println("  3) Delete a task")
		fmt.Println("  4) Reorder tasks (renumber IDs)")
		fmt.Println("  5) Done editing")
		fmt.Print("Choice (1‑5): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			tasks = editTask(tasks)
		case "2":
			tasks = addTask(tasks)
		case "3":
			tasks = deleteTask(tasks)
		case "4":
			tasks = reorderTasks(tasks)
		case "5":
			return tasks
		default:
			fmt.Println("Invalid choice. Please enter 1‑5.")
		}
	}
}

// printTasks displays tasks with their details.
func printTasks(tasks []ProjectTask) {
	for _, task := range tasks {
		fmt.Printf("  %d. %s (%s)\n", task.ID, task.Title, task.Priority)
		if task.Description != "" && task.Description != task.Title {
			fmt.Printf("     %s\n", task.Description)
		}
		if task.TestCommand != "" {
			fmt.Printf("     Test: %s\n", task.TestCommand)
		}
		if len(task.DependsOn) > 0 {
			fmt.Printf("     Depends on: %v\n", task.DependsOn)
		}
	}
}

// editTask allows editing a single task.
func editTask(tasks []ProjectTask) []ProjectTask {
	if len(tasks) == 0 {
		fmt.Println("No tasks to edit.")
		return tasks
	}

	fmt.Print("Enter task ID to edit: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var id int64
	if _, err := fmt.Sscanf(input, "%d", &id); err != nil {
		fmt.Println("Invalid ID.")
		return tasks
	}

	// Find the task
	var task *ProjectTask
	var idx int
	for i := range tasks {
		if tasks[i].ID == id {
			task = &tasks[i]
			idx = i
			break
		}
	}
	if task == nil {
		fmt.Printf("Task %d not found.\n", id)
		return tasks
	}

	fmt.Println("\nEditing task:", task.Title)
	fmt.Println("Leave field blank to keep current value.")

	// Title
	newTitle := promptWithDefault(fmt.Sprintf("Title [%s]", task.Title), task.Title)
	if newTitle != "" {
		task.Title = newTitle
	}

	// Description
	newDesc := promptWithDefault(fmt.Sprintf("Description [%s]", task.Description), task.Description)
	if newDesc != "" {
		task.Description = newDesc
	}

	// Test command
	newTest := promptWithDefault(fmt.Sprintf("Test command [%s]", task.TestCommand), task.TestCommand)
	if newTest != "" {
		task.TestCommand = newTest
	}

	// Priority
	newPriority := promptPriority(task.Priority)
	task.Priority = newPriority

	// Dependencies
	fmt.Printf("Current dependencies: %v\n", task.DependsOn)
	if promptYesNo("Edit dependencies?", false) {
		task.DependsOn = promptDependencies(tasks, id)
	}

	tasks[idx] = *task
	fmt.Println("Task updated.")
	return tasks
}

// addTask creates a new task.
func addTask(tasks []ProjectTask) []ProjectTask {
	fmt.Println("\nAdding new task")

	// Determine next ID
	var maxID int64
	for _, t := range tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	newID := maxID + 1

	var task ProjectTask
	task.ID = newID

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Title: ")
	title, _ := reader.ReadString('\n')
	task.Title = strings.TrimSpace(title)

	fmt.Print("Description (optional): ")
	desc, _ := reader.ReadString('\n')
	task.Description = strings.TrimSpace(desc)

	fmt.Print("Test command (optional): ")
	test, _ := reader.ReadString('\n')
	task.TestCommand = strings.TrimSpace(test)

	task.Priority = promptPriority("medium")

	if promptYesNo("Add dependencies?", false) {
		task.DependsOn = promptDependencies(tasks, newID)
	}

	tasks = append(tasks, task)
	fmt.Printf("Task %d added.\n", newID)
	return tasks
}

// deleteTask removes a task and updates dependencies.
func deleteTask(tasks []ProjectTask) []ProjectTask {
	if len(tasks) == 0 {
		fmt.Println("No tasks to delete.")
		return tasks
	}

	fmt.Print("Enter task ID to delete: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var id int64
	if _, err := fmt.Sscanf(input, "%d", &id); err != nil {
		fmt.Println("Invalid ID.")
		return tasks
	}

	// Find task index
	var idx = -1
	for i, t := range tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		fmt.Printf("Task %d not found.\n", id)
		return tasks
	}

	// Remove task
	tasks = append(tasks[:idx], tasks[idx+1:]...)

	// Remove this ID from all dependency lists
	for i := range tasks {
		var newDeps []int64
		for _, dep := range tasks[i].DependsOn {
			if dep != id {
				newDeps = append(newDeps, dep)
			}
		}
		tasks[i].DependsOn = newDeps
	}

	fmt.Printf("Task %d deleted and removed from dependencies.\n", id)
	return tasks
}

// reorderTasks renumbers tasks sequentially starting from 1.
func reorderTasks(tasks []ProjectTask) []ProjectTask {
	if len(tasks) == 0 {
		return tasks
	}

	// Create mapping from old ID to new ID
	idMap := make(map[int64]int64)
	for i := range tasks {
		idMap[tasks[i].ID] = int64(i + 1)
	}

	// Update IDs and dependencies
	for i := range tasks {
		tasks[i].ID = idMap[tasks[i].ID]
		var newDeps []int64
		for _, dep := range tasks[i].DependsOn {
			if newID, ok := idMap[dep]; ok {
				newDeps = append(newDeps, newID)
			}
		}
		tasks[i].DependsOn = newDeps
	}

	fmt.Println("Tasks renumbered sequentially.")
	return tasks
}

// promptPriority asks for priority with validation.
func promptPriority(defaultPriority string) string {
	for {
		fmt.Printf("Priority (high/medium/low) [%s]: ", defaultPriority)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return defaultPriority
		}
		if input == "high" || input == "medium" || input == "low" {
			return input
		}
		fmt.Println("Priority must be 'high', 'medium', or 'low'.")
	}
}

// promptDependencies asks for a list of task IDs this task depends on.
func promptDependencies(tasks []ProjectTask, excludeID int64) []int64 {
	if len(tasks) == 0 {
		fmt.Println("No other tasks available for dependencies.")
		return nil
	}

	fmt.Println("Available tasks:")
	for _, t := range tasks {
		if t.ID == excludeID {
			continue
		}
		fmt.Printf("  %d. %s\n", t.ID, t.Title)
	}

	fmt.Println("Enter task IDs separated by commas, or leave blank for none:")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var deps []int64
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var id int64
		if _, err := fmt.Sscanf(p, "%d", &id); err != nil {
			fmt.Printf("Invalid ID '%s' ignored.\n", p)
			continue
		}
		// Validate task exists
		found := false
		for _, t := range tasks {
			if t.ID == id && id != excludeID {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Task %d not found or cannot depend on itself.\n", id)
			continue
		}
		deps = append(deps, id)
	}
	return deps
}

// printProjectSummary displays the complete project plan.
func printProjectSummary(plan *ProjectPlan) {
	fmt.Printf("Project: %s\n", plan.Name)
	if plan.Description != "" {
		fmt.Printf("Description: %s\n", plan.Description)
	}
	fmt.Printf("Type: %s\n", plan.ProjectType)
	fmt.Printf("Tech Stack: %s\n", plan.TechStack.Language)
	if plan.TechStack.TestCommand != "" {
		fmt.Printf("Test Command: %s\n", plan.TechStack.TestCommand)
	}
	if len(plan.Features) > 0 {
		fmt.Println("\nKey Features:")
		for i, f := range plan.Features {
			fmt.Printf("  %d. %s\n", i+1, f)
		}
	}
	if plan.Architecture != "" {
		fmt.Printf("\nArchitecture: %s\n", plan.Architecture)
	}
	if plan.TestStrategy != "" {
		fmt.Printf("\nTest Strategy: %s\n", plan.TestStrategy)
	}
	fmt.Printf("\nagit Configuration:\n")
	fmt.Printf("  Install Git hooks: %v\n", plan.InstallHooks)
	fmt.Printf("  Generate agent configs: %v\n", plan.GenerateAgentCfgs)
	if plan.Requirements != nil {
		fmt.Printf("  Project requirements: %s\n", plan.Requirements.Summary())
	}
	fmt.Printf("\nTasks (%d total):\n", len(plan.Tasks))
	for _, task := range plan.Tasks {
		fmt.Printf("  %d. %s (%s)\n", task.ID, task.Title, task.Priority)
	}
}

// ConvertProjectPlanToManifest transforms a ProjectPlan into a manifest.Manifest.
func ConvertProjectPlanToManifest(plan *ProjectPlan) *manifest.Manifest {
	tasks := make([]manifest.ManifestTask, 0, len(plan.Tasks))

	for _, pt := range plan.Tasks {
		tasks = append(tasks, manifest.ManifestTask{
			ID:           pt.ID,
			Description:  pt.Title,
			Test:         pt.TestCommand,
			Dependencies: pt.DependsOn,
		})
	}

	desc := plan.Description
	if desc == "" {
		desc = fmt.Sprintf("%s project managed by agit", plan.ProjectType)
	}
	if plan.Architecture != "" {
		desc = fmt.Sprintf("%s | Architecture: %s", desc, plan.Architecture)
	}

	return &manifest.Manifest{
		Version:     "1.0.0",
		Name:        plan.Name,
		Description: desc,
		Tasks:       tasks,
		Agents: manifest.AgentsConfig{
			Preferred:    "claude-code",
			Integrations: []string{"opencode", "cursor"},
		},
	}
}
