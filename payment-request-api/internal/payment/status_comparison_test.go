package payment

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockComparisonRepo struct {
	payment LocalPaymentStatus
	err     error
}

func (m *mockComparisonRepo) CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	return CreatePaymentResponse{}, nil
}

func (m *mockComparisonRepo) GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error) {
	return PaymentStatusResponse{}, nil
}

func (m *mockComparisonRepo) GetPaymentDetails(ctx context.Context, identifier string) (LocalPaymentStatus, error) {
	if m.err != nil {
		return LocalPaymentStatus{}, m.err
	}
	return m.payment, nil
}

func (m *mockComparisonRepo) UpdatePaymentStatus(ctx context.Context, paymentUUID string, status string, amountCents int64) (int64, error) {
	return 1, nil
}

func (m *mockComparisonRepo) GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (PaymentGatewayValidationData, error) {
	return PaymentGatewayValidationData{}, nil
}

func (m *mockComparisonRepo) UpdatePaymentStatusByGatewayPaymentID(ctx context.Context, gatewayPaymentID string, status string) (int64, error) {
	return 1, nil
}

func (m *mockComparisonRepo) GetPendingPaymentsForReconciliation(ctx context.Context, maxUpdatedAt time.Time, limit int32) ([]ReconciliationItem, error) {
	return nil, nil
}

type mockComparisonGatewayReader struct {
	details *GatewayPaymentDetails
	err     error
}

func (m *mockComparisonGatewayReader) GetPayment(ctx context.Context, gatewayPaymentID string, sellerID string) (*GatewayPaymentDetails, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.details, nil
}

func TestGetPaymentStatusComparison(t *testing.T) {
	t.Run("returns in_sync true when local and gateway match", func(t *testing.T) {
		repo := &mockComparisonRepo{
			payment: LocalPaymentStatus{
				UUID:             "test-uuid-1",
				GatewayPaymentID: "gw-pay-1",
				Status:           "succeeded",
				AmountCents:      1000,
				Currency:         "BRL",
			},
		}
		reader := &mockComparisonGatewayReader{
			details: &GatewayPaymentDetails{
				GatewayPaymentID: "gw-pay-1",
				Status:           "approved",
				AmountCents:      1000,
				Currency:         "BRL",
			},
		}

		service, err := NewPaymentService(repo, &mockPublisher{}, reader)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		res, err := service.GetPaymentStatusComparison(context.Background(), "test-uuid-1")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if !res.InSync {
			t.Errorf("expected InSync to be true, got false. discrepancy: %s", res.Discrepancy)
		}
		if res.Gateway.NormalizedStatus != "succeeded" {
			t.Errorf("expected normalized status 'succeeded', got %s", res.Gateway.NormalizedStatus)
		}
	})

	t.Run("returns in_sync false when local is pending and gateway is approved", func(t *testing.T) {
		repo := &mockComparisonRepo{
			payment: LocalPaymentStatus{
				UUID:             "test-uuid-2",
				GatewayPaymentID: "gw-pay-2",
				Status:           "pending",
				AmountCents:      2500,
				Currency:         "BRL",
			},
		}
		reader := &mockComparisonGatewayReader{
			details: &GatewayPaymentDetails{
				GatewayPaymentID: "gw-pay-2",
				Status:           "approved",
				AmountCents:      2500,
				Currency:         "BRL",
			},
		}

		service, err := NewPaymentService(repo, &mockPublisher{}, reader)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		res, err := service.GetPaymentStatusComparison(context.Background(), "test-uuid-2")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if res.InSync {
			t.Errorf("expected InSync to be false")
		}
		if res.Discrepancy == "" {
			t.Errorf("expected non-empty discrepancy description")
		}
	})

	t.Run("returns in_sync false when payment not linked to gateway", func(t *testing.T) {
		repo := &mockComparisonRepo{
			payment: LocalPaymentStatus{
				UUID:             "test-uuid-3",
				GatewayPaymentID: "",
				Status:           "pending",
			},
		}
		reader := &mockComparisonGatewayReader{}

		service, err := NewPaymentService(repo, &mockPublisher{}, reader)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		res, err := service.GetPaymentStatusComparison(context.Background(), "test-uuid-3")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if res.InSync {
			t.Errorf("expected InSync to be false")
		}
		if res.Discrepancy == "" {
			t.Errorf("expected discrepancy explaining missing gateway id")
		}
	})

	t.Run("returns error when local payment does not exist", func(t *testing.T) {
		repo := &mockComparisonRepo{
			err: errors.New("not found"),
		}
		reader := &mockComparisonGatewayReader{}

		service, err := NewPaymentService(repo, &mockPublisher{}, reader)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		_, err = service.GetPaymentStatusComparison(context.Background(), "missing-id")
		if err == nil {
			t.Fatalf("expected error for missing payment, got nil")
		}
	})
}
