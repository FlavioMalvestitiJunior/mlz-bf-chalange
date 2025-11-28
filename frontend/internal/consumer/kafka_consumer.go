package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/yourusername/bf-offers/frontend/internal/bot"
	"github.com/yourusername/bf-offers/frontend/internal/models"
)

type KafkaConsumer struct {
	ready      chan bool
	botHandler *bot.BotHandler
}

func NewKafkaConsumer(botHandler *bot.BotHandler) *KafkaConsumer {
	return &KafkaConsumer{
		ready:      make(chan bool),
		botHandler: botHandler,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *KafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *KafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (consumer *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := consumer.processMessage(message.Value); err != nil {
			log.Printf("Error processing message: %v", err)
		}
		session.MarkMessage(message, "")
	}
	return nil
}

// processMessage processes a Kafka message
func (c *KafkaConsumer) processMessage(data []byte) error {
	// Try OfferNotification
	var offerNotification models.OfferNotification
	if err := json.Unmarshal(data, &offerNotification); err == nil && offerNotification.TelegramID != 0 {
		if offerNotification.ProductName != "" {
			log.Printf("Received offer notification for user %d: %s", offerNotification.TelegramID, offerNotification.ProductName)
			return c.botHandler.SendNotification(&offerNotification)
		}
	}

	// Try WishlistResponse
	var wishlistResponse models.WishlistResponse
	if err := json.Unmarshal(data, &wishlistResponse); err == nil && wishlistResponse.ChatID != 0 {
		log.Printf("Received wishlist response for chat %d", wishlistResponse.ChatID)
		return c.botHandler.SendWishlistResponse(&wishlistResponse)
	}

	// Try DeleteResponse
	var deleteResponse models.DeleteResponse
	if err := json.Unmarshal(data, &deleteResponse); err == nil && deleteResponse.ChatID != 0 {
		log.Printf("Received delete response for chat %d", deleteResponse.ChatID)
		return c.botHandler.SendDeleteResponse(&deleteResponse)
	}

	log.Printf("Unknown message type received: %s", string(data))
	return nil
}

// StartConsumerGroup starts the consumer group
func StartConsumerGroup(ctx context.Context, brokers []string, topic, groupID string, botHandler *bot.BotHandler) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return err
	}

	consumer := NewKafkaConsumer(botHandler)

	go func() {
		for {
			if err := client.Consume(ctx, []string{topic}, consumer); err != nil {
				log.Printf("Error from consumer: %v", err)
				time.Sleep(time.Second * 5)
			}
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready
	log.Println("Sarama consumer up and running!...")
	return nil
}
