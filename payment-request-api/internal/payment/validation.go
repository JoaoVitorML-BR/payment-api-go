// payment-request-api\internal\payment\validation.go
package payment

import (
	"errors"
	"strings"
)

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
	allowedMethods := map[string]bool{"pix": true}
	if !allowedMethods[paymentMethod] {
		return errors.New("payment_method must be pix for now")
	}

	if currency != strings.ToUpper("BRL") {
		return errors.New("currency must be BRL")
	}

	return s.validatePixPayment(req)
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
	if req.Customer == nil {
		return errors.New("customer information is required for pix payments")
	}

	if strings.TrimSpace(req.Customer.Name) == "" || strings.TrimSpace(req.Customer.Email) == "" || strings.TrimSpace(req.Customer.TaxID) == "" {
		return errors.New("customer name, email, and tax_id are required for pix payments")
	}

	if strings.TrimSpace(req.Customer.Address) == "" || strings.TrimSpace(req.Customer.City) == "" || strings.TrimSpace(req.Customer.State) == "" || strings.TrimSpace(req.Customer.PostalCode) == "" {
		return errors.New("customer address, city, state, and postal_code are required for pix payments")
	}

	return nil
}
