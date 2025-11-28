package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/FlavioMalvestitiJunior/bf-offers/s3-importer/internal/importer"
	"github.com/FlavioMalvestitiJunior/bf-offers/s3-importer/internal/repository"
	"github.com/IBM/sarama"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("S3 Importer Service Starting...")

	// Load configuration from environment
	config := loadConfig()

	// Initialize database connection
	db, err := initDB(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Kafka producer
	kafkaProducer, err := initKafkaProducer(config.KafkaBrokers)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Initialize repository
	importRepo := repository.NewImportTemplateRepository(db)

	// Initialize importer
	s3Importer := importer.NewS3Importer(
		importRepo,
		kafkaProducer,
		config.KafkaOffersTopic,
	)

	// Run import job
	if err := s3Importer.Run(); err != nil {
		log.Fatalf("Import job failed: %v", err)
	}

	log.Println("S3 Importer Service completed successfully")
}

// Config holds application configuration
type Config struct {
	KafkaBrokers     string
	KafkaOffersTopic string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPass     string
	PostgresDB       string
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		KafkaBrokers:     getEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaOffersTopic: getEnv("KAFKA_OFFERS_TOPIC", "offers"),
		PostgresHost:     getEnv("POSTGRES_HOST", "postgres"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPass:     getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "postgres"),
	}
}

// initDB initializes the database connection
func initDB(config Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
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

// initKafkaProducer initializes the Kafka producer
func initKafkaProducer(brokers string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	brokerList := strings.Split(brokers, ",")
	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Printf("Kafka producer connected to: %s", brokers)
	return producer, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
