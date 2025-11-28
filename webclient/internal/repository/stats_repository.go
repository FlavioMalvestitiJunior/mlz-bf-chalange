package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/yourusername/bf-offers/webclient/internal/models"
)

type StatsRepository struct {
	db    *sql.DB
	redis *redis.Client
	ctx   context.Context
}

func NewStatsRepository(db *sql.DB, redisClient *redis.Client) *StatsRepository {
	return &StatsRepository{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// GetDashboardStats returns overall dashboard statistics
func (r *StatsRepository) GetDashboardStats() (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// Get active users (last 24 hours)
	err := r.db.QueryRow(`
		SELECT COUNT(DISTINCT telegram_id) 
		FROM users 
		WHERE updated_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	// Get total users
	err = r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	// Get total wishlists
	err = r.db.QueryRow(`SELECT COUNT(*) FROM wishlists`).Scan(&stats.TotalWishlists)
	if err != nil {
		return nil, err
	}

	// Get recent offers (last 24 hours)
	err = r.db.QueryRow(`
		SELECT COUNT(*) 
		FROM offers 
		WHERE received_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.RecentOffers)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetActiveUsers returns list of active users with their activity
func (r *StatsRepository) GetActiveUsers(limit int) ([]models.UserActivity, error) {
	rows, err := r.db.Query(`
		SELECT 
			u.telegram_id,
			u.username,
			u.first_name,
			u.last_name,
			u.updated_at,
			COUNT(w.id) as wishlists
		FROM users u
		LEFT JOIN wishlists w ON u.telegram_id = w.telegram_id
		WHERE u.updated_at > NOW() - INTERVAL '24 hours'
		GROUP BY u.telegram_id, u.username, u.first_name, u.last_name, u.updated_at
		ORDER BY u.updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserActivity
	for rows.Next() {
		var user models.UserActivity
		err := rows.Scan(
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.LastActive,
			&user.Wishlists,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// SearchUsers searches for users by name or username
func (r *StatsRepository) SearchUsers(query string) ([]models.UserActivity, error) {
	rows, err := r.db.Query(`
		SELECT 
			u.telegram_id,
			u.username,
			u.first_name,
			u.last_name,
			u.updated_at,
			COUNT(w.id) as wishlists
		FROM users u
		LEFT JOIN wishlists w ON u.telegram_id = w.telegram_id
		WHERE 
			LOWER(u.first_name) LIKE LOWER($1) OR 
			LOWER(u.last_name) LIKE LOWER($1) OR 
			LOWER(u.username) LIKE LOWER($1)
		GROUP BY u.telegram_id, u.username, u.first_name, u.last_name, u.updated_at
		ORDER BY u.updated_at DESC
		LIMIT 20
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserActivity
	for rows.Next() {
		var user models.UserActivity
		err := rows.Scan(
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.LastActive,
			&user.Wishlists,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserWishlist returns the wishlist for a specific user
func (r *StatsRepository) GetUserWishlist(userID int64) ([]models.Wishlist, error) {
	// Try to get from Redis cache first
	cacheKey := fmt.Sprintf("wishlist:%d", userID)
	cached, err := r.redis.Get(r.ctx, cacheKey).Result()
	if err == nil {
		var wishlists []models.Wishlist
		if err := json.Unmarshal([]byte(cached), &wishlists); err == nil {
			return wishlists, nil
		}
	}

	// If not in cache, get from database
	rows, err := r.db.Query(`
		SELECT id, telegram_id, product_name, target_price, discount_percentage, created_at
		FROM wishlists
		WHERE telegram_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		wishlists = append(wishlists, w)
	}

	// Cache the result
	if data, err := json.Marshal(wishlists); err == nil {
		r.redis.Set(r.ctx, cacheKey, data, 5*time.Minute)
	}

	return wishlists, nil
}

// BlacklistUser adds a user to the blacklist
func (r *StatsRepository) BlacklistUser(userID int64) error {
	// Update database
	_, err := r.db.Exec(`UPDATE users SET is_blacklisted = true WHERE telegram_id = $1`, userID)
	if err != nil {
		return err
	}

	// Update Redis
	r.redis.Set(r.ctx, fmt.Sprintf("blacklist:%d", userID), "true", 0)
	
	return nil
}

// UnblacklistUser removes a user from the blacklist
func (r *StatsRepository) UnblacklistUser(userID int64) error {
	// Update database
	_, err := r.db.Exec(`UPDATE users SET is_blacklisted = false WHERE telegram_id = $1`, userID)
	if err != nil {
		return err
	}

	// Update Redis
	r.redis.Del(r.ctx, fmt.Sprintf("blacklist:%d", userID))
	
	return nil
}

// DeleteUser deletes a user and their data
func (r *StatsRepository) DeleteUser(userID int64) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete wishlists
	_, err = tx.Exec(`DELETE FROM wishlists WHERE telegram_id = $1`, userID)
	if err != nil {
		return err
	}

	// Delete user
	_, err = tx.Exec(`DELETE FROM users WHERE telegram_id = $1`, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Clean up Redis
	r.redis.Del(r.ctx, fmt.Sprintf("wishlist:%d", userID))
	r.redis.Del(r.ctx, fmt.Sprintf("blacklist:%d", userID))
	
	return nil
}
