// Service / Use Case
package payment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment/events"
)

type CreatePaymentResponse struct {
	ID            int64     `json:"id"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PaymentRepository interface {
	// ctx is a standard Go context that can be used for cancellation and timeouts.
	// It allows the caller to signal that the operation should be aborted if it takes too long or if the client disconnects.
	CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error)
}

type PaymentService struct {
	repo      PaymentRepository
	publisher events.PaymentRequestedEventPublisher
}

func NewPaymentService(repo PaymentRepository, publisher events.PaymentRequestedEventPublisher) (*PaymentService, error) {
	if repo == nil {
		return nil, errors.New("nil repository provided to NewPaymentService")
	}
	if publisher == nil {
		return nil, errors.New("nil publisher provided to NewPaymentService")
	}
	return &PaymentService{repo: repo, publisher: publisher}, nil
}

func (s *PaymentService) CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	if err := s.validateCreatePaymentRequest(req); err != nil {
		return CreatePaymentResponse{}, err
	}

	req = normalizeCreatePaymentRequest(req)

	// Call repository to create payment request
	resp, err := s.repo.CreatePaymentRequest(ctx, req)
	if err != nil {
		return CreatePaymentResponse{}, err
	}

	// Publish the payment requested event
	if err := s.publishPaymentRequestedEvent(req, resp); err != nil {
		return CreatePaymentResponse{}, err
	}

	return resp, nil
}

func (s *PaymentService) validateCreatePaymentRequest(req CreatePaymentRequest) error {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}

	if req.AmountCents <= 0 {
		return errors.New("amount_cents must be greater than zero")
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(currency) != 3 {
		return errors.New("currency must have 3 characters")
	}

	paymentMethod := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	allowedMethods := map[string]bool{"credit": true, "debit": true, "pix": true, "boleto": true}
	if !allowedMethods[paymentMethod] {
		return errors.New("payment_method must be one of: credit, debit, pix, boleto")
	}

	switch paymentMethod {
	case "credit":
		return s.validateCreditPayment(req)
	case "pix":
		return s.validatePixPayment(req)
	case "boleto":
		return s.validateBoletoPayment(req)
	case "debit":
		return s.validateDebitPayment(req)
	default:
		return nil
	}
}

func normalizeCreatePaymentRequest(req CreatePaymentRequest) CreatePaymentRequest {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.PaymentMethod = strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	return req
}

func (s *PaymentService) publishPaymentRequestedEvent(req CreatePaymentRequest, resp CreatePaymentResponse) error {
	if s.publisher == nil {
		return nil
	}

	event := events.NewPaymentRequestedEvent(
		resp.ID,
		req.IdempotencyKey,
		req.AmountCents,
		req.Currency,
		req.PaymentMethod,
		req.Installments,
	)

	return s.publisher.Publish(event)
}

func (s *PaymentService) validateCreditPayment(req CreatePaymentRequest) error {
	if req.Installments != nil {
		if *req.Installments < 1 || *req.Installments > 12 {
			return errors.New("installments must be between 1 and 12")
		}
	}
	return nil
}

func (s *PaymentService) validatePixPayment(req CreatePaymentRequest) error {
	// Future
	return nil
}

func (s *PaymentService) validateBoletoPayment(req CreatePaymentRequest) error {
	// Future
	return nil
}

func (s *PaymentService) validateDebitPayment(req CreatePaymentRequest) error {
	// Future
	return nil
}
