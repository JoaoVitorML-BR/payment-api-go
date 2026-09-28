package payment

import (
	"testing"
)

func TestExtractWebhookDataID(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "orders api string id in data.id",
			body:     `{"action":"payment.updated","data":{"id":"ORDTST01M379GN663CHAA4S5Y6B4H9F2"},"type":"payment"}`,
			expected: "ORDTST01M379GN663CHAA4S5Y6B4H9F2",
		},
		{
			name:     "payments api numeric id in data.id",
			body:     `{"action":"payment.updated","data":{"id":123456789},"type":"payment"}`,
			expected: "123456789",
		},
		{
			name:     "payments api string numeric id in data.id",
			body:     `{"action":"payment.updated","data":{"id":"123456789"},"type":"payment"}`,
			expected: "123456789",
		},
		{
			name:     "legacy top-level id numeric",
			body:     `{"id":987654321,"type":"payment"}`,
			expected: "987654321",
		},
		{
			name:     "legacy top-level id string",
			body:     `{"id":"ORDTST01M379GN663CHAA4S5Y6B4H9F2","type":"payment"}`,
			expected: "ORDTST01M379GN663CHAA4S5Y6B4H9F2",
		},
		{
			name:     "empty body",
			body:     ``,
			expected: "",
		},
		{
			name:     "invalid json",
			body:     `{not-valid}`,
			expected: "",
		},
		{
			name:     "missing data.id and id",
			body:     `{"action":"payment.updated","data":{}}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWebhookDataID([]byte(tt.body))
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
