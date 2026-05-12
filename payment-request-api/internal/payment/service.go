// Service / Use Case
package payment

import (
	"context"
	"errors"
	"strings"
	"time"
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
	repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return CreatePaymentResponse{}, errors.New("idempotency_key is required")
	}

	if req.AmountCents <= 0 {
		return CreatePaymentResponse{}, errors.New("amount_cents must be greater than zero")
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(currency) != 3 {
		return CreatePaymentResponse{}, errors.New("currency must have 3 characters")
	}

	paymentMethod := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	allowedMethods := map[string]bool{"credit": true, "debit": true, "pix": true, "boleto": true}
	if !allowedMethods[paymentMethod] {
		return CreatePaymentResponse{}, errors.New("payment_method must be one of: credit, debit, pix, boleto")
	}

	req.Currency = currency
	req.PaymentMethod = paymentMethod

	// Validate specific rules based on payment method
	switch paymentMethod {
	case "credit":
		if err := s.validateCreditPayment(req); err != nil {
			return CreatePaymentResponse{}, err
		}
	case "pix":
		if err := s.validatePixPayment(req); err != nil {
			return CreatePaymentResponse{}, err
		}
	case "boleto":
		if err := s.validateBoletoPayment(req); err != nil {
			return CreatePaymentResponse{}, err
		}
	case "debit":
		if err := s.validateDebitPayment(req); err != nil {
			return CreatePaymentResponse{}, err
		}
	}

	return s.repo.CreatePaymentRequest(ctx, req)
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
