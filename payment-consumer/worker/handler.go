package worker

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database/bridge"
	consumerstripe "github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentStripe"
	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlePaymentRequested(ctx context.Context, queries *bridge.Queries, stripeClient *consumerstripe.Client, d amqp.Delivery) error {
	var msg paymentRequestedMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return err
	}

	currency := strings.ToLower(msg.Currency)
	metadata := map[string]string{
		"payment_request_id": strconv.FormatInt(msg.PaymentID, 10),
		"payment_method":     msg.PaymentMethod,
	}
	if msg.Installments != nil {
		metadata["installments"] = strconv.Itoa(*msg.Installments)
	}

	intent, err := consumerstripe.CreatePaymentIntent(stripeClient, msg.AmountCents, currency, msg.IdempotencyKey, metadata)
	if err != nil {
		return err
	}

	if shouldConfirmPaymentIntent(msg.PaymentMethod) {
		confirmedIntent, err := consumerstripe.ConfirmPaymentIntent(stripeClient, intent.ID, msg.PaymentMethod)
		if err != nil {
			return err
		}

		intent = confirmedIntent
	} else if strings.TrimSpace(msg.PaymentMethod) != "" {
		log.Printf("skipping Stripe confirmation because payment_method=%q is not a Stripe payment method id", msg.PaymentMethod)
	}

	responseBody, _ := json.Marshal(map[string]any{
		"payment_intent_id": intent.ID,
		"client_secret":     intent.ClientSecret,
		"status":            intent.Status,
		"amount":            intent.Amount,
		"currency":          intent.Currency,
	})

	return queries.UpdatePaymentAttempt(ctx, bridge.UpdatePaymentAttemptParams{
		PaymentRequestID:      strconv.FormatInt(msg.PaymentID, 10),
		StripePaymentIntentID: pgtype.Text{String: intent.ID, Valid: true},
		AttemptNumber:         1,
		Currency:              strings.ToUpper(currency),
		Status:                string(intent.Status),
		ErrorCode:             pgtype.Text{},
		ErrorMessage:          pgtype.Text{},
		Response:              responseBody,
	})
}
