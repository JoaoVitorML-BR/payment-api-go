package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func computeOfficialHash(dataID, requestID, ts, secret string) string {
	parts := []string{}
	if dataID != "" {
		parts = append(parts, "id:"+strings.ToLower(dataID))
	}
	if requestID != "" {
		parts = append(parts, "request-id:"+requestID)
	}
	parts = append(parts, "ts:"+ts)
	manifest := ""
	for _, p := range parts {
		manifest += p + ";"
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(manifest))
	return hex.EncodeToString(h.Sum(nil))
}


func TestVerifySignature_OfficialCompliance(t *testing.T) {
	secret := "my_mercado_pago_secret_key"
	os.Setenv("MERCADO_PAGO_WEBHOOK_SECRET", secret)
	defer os.Unsetenv("MERCADO_PAGO_WEBHOOK_SECRET")

	now := time.Now().UTC()
	tsStr := fmt.Sprintf("%d", now.Unix())
	requestID := "2066ca19-c6f1-498a-be75-1923005edd06"
	numericDataID := "1234567890"

	t.Run("valid numeric dataID signature matches", func(t *testing.T) {
		hash := computeOfficialHash(numericDataID, requestID, tsStr, secret)
		sigHeader := fmt.Sprintf("ts=%s,v1=%s", tsStr, hash)

		err := VerifySignature(sigHeader, requestID, numericDataID, now)
		if err != nil {
			t.Fatalf("expected valid signature, got error: %v", err)
		}
	})

	t.Run("valid alphanumeric dataID in uppercase matches", func(t *testing.T) {
		alphaDataID := "ORD01JQ4S4KY8HWQ6NA5PXB65B3D3"
		hash := computeOfficialHash(alphaDataID, requestID, tsStr, secret)
		sigHeader := fmt.Sprintf("ts=%s,v1=%s", tsStr, hash)

		err := VerifySignature(sigHeader, requestID, alphaDataID, now)
		if err != nil {
			t.Fatalf("expected valid signature for uppercase dataID, got error: %v", err)
		}
	})

	t.Run("secret with surrounding quotes/whitespace is trimmed", func(t *testing.T) {
		os.Setenv("MERCADO_PAGO_WEBHOOK_SECRET", " \""+secret+"\" \n")
		hash := computeOfficialHash(numericDataID, requestID, tsStr, secret)
		sigHeader := fmt.Sprintf("ts=%s,v1=%s", tsStr, hash)

		err := VerifySignature(sigHeader, requestID, numericDataID, now)
		if err != nil {
			t.Fatalf("expected secret to be trimmed and validated, got: %v", err)
		}
	})

	t.Run("mismatched signature returns error", func(t *testing.T) {
		os.Setenv("MERCADO_PAGO_WEBHOOK_SECRET", secret)
		sigHeader := fmt.Sprintf("ts=%s,v1=tampered_hash_value_1234567890", tsStr)

		err := VerifySignature(sigHeader, requestID, numericDataID, now)
		if err == nil {
			t.Fatal("expected error for tampered signature, got nil")
		}
	})

	t.Run("expired timestamp returns error", func(t *testing.T) {
		oldTs := fmt.Sprintf("%d", now.Add(-10*time.Minute).Unix())
		hash := computeOfficialHash(numericDataID, requestID, oldTs, secret)
		sigHeader := fmt.Sprintf("ts=%s,v1=%s", oldTs, hash)

		err := VerifySignature(sigHeader, requestID, numericDataID, now)
		if err == nil {
			t.Fatal("expected error for expired timestamp, got nil")
		}
	})

	t.Run("missing header returns error", func(t *testing.T) {
		err := VerifySignature("", requestID, numericDataID, now)
		if err == nil {
			t.Fatal("expected error for empty signature header, got nil")
		}
	})
}

