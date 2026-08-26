// payment-consumer\worker\error.go
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5/pgconn"
)

// isRetryableWorkerError returns true when the message should be requeued.
func isRetryableWorkerError(err error) bool {
	return isRetryableServiceError(err)
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

// gatewayErrorPayload extracts human-readable fields and a JSON payload
// that can be stored in the database for later inspection.
func gatewayErrorPayload(err error) (string, string, []byte) {
	payload, _ := json.Marshal(map[string]any{
		"error_type":    fmt.Sprintf("%T", err),
		"error_message": err.Error(),
	})
	return "", err.Error(), payload
}
