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
	secret := os.Getenv("MERCADO_PAGO_WEBHOOK_SECRET")
	if strings.TrimSpace(secret) == "" {
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

	tsTime := time.Unix(ts, 0).UTC()
	if now.UTC().Sub(tsTime) > maxSignatureAge {
		return errors.New("webhook signature expired")
	}

	requestID = strings.TrimSpace(requestID)
	dataID = strings.TrimSpace(dataID)
	if requestID == "" || dataID == "" {
		return errors.New("missing request id or data.id for signature verification")
	}

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, requestID, parts["ts"])
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(manifest))
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	received := strings.ToLower(parts["v1"])

	if !hmac.Equal([]byte(received), []byte(expectedSignature)) {
		return errors.New("invalid signature")
	}

	return nil
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