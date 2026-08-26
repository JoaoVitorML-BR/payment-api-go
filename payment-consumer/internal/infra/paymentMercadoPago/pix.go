package paymentmercadopago

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
	"github.com/mercadopago/sdk-go/pkg/payment"
	"github.com/mercadopago/sdk-go/pkg/requestoptions"
)

func (c *Client) CreatePayment(ctx context.Context, input paymentgateway.CreatePaymentInput) (*paymentgateway.PaymentResult, error) {
	if input.PaymentMethod != "pix" {
		return nil, fmt.Errorf("paymentmercadopago: only 'pix' is implemented so far, got %q", input.PaymentMethod)
	}
	if strings.TrimSpace(input.PayerEmail) == "" || strings.TrimSpace(input.PayerName) == "" || strings.TrimSpace(input.PayerTaxID) == "" {
		return nil, fmt.Errorf("paymentmercadopago: payer email, name, and tax id are required for pix payments")
	}

	ctx = requestoptions.WithIdempotencyKey(ctx, input.IdempotencyKey)

	firstName, lastName := splitFullName(input.PayerName)
	identificationType := identificationTypeForTaxID(input.PayerTaxID)
	if identificationType == "" {
		return nil, fmt.Errorf("paymentmercadopago: payer tax id must be a valid CPF or CNPJ for pix payments")
	}

	client := payment.NewClient(c.cfg)

	amount := float64(input.AmountCents) / 100

	request := payment.Request{
		TransactionAmount: amount,
		Description:       input.Description,
		PaymentMethodID:   "pix",
		NotificationURL:   input.NotificationURL,
		ExternalReference: input.Metadata["payment_request_uuid"],
		Metadata:          map[string]any{},
		Payer: &payment.PayerRequest{
			Email:     strings.TrimSpace(input.PayerEmail),
			FirstName: firstName,
			LastName:  lastName,
			Identification: &payment.IdentificationRequest{
				Type:   identificationType,
				Number: strings.TrimSpace(input.PayerTaxID),
			},
			Address: &payment.AddressRequest{
				City:         strings.TrimSpace(input.PayerCity),
				FederalUnit:  strings.TrimSpace(input.PayerState),
				ZipCode:      strings.TrimSpace(input.PayerPostalCode),
				StreetName:   strings.TrimSpace(input.PayerAddress),
			},
		},
	}

	result, err := client.Create(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create pix payment: %w", err)
	}

	return toPaymentResult(result), nil
}

func (c *Client) GetPayment(ctx context.Context, gatewayPaymentID string) (*paymentgateway.PaymentResult, error) {
	client := payment.NewClient(c.cfg)

	id, err := strconv.Atoi(gatewayPaymentID)
	if err != nil {
		return nil, fmt.Errorf("invalid mercado pago payment id %q: %w", gatewayPaymentID, err)
	}

	result, err := client.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get payment %s: %w", gatewayPaymentID, err)
	}

	return toPaymentResult(result), nil
}

func toPaymentResult(p *payment.Response) *paymentgateway.PaymentResult {
	raw, _ := json.Marshal(p)
	qrCode := ""
	qrCodeBase64 := ""
	if p.PointOfInteraction.TransactionData.QRCode != "" {
		qrCode = p.PointOfInteraction.TransactionData.QRCode
	}
	if p.PointOfInteraction.TransactionData.QRCodeBase64 != "" {
		qrCodeBase64 = p.PointOfInteraction.TransactionData.QRCodeBase64
	}

	pixExpirationDate := ""
	if !p.DateOfExpiration.IsZero() {
		pixExpirationDate = p.DateOfExpiration.UTC().Format(time.RFC3339)
	}

	res := &paymentgateway.PaymentResult{
		GatewayPaymentID: strconv.Itoa(p.ID),
		RawStatus:        p.Status,
		Status:           normalizeStatus(p.Status),
		AmountCents:      int64(p.TransactionAmount * 100),
		Currency:         p.CurrencyID,
		PixQRCode:        qrCode,
		PixQRCodeBase64:  qrCodeBase64,
		PixExpirationDate: pixExpirationDate,
		RawResponse:      raw,
	}

	return res
}

func splitFullName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], parts[0]
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func identificationTypeForTaxID(taxID string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, taxID)

	switch len(digits) {
	case 11:
		return "CPF"
	case 14:
		return "CNPJ"
	default:
		return ""
	}
}

func normalizeStatus(mpStatus string) paymentgateway.PaymentStatus {
	switch mpStatus {
	case "approved":
		return paymentgateway.StatusApproved
	case "rejected", "cancelled":
		return paymentgateway.StatusRejected
	case "pending", "in_process", "in_mediation":
		return paymentgateway.StatusPending
	default:
		return paymentgateway.StatusFailed
	}
}
