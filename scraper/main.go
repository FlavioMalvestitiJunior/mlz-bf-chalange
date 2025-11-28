package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-redis/redis/v8"
)

// Offer represents the Kafka message schema
type Offer struct {
	ID                 int       `json:"id"`
	ProductName        string    `json:"titulo"`
	Price              float64   `json:"price"`
	OriginalPrice      float64   `json:"oldPrice"`
	Details            string    `json:"details"`
	CashbackPercentage int       `json:"percentCashback"`
	Source             string    `json:"source"`
	ReceivedAt         time.Time `json:"received_at"`
}

// WishlistEvent represents an event when a wishlist item is added
type WishlistEvent struct {
	Type               string    `json:"type"`
	TelegramID         int64     `json:"telegram_id"`
	ProductName        string    `json:"product_name"`
	TargetPrice        *float64  `json:"target_price,omitempty"`
	DiscountPercentage *int      `json:"discount_percentage,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// Promobit API Response Structures

type PromobitSearchResponse struct {
	Data struct {
		Offers []PromobitOffer `json:"offers"`
		Meta   struct {
			CurrentPage int `json:"current_page"`
			LastPage    int `json:"last_page"`
		} `json:"meta"`
	} `json:"data"`
}

type PromobitOffer struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	OldPrice    float64 `json:"old_price"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	IsActive    bool    `json:"is_active"`
	Cashback    struct {
		Percentage int `json:"percentage"`
	} `json:"cashback"`
}

type PromobitHomeResponse struct {
	PageProps struct {
		Offers []PromobitOffer `json:"offers"`
	} `json:"pageProps"`
}

func main() {
	log.Println("Starting Promobit Scraper Service...")

	config := loadConfig()

	// Initialize Redis
	redisClient := initRedis(config)
	defer redisClient.Close()

	// Initialize Kafka Producer
	producer, err := initKafkaProducer(config)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer producer.Close()

	// Initialize Kafka Consumer (for wishlist events)
	consumerGroup, err := initKafkaConsumer(config)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka consumer: %v", err)
	}
	defer consumerGroup.Close()

	// Context for shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping...")
		cancel()
	}()

	// Start periodic scraping (5 min) - General Promobit
	go startPeriodicScraping(ctx, producer, config, 5*time.Minute, scrapePromobitHome)

	// Start periodic wishlist scraping (10 min)
	go startPeriodicWishlistScraping(ctx, redisClient, producer, config, 10*time.Minute)

	// Start consumer for on-demand scraping
	go startWishlistEventConsumer(ctx, consumerGroup, producer, config)

	log.Println("Scraper service is running...")
	<-ctx.Done()
	log.Println("Scraper service stopped gracefully")
}

func startPeriodicScraping(ctx context.Context, producer sarama.SyncProducer, config Config, interval time.Duration, scrapeFunc func(sarama.SyncProducer, Config)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately once
	scrapeFunc(producer, config)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scrapeFunc(producer, config)
		}
	}
}

func startPeriodicWishlistScraping(ctx context.Context, redisClient *redis.Client, producer sarama.SyncProducer, config Config, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scrapeWishlistItems(redisClient, producer, config)
		}
	}
}

func startWishlistEventConsumer(ctx context.Context, consumerGroup sarama.ConsumerGroup, producer sarama.SyncProducer, config Config) {
	consumer := &WishlistConsumer{
		producer: producer,
		config:   config,
	}

	for {
		if err := consumerGroup.Consume(ctx, []string{config.KafkaWishlistEventsTopic}, consumer); err != nil {
			log.Printf("Error from consumer: %v", err)
			time.Sleep(5 * time.Second)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// WishlistConsumer implements sarama.ConsumerGroupHandler
type WishlistConsumer struct {
	producer sarama.SyncProducer
	config   Config
}

func (c *WishlistConsumer) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (c *WishlistConsumer) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (c *WishlistConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var event WishlistEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			log.Printf("Failed to unmarshal wishlist event: %v", err)
			session.MarkMessage(message, "")
			continue
		}

		if event.Type == "wishlist_item_added" {
			log.Printf("Received wishlist event for: %s", event.ProductName)
			scrapePromobitSearch(c.producer, c.config, event.ProductName)
		}

		session.MarkMessage(message, "")
	}
	return nil
}

// Scrape Logic using Promobit API

func scrapePromobitHome(producer sarama.SyncProducer, config Config) {
	log.Println("Fetching Promobit Home via API...")

	// Use the Next.js data endpoint
	resp, err := http.Get("https://www.promobit.com.br/_next/data/bcc3e837c1/index.json")
	if err != nil {
		log.Printf("Failed to fetch Promobit home: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Promobit home returned status: %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}

	var homeResp PromobitHomeResponse
	if err := json.Unmarshal(body, &homeResp); err != nil {
		log.Printf("Failed to unmarshal home response: %v", err)
		return
	}

	count := 0
	for _, promobitOffer := range homeResp.PageProps.Offers {
		// Only process active offers
		if !promobitOffer.IsActive {
			continue
		}

		offer := convertPromobitOffer(promobitOffer)
		publishOffer(producer, offer, config.KafkaOffersTopic)
		count++
	}

	log.Printf("Published %d offers from Promobit home", count)
}

func scrapePromobitSearch(producer sarama.SyncProducer, config Config, query string) {
	log.Printf("Searching Promobit API for: %s", query)

	encodedQuery := url.QueryEscape(query)
	page := 1
	totalCount := 0

	for {
		apiURL := fmt.Sprintf("https://api.promobit.com.br/search/result/offers?q=%s&page=%d", encodedQuery, page)
		
		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("Failed to fetch Promobit search page %d: %v", page, err)
			break
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Promobit search returned status: %d", resp.StatusCode)
			resp.Body.Close()
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			break
		}

		var searchResp PromobitSearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			log.Printf("Failed to unmarshal search response: %v", err)
			break
		}

		// Process offers from this page
		pageCount := 0
		for _, promobitOffer := range searchResp.Data.Offers {
			// Only process active offers
			if !promobitOffer.IsActive {
				continue
			}

			offer := convertPromobitOffer(promobitOffer)
			publishOffer(producer, offer, config.KafkaOffersTopic)
			pageCount++
			totalCount++
		}

		log.Printf("Published %d active offers from page %d", pageCount, page)

		// Check if there are more pages
		if page >= searchResp.Data.Meta.LastPage {
			break
		}

		page++
		time.Sleep(1 * time.Second) // Polite delay between pages
	}

	log.Printf("Total published %d active offers for query: %s", totalCount, query)
}

func scrapeWishlistItems(redisClient *redis.Client, producer sarama.SyncProducer, config Config) {
	log.Println("Scraping all wishlist items...")
	
	terms, err := redisClient.SMembers(context.Background(), "all_wishlist_terms").Result()
	if err != nil {
		log.Printf("Failed to get wishlist terms from Redis: %v", err)
		return
	}

	for _, term := range terms {
		scrapePromobitSearch(producer, config, term)
		time.Sleep(2 * time.Second) // Polite delay
	}
}

func convertPromobitOffer(promobitOffer PromobitOffer) *Offer {
	return &Offer{
		ID:                 promobitOffer.ID,
		ProductName:        promobitOffer.Title,
		Price:              promobitOffer.Price,
		OriginalPrice:      promobitOffer.OldPrice,
		Details:            promobitOffer.Description,
		CashbackPercentage: promobitOffer.Cashback.Percentage,
		Source:             "promobit-api",
		ReceivedAt:         time.Now(),
	}
}

func publishOffer(producer sarama.SyncProducer, offer *Offer, topic string) {
	bytes, err := json.Marshal(offer)
	if err != nil {
		log.Printf("Failed to marshal offer: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(bytes),
	}

	if _, _, err := producer.SendMessage(msg); err != nil {
		log.Printf("Failed to publish offer: %v", err)
	} else {
		log.Printf("Published offer: %s (R$ %.2f)", offer.ProductName, offer.Price)
	}
}

// Config and Init

type Config struct {
	KafkaBrokers             string
	KafkaOffersTopic         string
	KafkaWishlistEventsTopic string
	RedisHost                string
	RedisPort                string
	RedisPassword            string
	RedisDB                  int
}

func loadConfig() Config {
	return Config{
		KafkaBrokers:             getEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaOffersTopic:         getEnv("KAFKA_OFFERS_TOPIC", "offers"),
		KafkaWishlistEventsTopic: getEnv("KAFKA_WISHLIST_EVENTS_TOPIC", "wishlist-events"),
		RedisHost:                getEnv("REDIS_HOST", "redis"),
		RedisPort:                getEnv("REDIS_PORT", "6379"),
		RedisPassword:            getEnv("REDIS_PASSWORD", ""),
		RedisDB:                  0,
	}
}

func initRedis(config Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
}

func initKafkaProducer(config Config) (sarama.SyncProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	return sarama.NewSyncProducer(strings.Split(config.KafkaBrokers, ","), saramaConfig)
}

func initKafkaConsumer(config Config) (sarama.ConsumerGroup, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	return sarama.NewConsumerGroup(strings.Split(config.KafkaBrokers, ","), "scraper-consumer-group", saramaConfig)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
