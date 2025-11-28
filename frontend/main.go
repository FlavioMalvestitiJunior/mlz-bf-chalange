package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/bf-offers/frontend/internal/bot"
	"github.com/yourusername/bf-offers/frontend/internal/consumer"
)

func main() {
	log.Println("Starting Frontend Service (Telegram Bot)...")

	// Load configuration
	config := loadConfig()

	// Initialize Telegram bot
	telegramBot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	log.Printf("Authorized on account %s", telegramBot.Self.UserName)

	// Initialize Kafka writer for sending commands to backend (Sarama SyncProducer)
	configKafka := sarama.NewConfig()
	configKafka.Producer.Return.Successes = true
	configKafka.Producer.RequiredAcks = sarama.WaitForAll
	configKafka.Producer.Retry.Max = 5

	kafkaProducer, err := sarama.NewSyncProducer(strings.Split(config.KafkaBrokers, ","), configKafka)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Initialize bot handler
	botHandler := bot.NewBotHandler(telegramBot, kafkaProducer, config.KafkaCommandTopic)

	// Start health check server
	go startHealthServer(config.Port)

	// Create context for graceful shutdown
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

	// Start Kafka consumer for receiving responses from backend
	go func() {
		err := consumer.StartConsumerGroup(
			ctx,
			strings.Split(config.KafkaBrokers, ","),
			config.KafkaResponseTopic,
			config.KafkaGroupID,
			botHandler,
		)
		if err != nil {
			log.Printf("Kafka consumer error: %v", err)
		}
	}()

	// Start Telegram bot updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := telegramBot.GetUpdatesChan(u)

	log.Println("Frontend service is ready and listening for updates...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Telegram bot...")
			telegramBot.StopReceivingUpdates()
			log.Println("Frontend service stopped gracefully")
			return
		case update := <-updates:
			go botHandler.HandleUpdate(update)
		}
	}
}

// Config holds application configuration
type Config struct {
	TelegramToken       string
	KafkaBrokers        string
	KafkaCommandTopic   string
	KafkaResponseTopic  string
	KafkaGroupID        string
	Port                string
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		TelegramToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		KafkaBrokers:       getEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaCommandTopic:  getEnv("KAFKA_COMMAND_TOPIC", "bot-commands"),
		KafkaResponseTopic: getEnv("KAFKA_RESPONSE_TOPIC", "bot-responses"),
		KafkaGroupID:       getEnv("KAFKA_GROUP_ID", "telegram-bot-consumer"),
		Port:               getEnv("FRONTEND_PORT", "8081"),
	}
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
