package paymentgateway

import "context"

// Status normalized, independent of the provider.
type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusApproved PaymentStatus = "approved"
	StatusRejected PaymentStatus = "rejected"
	StatusFailed   PaymentStatus = "failed"
)

type CreatePaymentInput struct {
	AmountCents     int64
	Currency        string // always "BRL" for now
	PaymentMethod   string // "pix", "credit", "debit"
	IdempotencyKey  string
	Description     string
	PayerEmail      string
	PayerName       string
	PayerTaxID      string // CPF/CNPJ, required for Pix in Mercado Pago
	PayerAddress    string
	PayerCity       string
	PayerState      string
	PayerPostalCode string
	Metadata        map[string]string
	CardToken       string // used only when PaymentMethod == "credit"/"debit"
	Installments    *int
	NotificationURL string // webhook URL, for the provider to call back
}

type PaymentResult struct {
	GatewayPaymentID string
	Status           PaymentStatus
	RawStatus        string // original provider status, stored for auditing
	AmountCents      int64
	Currency         string
	// Pix-specific fields — empty for other payment methods.
	PixQRCode         string // "copy and paste"
	PixQRCodeBase64   string // QR code image in base64
	PixExpirationDate string // RFC3339

	RawResponse []byte // raw provider payload, useful for debugging/auditing
}

// Gateway is the interface that any payment provider must implement.
type Gateway interface {
	// CreatePayment creates a new charge. For Pix, the payment starts
	// as "pending" and only changes status when the webhook notifies.
	CreatePayment(ctx context.Context, input CreatePaymentInput) (*PaymentResult, error)

	// GetPayment queries the current status of an existing payment.
	// Used for both security polling and when receiving a webhook
	// (never trust the webhook payload alone — always re-query).
	GetPayment(ctx context.Context, gatewayPaymentID string) (*PaymentResult, error)
}
