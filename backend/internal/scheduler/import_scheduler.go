package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/tidwall/gjson"
	"github.com/yourusername/bf-offers/backend/internal/models"
	"github.com/yourusername/bf-offers/backend/internal/repository"
)

type ImportScheduler struct {
	repo           *repository.ImportTemplateRepository
	kafkaProducer  sarama.SyncProducer
	kafkaTopic     string
	interval       time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewImportScheduler(
	repo *repository.ImportTemplateRepository,
	kafkaProducer sarama.SyncProducer,
	kafkaTopic string,
	intervalMinutes int,
) *ImportScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &ImportScheduler{
		repo:          repo,
		kafkaProducer: kafkaProducer,
		kafkaTopic:    kafkaTopic,
		interval:      time.Duration(intervalMinutes) * time.Minute,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the scheduler
func (s *ImportScheduler) Start() {
	log.Printf("Starting import scheduler with interval: %v", s.interval)
	
	// Run immediately on start
	go s.runImports()
	
	ticker := time.NewTicker(s.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.runImports()
			case <-s.ctx.Done():
				ticker.Stop()
				log.Println("Import scheduler stopped")
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *ImportScheduler) Stop() {
	s.cancel()
}

// runImports executes all active import templates
func (s *ImportScheduler) runImports() {
	log.Println("Running scheduled imports...")
	
	templates, err := s.repo.GetActiveTemplates()
	if err != nil {
		log.Printf("Error fetching active templates: %v", err)
		return
	}

	if len(templates) == 0 {
		log.Println("No active import templates found")
		return
	}

	for _, template := range templates {
		if err := s.processTemplate(&template); err != nil {
			log.Printf("Error processing template %s: %v", template.Name, err)
		} else {
			// Update last run timestamp
			if err := s.repo.UpdateLastRunAt(template.ID); err != nil {
				log.Printf("Error updating last_run_at for template %s: %v", template.Name, err)
			}
		}
	}

	log.Printf("Completed scheduled imports for %d templates", len(templates))
}

// processTemplate fetches JSON from S3 and produces to Kafka
func (s *ImportScheduler) processTemplate(template *models.ImportTemplate) error {
	log.Printf("Processing template: %s (URL: %s)", template.Name, template.S3URL)

	// Fetch JSON from S3 URL
	resp, err := http.Get(template.S3URL)
	if err != nil {
		return fmt.Errorf("failed to fetch S3 URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("S3 URL returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse mapping schema
	var mappingSchema map[string]string
	if err := json.Unmarshal([]byte(template.MappingSchema), &mappingSchema); err != nil {
		return fmt.Errorf("failed to parse mapping schema: %w", err)
	}

	// Check if the JSON is an array or single object
	jsonStr := string(body)
	if gjson.Get(jsonStr, "#").Exists() {
		// It's an array
		result := gjson.Parse(jsonStr)
		if !result.IsArray() {
			return fmt.Errorf("expected JSON array but got different type")
		}

		count := 0
		result.ForEach(func(key, value gjson.Result) bool {
			offer, err := s.mapJSONToOffer(value.Raw, mappingSchema)
			if err != nil {
				log.Printf("Error mapping JSON to offer: %v", err)
				return true // Continue to next item
			}

			if err := s.produceOffer(offer); err != nil {
				log.Printf("Error producing offer to Kafka: %v", err)
			} else {
				count++
			}
			return true
		})

		log.Printf("Produced %d offers from template %s", count, template.Name)
	} else {
		// Single object
		offer, err := s.mapJSONToOffer(jsonStr, mappingSchema)
		if err != nil {
			return fmt.Errorf("failed to map JSON to offer: %w", err)
		}

		if err := s.produceOffer(offer); err != nil {
			return fmt.Errorf("failed to produce offer to Kafka: %w", err)
		}

		log.Printf("Produced 1 offer from template %s", template.Name)
	}

	return nil
}

// mapJSONToOffer maps JSON fields to Offer model using mapping schema
func (s *ImportScheduler) mapJSONToOffer(jsonStr string, mapping map[string]string) (*models.Offer, error) {
	offer := &models.Offer{
		ReceivedAt: time.Now(),
	}

	// Map ProductName
	if path, ok := mapping["ProductName"]; ok {
		offer.ProductName = gjson.Get(jsonStr, path).String()
	}

	// Map Price
	if path, ok := mapping["Price"]; ok {
		priceVal := gjson.Get(jsonStr, path)
		if priceVal.Exists() {
			if priceVal.Type == gjson.Number {
				offer.Price = priceVal.Float()
			} else {
				// Try to parse as string
				if price, err := strconv.ParseFloat(priceVal.String(), 64); err == nil {
					offer.Price = price
				}
			}
		}
	}

	// Map OriginalPrice
	if path, ok := mapping["OriginalPrice"]; ok {
		priceVal := gjson.Get(jsonStr, path)
		if priceVal.Exists() {
			if priceVal.Type == gjson.Number {
				offer.OriginalPrice = priceVal.Float()
			} else {
				if price, err := strconv.ParseFloat(priceVal.String(), 64); err == nil {
					offer.OriginalPrice = price
				}
			}
		}
	}

	// Map Details
	if path, ok := mapping["Details"]; ok {
		offer.Details = gjson.Get(jsonStr, path).String()
	}

	// Map CashbackPercentage
	if path, ok := mapping["CashbackPercentage"]; ok {
		cashbackVal := gjson.Get(jsonStr, path)
		if cashbackVal.Exists() {
			if cashbackVal.Type == gjson.Number {
				offer.CashbackPercentage = int(cashbackVal.Int())
			} else {
				if cb, err := strconv.Atoi(cashbackVal.String()); err == nil {
					offer.CashbackPercentage = cb
				}
			}
		}
	}

	// Map Source
	if path, ok := mapping["Source"]; ok {
		offer.Source = gjson.Get(jsonStr, path).String()
	} else {
		offer.Source = "s3-import"
	}

	// Validate required fields
	if offer.ProductName == "" {
		return nil, fmt.Errorf("ProductName is required but not found in mapping")
	}

	return offer, nil
}

// produceOffer sends an offer to Kafka
func (s *ImportScheduler) produceOffer(offer *models.Offer) error {
	offerJSON, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("failed to marshal offer: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: s.kafkaTopic,
		Value: sarama.StringEncoder(offerJSON),
	}

	_, _, err = s.kafkaProducer.SendMessage(msg)
	return err
}
