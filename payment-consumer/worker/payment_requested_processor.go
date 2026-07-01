package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database/bridge"
	consumerstripe "github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentStripe"
	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stripe/stripe-go/v85"
)

type PaymentRequestedProcessor struct {
	queries      *bridge.Queries
	stripeClient *consumerstripe.Client
}

func NewPaymentRequestedProcessor(queries *bridge.Queries, stripeClient *consumerstripe.Client) *PaymentRequestedProcessor {
	return &PaymentRequestedProcessor{
		queries:      queries,
		stripeClient: stripeClient,
	}
}

func (p *PaymentRequestedProcessor) Handle(ctx context.Context, d amqp.Delivery) error {
	msg, err := parsePaymentRequestedMessage(d.Body)
	if err != nil {
		return err
	}

	hasAttempt, err := hasSuccessfulAttempt(ctx, p.queries, msg.PaymentID)
	if err != nil {
		return fmt.Errorf("check existing successful attempt: %w", err)
	}
	if hasAttempt {
		log.Printf("[INFO] Payment %s already has a successful attempt, skipping creation", msg.PaymentID)
		return nil
	}

	currency := strings.ToLower(msg.Currency)
	metadata := buildPaymentIntentMetadata(msg)

	intent, err := consumerstripe.CreatePaymentIntent(p.stripeClient, msg.AmountCents, currency, msg.IdempotencyKey, metadata)
	if err != nil {
		return saveFailedPaymentAttempt(ctx, p.queries, msg.PaymentID, "", currency, "create_payment_intent", err)
	}

	intent, err = confirmPaymentIntentIfNeeded(p.stripeClient, intent, msg)
	if err != nil {
		return saveFailedPaymentAttempt(ctx, p.queries, msg.PaymentID, intent.ID, currency, "confirm_payment_intent", err)
	}

	if err := saveSuccessfulPaymentAttempt(ctx, p.queries, msg.PaymentID, intent, currency); err != nil {
		return fmt.Errorf("save successful payment attempt: %w", err)
	}

	return updatePaymentRequestSuccess(ctx, p.queries, msg.PaymentID, intent)
}

func parsePaymentRequestedMessage(body []byte) (paymentRequestedMessage, error) {
	var msg paymentRequestedMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return paymentRequestedMessage{}, err
	}
	return msg, nil
}

func confirmPaymentIntentIfNeeded(
	stripeClient *consumerstripe.Client,
	intent *stripe.PaymentIntent,
	msg paymentRequestedMessage,
) (*stripe.PaymentIntent, error) {
	if strings.ToLower(strings.TrimSpace(msg.PaymentMethod)) != "credit" {
		return intent, nil
	}

	stripePaymentMethodID := strings.TrimSpace(msg.StripePaymentMethodID)
	if stripePaymentMethodID == "" {
		return nil, fmt.Errorf("missing stripe_payment_method_id for credit payment")
	}

	return consumerstripe.ConfirmPaymentIntent(stripeClient, intent.ID, stripePaymentMethodID)
}

func hasSuccessfulAttempt(ctx context.Context, queries *bridge.Queries, paymentID string) (bool, error) {
	existingAttempt, err := queries.GetLatestPaymentAttempt(ctx, parseStringToUUID(paymentID))
	if err != nil {
		return false, err
	}

	return existingAttempt.StripePaymentIntentID.Valid && existingAttempt.Status == "succeeded", nil
}

func buildPaymentIntentMetadata(msg paymentRequestedMessage) map[string]string {
	metadata := map[string]string{
		"payment_request_uuid": msg.PaymentID,
		"payment_method":       msg.PaymentMethod,
	}
	if msg.Installments != nil {
		metadata["installments"] = strconv.Itoa(*msg.Installments)
	}
	return metadata
}

func saveFailedPaymentAttempt(ctx context.Context, queries *bridge.Queries, paymentID string, intentID string, currency string, operation string, err error) error {
	errorCode, errorMessage, errorPayload := stripeErrorPayload(err)
	log.Printf("[ERROR] %s failed: %s", operation, errorMessage)

	stripeIntent := pgtype.Text{}
	if strings.TrimSpace(intentID) != "" {
		stripeIntent = pgtype.Text{String: intentID, Valid: true}
	}

	saveErr := queries.UpdatePaymentAttempt(ctx, bridge.UpdatePaymentAttemptParams{
		PaymentRequestUuid:    parseStringToUUID(paymentID),
		StripePaymentIntentID: stripeIntent,
		StripeClientSecret:    pgtype.Text{},
		AttemptNumber:         1,
		Currency:              strings.ToUpper(currency),
		Status:                "failed",
		ErrorCode:             pgtype.Text{String: errorCode, Valid: errorCode != ""},
		ErrorMessage:          pgtype.Text{String: errorMessage, Valid: true},
		Response:              appendOperationToPayload(operation, errorPayload),
	})
	if saveErr != nil {
		log.Printf("failed to save payment attempt error for %s: %v", operation, saveErr)
		return errors.Join(err, fmt.Errorf("save failed payment attempt: %w", saveErr))
	}

	return err
}

func saveSuccessfulPaymentAttempt(ctx context.Context, queries *bridge.Queries, paymentID string, intent *stripe.PaymentIntent, currency string) error {
	responseBody, _ := json.Marshal(map[string]any{
		"payment_intent_id": intent.ID,
		"client_secret":     intent.ClientSecret,
		"status":            intent.Status,
		"amount":            intent.Amount,
		"currency":          intent.Currency,
	})

	if err := queries.UpdatePaymentAttempt(ctx, bridge.UpdatePaymentAttemptParams{
		PaymentRequestUuid:    parseStringToUUID(paymentID),
		StripePaymentIntentID: pgtype.Text{String: intent.ID, Valid: true},
		StripeClientSecret:    pgtype.Text{String: intent.ClientSecret, Valid: true},
		AttemptNumber:         1,
		Currency:              strings.ToUpper(currency),
		Status:                string(intent.Status),
		ErrorCode:             pgtype.Text{},
		ErrorMessage:          pgtype.Text{},
		Response:              responseBody,
	}); err != nil {
		log.Printf("[ERROR] UpdatePaymentAttempt failed: %v", err)
		return err
	}
	return nil
}

func updatePaymentRequestSuccess(ctx context.Context, queries *bridge.Queries, paymentID string, intent *stripe.PaymentIntent) error {
	parsedUUID := parseStringToUUID(paymentID)
	if err := queries.UpdatePaymentRequestSuccess(ctx, bridge.UpdatePaymentRequestSuccessParams{
		Uuid:                  parsedUUID,
		StripePaymentIntentID: pgtype.Text{String: intent.ID, Valid: true},
		Status:                string(intent.Status),
	}); err != nil {
		log.Printf("[ERROR] UpdatePaymentRequestSuccess failed: %v", err)
		return fmt.Errorf("update payment request success: %w", err)
	}

	log.Printf("[SUCCESS] Payment %s processed and payment_requests updated successfully", paymentID)
	return nil
}

func appendOperationToPayload(operation string, payload []byte) []byte {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	data["operation"] = operation
	updatedPayload, err := json.Marshal(data)
	if err != nil {
		return payload
	}

	return updatedPayload
}
