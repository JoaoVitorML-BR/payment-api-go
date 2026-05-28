package events

import (
	"time"
)

type PaymentRequestedEvent struct {
	EventName      string    `json:"event_name"`
	PaymentID      string `json:"payment_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	AmountCents    int64     `json:"amount_cents"`
	Currency       string    `json:"currency"`
	PaymentMethod  string    `json:"payment_method"`
	StripePaymentMethodID string `json:"stripe_payment_method_id,omitempty"`
	Installments   *int      `json:"installments,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

func NewPaymentRequestedEvent(paymentUUID string, idempotencyKey string, amountCents int64, currency string, paymentMethod string, stripePaymentMethodID string, installments *int) *PaymentRequestedEvent {
	return &PaymentRequestedEvent{
		EventName:      "payment.requested.v1",
		PaymentID:      paymentUUID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    amountCents,
		Currency:       currency,
		PaymentMethod:  paymentMethod,
		StripePaymentMethodID: stripePaymentMethodID,
		Installments:   installments,
		OccurredAt:     time.Now().UTC(),
	}
}
