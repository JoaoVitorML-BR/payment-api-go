// payment-request-api/internal/payment/webhook_concurrency_test.go
package payment

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockConcurrentRepo struct {
	mu              sync.Mutex
	status          string
	updateCallCount int32
}

func (m *mockConcurrentRepo) CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	return CreatePaymentResponse{}, nil
}

func (m *mockConcurrentRepo) GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error) {
	return PaymentStatusResponse{}, nil
}

func (m *mockConcurrentRepo) UpdatePaymentStatus(ctx context.Context, paymentUUID string, status string, amountCents int64) (int64, error) {
	return 1, nil
}

func (m *mockConcurrentRepo) GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (PaymentGatewayValidationData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return PaymentGatewayValidationData{
		PaymentUUID:      "uuid-concurrent-1",
		ExpectedAmount:   1500,
		ExpectedCurrency: "BRL",
		CurrentStatus:    m.status,
	}, nil
}

func (m *mockConcurrentRepo) UpdatePaymentStatusByGatewayPaymentID(ctx context.Context, gatewayPaymentID string, newStatus string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate SQL optimistic locking: WHERE gateway_payment_id = $2 AND status NOT IN ('succeeded', 'failed', 'canceled')
	if m.status == "succeeded" || m.status == "failed" || m.status == "canceled" {
		return 0, nil
	}

	m.status = newStatus
	atomic.AddInt32(&m.updateCallCount, 1)
	return 1, nil
}

func (m *mockConcurrentRepo) GetPendingPaymentsForReconciliation(ctx context.Context, maxUpdatedAt time.Time, limit int32) ([]ReconciliationItem, error) {
	return nil, nil
}

type mockGatewayReaderSingle struct {
	payment *GatewayPaymentDetails
}

func (m *mockGatewayReaderSingle) GetPayment(ctx context.Context, gatewayPaymentID string, sellerID string) (*GatewayPaymentDetails, error) {
	return m.payment, nil
}

func TestProcessMercadoPagoWebhook_ConcurrentExecutions(t *testing.T) {
	repo := &mockConcurrentRepo{
		status: "pending",
	}

	reader := &mockGatewayReaderSingle{
		payment: &GatewayPaymentDetails{
			GatewayPaymentID:  "gw-concurrent-1",
			Status:            "approved",
			ExternalReference: "uuid-concurrent-1",
			AmountCents:       1500,
			Currency:          "BRL",
		},
	}

	publisher := &mockPublisher{}
	service, err := NewPaymentService(repo, publisher, reader)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := service.ProcessMercadoPagoWebhook(context.Background(), "gw-concurrent-1")
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error in concurrent webhook processing: %v", err)
	}

	if repo.updateCallCount != 1 {
		t.Errorf("expected DB update side-effect to execute EXACTLY ONCE, but executed %d times", repo.updateCallCount)
	}

	if repo.status != "succeeded" {
		t.Errorf("expected final status 'succeeded', got '%s'", repo.status)
	}
}

func TestProcessMercadoPagoWebhook_IdempotentAlreadyTerminal(t *testing.T) {
	repo := &mockConcurrentRepo{
		status: "succeeded",
	}

	reader := &mockGatewayReaderSingle{
		payment: &GatewayPaymentDetails{
			GatewayPaymentID:  "gw-concurrent-1",
			Status:            "approved",
			ExternalReference: "uuid-concurrent-1",
			AmountCents:       1500,
			Currency:          "BRL",
		},
	}

	service, err := NewPaymentService(repo, &mockPublisher{}, reader)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	err = service.ProcessMercadoPagoWebhook(context.Background(), "gw-concurrent-1")
	if err != nil {
		t.Fatalf("expected nil error for already terminal state, got: %v", err)
	}

	if repo.updateCallCount != 0 {
		t.Errorf("expected 0 DB updates for already terminal state, got %d", repo.updateCallCount)
	}
}
