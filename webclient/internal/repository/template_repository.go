package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/FlavioMalvestitiJunior/bf-offers/webclient/internal/models"
)

type TemplateRepository struct {
	db *sql.DB
}

func NewTemplateRepository(db *sql.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

// GetAllTemplates returns all message templates
func (r *TemplateRepository) GetAllTemplates() ([]models.MessageTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, product_model, title_field, description_field, price_field, 
		       discount_field, details_fields, message_schema, sns_topic_arn, 
		       is_active, created_at, updated_at
		FROM message_templates
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.MessageTemplate
	for rows.Next() {
		var t models.MessageTemplate
		var schemaBytes []byte
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.ProductModel,
			&t.TitleField,
			&t.DescriptionField,
			&t.PriceField,
			&t.DiscountField,
			&t.DetailsFields,
			&schemaBytes,
			&t.SNSTopicARN,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		t.MessageSchema = string(schemaBytes)
		templates = append(templates, t)
	}

	return templates, nil
}

// GetTemplateByID returns a specific template by ID
func (r *TemplateRepository) GetTemplateByID(id int) (*models.MessageTemplate, error) {
	var t models.MessageTemplate
	var schemaBytes []byte

	err := r.db.QueryRow(`
		SELECT id, name, product_model, title_field, description_field, price_field,
		       discount_field, details_fields, message_schema, sns_topic_arn, 
		       is_active, created_at, updated_at
		FROM message_templates
		WHERE id = $1
	`, id).Scan(
		&t.ID,
		&t.Name,
		&t.ProductModel,
		&t.TitleField,
		&t.DescriptionField,
		&t.PriceField,
		&t.DiscountField,
		&t.DetailsFields,
		&schemaBytes,
		&t.SNSTopicARN,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.MessageSchema = string(schemaBytes)
	return &t, nil
}

// CreateTemplate creates a new message template
func (r *TemplateRepository) CreateTemplate(t *models.MessageTemplate) error {
	// Validate JSON schema
	var js interface{}
	if err := json.Unmarshal([]byte(t.MessageSchema), &js); err != nil {
		return err
	}

	return r.db.QueryRow(`
		INSERT INTO message_templates (name, product_model, title_field, description_field, 
		                               price_field, discount_field, details_fields, 
		                               message_schema, sns_topic_arn, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, t.Name, t.ProductModel, t.TitleField, t.DescriptionField, t.PriceField,
		t.DiscountField, t.DetailsFields, t.MessageSchema, t.SNSTopicARN, t.IsActive).Scan(
		&t.ID,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
}

// UpdateTemplate updates an existing template
func (r *TemplateRepository) UpdateTemplate(t *models.MessageTemplate) error {
	// Validate JSON schema
	var js interface{}
	if err := json.Unmarshal([]byte(t.MessageSchema), &js); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		UPDATE message_templates
		SET name = $1, product_model = $2, title_field = $3, description_field = $4,
		    price_field = $5, discount_field = $6, details_fields = $7,
		    message_schema = $8, sns_topic_arn = $9, is_active = $10
		WHERE id = $11
	`, t.Name, t.ProductModel, t.TitleField, t.DescriptionField, t.PriceField,
		t.DiscountField, t.DetailsFields, t.MessageSchema, t.SNSTopicARN, t.IsActive, t.ID)

	return err
}

// DeleteTemplate deletes a template by ID
func (r *TemplateRepository) DeleteTemplate(id int) error {
	_, err := r.db.Exec(`DELETE FROM message_templates WHERE id = $1`, id)
	return err
}
