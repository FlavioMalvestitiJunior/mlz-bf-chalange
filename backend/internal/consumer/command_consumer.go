package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

// Command represents a command from the frontend
type Command struct {
	Type               string    `json:"type"`
	TelegramID         int64     `json:"telegram_id"`
	ChatID             int64     `json:"chat_id,omitempty"`
	Username           string    `json:"username,omitempty"`
	FirstName          string    `json:"first_name,omitempty"`
	LastName           string    `json:"last_name,omitempty"`
	ProductName        string    `json:"product_name,omitempty"`
	TargetPrice        *float64  `json:"target_price,omitempty"`
	DiscountPercentage *int      `json:"discount_percentage,omitempty"`
	WishlistID         int       `json:"wishlist_id,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// ParseCommand parses a command from JSON
func ParseCommand(data []byte) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

type CommandConsumer struct {
	ready   chan bool
	handler func([]byte) error
}

func NewCommandConsumer(handler func([]byte) error) *CommandConsumer {
	return &CommandConsumer{
		ready:   make(chan bool),
		handler: handler,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *CommandConsumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *CommandConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (consumer *CommandConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// Process the message
		if err := consumer.handler(message.Value); err != nil {
			log.Printf("Error handling command: %v", err)
		}
		session.MarkMessage(message, "")
	}
	return nil
}

// StartConsumerGroup starts the consumer group
func StartConsumerGroup(ctx context.Context, brokers []string, topic, groupID string, handler func([]byte) error) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0 // Specify a version
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return err
	}

	consumer := NewCommandConsumer(handler)

	go func() {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, []string{topic}, consumer); err != nil {
				log.Printf("Error from consumer: %v", err)
				time.Sleep(time.Second * 5) // Wait before retrying
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Wait till the consumer has been set up
	log.Println("Sarama consumer up and running!...")
	return nil
}
