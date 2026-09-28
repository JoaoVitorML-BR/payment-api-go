// payment-request-api/internal/payment/status_test.go
package payment

import "testing"

func TestNormalizeGatewayStatus(t *testing.T) {
	tests := []struct {
		name          string
		gatewayStatus string
		want          string
		wantErr       bool
	}{
		{
			name:          "approved status",
			gatewayStatus: "approved",
			want:          "succeeded",
			wantErr:       false,
		},
		{
			name:          "cancelled status with double l",
			gatewayStatus: "cancelled",
			want:          "canceled",
			wantErr:       false,
		},
		{
			name:          "canceled status with single l",
			gatewayStatus: "canceled",
			want:          "canceled",
			wantErr:       false,
		},
		{
			name:          "rejected status",
			gatewayStatus: "rejected",
			want:          "failed",
			wantErr:       false,
		},
		{
			name:          "pending status",
			gatewayStatus: "pending",
			want:          "pending",
			wantErr:       false,
		},
		{
			name:          "in_process status",
			gatewayStatus: "in_process",
			want:          "pending",
			wantErr:       false,
		},
		{
			name:          "processed status (Orders API)",
			gatewayStatus: "processed",
			want:          "succeeded",
			wantErr:       false,
		},
		{
			name:          "accredited status (Orders API)",
			gatewayStatus: "accredited",
			want:          "succeeded",
			wantErr:       false,
		},
		{
			name:          "paid status (Orders API)",
			gatewayStatus: "paid",
			want:          "succeeded",
			wantErr:       false,
		},
		{
			name:          "action_required status (Orders API)",
			gatewayStatus: "action_required",
			want:          "pending",
			wantErr:       false,
		},
		{
			name:          "created status (Orders API)",
			gatewayStatus: "created",
			want:          "pending",
			wantErr:       false,
		},
		{
			name:          "expired status (Orders API)",
			gatewayStatus: "expired",
			want:          "canceled",
			wantErr:       false,
		},
		{
			name:          "unknown status",
			gatewayStatus: "unknown_status_xyz",
			want:          "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeGatewayStatus(tt.gatewayStatus)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeGatewayStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("normalizeGatewayStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAllowedStatusTransition(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus string
		nextStatus    string
		want          bool
	}{
		{
			name:          "same status transition",
			currentStatus: "pending",
			nextStatus:    "pending",
			want:          true,
		},
		{
			name:          "pending to succeeded",
			currentStatus: "pending",
			nextStatus:    "succeeded",
			want:          true,
		},
		{
			name:          "pending to failed",
			currentStatus: "pending",
			nextStatus:    "failed",
			want:          true,
		},
		{
			name:          "pending to canceled",
			currentStatus: "pending",
			nextStatus:    "canceled",
			want:          true,
		},
		{
			name:          "succeeded to succeeded (already terminal)",
			currentStatus: "succeeded",
			nextStatus:    "succeeded",
			want:          true,
		},
		{
			name:          "succeeded to failed (invalid transition from terminal)",
			currentStatus: "succeeded",
			nextStatus:    "failed",
			want:          false,
		},
		{
			name:          "canceled to succeeded (invalid transition from terminal)",
			currentStatus: "canceled",
			nextStatus:    "succeeded",
			want:          false,
		},
		{
			name:          "failed to succeeded (invalid transition from terminal)",
			currentStatus: "failed",
			nextStatus:    "succeeded",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowedStatusTransition(tt.currentStatus, tt.nextStatus)
			if got != tt.want {
				t.Errorf("isAllowedStatusTransition(%q, %q) = %v, want %v", tt.currentStatus, tt.nextStatus, got, tt.want)
			}
		})
	}
}
