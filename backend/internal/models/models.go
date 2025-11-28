package models

import "time"

// Offer represents a product offer from Kafka
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

// WishlistEvent represents an event when a wishlist item is added
type WishlistEvent struct {
	Type               string  `json:"type"` // "wishlist_item_added"
	TelegramID         int64   `json:"telegram_id"`
	ProductName        string  `json:"product_name"`
	TargetPrice        *float64 `json:"target_price,omitempty"`
	DiscountPercentage *int     `json:"discount_percentage,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// User represents a Telegram user
type User struct {
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Wishlist represents a user's wishlist item
type Wishlist struct {
	ID                 int      `json:"id"`
	TelegramID         int64    `json:"telegram_id"`
	ProductName        string   `json:"product_name"`
	TargetPrice        *float64 `json:"target_price,omitempty"`
	DiscountPercentage *int     `json:"discount_percentage,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Notification represents a sent notification
type Notification struct {
	ID         int       `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	WishlistID *int      `json:"wishlist_id,omitempty"`
	OfferID    *int      `json:"offer_id,omitempty"`
	SentAt     time.Time `json:"sent_at"`
}

// OfferNotification represents a notification to be sent via Kafka
type OfferNotification struct {
	TelegramID         int64   `json:"telegram_id"`
	ProductName        string  `json:"product_name"`
	Price              float64 `json:"price"`
	OriginalPrice      float64 `json:"original_price"`
	DiscountPercentage int     `json:"discount_percentage"`
	CashbackPercentage int     `json:"cashback_percentage"`
	WishlistID         int     `json:"wishlist_id"`
	MatchType          string  `json:"match_type"` // "price" or "discount"
}
