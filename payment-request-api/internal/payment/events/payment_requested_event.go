package events

import (
	"time"
)

type PaymentRequestedEvent struct {
	EventName      string    `json:"event_name"`
	PaymentID      int64     `json:"payment_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	AmountCents    int64     `json:"amount_cents"`
	Currency       string    `json:"currency"`
	PaymentMethod  string    `json:"payment_method"`
	Installments   *int      `json:"installments,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

func NewPaymentRequestedEvent(paymentID int64, idempotencyKey string, amountCents int64, currency string, paymentMethod string, installments *int) *PaymentRequestedEvent {
	return &PaymentRequestedEvent{
		EventName:      "payment.requested.v1",
		PaymentID:      paymentID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    amountCents,
		Currency:       currency,
		PaymentMethod:  paymentMethod,
		Installments:   installments,
		OccurredAt:     time.Now().UTC(),
	}
}
