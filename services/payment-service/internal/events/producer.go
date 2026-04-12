package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type PaymentEventSucceeded struct {
	OrderID  int64  `json:"order_id"`
	UserID   int64  `json:"user_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type KafkaProducer struct {
	Writer *kafka.Writer
}

func NewKafkaProducer(brokerURL string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerURL),
		Topic:    "payment-events",
		Balancer: &kafka.LeastBytes{},
	}
	return &KafkaProducer{Writer: w}
}

func (p *KafkaProducer) PublishPaymentSucceeded(ctx context.Context, event PaymentEventSucceeded) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.Writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("payment-succeeded"),
		Value: payload,
		Time:  time.Now(),
	})
	if err != nil {
		log.Printf("Failed to publish to Kafka: %v", err)
		return err
	}
	return nil
}

func (p *KafkaProducer) Close() error {
	return p.Writer.Close()
}
