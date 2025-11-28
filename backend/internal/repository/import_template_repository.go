package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/yourusername/bf-offers/backend/internal/models"
)

type ImportTemplateRepository struct {
	db *sql.DB
}

func NewImportTemplateRepository(db *sql.DB) *ImportTemplateRepository {
	return &ImportTemplateRepository{db: db}
}

// GetAllTemplates returns all import templates
func (r *ImportTemplateRepository) GetAllTemplates() ([]models.ImportTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, s3_url, mapping_schema, is_active, last_run_at, created_at, updated_at
		FROM import_templates
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.ImportTemplate
	for rows.Next() {
		var t models.ImportTemplate
		var schemaBytes []byte
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.S3URL,
			&schemaBytes,
			&t.IsActive,
			&t.LastRunAt,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		t.MappingSchema = string(schemaBytes)
		templates = append(templates, t)
	}

	return templates, nil
}

// GetActiveTemplates returns all active import templates
func (r *ImportTemplateRepository) GetActiveTemplates() ([]models.ImportTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, s3_url, mapping_schema, is_active, last_run_at, created_at, updated_at
		FROM import_templates
		WHERE is_active = true
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.ImportTemplate
	for rows.Next() {
		var t models.ImportTemplate
		var schemaBytes []byte
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.S3URL,
			&schemaBytes,
			&t.IsActive,
			&t.LastRunAt,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		t.MappingSchema = string(schemaBytes)
		templates = append(templates, t)
	}

	return templates, nil
}

// GetTemplateByID returns a specific template by ID
func (r *ImportTemplateRepository) GetTemplateByID(id int) (*models.ImportTemplate, error) {
	var t models.ImportTemplate
	var schemaBytes []byte

	err := r.db.QueryRow(`
		SELECT id, name, s3_url, mapping_schema, is_active, last_run_at, created_at, updated_at
		FROM import_templates
		WHERE id = $1
	`, id).Scan(
		&t.ID,
		&t.Name,
		&t.S3URL,
		&schemaBytes,
		&t.IsActive,
		&t.LastRunAt,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.MappingSchema = string(schemaBytes)
	return &t, nil
}

// CreateTemplate creates a new import template
func (r *ImportTemplateRepository) CreateTemplate(t *models.ImportTemplate) error {
	// Validate JSON schema
	var js interface{}
	if err := json.Unmarshal([]byte(t.MappingSchema), &js); err != nil {
		return err
	}

	return r.db.QueryRow(`
		INSERT INTO import_templates (name, s3_url, mapping_schema, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, t.Name, t.S3URL, t.MappingSchema, t.IsActive).Scan(
		&t.ID,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
}

// UpdateTemplate updates an existing template
func (r *ImportTemplateRepository) UpdateTemplate(t *models.ImportTemplate) error {
	// Validate JSON schema
	var js interface{}
	if err := json.Unmarshal([]byte(t.MappingSchema), &js); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		UPDATE import_templates
		SET name = $1, s3_url = $2, mapping_schema = $3, is_active = $4
		WHERE id = $5
	`, t.Name, t.S3URL, t.MappingSchema, t.IsActive, t.ID)

	return err
}

// UpdateLastRunAt updates the last_run_at timestamp
func (r *ImportTemplateRepository) UpdateLastRunAt(id int) error {
	_, err := r.db.Exec(`
		UPDATE import_templates
		SET last_run_at = $1
		WHERE id = $2
	`, time.Now(), id)

	return err
}

// DeleteTemplate deletes a template by ID
func (r *ImportTemplateRepository) DeleteTemplate(id int) error {
	_, err := r.db.Exec(`DELETE FROM import_templates WHERE id = $1`, id)
	return err
}
