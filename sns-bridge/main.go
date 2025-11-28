package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	_ "github.com/lib/pq"
)

// MessageTemplate represents a template for SNS messages
type MessageTemplate struct {
	ID               int
	Name             string
	ProductModel     string
	TitleField       string
	DescriptionField *string
	PriceField       string
	DiscountField    *string
	DetailsFields    *string // JSON array string
	IsActive         bool
}

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

func main() {
	log.Println("Starting SNS Bridge Service...")

	config := loadConfig()

	// Initialize Database
	db, err := initDB(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Kafka Producer
	producer, err := initKafkaProducer(config)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer producer.Close()

	// Initialize SNS/SQS Consumer
	sqsClient, err := initSQSClient(config)
	if err != nil {
		log.Fatalf("Failed to initialize SQS client: %v", err)
	}

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

	// Start polling loop
	log.Println("SNS Bridge service is ready and polling...")
	pollLoop(ctx, sqsClient, db, producer, config)

	log.Println("SNS Bridge service stopped gracefully")
}

func pollLoop(ctx context.Context, sqsClient *sqs.SQS, db *sql.DB, producer sarama.SyncProducer, config Config) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Receive messages
			result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(config.SNSQueueURL),
				MaxNumberOfMessages: aws.Int64(10),
				WaitTimeSeconds:     aws.Int64(20), // Long polling
				VisibilityTimeout:   aws.Int64(30),
			})

			if err != nil {
				log.Printf("Error receiving messages: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Process messages
			for _, message := range result.Messages {
				processMessage(message, db, producer, config)
				
				// Delete message
				_, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      aws.String(config.SNSQueueURL),
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Printf("Error deleting message: %v", err)
				}
			}
		}
	}
}

func processMessage(message *sqs.Message, db *sql.DB, producer sarama.SyncProducer, config Config) {
	if message.Body == nil {
		return
	}

	// Parse SNS wrapper if present
	var bodyStr string
	var snsMessage struct {
		Message string `json:"Message"`
	}
	if err := json.Unmarshal([]byte(*message.Body), &snsMessage); err == nil && snsMessage.Message != "" {
		bodyStr = snsMessage.Message
	} else {
		bodyStr = *message.Body
	}

	// Parse JSON body to map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(bodyStr), &data); err != nil {
		log.Printf("Failed to parse message body: %v", err)
		return
	}

	// Load active templates
	templates, err := loadActiveTemplates(db)
	if err != nil {
		log.Printf("Failed to load templates: %v", err)
		return
	}

	// Try to match/map using templates
	for _, tmpl := range templates {
		offer, err := mapToOffer(data, tmpl)
		if err != nil {
			continue // Template didn't match or error mapping
		}

		// Publish to Kafka
		if err := publishOffer(producer, offer, config.KafkaOffersTopic); err != nil {
			log.Printf("Failed to publish offer: %v", err)
		} else {
			log.Printf("Published offer: %s (Template: %s)", offer.ProductName, tmpl.Name)
		}
	}
}

func loadActiveTemplates(db *sql.DB) ([]MessageTemplate, error) {
	rows, err := db.Query(`
		SELECT id, name, product_model, title_field, description_field, price_field, 
		       discount_field, details_fields 
		FROM message_templates 
		WHERE is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []MessageTemplate
	for rows.Next() {
		var t MessageTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.ProductModel, &t.TitleField, 
			&t.DescriptionField, &t.PriceField, &t.DiscountField, &t.DetailsFields); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func mapToOffer(data map[string]interface{}, tmpl MessageTemplate) (*Offer, error) {
	// Extract fields based on template
	title, ok := getString(data, tmpl.TitleField)
	if !ok {
		return nil, fmt.Errorf("title field not found")
	}

	price, ok := getFloat(data, tmpl.PriceField)
	if !ok {
		return nil, fmt.Errorf("price field not found")
	}

	var oldPrice float64
	// Logic for oldPrice? Maybe just same as price if not found, or 0.
	// The requirement says "oldPrice". If not in template, maybe 0.
	// I'll assume 0 for now as it's not in the template explicit mapping (except maybe description/details).
	
	var details string
	if tmpl.DetailsFields != nil && *tmpl.DetailsFields != "" {
		var fields []string
		if err := json.Unmarshal([]byte(*tmpl.DetailsFields), &fields); err == nil {
			var parts []string
			for _, f := range fields {
				if val, ok := getString(data, f); ok {
					parts = append(parts, val)
				}
			}
			details = strings.Join(parts, " | ")
		}
	}
	if details == "" && tmpl.DescriptionField != nil {
		details, _ = getString(data, *tmpl.DescriptionField)
	}

	// Cashback? Not in template currently. Default to 0.
	
	return &Offer{
		ProductName:        title,
		Price:              price,
		OriginalPrice:      oldPrice,
		Details:            details,
		CashbackPercentage: 0,
		Source:             "sns-bridge",
		ReceivedAt:         time.Now(),
	}, nil
}

func getString(data map[string]interface{}, key string) (string, bool) {
	val, ok := data[key]
	if !ok {
		return "", false
	}
	if str, ok := val.(string); ok {
		return str, true
	}
	return fmt.Sprintf("%v", val), true
}

func getFloat(data map[string]interface{}, key string) (float64, bool) {
	val, ok := data[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case string:
		// Try parsing? For now assume number type in JSON
		return 0, false
	default:
		return 0, false
	}
}

func publishOffer(producer sarama.SyncProducer, offer *Offer, topic string) error {
	bytes, err := json.Marshal(offer)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(bytes),
	}

	_, _, err = producer.SendMessage(msg)
	return err
}

// Config and Init functions...

type Config struct {
	AWSRegion        string
	SNSQueueURL      string
	KafkaBrokers     string
	KafkaOffersTopic string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPass     string
	PostgresDB       string
}

func loadConfig() Config {
	return Config{
		AWSRegion:        getEnv("AWS_REGION", "us-east-1"),
		SNSQueueURL:      getEnv("SNS_QUEUE_URL", ""),
		KafkaBrokers:     getEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaOffersTopic: getEnv("KAFKA_OFFERS_TOPIC", "offers"),
		PostgresHost:     getEnv("POSTGRES_HOST", "postgres"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "offerbot"),
		PostgresPass:     getEnv("POSTGRES_PASSWORD", "offerbot123"),
		PostgresDB:       getEnv("POSTGRES_DB", "offerbot"),
	}
}

func initDB(config Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPass, config.PostgresDB)
	return sql.Open("postgres", connStr)
}

func initKafkaProducer(config Config) (sarama.SyncProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	return sarama.NewSyncProducer(strings.Split(config.KafkaBrokers, ","), saramaConfig)
}

func initSQSClient(config Config) (*sqs.SQS, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion),
	})
	if err != nil {
		return nil, err
	}
	return sqs.New(sess), nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
