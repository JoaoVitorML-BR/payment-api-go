package worker

import (
	"testing"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
)

func TestPaymentRequestStatusMapsApprovedToSucceeded(t *testing.T) {
	if got := paymentRequestStatus(paymentgateway.StatusApproved); got != "succeeded" {
		t.Fatalf("expected approved to map to succeeded, got %q", got)
	}
}

func TestPaymentRequestStatusKeepsPending(t *testing.T) {
	if got := paymentRequestStatus(paymentgateway.StatusPending); got != "pending" {
		t.Fatalf("expected pending to remain pending, got %q", got)
	}
}
