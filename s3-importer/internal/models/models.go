package models

import "time"

// ImportTemplate represents an S3 import configuration
type ImportTemplate struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	S3URL         string     `json:"s3_url"`
	MappingSchema string     `json:"mapping_schema"` // JSON string mapping Offer fields to JSON paths
	IsActive      bool       `json:"is_active"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Offer represents a product offer to be sent to Kafka
type Offer struct {
	ID                 int       `json:"id"`
	ProductName        string    `json:"titulo"`
	Price              float64   `json:"price"`
	OriginalPrice      float64   `json:"oldPrice"`
	Details            string    `json:"details"`
	CashbackPercentage int       `json:"percentCashback"`
	DiscountPercentage int       `json:"-"` // Calculated or not present in new schema
	Source             string    `json:"source,omitempty"`
	ReceivedAt         time.Time `json:"received_at"`
}
