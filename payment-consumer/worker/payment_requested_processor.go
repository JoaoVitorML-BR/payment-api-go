// payment-consumer\worker\payment_requested_processor.go
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
	"github.com/mercadopago/sdk-go/pkg/requestoptions"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database/bridge"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentRequestedProcessor struct {
	queries *bridge.Queries
	gateway paymentgateway.Gateway
	cfg     *config.Config
}

func NewPaymentRequestedProcessor(queries *bridge.Queries, gateway paymentgateway.Gateway, cfg *config.Config) *PaymentRequestedProcessor {
	return &PaymentRequestedProcessor{
		queries: queries,
		gateway: gateway,
		cfg:     cfg,
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

	currency := strings.ToUpper(strings.TrimSpace(msg.Currency))
	metadata := buildPaymentMetadata(msg)
	ctx = requestoptions.WithIdempotencyKey(ctx, msg.IdempotencyKey)
	input := paymentgateway.CreatePaymentInput{
		AmountCents:     msg.AmountCents,
		Currency:        currency,
		PaymentMethod:   strings.ToLower(strings.TrimSpace(msg.PaymentMethod)),
		IdempotencyKey:  msg.IdempotencyKey,
		Description:     fmt.Sprintf("consultoria online - %s", msg.PaymentID),
		PayerEmail:      customerEmail(msg.Customer),
		PayerName:       customerName(msg.Customer),
		PayerTaxID:      customerTaxID(msg.Customer),
		PayerAddress:    customerAddress(msg.Customer),
		PayerCity:       customerCity(msg.Customer),
		PayerState:      customerState(msg.Customer),
		PayerPostalCode: customerPostalCode(msg.Customer),
		Metadata:        metadata,
		NotificationURL: p.cfg.MercadoPagoWebhookURL,
	}

	paymentResult, err := p.gateway.CreatePayment(ctx, input)
	if err != nil {
		return saveFailedPaymentAttempt(ctx, p.queries, msg.PaymentID, "", currency, "create_payment", err)
	}

	if err := saveSuccessfulPaymentAttempt(ctx, p.queries, msg.PaymentID, paymentResult, currency); err != nil {
		return fmt.Errorf("save successful payment attempt: %w", err)
	}

	return updatePaymentRequestSuccess(ctx, p.queries, msg.PaymentID, paymentResult)
}

func parsePaymentRequestedMessage(body []byte) (paymentRequestedMessage, error) {
	var msg paymentRequestedMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return paymentRequestedMessage{}, err
	}
	return msg, nil
}

func hasSuccessfulAttempt(ctx context.Context, queries *bridge.Queries, paymentID string) (bool, error) {
	existingAttempt, err := queries.GetLatestPaymentAttempt(ctx, parseStringToUUID(paymentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return existingAttempt.GatewayPaymentID.Valid && existingAttempt.Status != "failed", nil
}

func buildPaymentMetadata(msg paymentRequestedMessage) map[string]string {
	metadata := map[string]string{
		"payment_request_uuid": msg.PaymentID,
		"payment_method":       msg.PaymentMethod,
	}
	if msg.Installments != nil {
		metadata["installments"] = strconv.Itoa(*msg.Installments)
	}
	if msg.Customer != nil {
		metadata["customer_email"] = strings.TrimSpace(msg.Customer.Email)
		metadata["customer_tax_id"] = strings.TrimSpace(msg.Customer.TaxID)
	}
	return metadata
}

func saveFailedPaymentAttempt(ctx context.Context, queries *bridge.Queries, paymentID string, gatewayPaymentID string, currency string, operation string, err error) error {
	errorCode, errorMessage, errorPayload := gatewayErrorPayload(err)
	log.Printf("[ERROR] %s failed: %s", operation, errorMessage)

	gatewayPayment := pgtype.Text{}
	if strings.TrimSpace(gatewayPaymentID) != "" {
		gatewayPayment = pgtype.Text{String: gatewayPaymentID, Valid: true}
	}

	saveErr := queries.UpdatePaymentAttempt(ctx, bridge.UpdatePaymentAttemptParams{
		PaymentRequestUuid:    parseStringToUUID(paymentID),
		StripePaymentIntentID: pgtype.Text{},
		StripeClientSecret:    pgtype.Text{},
		Gateway:               "mercado_pago",
		GatewayPaymentID:      gatewayPayment,
		PixQrCode:             pgtype.Text{},
		PixQrCodeBase64:       pgtype.Text{},
		PixExpirationAt:       pgtype.Timestamptz{},
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

func saveSuccessfulPaymentAttempt(ctx context.Context, queries *bridge.Queries, paymentID string, result *paymentgateway.PaymentResult, currency string) error {
	expirationAt := pgtype.Timestamptz{}
	if result.PixExpirationDate != "" {
		parsedExpiration, err := time.Parse(time.RFC3339, result.PixExpirationDate)
		if err == nil {
			expirationAt = pgtype.Timestamptz{Time: parsedExpiration, Valid: true}
		}
	}

	pixQRCode := pgtype.Text{}
	if strings.TrimSpace(result.PixQRCode) != "" {
		pixQRCode = pgtype.Text{String: result.PixQRCode, Valid: true}
	}

	pixQRCodeBase64 := pgtype.Text{}
	if strings.TrimSpace(result.PixQRCodeBase64) != "" {
		pixQRCodeBase64 = pgtype.Text{String: result.PixQRCodeBase64, Valid: true}
	}

	responseBody, _ := json.Marshal(map[string]any{
		"gateway_payment_id":  result.GatewayPaymentID,
		"status":              result.RawStatus,
		"amount_cents":        result.AmountCents,
		"currency":            result.Currency,
		"pix_qr_code":         result.PixQRCode,
		"pix_qr_code_base64":  result.PixQRCodeBase64,
		"pix_expiration_date": result.PixExpirationDate,
	})

	if err := queries.UpdatePaymentAttempt(ctx, bridge.UpdatePaymentAttemptParams{
		PaymentRequestUuid:    parseStringToUUID(paymentID),
		StripePaymentIntentID: pgtype.Text{},
		StripeClientSecret:    pgtype.Text{},
		Gateway:               "mercado_pago",
		GatewayPaymentID:      pgtype.Text{String: result.GatewayPaymentID, Valid: true},
		PixQrCode:             pixQRCode,
		PixQrCodeBase64:       pixQRCodeBase64,
		PixExpirationAt:       expirationAt,
		AttemptNumber:         1,
		Currency:              strings.ToUpper(currency),
		Status:                string(result.Status),
		ErrorCode:             pgtype.Text{},
		ErrorMessage:          pgtype.Text{},
		Response:              responseBody,
	}); err != nil {
		log.Printf("[ERROR] UpdatePaymentAttempt failed: %v", err)
		return err
	}
	return nil
}

func updatePaymentRequestSuccess(ctx context.Context, queries *bridge.Queries, paymentID string, result *paymentgateway.PaymentResult) error {
	parsedUUID := parseStringToUUID(paymentID)
	if err := queries.UpdatePaymentRequestSuccess(ctx, bridge.UpdatePaymentRequestSuccessParams{
		Uuid:             parsedUUID,
		Gateway:          "mercado_pago",
		GatewayPaymentID: pgtype.Text{String: result.GatewayPaymentID, Valid: true},
		Status:           string(result.Status),
	}); err != nil {
		log.Printf("[ERROR] UpdatePaymentRequestSuccess failed: %v", err)
		return fmt.Errorf("update payment request success: %w", err)
	}

	log.Printf("[SUCCESS] Payment %s processed and payment_requests updated successfully", paymentID)
	return nil
}

func customerName(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.Name)
}

func customerEmail(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.Email)
}

func customerTaxID(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.TaxID)
}

func customerAddress(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.Address)
}

func customerCity(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.City)
}

func customerState(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.State)
}

func customerPostalCode(customer *CustomerInfo) string {
	if customer == nil {
		return ""
	}

	return strings.TrimSpace(customer.PostalCode)
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
