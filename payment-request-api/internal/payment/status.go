// payment-request-api\internal\payment\status.go
package payment

import (
	"fmt"
	"strings"
)

func normalizeGatewayStatus(gatewayStatus string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(gatewayStatus)) {

	// Payments API legacy statuses & Orders API success statuses
	case "approved", "processed", "accredited", "paid":
		return "succeeded", nil

	case "cancelled", "canceled", "expired":
		return "canceled", nil

	case "rejected":
		return "failed", nil

	case "pending", "in_process", "in_mediation", "action_required", "created":
		return "pending", nil

	case "refunded":
		return "refunded", nil

	case "partially_refunded":
		return "partially_refunded", nil

	case "charged_back":
		return "failed", nil

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
