// payment-request-api\internal\infra\webhook\webhook.go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxSignatureAge = 5 * time.Minute

func VerifySignature(signatureHeader string, requestID string, dataID string, now time.Time) error {
	secret := strings.Trim(os.Getenv("MERCADO_PAGO_WEBHOOK_SECRET"), "\"' \t\r\n")

	if secret == "" {
		return errors.New("mercado pago webhook secret is not configured")
	}

	parts, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}

	ts, err := strconv.ParseInt(parts["ts"], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ts in x-signature: %w", err)
	}

	var tsTime time.Time

	if ts > 1_000_000_000_000 {
		tsTime = time.UnixMilli(ts).UTC()
	} else {
		tsTime = time.Unix(ts, 0).UTC()
	}

	age := now.UTC().Sub(tsTime)

	if age < -maxSignatureAge || age > maxSignatureAge {
		return errors.New("webhook signature expired")
	}

	requestID = strings.TrimSpace(requestID)
	dataID = strings.ToLower(strings.TrimSpace(dataID))

	if requestID == "" || dataID == "" {
		return errors.New("missing request id or data.id for signature verification")
	}

	received := strings.ToLower(parts["v1"])

	manifest := fmt.Sprintf(
		"id:%s;request-id:%s;ts:%s;",
		dataID,
		requestID,
		parts["ts"],
	)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(manifest))

	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if hmac.Equal([]byte(expectedSignature), []byte(received)) {
		return nil
	}

	return errors.New("invalid signature")
}

func parseSignatureHeader(signatureHeader string) (map[string]string, error) {
	signatureHeader = strings.TrimSpace(signatureHeader)
	if signatureHeader == "" {
		return nil, errors.New("missing x-signature header")
	}

	parts := strings.Split(signatureHeader, ",")
	result := make(map[string]string, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		result[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
	}

	if result["ts"] == "" || result["v1"] == "" {
		return nil, errors.New("x-signature must include ts and v1")
	}

	return result, nil
}
