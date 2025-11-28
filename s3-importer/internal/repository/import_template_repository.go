package repository

import (
	"database/sql"
	"time"

	"github.com/FlavioMalvestitiJunior/bf-offers/s3-importer/internal/models"
)

type ImportTemplateRepository struct {
	db *sql.DB
}

func NewImportTemplateRepository(db *sql.DB) *ImportTemplateRepository {
	return &ImportTemplateRepository{db: db}
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

// UpdateLastRunAt updates the last_run_at timestamp
func (r *ImportTemplateRepository) UpdateLastRunAt(id int) error {
	_, err := r.db.Exec(`
		UPDATE import_templates
		SET last_run_at = $1
		WHERE id = $2
	`, time.Now(), id)

	return err
}
