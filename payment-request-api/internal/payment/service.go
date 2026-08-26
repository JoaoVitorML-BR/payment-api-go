// payment-request-api\internal\payment\service.go
package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment/events"
)

type CreatePaymentResponse struct {
	PaymentUUID   string    `json:"payment_uuid"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PaymentStatusResponse struct {
	Status           string     `json:"status"`
	ClientSecret     string     `json:"client_secret,omitempty"`
	Gateway          string     `json:"gateway,omitempty"`
	GatewayPaymentID string     `json:"gateway_payment_id,omitempty"`
	PixQRCode        string     `json:"pix_qr_code,omitempty"`
	PixQRCodeBase64  string     `json:"pix_qr_code_base64,omitempty"`
	PixExpirationAt  *time.Time `json:"pix_expiration_at,omitempty"`
}

type PaymentRepository interface {
	// ctx is a standard Go context that can be used for cancellation and timeouts.
	// It allows the caller to signal that the operation should be aborted if it takes too long or if the client disconnects.
	CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error)
	GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error)
	UpdatePaymentStatus(ctx context.Context, paymentUUID string, status string, amountCents int64) error
	GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (PaymentGatewayValidationData, error)
	UpdatePaymentStatusByGatewayPaymentID(
		ctx context.Context,
		gatewayPaymentID string,
		status string,
	) (int64, error)
}

type PaymentGatewayValidationData struct {
	PaymentUUID      string
	ExpectedAmount   int64
	ExpectedCurrency string
	CurrentStatus    string
}

type GatewayPaymentDetails struct {
	GatewayPaymentID  string
	ExternalReference string
	Status            string
	AmountCents       int64
	Currency          string
}

type GatewayPaymentReader interface {
	GetPayment(ctx context.Context, gatewayPaymentID string) (*GatewayPaymentDetails, error)
}

var (
	ErrWebhookRetryable    = errors.New("webhook should be retried")
	ErrWebhookInvalidState = errors.New("webhook event rejected due to invalid state")
)

type PaymentService struct {
	repo          PaymentRepository
	publisher     events.PaymentRequestedEventPublisher
	gatewayReader GatewayPaymentReader
}

type RefundRequest struct {
	PaymentID   string `json:"payment_id"`
	AmountCents int64  `json:"amount_cents"` // Partial or full refund in cents
	SplitRule   string `json:"split_rule"`   // Optionally specify '50/50'
}

func (s *PaymentService) ProcessRefund(ctx context.Context, req RefundRequest) error {
	if req.PaymentID == "" {
		return errors.New("missing payment ID")
	}

	if req.AmountCents <= 0 {
		return errors.New("amount_cents must be positive")
	}

	refundCents := req.AmountCents
	if req.SplitRule == "50/50" {
		refundCents = req.AmountCents / 2 // Handle proportional refunds
	}

	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentID, "refunded", refundCents)
	if err != nil {
		return err
	}

	return nil
}

func NewPaymentService(repo PaymentRepository, publisher events.PaymentRequestedEventPublisher, gatewayReader GatewayPaymentReader) (*PaymentService, error) {
	if repo == nil {
		return nil, errors.New("nil repository provided to NewPaymentService")
	}
	if publisher == nil {
		return nil, errors.New("nil publisher provided to NewPaymentService")
	}
	if gatewayReader == nil {
		return nil, errors.New("nil gateway reader provided to NewPaymentService")
	}
	return &PaymentService{repo: repo, publisher: publisher, gatewayReader: gatewayReader}, nil
}

func (s *PaymentService) GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error) {
	fmt.Println("payment UUID received on service.go: ", paymentUUID)
	return s.repo.GetPaymentClientSecret(ctx, paymentUUID)
}

func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, paymentID string, status string, amountCents int64) error {
	return s.repo.UpdatePaymentStatus(ctx, paymentID, status, amountCents)
}

func (s *PaymentService) ProcessMercadoPagoWebhook(ctx context.Context, gatewayPaymentID string) error {
	gatewayPaymentID = strings.TrimSpace(gatewayPaymentID)
	if gatewayPaymentID == "" {
		return errors.New("missing Mercado Pago payment id")
	}

	localPayment, err := s.repo.GetPaymentRequestByGatewayPaymentID(ctx, gatewayPaymentID)
	if err != nil {
		return fmt.Errorf("%w: local payment not linked yet", ErrWebhookRetryable)
	}

	gatewayPayment, err := s.gatewayReader.GetPayment(ctx, gatewayPaymentID)
	if err != nil {
		return fmt.Errorf("%w: failed to fetch payment from Mercado Pago: %v", ErrWebhookRetryable, err)
	}

	if strings.TrimSpace(gatewayPayment.ExternalReference) != strings.TrimSpace(localPayment.PaymentUUID) {
		return fmt.Errorf("%w: external_reference mismatch", ErrWebhookInvalidState)
	}

	if !strings.EqualFold(strings.TrimSpace(gatewayPayment.Currency), strings.TrimSpace(localPayment.ExpectedCurrency)) {
		return fmt.Errorf("%w: currency mismatch", ErrWebhookInvalidState)
	}

	if gatewayPayment.AmountCents != localPayment.ExpectedAmount {
		return fmt.Errorf("%w: amount mismatch", ErrWebhookInvalidState)
	}

	nextStatus, err := normalizeGatewayStatus(gatewayPayment.Status)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookInvalidState, err)
	}

	if !isAllowedStatusTransition(localPayment.CurrentStatus, nextStatus) {
		return nil
	}

	rowsAffected, err := s.repo.UpdatePaymentStatusByGatewayPaymentID(
		ctx,
		gatewayPaymentID,
		nextStatus,
	)

	if err != nil {
		return fmt.Errorf(
			"%w: failed to update payment status: %v",
			ErrWebhookRetryable,
			err,
		)
	}

	if rowsAffected == 0 {
		log.Printf(
			"[INFO] Payment %s was not updated; likely already in terminal state",
			gatewayPaymentID,
		)
		return nil
	}

	log.Printf(
		"[INFO] Payment %s status updated to %s",
		gatewayPaymentID,
		nextStatus,
	)

	return nil
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

func normalizeCreatePaymentRequest(req CreatePaymentRequest) CreatePaymentRequest {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.PaymentMethod = strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	return req
}

func (s *PaymentService) publishPaymentRequestedEvent(req CreatePaymentRequest, resp CreatePaymentResponse) error {
	if s.publisher == nil {
		return nil
	}

	var customer *events.CustomerInfo
	if req.Customer != nil {
		customer = &events.CustomerInfo{
			Name:       req.Customer.Name,
			Email:      req.Customer.Email,
			Phone:      req.Customer.Phone,
			TaxID:      req.Customer.TaxID,
			Address:    req.Customer.Address,
			City:       req.Customer.City,
			State:      req.Customer.State,
			PostalCode: req.Customer.PostalCode,
		}
	}

	event := events.NewPaymentRequestedEvent(
		resp.PaymentUUID,
		req.IdempotencyKey,
		req.AmountCents,
		req.Currency,
		req.PaymentMethod,
		customer,
		req.Installments,
	)

	return s.publisher.Publish(event)
}
