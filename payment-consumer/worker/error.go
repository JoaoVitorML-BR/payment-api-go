package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5/pgconn"
	sdkstripe "github.com/stripe/stripe-go/v85"
)

// isRetryableWorkerError returns true when the message should be requeued.
func isRetryableWorkerError(err error) bool {
	return isRetryableServiceError(err) || isRetryableStripeError(err)
}

// isRetryableStripeError returns true when the error should be retried.
func isRetryableStripeError(err error) bool {
	var stripeErr *sdkstripe.Error
	if errors.As(err, &stripeErr) {
		switch stripeErr.Type {
		// API and rate limit errors are transient and worth retrying
		case sdkstripe.ErrorTypeAPI, sdkstripe.ErrorTypeRateLimit:
			return true
		// Idempotency errors indicate the same idempotency key was used with
		// different parameters — these are not retryable and should be discarded
		case sdkstripe.ErrorTypeIdempotency:
			return false
		case sdkstripe.ErrorTypeCard, sdkstripe.ErrorTypeInvalidRequest:
			return false
		default:
			return false
		}
	}

	var cardErr *sdkstripe.CardError
	if errors.As(err, &cardErr) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

// isRetryableServiceError returns true for transient service/database failures.
func isRetryableServiceError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001", "40P01":
			return true
		}

		if len(pgErr.Code) >= 2 {
			switch pgErr.Code[:2] {
			case "08", "53", "57":
				return true
			}
		}
	}

	return false
}

// stripeErrorPayload extracts human-readable fields and a JSON payload
// that can be stored in the database for later inspection.
func stripeErrorPayload(err error) (string, string, []byte) {
	var stripeErr *sdkstripe.Error
	if errors.As(err, &stripeErr) {
		payload, _ := json.Marshal(map[string]any{
			"error_type":    string(stripeErr.Type),
			"error_code":    string(stripeErr.Code),
			"error_message": stripeErr.Msg,
			"request_id":    stripeErr.RequestID,
			"http_status":   stripeErr.HTTPStatusCode,
			"decline_code":  string(stripeErr.DeclineCode),
		})
		return string(stripeErr.Code), stripeErr.Msg, payload
	}

	payload, _ := json.Marshal(map[string]any{
		"error_type":    fmt.Sprintf("%T", err),
		"error_message": err.Error(),
	})
	return "", err.Error(), payload
}
