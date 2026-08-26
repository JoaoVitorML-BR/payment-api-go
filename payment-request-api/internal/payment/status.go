// payment-request-api\internal\payment\status.go
package payment

import (
	"fmt"
	"strings"
)

func normalizeGatewayStatus(gatewayStatus string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(gatewayStatus)) {

	case "approved":
		return "succeeded", nil

	case "cancelled", "canceled":
		return "canceled", nil

	case "rejected":
		return "failed", nil

	case "pending", "in_process", "in_mediation":
		return "pending", nil

	default:
		return "", fmt.Errorf(
			"unknown Mercado Pago status: %q",
			gatewayStatus,
		)
	}
}

func isAllowedStatusTransition(currentStatus string, nextStatus string) bool {
	current := strings.ToLower(strings.TrimSpace(currentStatus))
	next := strings.ToLower(strings.TrimSpace(nextStatus))

	if current == next {
		return true
	}

	switch current {
	case "pending", "approved", "authorized", "in_process", "in_mediation", "rejected", "cancelled", "refunded", "charged_back":
		return next == "pending" || next == "succeeded" || next == "failed" || next == "canceled"
	case "succeeded", "failed", "canceled":
		return false
	default:
		return false
	}
}
