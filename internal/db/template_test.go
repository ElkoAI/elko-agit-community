package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	// Initialize database
	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Create template
	templateID, err := CreateTemplate(dbPath, "dark.css", "css", "body { background: #222; }")
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}
	if templateID <= 0 {
		t.Errorf("Expected positive template ID, got %d", templateID)
	}

	// Verify template exists and has correct fields
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var name, templateType, content string
	var isActive bool
	var createdAt, updatedAt time.Time
	err = db.QueryRow(`
		SELECT name, type, content, is_active, created_at, updated_at
		FROM templates WHERE id = ?
	`, templateID).Scan(&name, &templateType, &content, &isActive, &createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("Failed to query created template: %v", err)
	}
	if name != "dark.css" {
		t.Errorf("Expected name 'dark.css', got %q", name)
	}
	if templateType != "css" {
		t.Errorf("Expected type 'css', got %q", templateType)
	}
	if content != "body { background: #222; }" {
		t.Errorf("Expected content 'body { background: #222; }', got %q", content)
	}
	if isActive {
		t.Errorf("Expected is_active false for new template, got true")
	}
	if time.Since(createdAt) > 5*time.Second {
		t.Errorf("CreatedAt is too old: %v", createdAt)
	}
	if time.Since(updatedAt) > 5*time.Second {
		t.Errorf("UpdatedAt is too old: %v", updatedAt)
	}
}

func TestCreateTemplateInvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	_, err := CreateTemplate(dbPath, "invalid.txt", "txt", "content")
	if err == nil {
		t.Error("Expected error for invalid template type, got nil")
	}
}

func TestGetTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	// Create template via direct SQL
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active)
		VALUES (?, ?, ?, ?)
	`, "base.html", "html", "<html></html>", false)
	if err != nil {
		t.Fatalf("Failed to insert template: %v", err)
	}

	var templateID int64
	err = db.QueryRow("SELECT id FROM templates WHERE name = ?", "base.html").Scan(&templateID)
	if err != nil {
		t.Fatalf("Failed to get template ID: %v", err)
	}

	// Retrieve using GetTemplate
	template, err := GetTemplate(dbPath, templateID)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if template == nil {
		t.Fatal("Expected non‑nil template, got nil")
	}
	if template.Name != "base.html" {
		t.Errorf("Expected name 'base.html', got %q", template.Name)
	}
	if template.Type != "html" {
		t.Errorf("Expected type 'html', got %q", template.Type)
	}
	if template.Content != "<html></html>" {
		t.Errorf("Expected content '<html></html>', got %q", template.Content)
	}
	if template.IsActive {
		t.Errorf("Expected IsActive false, got true")
	}
}

func TestGetTemplateByName(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active)
		VALUES (?, ?, ?, ?)
	`, "script.js", "js", "console.log('hello');", false)
	if err != nil {
		t.Fatalf("Failed to insert template: %v", err)
	}

	template, err := GetTemplateByName(dbPath, "script.js")
	if err != nil {
		t.Fatalf("GetTemplateByName failed: %v", err)
	}
	if template == nil {
		t.Fatal("Expected non‑nil template, got nil")
	}
	if template.Name != "script.js" {
		t.Errorf("Expected name 'script.js', got %q", template.Name)
	}
	if template.Type != "js" {
		t.Errorf("Expected type 'js', got %q", template.Type)
	}
	if template.Content != "console.log('hello');" {
		t.Errorf("Expected content 'console.log('hello');', got %q", template.Content)
	}
}

func TestListTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active) VALUES
		('a.html', 'html', 'html1', false),
		('b.css', 'css', 'css1', true),
		('c.js', 'js', 'js1', false),
		('d.html', 'html', 'html2', true)
	`)
	if err != nil {
		t.Fatalf("Failed to insert templates: %v", err)
	}

	// List all templates
	templates, err := ListTemplates(dbPath, "")
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(templates) != 4 {
		t.Errorf("Expected 4 templates, got %d", len(templates))
	}
	// Verify ordering: type, name
	if templates[0].Type != "css" || templates[0].Name != "b.css" {
		t.Errorf("First template should be css/b.css, got %s/%s", templates[0].Type, templates[0].Name)
	}
	if templates[1].Type != "html" || templates[1].Name != "a.html" {
		t.Errorf("Second template should be html/a.html, got %s/%s", templates[1].Type, templates[1].Name)
	}
	if templates[2].Type != "html" || templates[2].Name != "d.html" {
		t.Errorf("Third template should be html/d.html, got %s/%s", templates[2].Type, templates[2].Name)
	}
	if templates[3].Type != "js" || templates[3].Name != "c.js" {
		t.Errorf("Fourth template should be js/c.js, got %s/%s", templates[3].Type, templates[3].Name)
	}

	// List only html templates
	htmlTemplates, err := ListTemplates(dbPath, "html")
	if err != nil {
		t.Fatalf("ListTemplates(html) failed: %v", err)
	}
	if len(htmlTemplates) != 2 {
		t.Errorf("Expected 2 html templates, got %d", len(htmlTemplates))
	}
	for _, tmpl := range htmlTemplates {
		if tmpl.Type != "html" {
			t.Errorf("Expected type html, got %s", tmpl.Type)
		}
	}
}

func TestUpdateTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active)
		VALUES (?, ?, ?, ?)
	`, "old.css", "css", "old content", false)
	if err != nil {
		t.Fatalf("Failed to insert template: %v", err)
	}

	var templateID int64
	err = db.QueryRow("SELECT id FROM templates WHERE name = ?", "old.css").Scan(&templateID)
	if err != nil {
		t.Fatalf("Failed to get template ID: %v", err)
	}

	// Update content
	newContent := "new content"
	err = UpdateTemplate(dbPath, templateID, newContent)
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}

	// Verify update
	var content string
	var updatedAt time.Time
	err = db.QueryRow("SELECT content, updated_at FROM templates WHERE id = ?", templateID).Scan(&content, &updatedAt)
	if err != nil {
		t.Fatalf("Failed to query updated template: %v", err)
	}
	if content != newContent {
		t.Errorf("Expected content %q, got %q", newContent, content)
	}
	if time.Since(updatedAt) > 5*time.Second {
		t.Errorf("UpdatedAt not updated: %v", updatedAt)
	}
}

func TestDeleteTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active)
		VALUES (?, ?, ?, ?)
	`, "toDelete.html", "html", "content", false)
	if err != nil {
		t.Fatalf("Failed to insert template: %v", err)
	}

	var templateID int64
	err = db.QueryRow("SELECT id FROM templates WHERE name = ?", "toDelete.html").Scan(&templateID)
	if err != nil {
		t.Fatalf("Failed to get template ID: %v", err)
	}

	// Delete template
	err = DeleteTemplate(dbPath, templateID)
	if err != nil {
		t.Fatalf("DeleteTemplate failed: %v", err)
	}

	// Verify deletion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM templates WHERE id = ?", templateID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected template to be deleted, count = %d", count)
	}
}

func TestActivateTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	// Insert two css templates, one active
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active) VALUES
		('css1', 'css', 'css1', true),
		('css2', 'css', 'css2', false),
		('html1', 'html', 'html1', false)
	`)
	if err != nil {
		t.Fatalf("Failed to insert templates: %v", err)
	}

	var css2ID int64
	err = db.QueryRow("SELECT id FROM templates WHERE name = ?", "css2").Scan(&css2ID)
	if err != nil {
		t.Fatalf("Failed to get css2 ID: %v", err)
	}

	// Activate css2
	err = ActivateTemplate(dbPath, css2ID)
	if err != nil {
		t.Fatalf("ActivateTemplate failed: %v", err)
	}

	// Verify css1 is deactivated, css2 is active, html1 unchanged
	var css1Active, css2Active, html1Active bool
	err = db.QueryRow("SELECT is_active FROM templates WHERE name = ?", "css1").Scan(&css1Active)
	if err != nil {
		t.Fatalf("Failed to query css1: %v", err)
	}
	err = db.QueryRow("SELECT is_active FROM templates WHERE name = ?", "css2").Scan(&css2Active)
	if err != nil {
		t.Fatalf("Failed to query css2: %v", err)
	}
	err = db.QueryRow("SELECT is_active FROM templates WHERE name = ?", "html1").Scan(&html1Active)
	if err != nil {
		t.Fatalf("Failed to query html1: %v", err)
	}
	if css1Active {
		t.Error("css1 should be deactivated")
	}
	if !css2Active {
		t.Error("css2 should be activated")
	}
	if html1Active {
		t.Error("html1 should remain inactive")
	}
}

func TestGetActiveTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agit.db")

	if err := InitDatabase(dbPath); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO templates (name, type, content, is_active) VALUES
		('active.css', 'css', 'css', true),
		('inactive.css', 'css', 'css2', false),
		('active.html', 'html', 'html', true)
	`)
	if err != nil {
		t.Fatalf("Failed to insert templates: %v", err)
	}

	// Get active css template
	template, err := GetActiveTemplate(dbPath, "css")
	if err != nil {
		t.Fatalf("GetActiveTemplate failed: %v", err)
	}
	if template == nil {
		t.Fatal("Expected non‑nil active css template, got nil")
	}
	if template.Name != "active.css" {
		t.Errorf("Expected active.css, got %q", template.Name)
	}
	if !template.IsActive {
		t.Error("Expected IsActive true")
	}

	// Get active js template (none)
	jsTemplate, err := GetActiveTemplate(dbPath, "js")
	if err != nil {
		t.Fatalf("GetActiveTemplate for js failed: %v", err)
	}
	if jsTemplate != nil {
		t.Errorf("Expected nil active js template, got %v", jsTemplate)
	}
}
