package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"notification-service/internal/clients"
	"notification-service/internal/providers"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

// This MUST match the struct you created in the Payment Service exactly!
type PaymentEventSucceeded struct {
	OrderID  int64  `json:"order_id"`
	UserID   int64  `json:"user_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type KafkaConsumer struct {
	Reader        *kafka.Reader
	DB            *pgxpool.Pool
	EmailProvider providers.EmailProvider
	UserClient    *clients.UserServiceClient
}

func NewKafkaConsumer(brokerURL string, db *pgxpool.Pool, emailProvider providers.EmailProvider, userClient *clients.UserServiceClient) *KafkaConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerURL},
		GroupID: "notification-service-group", // CRITICAL: This tracks which messages this service has already read
		Topic:   "payment-events",
		MaxWait: 1 * time.Second, // Wait at most 1 second for new messages to batch
	})

	return &KafkaConsumer{
		Reader:        r,
		DB:            db,
		EmailProvider: emailProvider,
		UserClient:    userClient,
	}
}

// Start runs the infinite loop that processes messages
func (c *KafkaConsumer) Start(ctx context.Context) {
	log.Println("Starting Kafka Consumer for topic: payment-events...")

	for {
		// 1. Fetch the message, but do NOT commit it yet
		msg, err := c.Reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Consumer context cancelled, shutting down gracefully...")
				return // Context was cancelled by main.go shutdown
			}
			log.Printf("Error fetching message: %v", err)
			time.Sleep(1 * time.Second) // Prevent tight loop on error
			continue
		}

		// 2. Route the message based on its Key
		if string(msg.Key) == "payment-succeeded" {
			err = c.handlePaymentSucceeded(ctx, msg.Value)
			if err != nil {
				log.Printf("Failed to process payment event: %v", err)
				// Here, we choose NOT to commit. The message stays in Kafka.
				// When the service restarts, it will try again.
				continue
			}
		}

		// 3. Mark the message as processed ONLY if we successfully sent the email
		if err := c.Reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Failed to commit message: %v", err)
		}
	}
}

func (c *KafkaConsumer) handlePaymentSucceeded(ctx context.Context, payload []byte) error {
	var event PaymentEventSucceeded
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	log.Printf("Received payment_succeeded event for Order #%d", event.OrderID)

	userEmail, err := c.UserClient.GetUserEmail(ctx, event.UserID)
	if err != nil {
		log.Printf("Failed to fetch user email: %v", err)
		return err // Returning an error means Kafka will not commit, and it will retry later!
	}

	err = c.EmailProvider.SendReceipt(ctx, userEmail, event.OrderID, event.Amount, event.Currency)
	if err != nil {
		return err
	}

	return nil
}

func (c *KafkaConsumer) Close() error {
	return c.Reader.Close()
}
