package models

import "time"

// MessageTemplate represents a template for SNS messages
type MessageTemplate struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	ProductModel  string    `json:"product_model"`
	
	// Structured fields for offer data
	TitleField       string  `json:"title_field"`        // Campo para título
	DescriptionField *string `json:"description_field,omitempty"` // Campo para descrição
	PriceField       string  `json:"price_field"`        // Campo para preço
	DiscountField    *string `json:"discount_field,omitempty"`    // Campo para desconto
	DetailsFields    *string `json:"details_fields,omitempty"`    // Campos para busca (JSON array)
	
	MessageSchema string    `json:"message_schema"` // JSON string
	SNSTopicARN   *string   `json:"sns_topic_arn,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	ActiveUsers   int `json:"active_users"`
	TotalUsers    int `json:"total_users"`
	TotalWishlists int `json:"total_wishlists"`
	RecentOffers  int `json:"recent_offers"`
}

// UserActivity represents user activity information
type UserActivity struct {
	TelegramID int64     `json:"telegram_id"`
	Username   *string   `json:"username,omitempty"`
	FirstName  *string   `json:"first_name,omitempty"`
	LastName   *string   `json:"last_name,omitempty"`
	LastActive time.Time `json:"last_active"`
	Wishlists  int       `json:"wishlists"`
	IsBlacklisted bool   `json:"is_blacklisted,omitempty"`
}

// Wishlist represents a user's wishlist item
type Wishlist struct {
	ID                 int       `json:"id"`
	TelegramID         int64     `json:"telegram_id"`
	ProductName        string    `json:"product_name"`
	TargetPrice        float64   `json:"target_price"`
	DiscountPercentage int       `json:"discount_percentage"`
	CreatedAt          time.Time `json:"created_at"`
}
