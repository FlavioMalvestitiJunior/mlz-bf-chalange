package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/consumer"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/handler"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/matcher"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/models"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/producer"
	"github.com/FlavioMalvestitiJunior/bf-offers/backend/internal/repository"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting Backend Service...")
	ctx, cancel := context.WithCancel(context.Background())
	// Load configuration from environment
	config := loadConfig()

	// Initialize database connection
	db, err := initDB(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient := initRedis(config)
	defer redisClient.Close()

	// Initialize repository
	repo := repository.NewWishlistRepository(db, redisClient)

	// Initialize Kafka producer for notifications
	kafkaNotificationProducer := producer.NewKafkaProducer(
		strings.Split(config.KafkaBrokers, ","),
		config.KafkaNotificationTopic,
	)
	defer kafkaNotificationProducer.Close()

	// Initialize Kafka writer for responses (Sarama SyncProducer)
	kafkaResponseWriter, err := producer.NewKafkaWriter(strings.Split(config.KafkaBrokers, ","))
	if err != nil {
		log.Fatalf("Failed to create Kafka response writer: %v", err)
	}
	defer kafkaResponseWriter.Close()

	// Initialize offer matcher
	offerMatcher := matcher.NewOfferMatcher()

	// Initialize command handler
	cmdHandler := handler.NewCommandHandler(db, redisClient, kafkaResponseWriter, config.KafkaNotificationTopic, config.KafkaWishlistEventsTopic)

	// Start command consumer
	err = consumer.StartConsumerGroup(
		ctx,
		strings.Split(config.KafkaBrokers, ","),
		config.KafkaCommandTopic,
		"backend-command-consumer",
		cmdHandler.HandleCommand,
	)
	if err != nil {
		log.Printf("Failed to start command consumer: %v", err)
	}

	// Start offers consumer
	err = consumer.StartConsumerGroup(
		ctx,
		strings.Split(config.KafkaBrokers, ","),
		config.KafkaOffersTopic,
		"backend-offers-consumer",
		func(data []byte) error {
			var offer models.Offer
			if err := json.Unmarshal(data, &offer); err != nil {
				log.Printf("Failed to unmarshal offer: %v", err)
				return nil // Don't retry malformed messages
			}
			offer.ReceivedAt = time.Now()
			return handleOffer(&offer, repo, offerMatcher, kafkaNotificationProducer)
		},
	)
	if err != nil {
		log.Fatalf("Failed to start offers consumer: %v", err)
	}

	// Start health check server
	go startHealthServer(config.Port)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Backend service is ready and listening for offers and commands...")

	<-sigChan
	log.Println("Shutdown signal received, stopping...")
	cancel()

	log.Println("Backend service stopped gracefully")
}

// handleOffer processes an incoming offer
func handleOffer(offer *models.Offer, repo *repository.WishlistRepository,
	matcher *matcher.OfferMatcher, producer *producer.KafkaProducer) error {

	log.Printf("Processing offer: %s - R$ %.2f", offer.ProductName, offer.Price)

	// Save offer to database
	if err := repo.SaveOffer(offer); err != nil {
		log.Printf("Failed to save offer: %v", err)
	}

	// Get all wishlists
	wishlists, err := repo.GetAllWishlists()
	if err != nil {
		return fmt.Errorf("failed to get wishlists: %w", err)
	}

	// Match offer against wishlists
	notifications := matcher.MatchOffer(offer, wishlists)

	// Send notifications via Kafka
	if len(notifications) > 0 {
		if err := producer.SendNotifications(notifications); err != nil {
			return fmt.Errorf("failed to send notifications: %w", err)
		}
		log.Printf("Sent %d notifications for offer: %s", len(notifications), offer.ProductName)
	}

	return nil
}

// Config holds application configuration
type Config struct {
	KafkaBrokers             string
	KafkaNotificationTopic   string
	KafkaCommandTopic        string
	KafkaOffersTopic         string
	KafkaWishlistEventsTopic string
	RedisHost                string
	RedisPort                string
	RedisPassword            string
	RedisDB                  int
	PostgresHost             string
	PostgresPort             string
	PostgresUser             string
	PostgresPass             string
	PostgresDB               string
	Port                     string
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		KafkaBrokers:             getEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaNotificationTopic:   getEnv("KAFKA_NOTIFICATION_TOPIC", "bot-responses"),
		KafkaCommandTopic:        getEnv("KAFKA_COMMAND_TOPIC", "bot-commands"),
		KafkaOffersTopic:         getEnv("KAFKA_OFFERS_TOPIC", "offers"),
		KafkaWishlistEventsTopic: getEnv("KAFKA_WISHLIST_EVENTS_TOPIC", "wishlist-events"),
		RedisHost:                getEnv("REDIS_HOST", "redis"),
		RedisPort:                getEnv("REDIS_PORT", "6379"),
		RedisPassword:            getEnv("REDIS_PASSWORD", ""),
		RedisDB:                  0,
		PostgresHost:             getEnv("POSTGRES_HOST", "postgres"),
		PostgresPort:             getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:             getEnv("POSTGRES_USER", "postgres"),
		PostgresPass:             getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:               getEnv("POSTGRES_DB", "postgres"),
		Port:                     getEnv("BACKEND_PORT", "8080"),
	}
}

// initDB initializes the database connection
func initDB(config Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPass, config.PostgresDB)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			log.Println("Database connection established")
			return db, nil
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("database connection timeout")
}

// initRedis initializes the Redis client
func initRedis(config Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Wait for Redis to be ready
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		if err := client.Ping(ctx).Err(); err == nil {
			log.Println("Redis connection established")
			return client
		}
		log.Printf("Waiting for Redis... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Println("Warning: Redis connection failed, continuing without cache")
	return client
}

// startHealthServer starts a simple health check HTTP server
func startHealthServer(port string) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Health check server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Printf("Health server error: %v", err)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
