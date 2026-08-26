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
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
)

const (
	defaultBaseURL = "https://api.mercadopago.com"
	getPaymentPath = "/v1/payments/%s"
	httpTimeout    = 10 * time.Second
)

// GatewayReader queries Mercado Pago for the authoritative state of a payment.
type GatewayReader struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewGatewayReader(accessToken string) *GatewayReader {
	return &GatewayReader{
		baseURL: defaultBaseURL,
		token:   accessToken,
		http:    &http.Client{Timeout: httpTimeout},
	}
}

// rawPayment models only the fields we need from GET /v1/payments/{id}.
type rawPayment struct {
	ID                json.Number `json:"id"`
	Status            string      `json:"status"`
	StatusDetail      string      `json:"status_detail"`
	ExternalReference string      `json:"external_reference"`
	TransactionAmount float64     `json:"transaction_amount"`
	CurrencyID        string      `json:"currency_id"`
}

// GetPayment fetches the payment by id and converts it to normalized details.
// Returns *payment.GatewayPaymentDetails to satisfy the payment.GatewayPaymentReader interface.
func (g *GatewayReader) GetPayment(ctx context.Context, gatewayPaymentID string) (*payment.GatewayPaymentDetails, error) {
	id := strings.TrimSpace(gatewayPaymentID)
	if id == "" {
		return nil, fmt.Errorf("mercado pago: empty payment id")
	}
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return nil, fmt.Errorf("mercado pago: invalid payment id %q", id)
	}

	url := g.baseURL + fmt.Sprintf(getPaymentPath, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mercado pago: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mercado pago: get payment %s: %w", id, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("mercado pago: payment %s not found (404)", id)
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("mercado pago: upstream error (status %d)", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("mercado pago: unexpected status %d", resp.StatusCode)
	}

	var raw rawPayment
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("mercado pago: decode response: %w", err)
	}

	amountCents, err := floatToCents(raw.TransactionAmount)
	if err != nil {
		return nil, fmt.Errorf("mercado pago: invalid transaction_amount for payment %s: %w", id, err)
	}

	return &payment.GatewayPaymentDetails{
		GatewayPaymentID:  raw.ID.String(),
		ExternalReference: strings.TrimSpace(raw.ExternalReference),
		Status:            strings.TrimSpace(raw.Status),
		AmountCents:       amountCents,
		Currency:          strings.TrimSpace(raw.CurrencyID),
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
