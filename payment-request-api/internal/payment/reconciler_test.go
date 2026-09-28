// payment-request-api/internal/payment/reconciler_test.go
package payment

import (
	"context"
	"testing"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment/events"
)

type mockRepoForReconciliation struct {
	pendingItems    []ReconciliationItem
	validationData  map[string]PaymentGatewayValidationData
	updatedStatuses map[string]string
}

func (m *mockRepoForReconciliation) CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	return CreatePaymentResponse{}, nil
}

func (m *mockRepoForReconciliation) GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error) {
	return PaymentStatusResponse{}, nil
}

func (m *mockRepoForReconciliation) UpdatePaymentStatus(ctx context.Context, paymentUUID string, status string, amountCents int64) (int64, error) {
	return 1, nil
}

func (m *mockRepoForReconciliation) GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (PaymentGatewayValidationData, error) {
	if data, ok := m.validationData[gatewayPaymentID]; ok {
		return data, nil
	}
	return PaymentGatewayValidationData{}, ErrWebhookRetryable
}

func (m *mockRepoForReconciliation) UpdatePaymentStatusByGatewayPaymentID(ctx context.Context, gatewayPaymentID string, status string) (int64, error) {
	m.updatedStatuses[gatewayPaymentID] = status
	return 1, nil
}

func (m *mockRepoForReconciliation) GetPendingPaymentsForReconciliation(ctx context.Context, maxUpdatedAt time.Time, limit int32) ([]ReconciliationItem, error) {
	return m.pendingItems, nil
}

type mockGatewayReaderForReconciliation struct {
	payments map[string]*GatewayPaymentDetails
}

func (m *mockGatewayReaderForReconciliation) GetPayment(ctx context.Context, gatewayPaymentID string, sellerID string) (*GatewayPaymentDetails, error) {
	if p, ok := m.payments[gatewayPaymentID]; ok {
		return p, nil
	}
	return nil, ErrWebhookRetryable
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(event *events.PaymentRequestedEvent) error {
	return nil
}

func TestReconcilePendingPayments_Approved(t *testing.T) {
	repo := &mockRepoForReconciliation{
		pendingItems: []ReconciliationItem{
			{GatewayPaymentID: "gw-100", PaymentUUID: "uuid-100"},
		},
		validationData: map[string]PaymentGatewayValidationData{
			"gw-100": {
				PaymentUUID:      "uuid-100",
				ExpectedAmount:   2500,
				ExpectedCurrency: "BRL",
				CurrentStatus:    "pending",
			},
		},
		updatedStatuses: make(map[string]string),
	}

	reader := &mockGatewayReaderForReconciliation{
		payments: map[string]*GatewayPaymentDetails{
			"gw-100": {
				GatewayPaymentID:  "gw-100",
				Status:            "approved",
				ExternalReference: "uuid-100",
				AmountCents:       2500,
				Currency:          "BRL",
			},
		},
	}

	service, err := NewPaymentService(repo, &mockPublisher{}, reader)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	reconciled, err := service.ReconcilePendingPayments(context.Background(), 1*time.Minute, 10)
	if err != nil {
		t.Fatalf("unexpected error during reconciliation: %v", err)
	}

	if reconciled != 1 {
		t.Errorf("expected 1 reconciled payment, got %d", reconciled)
	}

	if status, ok := repo.updatedStatuses["gw-100"]; !ok || status != "succeeded" {
		t.Errorf("expected status 'succeeded' for gw-100, got '%s'", status)
	}
}

func TestReconcilePendingPayments_Cancelled(t *testing.T) {
	repo := &mockRepoForReconciliation{
		pendingItems: []ReconciliationItem{
			{GatewayPaymentID: "gw-200", PaymentUUID: "uuid-200"},
		},
		validationData: map[string]PaymentGatewayValidationData{
			"gw-200": {
				PaymentUUID:      "uuid-200",
				ExpectedAmount:   5000,
				ExpectedCurrency: "BRL",
				CurrentStatus:    "pending",
			},
		},
		updatedStatuses: make(map[string]string),
	}

	reader := &mockGatewayReaderForReconciliation{
		payments: map[string]*GatewayPaymentDetails{
			"gw-200": {
				GatewayPaymentID:  "gw-200",
				Status:            "cancelled",
				ExternalReference: "uuid-200",
				AmountCents:       5000,
				Currency:          "BRL",
			},
		},
	}

	service, err := NewPaymentService(repo, &mockPublisher{}, reader)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	reconciled, err := service.ReconcilePendingPayments(context.Background(), 1*time.Minute, 10)
	if err != nil {
		t.Fatalf("unexpected error during reconciliation: %v", err)
	}

	if reconciled != 1 {
		t.Errorf("expected 1 reconciled payment, got %d", reconciled)
	}

	if status, ok := repo.updatedStatuses["gw-200"]; !ok || status != "canceled" {
		t.Errorf("expected status 'canceled' for gw-200, got '%s'", status)
	}
}

func TestReconcilerWorker_StartStop(t *testing.T) {
	repo := &mockRepoForReconciliation{
		pendingItems:    []ReconciliationItem{},
		validationData:  make(map[string]PaymentGatewayValidationData),
		updatedStatuses: make(map[string]string),
	}
	reader := &mockGatewayReaderForReconciliation{payments: make(map[string]*GatewayPaymentDetails)}
	service, _ := NewPaymentService(repo, &mockPublisher{}, reader)

	worker := NewReconcilerWorker(service, 10*time.Millisecond, 1*time.Minute, 10)
	ctx, cancel := context.WithCancel(context.Background())

	worker.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	worker.Stop()
}
