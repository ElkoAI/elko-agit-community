package setup

import (
	"os"
)

// TechStack represents the detected technology stack of a project.
type TechStack struct {
	Language      string   // e.g., "Go", "JavaScript", "Python", "Rust"
	BuildTool     string   // e.g., "go", "npm", "yarn", "pip", "cargo"
	TestCommand   string   // suggested test command (e.g., "go test ./...")
	DetectedFiles []string // files that indicate this stack
}

// Detect examines the current directory and returns the detected tech stack.
// It returns nil if no known stack is detected.
func Detect() *TechStack {
	// List of detectors in order of priority
	detectors := []func() *TechStack{
		detectGo,
		detectNode,
		detectPython,
		detectRust,
	}
	for _, detector := range detectors {
		if stack := detector(); stack != nil {
			return stack
		}
	}
	return nil
}

// detectGo checks for Go projects (go.mod file).
func detectGo() *TechStack {
	if fileExists("go.mod") {
		return &TechStack{
			Language:      "Go",
			BuildTool:     "go",
			TestCommand:   "go test ./...",
			DetectedFiles: []string{"go.mod"},
		}
	}
	return nil
}

// detectNode checks for Node.js projects (package.json).
func detectNode() *TechStack {
	if fileExists("package.json") {
		return &TechStack{
			Language:      "JavaScript",
			BuildTool:     "npm",
			TestCommand:   "npm test",
			DetectedFiles: []string{"package.json"},
		}
	}
	return nil
}

// detectPython checks for Python projects (requirements.txt, pyproject.toml, setup.py).
func detectPython() *TechStack {
	var files []string
	if fileExists("requirements.txt") {
		files = append(files, "requirements.txt")
	}
	if fileExists("pyproject.toml") {
		files = append(files, "pyproject.toml")
	}
	if fileExists("setup.py") {
		files = append(files, "setup.py")
	}
	if len(files) > 0 {
		return &TechStack{
			Language:      "Python",
			BuildTool:     "pip",
			TestCommand:   "pytest",
			DetectedFiles: files,
		}
	}
	return nil
}

// detectRust checks for Rust projects (Cargo.toml).
func detectRust() *TechStack {
	if fileExists("Cargo.toml") {
		return &TechStack{
			Language:      "Rust",
			BuildTool:     "cargo",
			TestCommand:   "cargo test",
			DetectedFiles: []string{"Cargo.toml"},
		}
	}
	return nil
}

// fileExists returns true if the given file exists in the current directory.
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// SuggestTestCommand returns a suggested test command for the current project.
// It uses Detect() internally and returns a default command if no stack is detected.
func SuggestTestCommand() string {
	stack := Detect()
	if stack != nil {
		return stack.TestCommand
	}
	// Default fallback (generic shell command that exits successfully)
	return "echo 'No tests configured'"
}
