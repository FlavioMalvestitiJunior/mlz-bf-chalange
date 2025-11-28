package models

import "time"

// OfferNotification represents a notification from backend about a matched offer
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

// Command represents a command sent from frontend to backend
type Command struct {
	Type               string    `json:"type"` // register_user, add_wishlist, list_wishlist, delete_wishlist
	TelegramID         int64     `json:"telegram_id"`
	ChatID             int64     `json:"chat_id,omitempty"`
	Username           string    `json:"username,omitempty"`
	FirstName          string    `json:"first_name,omitempty"`
	LastName           string    `json:"last_name,omitempty"`
	ProductName        string    `json:"product_name,omitempty"`
	TargetPrice        *float64  `json:"target_price,omitempty"`
	DiscountPercentage *int      `json:"discount_percentage,omitempty"`
	WishlistID         int       `json:"wishlist_id,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// WishlistItem represents a single wishlist item
type WishlistItem struct {
	ID                 int      `json:"id"`
	ProductName        string   `json:"product_name"`
	TargetPrice        *float64 `json:"target_price,omitempty"`
	DiscountPercentage *int     `json:"discount_percentage,omitempty"`
}

// WishlistResponse represents the response to a list command
type WishlistResponse struct {
	ChatID int64          `json:"chat_id"`
	Items  []WishlistItem `json:"items"`
}

// DeleteResponse represents the response to a delete command
type DeleteResponse struct {
	ChatID  int64 `json:"chat_id"`
	Success bool  `json:"success"`
}
