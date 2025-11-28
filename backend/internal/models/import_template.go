package models

import "time"

// ImportTemplate represents an S3 import configuration
type ImportTemplate struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	S3URL         string    `json:"s3_url"`
	MappingSchema string    `json:"mapping_schema"` // JSON string mapping Offer fields to JSON paths
	IsActive      bool      `json:"is_active"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
