// payment-request-api\internal\payment\events\payment_requested_event.go
package events

import (
	"time"
)

type PaymentRequestedEvent struct {
	EventName      string        `json:"event_name"`
	PaymentID      string        `json:"payment_id"`
	IdempotencyKey string        `json:"idempotency_key"`
	AmountCents    int64         `json:"amount_cents"`
	Currency       string        `json:"currency"`
	PaymentMethod  string        `json:"payment_method"`
	Customer       *CustomerInfo `json:"customer,omitempty"`
	Installments   *int          `json:"installments,omitempty"`
	OccurredAt     time.Time     `json:"occurred_at"`
}

type CustomerInfo struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	TaxID      string `json:"tax_id"`
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

func NewPaymentRequestedEvent(paymentUUID string, idempotencyKey string, amountCents int64, currency string, paymentMethod string, customer *CustomerInfo, installments *int) *PaymentRequestedEvent {
	return &PaymentRequestedEvent{
		EventName:      "payment.requested.v1",
		PaymentID:      paymentUUID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    amountCents,
		Currency:       currency,
		PaymentMethod:  paymentMethod,
		Customer:       customer,
		Installments:   installments,
		OccurredAt:     time.Now().UTC(),
	}
}
