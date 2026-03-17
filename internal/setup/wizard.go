package setup

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// WizardConfig holds the choices made by the user during interactive setup.
type WizardConfig struct {
	TechStack         *TechStack // can be nil if custom/unknown
	Language          string     // e.g., "Go", "Node.js", "Python", "Rust", "Other"
	CustomTestCmd     string     // if Language == "Other"
	InstallHooks      bool
	GenerateManifest  bool
	GenerateAgentCfgs bool
	ProjectName       string
}

// RunInteractiveWizard presents a series of prompts to the user and returns their choices.
func RunInteractiveWizard() *WizardConfig {
	fmt.Println("\n=== agit Bootstrap Wizard ===")
	fmt.Println("This wizard will help you configure agit for your project.")
	fmt.Println()

	cfg := &WizardConfig{}

	// 1. Project name
	cfg.ProjectName = promptProjectName()

	// 2. Tech stack / language
	cfg.Language, cfg.CustomTestCmd = promptTechStack()
	if cfg.Language != "Other" {
		// Map language to TechStack
		cfg.TechStack = detectStackForLanguage(cfg.Language)
	} else {
		// For "Other", we still need a test command
		if cfg.CustomTestCmd == "" {
			cfg.CustomTestCmd = "echo 'No tests configured'"
		}
	}

	// 3. Install Git hooks?
	cfg.InstallHooks = promptYesNo("Install Git post‑checkout hook for automatic workspace switching?", true)

	// 4. Generate manifest?
	cfg.GenerateManifest = promptYesNo("Generate .agit-manifest.json with initial tasks?", true)

	// 5. Generate agent configs?
	cfg.GenerateAgentCfgs = promptYesNo("Generate MCP configuration files for AI agents (Claude Code, OpenCode, Cursor)?", true)

	fmt.Println("\n--- Summary ---")
	fmt.Printf("Project name:      %s\n", cfg.ProjectName)
	fmt.Printf("Primary language:  %s\n", cfg.Language)
	if cfg.TechStack != nil {
		fmt.Printf("Test command:      %s\n", cfg.TechStack.TestCommand)
	} else if cfg.CustomTestCmd != "" {
		fmt.Printf("Test command:      %s\n", cfg.CustomTestCmd)
	}
	fmt.Printf("Install hooks:     %v\n", cfg.InstallHooks)
	fmt.Printf("Generate manifest: %v\n", cfg.GenerateManifest)
	fmt.Printf("Generate agent cfgs: %v\n", cfg.GenerateAgentCfgs)
	fmt.Println()

	if !promptYesNo("Proceed with this configuration?", true) {
		fmt.Println("Setup cancelled.")
		os.Exit(0)
	}

	return cfg
}

// promptProjectName asks for a project name (defaults to current directory base name).
func promptProjectName() string {
	defaultName := GetCurrentDir()
	fmt.Printf("Project name [%s]: ", defaultName)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultName
	}
	return input
}

// promptTechStack asks the user to select a primary language and returns the language
// and a custom test command (if language is "Other").
func promptTechStack() (language, customTestCmd string) {
	fmt.Println("Select your project's primary language:")
	fmt.Println("  1) Go")
	fmt.Println("  2) Node.js (JavaScript/TypeScript)")
	fmt.Println("  3) Python")
	fmt.Println("  4) Rust")
	fmt.Println("  5) Other (custom test command)")

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
			fmt.Print("Enter custom test command (e.g., 'make test', 'echo \"no tests\"'): ")
			cmd, _ := reader.ReadString('\n')
			cmd = strings.TrimSpace(cmd)
			return "Other", cmd
		default:
			fmt.Println("Invalid choice. Please enter 1‑5.")
		}
	}
}

// promptYesNo asks a yes/no question and returns true for yes, false for no.
func promptYesNo(question string, defaultYes bool) bool {
	options := "(Y/n)"
	if !defaultYes {
		options = "(y/N)"
	}
	fmt.Printf("%s %s: ", question, options)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// detectStackForLanguage returns a TechStack for a known language.
func detectStackForLanguage(lang string) *TechStack {
	switch lang {
	case "Go":
		return &TechStack{
			Language:      "Go",
			BuildTool:     "go",
			TestCommand:   "go test ./...",
			DetectedFiles: []string{"go.mod"},
		}
	case "Node.js":
		return &TechStack{
			Language:      "JavaScript",
			BuildTool:     "npm",
			TestCommand:   "npm test",
			DetectedFiles: []string{"package.json"},
		}
	case "Python":
		return &TechStack{
			Language:      "Python",
			BuildTool:     "pip",
			TestCommand:   "pytest",
			DetectedFiles: []string{"requirements.txt", "pyproject.toml", "setup.py"},
		}
	case "Rust":
		return &TechStack{
			Language:      "Rust",
			BuildTool:     "cargo",
			TestCommand:   "cargo test",
			DetectedFiles: []string{"Cargo.toml"},
		}
	default:
		return nil
	}
}
