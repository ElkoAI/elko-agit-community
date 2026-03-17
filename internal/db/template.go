package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Template represents a dashboard template row in the database.
type Template struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // html, css, js
	Content   string    `json:"content"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTemplate inserts a new template.
// Returns the newly created template ID.
func CreateTemplate(dbPath, name, templateType, content string) (int64, error) {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Validate template type
	if templateType != "html" && templateType != "css" && templateType != "js" {
		return 0, fmt.Errorf("invalid template type: %s (must be html, css, or js)", templateType)
	}

	// Insert template (initially inactive)
	result, err := db.Exec(`
		INSERT INTO templates (name, type, content, is_active)
		VALUES (?, ?, ?, FALSE)
	`, name, templateType, content)
	if err != nil {
		return 0, fmt.Errorf("failed to insert template: %w", err)
	}

	templateID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	return templateID, nil
}

// GetTemplate retrieves a template by its ID.
// Returns nil if not found.
func GetTemplate(dbPath string, templateID int64) (*Template, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var t Template
	err = db.QueryRow(`
		SELECT id, name, type, content, is_active, created_at, updated_at
		FROM templates
		WHERE id = ?
	`, templateID).Scan(
		&t.ID,
		&t.Name,
		&t.Type,
		&t.Content,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}
	return &t, nil
}

// GetTemplateByName retrieves a template by its name.
// Returns nil if not found.
func GetTemplateByName(dbPath, name string) (*Template, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var t Template
	err = db.QueryRow(`
		SELECT id, name, type, content, is_active, created_at, updated_at
		FROM templates
		WHERE name = ?
	`, name).Scan(
		&t.ID,
		&t.Name,
		&t.Type,
		&t.Content,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}
	return &t, nil
}

// ListTemplates returns all templates, optionally filtered by type.
func ListTemplates(dbPath, templateType string) ([]Template, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var rows *sql.Rows
	if templateType == "" {
		rows, err = db.Query(`
			SELECT id, name, type, content, is_active, created_at, updated_at
			FROM templates
			ORDER BY type, name
		`)
	} else {
		rows, err = db.Query(`
			SELECT id, name, type, content, is_active, created_at, updated_at
			FROM templates
			WHERE type = ?
			ORDER BY name
		`, templateType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	templates := []Template{}
	for rows.Next() {
		var t Template
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Type,
			&t.Content,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template row: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return templates, nil
}

// UpdateTemplate modifies an existing template's content.
func UpdateTemplate(dbPath string, templateID int64, content string) error {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		UPDATE templates
		SET content = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, content, templateID)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}
	return nil
}

// DeleteTemplate removes a template by its ID.
func DeleteTemplate(dbPath string, templateID int64) error {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM templates WHERE id = ?", templateID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	return nil
}

// ActivateTemplate sets a template as active and deactivates others of the same type.
func ActivateTemplate(dbPath string, templateID int64) error {
	// Acquire file‑based lock for concurrent multi‑agent access
	unlock, err := Lock(dbPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// First, get the template type
	var templateType string
	err = db.QueryRow("SELECT type FROM templates WHERE id = ?", templateID).Scan(&templateType)
	if err == sql.ErrNoRows {
		return fmt.Errorf("template %d not found", templateID)
	} else if err != nil {
		return fmt.Errorf("failed to query template: %w", err)
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Deactivate all templates of the same type
	_, err = tx.Exec(`
		UPDATE templates
		SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE type = ?
	`, templateType)
	if err != nil {
		return fmt.Errorf("failed to deactivate templates: %w", err)
	}

	// Activate the specified template
	_, err = tx.Exec(`
		UPDATE templates
		SET is_active = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, templateID)
	if err != nil {
		return fmt.Errorf("failed to activate template: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetActiveTemplate returns the active template for a given type, or nil if none.
func GetActiveTemplate(dbPath, templateType string) (*Template, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var t Template
	err = db.QueryRow(`
		SELECT id, name, type, content, is_active, created_at, updated_at
		FROM templates
		WHERE type = ? AND is_active = TRUE
	`, templateType).Scan(
		&t.ID,
		&t.Name,
		&t.Type,
		&t.Content,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query active template: %w", err)
	}
	return &t, nil
}
