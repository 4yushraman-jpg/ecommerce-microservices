package providers

import (
	"context"
	"fmt"
	"log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// The interface our Kafka consumer will rely on
type EmailProvider interface {
	SendReceipt(ctx context.Context, toEmail string, orderID int64, amount int64, currency string) error
}

type SendGridProvider struct {
	Client      *sendgrid.Client
	FromAddress string
	FromName    string
}

func NewSendGridProvider(apiKey, fromEmail, fromName string) *SendGridProvider {
	return &SendGridProvider{
		Client:      sendgrid.NewSendClient(apiKey),
		FromAddress: fromEmail,
		FromName:    fromName,
	}
}

func (p *SendGridProvider) SendReceipt(ctx context.Context, toEmail string, orderID int64, amount int64, currency string) error {
	from := mail.NewEmail(p.FromName, p.FromAddress)
	to := mail.NewEmail("Valued Customer", toEmail)

	subject := fmt.Sprintf("Receipt for Order #%d", orderID)
	displayAmount := float64(amount) / 100.0

	plainTextContent := fmt.Sprintf("Thank you for your purchase! Your order #%d has been successfully paid. Total: %.2f %s", orderID, displayAmount, currency)
	htmlContent := fmt.Sprintf("<strong>Thank you for your purchase!</strong><br>Your order #%d has been successfully paid.<br>Total: %.2f %s", orderID, displayAmount, currency)

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	response, err := p.Client.SendWithContext(ctx, message)
	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API returned status %d: %s", response.StatusCode, response.Body)
	}

	log.Printf("Successfully sent receipt email to %s for Order #%d", toEmail, orderID)
	return nil
}
