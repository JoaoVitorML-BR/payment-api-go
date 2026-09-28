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
	Created       bool      `json:"-"`
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

type ReconciliationItem struct {
	GatewayPaymentID string
	PaymentUUID      string
}

type PaymentRepository interface {
	// ctx is a standard Go context that can be used for cancellation and timeouts.
	// It allows the caller to signal that the operation should be aborted if it takes too long or if the client disconnects.
	CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error)
	GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error)
	UpdatePaymentStatus(ctx context.Context, paymentUUID string, status string, amountCents int64) (int64, error)
	GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (PaymentGatewayValidationData, error)
	UpdatePaymentStatusByGatewayPaymentID(
		ctx context.Context,
		gatewayPaymentID string,
		status string,
	) (int64, error)
	GetPendingPaymentsForReconciliation(ctx context.Context, maxUpdatedAt time.Time, limit int32) ([]ReconciliationItem, error)
}

type RefundRepository interface {
	GetRefundPaymentInfo(context.Context, string) (RefundPaymentInfo, error)
	ReserveRefund(context.Context, string, string, int64, string) (RefundRecord, error)
	MarkRefundSucceeded(context.Context, string, string) error
	MarkRefundFailed(context.Context, string, string, string) error
	ListProcessingRefunds(context.Context, int32) ([]RefundRecord, error)
}

type RefundPaymentInfo struct {
	PaymentUUID      string
	GatewayPaymentID string
	AmountCents      int64
	Status           string
	SellerID         string
}
type RefundRecord struct {
	PaymentID       string
	IdempotencyKey  string
	AmountCents     int64
	Status          string
	GatewayRefundID string
}
type RefundGateway interface {
	Refund(context.Context, string, string, int64) (string, error)
	FindRefund(context.Context, string, string, int64) (string, bool, error)
}

type PaymentGatewayValidationData struct {
	PaymentUUID      string
	ExpectedAmount   int64
	ExpectedCurrency string
	CurrentStatus    string
	SellerID         string
}

type GatewayPaymentDetails struct {
	GatewayPaymentID  string
	ExternalReference string
	Status            string
	AmountCents       int64
	Currency          string
}

type GatewayPaymentReader interface {
	GetPayment(ctx context.Context, gatewayPaymentID string, sellerID string) (*GatewayPaymentDetails, error)
}

var (
	ErrWebhookRetryable    = errors.New("webhook should be retried")
	ErrWebhookInvalidState = errors.New("webhook event rejected due to invalid state")
)

type PaymentService struct {
	repo          PaymentRepository
	publisher     events.PaymentRequestedEventPublisher
	gatewayReader GatewayPaymentReader
	refundRepo    RefundRepository
	refundGateway RefundGateway
}

type RefundRequest struct {
	PaymentID      string `json:"payment_id"`
	AmountCents    int64  `json:"amount_cents"` // Partial or full refund in cents
	SplitRule      string `json:"split_rule"`   // Optionally specify '50/50'
	IdempotencyKey string `json:"idempotency_key"`
	Reason         string `json:"reason"`
}

func (s *PaymentService) SetRefundDependencies(repo RefundRepository, gateway RefundGateway) {
	s.refundRepo = repo
	s.refundGateway = gateway
}

func (s *PaymentService) ProcessRefund(ctx context.Context, req RefundRequest) error {
	if strings.TrimSpace(req.PaymentID) == "" {
		return errors.New("missing payment ID")
	}

	if req.AmountCents <= 0 {
		return errors.New("amount_cents must be positive")
	}
	if strings.TrimSpace(req.SplitRule) != "" {
		return errors.New("split_rule is not supported; refund allocation must come from the consulting service")
	}

	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}
	if s.refundRepo == nil || s.refundGateway == nil {
		return errors.New("refund flow is not configured")
	}
	info, err := s.refundRepo.GetRefundPaymentInfo(ctx, req.PaymentID)
	if err != nil {
		return fmt.Errorf("get payment for refund: %w", err)
	}
	if info.Status != "succeeded" && info.Status != "partially_refunded" {
		return fmt.Errorf("payment status %q cannot be refunded", info.Status)
	}
	if req.AmountCents > info.AmountCents {
		return errors.New("refund amount exceeds original payment")
	}
	targetUUID := info.PaymentUUID
	if targetUUID == "" {
		targetUUID = req.PaymentID
	}
	reservation, err := s.refundRepo.ReserveRefund(ctx, targetUUID, req.IdempotencyKey, req.AmountCents, req.Reason)
	if err != nil {
		return fmt.Errorf("reserve refund: %w", err)
	}
	if reservation.Status == "succeeded" {
		return nil
	}
	gatewayRefundID, err := s.refundGateway.Refund(ctx, info.GatewayPaymentID, info.SellerID, req.AmountCents)
	if err != nil {
		_ = s.refundRepo.MarkRefundFailed(ctx, req.IdempotencyKey, "gateway_refund_failed", err.Error())
		return fmt.Errorf("execute gateway refund: %w", err)
	}
	if err := s.refundRepo.MarkRefundSucceeded(ctx, req.IdempotencyKey, gatewayRefundID); err != nil {
		return fmt.Errorf("persist refund success: %w", err)
	}
	return nil
}

func (s *PaymentService) ReconcileProcessingRefunds(ctx context.Context, limit int32) (int, error) {
	if s.refundRepo == nil || s.refundGateway == nil {
		return 0, errors.New("refund flow is not configured")
	}
	items, err := s.refundRepo.ListProcessingRefunds(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("list processing refunds: %w", err)
	}
	reconciled := 0
	for _, item := range items {
		info, err := s.refundRepo.GetRefundPaymentInfo(ctx, item.PaymentID)
		if err != nil {
			continue
		}
		gatewayID, found, err := s.refundGateway.FindRefund(ctx, info.GatewayPaymentID, info.SellerID, item.AmountCents)
		if err != nil || !found {
			continue
		}
		if err := s.refundRepo.MarkRefundSucceeded(ctx, item.IdempotencyKey, gatewayID); err != nil {
			return reconciled, fmt.Errorf("mark reconciled refund: %w", err)
		}
		reconciled++
	}
	return reconciled, nil
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
	rowsAffected, err := s.repo.UpdatePaymentStatus(ctx, paymentID, status, amountCents)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		log.Printf("[INFO] Payment %s status was not updated; likely already in terminal state", paymentID)
	}
	return nil
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

	gatewayPayment, err := s.gatewayReader.GetPayment(ctx, gatewayPaymentID, localPayment.SellerID)
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

// ReconcilePendingPayments scans payment requests stuck in 'pending' status older than minAge
// and re-queries Mercado Pago authoritatively to synchronize local database state.
func (s *PaymentService) ReconcilePendingPayments(ctx context.Context, minAge time.Duration, limit int32) (int, error) {
	maxUpdatedAt := time.Now().Add(-minAge)
	items, err := s.repo.GetPendingPaymentsForReconciliation(ctx, maxUpdatedAt, limit)
	if err != nil {
		return 0, fmt.Errorf("fetch pending payments for reconciliation: %w", err)
	}

	if len(items) == 0 {
		return 0, nil
	}

	reconciledCount := 0
	for _, item := range items {
		if strings.TrimSpace(item.GatewayPaymentID) == "" {
			continue
		}

		log.Printf("[RECONCILIATION] Reconciling payment_uuid=%s, gateway_payment_id=%s", item.PaymentUUID, item.GatewayPaymentID)
		err := s.ProcessMercadoPagoWebhook(ctx, item.GatewayPaymentID)
		if err != nil {
			if errors.Is(err, ErrWebhookRetryable) {
				log.Printf("[RECONCILIATION] Retryable error reconciling payment %s: %v", item.PaymentUUID, err)
			} else {
				log.Printf("[RECONCILIATION] Permanent error reconciling payment %s: %v", item.PaymentUUID, err)
			}
			continue
		}

		reconciledCount++
	}

	return reconciledCount, nil
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
	if resp.Created {
		if err := s.publishPaymentRequestedEvent(req, resp); err != nil {
			return CreatePaymentResponse{}, err
		}
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
		req.SellerID,
		req.MarketplaceFeeCents,
	)

	return s.publisher.Publish(event)
}
