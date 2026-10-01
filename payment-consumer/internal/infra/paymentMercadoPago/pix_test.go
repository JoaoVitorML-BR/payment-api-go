package paymentmercadopago

import (
	"testing"
)

func TestSplitFullName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFirst string
		wantLast  string
	}{
		{"empty", "", "", ""},
		{"single name", "João", "João", "João"},
		{"two names", "João Silva", "João", "Silva"},
		{"multiple names", "João Vitor Martins Lima", "João", "Vitor Martins Lima"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := splitFullName(tt.input)
			if first != tt.wantFirst || last != tt.wantLast {
				t.Errorf("splitFullName(%q) = (%q, %q), want (%q, %q)", tt.input, first, last, tt.wantFirst, tt.wantLast)
			}
		})
	}
}

func TestParsePhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArea string
		wantNum  string
	}{
		{"brazil formatted mobile", "(11) 98765-4321", "11", "987654321"},
		{"brazil country code prefixed", "+5511987654321", "11", "987654321"},
		{"10 digits landline", "1133334444", "11", "33334444"},
		{"digits only", "21999998888", "21", "999998888"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area, num := parsePhone(tt.input)
			if area != tt.wantArea || num != tt.wantNum {
				t.Errorf("parsePhone(%q) = (%q, %q), want (%q, %q)", tt.input, area, num, tt.wantArea, tt.wantNum)
			}
		})
	}
}

func TestParseStreetAndNumber(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStreet string
		wantNum    string
	}{
		{"empty", "", "Rua", "1"},
		{"comma separated", "Rua das Flores, 123", "Rua das Flores", "123"},
		{"comma separated with complement", "Av. Paulista, 1000 - Cj 42", "Av. Paulista", "1000"},
		{"space separated ending with number", "Rua dos Andradas 450", "Rua dos Andradas", "450"},
		{"no number fallback", "Avenida Sem Fim", "Avenida Sem Fim", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			street, number := parseStreetAndNumber(tt.input)
			if street != tt.wantStreet || number != tt.wantNum {
				t.Errorf("parseStreetAndNumber(%q) = (%q, %q), want (%q, %q)", tt.input, street, number, tt.wantStreet, tt.wantNum)
			}
		})
	}
}

func TestIdentificationTypeForTaxID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid cpf with punct", "123.456.789-01", "CPF"},
		{"valid cpf digits", "12345678901", "CPF"},
		{"valid cnpj with punct", "12.345.678/0001-90", "CNPJ"},
		{"valid cnpj digits", "12345678000190", "CNPJ"},
		{"invalid length", "12345", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := identificationTypeForTaxID(tt.input)
			if got != tt.want {
				t.Errorf("identificationTypeForTaxID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}