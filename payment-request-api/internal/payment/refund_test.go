package payment

import (
	"context"
	"testing"
)

type refundRepositoryMock struct {
	info     RefundPaymentInfo
	reserved RefundRecord
	markedID string
	failed   bool
}

func (m *refundRepositoryMock) GetRefundPaymentInfo(context.Context, string) (RefundPaymentInfo, error) {
	return m.info, nil
}

func (m *refundRepositoryMock) ReserveRefund(_ context.Context, paymentID, key string, amount int64, _ string) (RefundRecord, error) {
	m.reserved = RefundRecord{PaymentID: paymentID, IdempotencyKey: key, AmountCents: amount, Status: "processing"}
	return m.reserved, nil
}

func (m *refundRepositoryMock) MarkRefundSucceeded(_ context.Context, _, gatewayID string) error {
	m.markedID = gatewayID
	return nil
}

func (m *refundRepositoryMock) MarkRefundFailed(context.Context, string, string, string) error {
	m.failed = true
	return nil
}

type refundGatewayMock struct {
	calls int
}

func (m *refundGatewayMock) Refund(context.Context, string, string, int64) (string, error) {
	m.calls++
	return "refund-123", nil
}

func TestProcessRefundExecutesAndPersistsGatewayRefund(t *testing.T) {
	repo := &refundRepositoryMock{info: RefundPaymentInfo{
		GatewayPaymentID: "123456",
		AmountCents:      10000,
		Status:           "succeeded",
		SellerID:         "seller-1",
	}}
	gateway := &refundGatewayMock{}
	service, err := NewPaymentService(&mockRepoForReconciliation{}, &mockPublisher{}, &mockGatewayReaderForReconciliation{})
	if err != nil {
		t.Fatal(err)
	}
	service.SetRefundDependencies(repo, gateway)

	err = service.ProcessRefund(context.Background(), RefundRequest{
		PaymentID:      "payment-1",
		AmountCents:    7500,
		IdempotencyKey: "refund-key-1",
		Reason:         "customer_cancelled",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gateway.calls != 1 || repo.markedID != "refund-123" || repo.failed {
		t.Fatalf("unexpected refund execution: calls=%d marked=%q failed=%v", gateway.calls, repo.markedID, repo.failed)
	}
}
