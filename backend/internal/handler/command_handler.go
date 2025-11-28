package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/consumer"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/models"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/repository"
	"github.com/IBM/sarama"
	"github.com/go-redis/redis/v8"
)

type CommandHandler struct {
	repo                *repository.WishlistRepository
	responseWriter      sarama.SyncProducer
	responseTopic       string
	wishlistEventsTopic string
}

func NewCommandHandler(db *sql.DB, redisClient *redis.Client, responseWriter sarama.SyncProducer, responseTopic, wishlistEventsTopic string) *CommandHandler {
	return &CommandHandler{
		repo:                repository.NewWishlistRepository(db, redisClient),
		responseWriter:      responseWriter,
		responseTopic:       responseTopic,
		wishlistEventsTopic: wishlistEventsTopic,
	}
}

// HandleCommand processes a command from the frontend
func (h *CommandHandler) HandleCommand(data []byte) error {
	cmd, err := consumer.ParseCommand(data)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	log.Printf("Handling command: %s for user %d", cmd.Type, cmd.TelegramID)

	switch cmd.Type {
	case "register_user":
		return h.handleRegisterUser(cmd)
	case "add_wishlist":
		return h.handleAddWishlist(cmd)
	case "list_wishlist":
		return h.handleListWishlist(cmd)
	case "delete_wishlist":
		return h.handleDeleteWishlist(cmd)
	default:
		log.Printf("Unknown command type: %s", cmd.Type)
	}

	return nil
}

// handleRegisterUser registers or updates a user
func (h *CommandHandler) handleRegisterUser(cmd *consumer.Command) error {
	user := &models.User{
		TelegramID: cmd.TelegramID,
		Username:   cmd.Username,
		FirstName:  cmd.FirstName,
		LastName:   cmd.LastName,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (telegram_id) 
		DO UPDATE SET username = $2, first_name = $3, last_name = $4, updated_at = $6
	`

	_, err := h.repo.GetDB().Exec(query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		log.Printf("Error registering user: %v", err)
		return err
	}

	log.Printf("User registered: %d (%s)", user.TelegramID, user.Username)
	return nil
}

// handleAddWishlist adds a wishlist item
func (h *CommandHandler) handleAddWishlist(cmd *consumer.Command) error {
	wishlist := &models.Wishlist{
		TelegramID:         cmd.TelegramID,
		ProductName:        cmd.ProductName,
		TargetPrice:        cmd.TargetPrice,
		DiscountPercentage: cmd.DiscountPercentage,
		CreatedAt:          time.Now(),
	}

	query := `
		INSERT INTO wishlists (telegram_id, product_name, target_price, discount_percentage, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := h.repo.GetDB().QueryRow(
		query,
		wishlist.TelegramID,
		wishlist.ProductName,
		wishlist.TargetPrice,
		wishlist.DiscountPercentage,
		wishlist.CreatedAt,
	).Scan(&wishlist.ID)

	if err != nil {
		log.Printf("Error adding wishlist item: %v", err)
		return err
	}

	h.repo.InvalidateUserCache(wishlist.TelegramID)
	log.Printf("Wishlist item added: %d for user %d", wishlist.ID, wishlist.TelegramID)

	// Publish event to Kafka
	event := models.WishlistEvent{
		Type:               "wishlist_item_added",
		TelegramID:         wishlist.TelegramID,
		ProductName:        wishlist.ProductName,
		TargetPrice:        wishlist.TargetPrice,
		DiscountPercentage: wishlist.DiscountPercentage,
		Timestamp:          time.Now(),
	}

	if err := h.publishEvent(event); err != nil {
		log.Printf("Failed to publish wishlist event: %v", err)
		// Don't fail the request, just log the error
	}

	return nil
}

// handleListWishlist retrieves and sends wishlist items
func (h *CommandHandler) handleListWishlist(cmd *consumer.Command) error {
	wishlists, err := h.repo.GetWishlistsByTelegramID(cmd.TelegramID)
	if err != nil {
		log.Printf("Error getting wishlists: %v", err)
		return err
	}

	// Convert to response format
	items := make([]WishlistItem, len(wishlists))
	for i, w := range wishlists {
		items[i] = WishlistItem{
			ID:                 w.ID,
			ProductName:        w.ProductName,
			TargetPrice:        w.TargetPrice,
			DiscountPercentage: w.DiscountPercentage,
		}
	}

	response := WishlistResponse{
		ChatID: cmd.ChatID,
		Items:  items,
	}

	return h.sendResponse(response)
}

// handleDeleteWishlist deletes a wishlist item
func (h *CommandHandler) handleDeleteWishlist(cmd *consumer.Command) error {
	query := `
		DELETE FROM wishlists
		WHERE id = $1 AND telegram_id = $2
	`

	result, err := h.repo.GetDB().Exec(query, cmd.WishlistID, cmd.TelegramID)
	if err != nil {
		log.Printf("Error deleting wishlist item: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	success := rowsAffected > 0

	if success {
		h.repo.InvalidateUserCache(cmd.TelegramID)
		log.Printf("Wishlist item deleted: %d for user %d", cmd.WishlistID, cmd.TelegramID)
	}

	response := DeleteResponse{
		ChatID:  cmd.ChatID,
		Success: success,
	}

	return h.sendResponse(response)
}

// sendResponse sends a response back to the frontend via Kafka
func (h *CommandHandler) sendResponse(response interface{}) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: h.responseTopic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = h.responseWriter.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	return nil
}

// publishEvent publishes a wishlist event to Kafka
func (h *CommandHandler) publishEvent(event models.WishlistEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: h.wishlistEventsTopic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = h.responseWriter.SendMessage(msg)
	return err
}

// WishlistItem represents a wishlist item in the response
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
