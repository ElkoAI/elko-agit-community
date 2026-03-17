package setup

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ProjectRequirements defines minimum requirements for a project that agit can validate.
type ProjectRequirements struct {
	Version     string                   `json:"version"`
	Structure   *StructureRequirements   `json:"structure,omitempty"`
	CodeQuality *CodeQualityRequirements `json:"code_quality,omitempty"`
	Workflow    *WorkflowRequirements    `json:"workflow,omitempty"`
	Security    *SecurityRequirements    `json:"security,omitempty"`
}

// StructureRequirements defines required project structure.
type StructureRequirements struct {
	RequiredDirectories []string          `json:"required_directories,omitempty"`
	RequiredFiles       map[string]string `json:"required_files,omitempty"`   // filename -> description
	IgnoredPatterns     []string          `json:"ignored_patterns,omitempty"` // for .gitignore
}

// CodeQualityRequirements defines code quality standards.
type CodeQualityRequirements struct {
	Linters                 []LinterConfig    `json:"linters,omitempty"`
	Formatters              []FormatterConfig `json:"formatters,omitempty"`
	TestCoverageThreshold   *int              `json:"test_coverage_threshold,omitempty"` // percentage
	MinTestCasesPerFunction *int              `json:"min_test_cases_per_function,omitempty"`
	ComplexityThreshold     *int              `json:"complexity_threshold,omitempty"` // cyclomatic complexity
}

// LinterConfig defines a linter tool configuration.
type LinterConfig struct {
	Tool        string `json:"tool"`    // e.g., "golangci-lint", "eslint", "pylint"
	Command     string `json:"command"` // e.g., "golangci-lint run"
	ConfigFile  string `json:"config_file,omitempty"`
	Description string `json:"description,omitempty"`
}

// FormatterConfig defines a code formatter configuration.
type FormatterConfig struct {
	Tool        string `json:"tool"`    // e.g., "gofmt", "prettier", "black"
	Command     string `json:"command"` // e.g., "gofmt -w ."
	ConfigFile  string `json:"config_file,omitempty"`
	Description string `json:"description,omitempty"`
}

// WorkflowRequirements defines development workflow standards.
type WorkflowRequirements struct {
	PreCommitHooks           []string                `json:"pre_commit_hooks,omitempty"`
	CommitMessageConventions *CommitConventions      `json:"commit_message_conventions,omitempty"`
	BranchNamingPattern      *string                 `json:"branch_naming_pattern,omitempty"`
	PRTemplate               *string                 `json:"pr_template,omitempty"`
	CodeReviewRequirements   *CodeReviewRequirements `json:"code_review_requirements,omitempty"`
}

// CommitConventions defines commit message conventions.
type CommitConventions struct {
	Types         []string `json:"types,omitempty"` // e.g., ["feat", "fix", "docs"]
	ScopeOptional bool     `json:"scope_optional,omitempty"`
	SubjectLength *int     `json:"subject_length,omitempty"` // max characters
	BodyRequired  bool     `json:"body_required,omitempty"`
	FooterPattern *string  `json:"footer_pattern,omitempty"` // regex pattern
}

// CodeReviewRequirements defines code review expectations.
type CodeReviewRequirements struct {
	MinReviewers *int     `json:"min_reviewers,omitempty"`
	RequireTests bool     `json:"require_tests,omitempty"`
	RequireDocs  bool     `json:"require_docs,omitempty"`
	AutoAssign   []string `json:"auto_assign,omitempty"`
}

// SecurityRequirements defines security standards.
type SecurityRequirements struct {
	DependencyScanning  bool     `json:"dependency_scanning,omitempty"`
	SecretDetection     bool     `json:"secret_detection,omitempty"`
	VulnerabilityChecks bool     `json:"vulnerability_checks,omitempty"`
	SASTTools           []string `json:"sast_tools,omitempty"` // Static Application Security Testing
	ContainerScanning   bool     `json:"container_scanning,omitempty"`
	LicenseCompliance   bool     `json:"license_compliance,omitempty"`
}

// DefaultRequirements returns a baseline set of requirements for a given project type and tech stack.
func DefaultRequirements(projectType string, stack *TechStack) *ProjectRequirements {
	req := &ProjectRequirements{
		Version: "1.0.0",
	}

	// Structure defaults
	req.Structure = defaultStructureRequirements(projectType, stack)

	// Code quality defaults based on language
	req.CodeQuality = defaultCodeQualityRequirements(stack)

	// Workflow defaults
	req.Workflow = defaultWorkflowRequirements(projectType)

	// Security defaults
	req.Security = defaultSecurityRequirements(projectType)

	return req
}

func defaultStructureRequirements(projectType string, stack *TechStack) *StructureRequirements {
	s := &StructureRequirements{
		RequiredFiles: make(map[string]string),
	}

	// Common required files
	s.RequiredFiles["README.md"] = "Project documentation"
	s.RequiredFiles["LICENSE"] = "License file"
	s.RequiredFiles[".gitignore"] = "Git ignore patterns"

	// Project-type specific directories
	switch projectType {
	case "web-app":
		s.RequiredDirectories = []string{"src/", "public/", "tests/"}
	case "cli-tool":
		s.RequiredDirectories = []string{"cmd/", "internal/", "pkg/"}
	case "library":
		s.RequiredDirectories = []string{"lib/", "src/", "test/"}
	case "api":
		s.RequiredDirectories = []string{"api/", "handlers/", "middleware/", "models/"}
	case "data-pipeline":
		s.RequiredDirectories = []string{"src/", "config/", "data/", "scripts/"}
	default:
		s.RequiredDirectories = []string{"src/", "tests/"}
	}

	// Language-specific files
	if stack != nil {
		switch stack.Language {
		case "Go":
			s.RequiredFiles["go.mod"] = "Go module definition"
			s.RequiredFiles["go.sum"] = "Go dependencies checksum"
		case "JavaScript":
			s.RequiredFiles["package.json"] = "Node.js package definition"
			s.RequiredFiles["package-lock.json"] = "Dependency lockfile"
		case "Python":
			s.RequiredFiles["requirements.txt"] = "Python dependencies"
			s.RequiredFiles["pyproject.toml"] = "Python project configuration"
		case "Rust":
			s.RequiredFiles["Cargo.toml"] = "Rust package manifest"
			s.RequiredFiles["Cargo.lock"] = "Rust dependency lockfile"
		}
	}

	// Common ignored patterns
	s.IgnoredPatterns = []string{
		"node_modules/",
		"__pycache__/",
		"*.pyc",
		"*.log",
		".DS_Store",
		"dist/",
		"build/",
		".env",
		".env.local",
	}

	return s
}

func defaultCodeQualityRequirements(stack *TechStack) *CodeQualityRequirements {
	cq := &CodeQualityRequirements{}

	if stack == nil {
		return cq
	}

	switch stack.Language {
	case "Go":
		cq.Linters = []LinterConfig{
			{Tool: "golangci-lint", Command: "golangci-lint run", Description: "Go linter aggregator"},
		}
		cq.Formatters = []FormatterConfig{
			{Tool: "gofmt", Command: "gofmt -w .", Description: "Go code formatter"},
		}
		cq.TestCoverageThreshold = intPtr(80)
		cq.ComplexityThreshold = intPtr(10)
	case "JavaScript":
		cq.Linters = []LinterConfig{
			{Tool: "eslint", Command: "eslint .", Description: "JavaScript/TypeScript linter"},
		}
		cq.Formatters = []FormatterConfig{
			{Tool: "prettier", Command: "prettier --write .", Description: "Code formatter"},
		}
		cq.TestCoverageThreshold = intPtr(70)
	case "Python":
		cq.Linters = []LinterConfig{
			{Tool: "pylint", Command: "pylint **/*.py", Description: "Python linter"},
		}
		cq.Formatters = []FormatterConfig{
			{Tool: "black", Command: "black .", Description: "Python code formatter"},
		}
		cq.TestCoverageThreshold = intPtr(75)
	case "Rust":
		cq.Linters = []LinterConfig{
			{Tool: "clippy", Command: "cargo clippy", Description: "Rust linter"},
		}
		cq.Formatters = []FormatterConfig{
			{Tool: "rustfmt", Command: "cargo fmt", Description: "Rust code formatter"},
		}
		cq.TestCoverageThreshold = intPtr(85)
	}

	return cq
}

func defaultWorkflowRequirements(projectType string) *WorkflowRequirements {
	w := &WorkflowRequirements{}

	// Common pre-commit hooks
	w.PreCommitHooks = []string{
		"run tests",
		"check formatting",
		"run linter",
	}

	// Commit conventions (Conventional Commits)
	w.CommitMessageConventions = &CommitConventions{
		Types:         []string{"feat", "fix", "docs", "style", "refactor", "test", "chore"},
		ScopeOptional: true,
		SubjectLength: intPtr(50),
		BodyRequired:  false,
	}

	// Branch naming pattern
	pattern := "^(feature|fix|hotfix|release)/[a-z0-9-]+$"
	w.BranchNamingPattern = &pattern

	// PR template suggestion
	template := "# Description\n\n# Testing\n\n# Checklist\n- [ ] Tests added/updated\n- [ ] Documentation updated\n- [ ] Code follows style guidelines"
	w.PRTemplate = &template

	// Code review requirements
	w.CodeReviewRequirements = &CodeReviewRequirements{
		MinReviewers: intPtr(1),
		RequireTests: true,
		RequireDocs:  true,
	}

	return w
}

func defaultSecurityRequirements(projectType string) *SecurityRequirements {
	s := &SecurityRequirements{
		DependencyScanning:  true,
		SecretDetection:     true,
		VulnerabilityChecks: true,
	}

	// SAST tools based on project type
	switch projectType {
	case "web-app", "api":
		s.SASTTools = []string{"semgrep", "bandit"}
	case "cli-tool", "library":
		s.SASTTools = []string{"semgrep"}
	}

	// Container scanning for deployable projects
	if projectType == "web-app" || projectType == "api" || projectType == "data-pipeline" {
		s.ContainerScanning = true
	}

	// License compliance for libraries
	if projectType == "library" {
		s.LicenseCompliance = true
	}

	return s
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// Summary returns a human-readable summary of the requirements.
func (r *ProjectRequirements) Summary() string {
	var parts []string

	if r.Structure != nil {
		dirs := len(r.Structure.RequiredDirectories)
		files := len(r.Structure.RequiredFiles)
		parts = append(parts, fmt.Sprintf("Structure: %d directories, %d files", dirs, files))
	}

	if r.CodeQuality != nil {
		linters := len(r.CodeQuality.Linters)
		formatters := len(r.CodeQuality.Formatters)
		coverage := ""
		if r.CodeQuality.TestCoverageThreshold != nil {
			coverage = fmt.Sprintf(", %d%% test coverage", *r.CodeQuality.TestCoverageThreshold)
		}
		parts = append(parts, fmt.Sprintf("Code Quality: %d linters, %d formatters%s", linters, formatters, coverage))
	}

	if r.Workflow != nil {
		hooks := len(r.Workflow.PreCommitHooks)
		parts = append(parts, fmt.Sprintf("Workflow: %d pre‑commit hooks", hooks))
	}

	if r.Security != nil {
		var securityItems []string
		if r.Security.DependencyScanning {
			securityItems = append(securityItems, "dependency scanning")
		}
		if r.Security.SecretDetection {
			securityItems = append(securityItems, "secret detection")
		}
		if r.Security.VulnerabilityChecks {
			securityItems = append(securityItems, "vulnerability checks")
		}
		if len(securityItems) > 0 {
			parts = append(parts, fmt.Sprintf("Security: %s", strings.Join(securityItems, ", ")))
		}
	}

	if len(parts) == 0 {
		return "No requirements defined"
	}
	return strings.Join(parts, "; ")
}

// WriteRequirements writes the requirements to a JSON file.
func WriteRequirements(req *ProjectRequirements, filename string, overwrite bool) error {
	if req == nil {
		return nil // nothing to write
	}

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil && !overwrite {
		return fmt.Errorf("%s already exists. Use --overwrite to replace.", filename)
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal requirements: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write requirements file: %w", err)
	}

	log.Printf("Requirements written to %s", filename)
	return nil
}

// LoadRequirements reads and parses a requirements JSON file.
func LoadRequirements(filename string) (*ProjectRequirements, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read requirements file: %w", err)
	}
	var req ProjectRequirements
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse requirements JSON: %w", err)
	}
	return &req, nil
}

// Validate checks the project against the requirements and returns a list of issues.
func (r *ProjectRequirements) Validate(root string) []string {
	var issues []string

	if r.Structure != nil {
		// Check required directories
		for _, dir := range r.Structure.RequiredDirectories {
			path := filepath.Join(root, dir)
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				issues = append(issues, fmt.Sprintf("missing required directory: %s", dir))
			}
		}
		// Check required files
		for filename := range r.Structure.RequiredFiles {
			path := filepath.Join(root, filename)
			if _, err := os.Stat(path); err != nil {
				issues = append(issues, fmt.Sprintf("missing required file: %s", filename))
			}
		}
	}

	// Note: Code quality, workflow, and security validation would require
	// tool‑specific checks; those are left as future extensions.

	return issues
}
