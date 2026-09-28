// payment-request-api\internal\infra\paymentmercadopago\gateway_reader.go
//
// Package paymentmercadopago provides a read-only client used by the
// payment-request-api to re-query payments directly from the Mercado Pago API
// when a webhook notification arrives.
//
// SECURITY: this is the "trust boundary" of the webhook flow. The webhook
// payload only carries an id; the financial truth (status, amount, currency,
// external_reference) always comes from THIS query, never from the payload.
package paymentmercadopago

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/order"
	mppayment "github.com/mercadopago/sdk-go/pkg/payment"
)

// GatewayReader queries Mercado Pago for the authoritative state of a payment.
type GatewayReader struct {
	token string
	store *EncryptedFileTokenStore
}

func NewGatewayReader(accessToken string, store *EncryptedFileTokenStore) *GatewayReader {
	return &GatewayReader{
		token: accessToken,
		store: store,
	}
}

// GetPayment fetches the payment by id and converts it to normalized details.
// Returns *payment.GatewayPaymentDetails to satisfy the payment.GatewayPaymentReader interface.
func (g *GatewayReader) GetPayment(ctx context.Context, gatewayPaymentID string, sellerID string) (*payment.GatewayPaymentDetails, error) {
	id := strings.TrimSpace(gatewayPaymentID)
	if id == "" {
		return nil, fmt.Errorf("mercado pago: empty payment id")
	}
	accessToken := g.token
	if strings.TrimSpace(sellerID) != "" {
		if g.store == nil {
			return nil, fmt.Errorf("mercado pago: seller token store is not configured")
		}
		token, err := g.store.Load(sellerID)
		if err != nil {
			return nil, fmt.Errorf("mercado pago: load seller token: %w", err)
		}
		accessToken = token.AccessToken
	}
	sdkConfig, err := config.New(accessToken)
	if err != nil {
		return nil, fmt.Errorf("mercado pago: configure Orders reader: %w", err)
	}

	// Support legacy Payments API IDs (all digits) as well as new Orders API IDs
	if numericID, err := strconv.ParseInt(id, 10, 64); err == nil && numericID > 0 {
		payClient := mppayment.NewClient(sdkConfig)
		payResult, payErr := payClient.Get(ctx, int(numericID))
		if payErr != nil {
			return nil, fmt.Errorf("mercado pago: get legacy payment %s: %w", id, payErr)
		}
		if payResult == nil {
			return nil, fmt.Errorf("mercado pago: legacy payment %s not found", id)
		}
		cents, _ := floatToCents(payResult.TransactionAmount)
		return &payment.GatewayPaymentDetails{
			GatewayPaymentID:  strconv.Itoa(payResult.ID),
			ExternalReference: strings.TrimSpace(payResult.ExternalReference),
			Status:            strings.TrimSpace(payResult.Status),
			AmountCents:       cents,
			Currency:          strings.TrimSpace(payResult.CurrencyID),
		}, nil
	}

	result, err := order.NewClient(sdkConfig).Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("mercado pago: get order %s: %w", id, err)
	}
	if result == nil || len(result.Transactions.Payments) == 0 {
		return nil, fmt.Errorf("mercado pago: order %s has no payment transaction", id)
	}
	transaction := result.Transactions.Payments[0]
	amount, err := strconv.ParseFloat(transaction.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("mercado pago: invalid order amount: %w", err)
	}

	status := transaction.Status
	if strings.TrimSpace(status) == "" {
		status = result.Status
	}

	return &payment.GatewayPaymentDetails{
		GatewayPaymentID:  result.ID,
		ExternalReference: strings.TrimSpace(result.ExternalReference),
		Status:            strings.TrimSpace(status),
		AmountCents:       int64(math.Round(amount * 100)),
		Currency:          strings.TrimSpace(result.Currency),
	}, nil
}

// floatToCents converts the decimal amount returned by Mercado Pago into
// integer cents using round-half-up to avoid float truncation errors
// (e.g. 149.99999999997 must become 15000, not 14999).
func floatToCents(v float64) (int64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("amount is not finite")
	}
	cents := math.Round(v * 100)
	if cents > math.MaxInt64 || cents < 0 {
		return 0, fmt.Errorf("amount out of range")
	}
	return int64(cents), nil
}
