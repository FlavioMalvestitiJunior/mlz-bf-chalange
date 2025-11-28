package producer

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/yourusername/bf-offers/backend/internal/models"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Failed to start Sarama producer: %v", err)
	}

	return &KafkaProducer{
		producer: producer,
		topic:    topic,
	}
}

// NewKafkaWriter creates a new Sarama SyncProducer (helper function)
func NewKafkaWriter(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	return sarama.NewSyncProducer(brokers, config)
}

// SendNotification sends an offer notification to Kafka
func (p *KafkaProducer) SendNotification(notification *models.OfferNotification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(fmt.Sprintf("%d", notification.TelegramID)),
		Value: sarama.ByteEncoder(data),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Message sent to partition %d at offset %d", partition, offset)
	return nil
}

// SendNotifications sends multiple notifications
func (p *KafkaProducer) SendNotifications(notifications []models.OfferNotification) error {
	var msgs []*sarama.ProducerMessage

	for _, n := range notifications {
		data, err := json.Marshal(n)
		if err != nil {
			log.Printf("Failed to marshal notification: %v", err)
			continue
		}

		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: p.topic,
			Key:   sarama.StringEncoder(fmt.Sprintf("%d", n.TelegramID)),
			Value: sarama.ByteEncoder(data),
		})
	}

	if len(msgs) == 0 {
		return nil
	}

	err := p.producer.SendMessages(msgs)
	if err != nil {
		return fmt.Errorf("failed to write batch to kafka: %w", err)
	}

	return nil
}

// Close closes the producer
func (p *KafkaProducer) Close() error {
	return p.producer.Close()
}
