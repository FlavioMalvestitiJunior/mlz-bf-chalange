package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/yourusername/bf-offers/backend/internal/models"
)

type WishlistRepository struct {
	db    *sql.DB
	redis *redis.Client
	ctx   context.Context
}

func NewWishlistRepository(db *sql.DB, redisClient *redis.Client) *WishlistRepository {
	return &WishlistRepository{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// GetAllWishlists retrieves all wishlists from cache or database
func (r *WishlistRepository) GetAllWishlists() ([]models.Wishlist, error) {
	// Try to get from Redis cache first
	cacheKey := "wishlists:all"
	cached, err := r.redis.Get(r.ctx, cacheKey).Result()
	if err == nil {
		var wishlists []models.Wishlist
		if err := json.Unmarshal([]byte(cached), &wishlists); err == nil {
			return wishlists, nil
		}
	}

	// If not in cache, get from database
	query := `
		SELECT id, telegram_id, product_name, target_price, discount_percentage, created_at
		FROM wishlists
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query wishlists: %w", err)
	}
	defer rows.Close()

	var wishlists []models.Wishlist
	for rows.Next() {
		var w models.Wishlist
		err := rows.Scan(
			&w.ID,
			&w.TelegramID,
			&w.ProductName,
			&w.TargetPrice,
			&w.DiscountPercentage,
			&w.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wishlist: %w", err)
		}
		wishlists = append(wishlists, w)
	}

	// Cache the result for 5 minutes
	if data, err := json.Marshal(wishlists); err == nil {
		r.redis.Set(r.ctx, cacheKey, data, 5*time.Minute)
	}

	return wishlists, nil
}

// GetWishlistsByTelegramID retrieves wishlists for a specific user
func (r *WishlistRepository) GetWishlistsByTelegramID(telegramID int64) ([]models.Wishlist, error) {
	cacheKey := fmt.Sprintf("wishlist:%d", telegramID)
	cached, err := r.redis.Get(r.ctx, cacheKey).Result()
	if err == nil {
		var wishlists []models.Wishlist
		if err := json.Unmarshal([]byte(cached), &wishlists); err == nil {
			return wishlists, nil
		}
	}

	query := `
		SELECT id, telegram_id, product_name, target_price, discount_percentage, created_at
		FROM wishlists
		WHERE telegram_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wishlists: %w", err)
	}
	defer rows.Close()

	var wishlists []models.Wishlist
	for rows.Next() {
		var w models.Wishlist
		err := rows.Scan(
			&w.ID,
			&w.TelegramID,
			&w.ProductName,
			&w.TargetPrice,
			&w.DiscountPercentage,
			&w.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wishlist: %w", err)
		}
		wishlists = append(wishlists, w)
	}

	// Cache the result
	if data, err := json.Marshal(wishlists); err == nil {
		r.redis.Set(r.ctx, cacheKey, data, 5*time.Minute)
	}

	return wishlists, nil
}

// SaveOffer saves an offer to the database
func (r *WishlistRepository) SaveOffer(offer *models.Offer) error {
	query := `
		INSERT INTO offers (product_name, price, original_price, discount_percentage, cashback_percentage, source, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		offer.ProductName,
		offer.Price,
		offer.OriginalPrice,
		offer.DiscountPercentage,
		offer.CashbackPercentage,
		offer.Source,
		offer.ReceivedAt,
	).Scan(&offer.ID)

	if err != nil {
		return fmt.Errorf("failed to save offer: %w", err)
	}

	return nil
}

// InvalidateCache invalidates the wishlist cache
func (r *WishlistRepository) InvalidateCache() {
	r.redis.Del(r.ctx, "wishlists:all")
}

// InvalidateUserCache invalidates cache for a specific user
func (r *WishlistRepository) InvalidateUserCache(telegramID int64) {
	cacheKey := fmt.Sprintf("wishlist:%d", telegramID)
	r.redis.Del(r.ctx, cacheKey)
	r.InvalidateCache()
}

// GetDB returns the database connection
func (r *WishlistRepository) GetDB() *sql.DB {
	return r.db
}

