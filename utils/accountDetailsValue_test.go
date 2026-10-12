package utils

import (
	"strings"
	"testing"
)

func TestGenerateAccountDetail(t *testing.T) {
	tests := []struct {
		name        string
		detailType  string
		countryCode string
		wantLength  int
		wantErr     bool
	}{
		{
			name:       "account number",
			detailType: "account_number",
			wantLength: 16,
		},
		{
			name:        "German IBAN",
			detailType:  "iban",
			countryCode: "DE",
			wantLength: 22,
		},
		{
			name:        "UK IBAN",
			detailType:  "iban",
			countryCode: "GB",
			wantLength: 22,
		},
		{
			name:        "Moroccan IBAN",
			detailType:  "iban",
			countryCode: "MA",
			wantLength: 28,
		},
		{
			name:        "lowercase country code",
			detailType:  "iban",
			countryCode: "de",
			wantLength: 22,
		},
		{
			name:        "unsupported IBAN country",
			detailType:  "iban",
			countryCode: "XX",
			wantErr:     true,
		},
		{
			name:       "routing number",
			detailType: "routing_number",
			wantErr:    true,
		},
		{
			name:       "unsupported detail type",
			detailType: "unknown",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateAccountDetail(
				tt.detailType,
				tt.countryCode,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != tt.wantLength {
				t.Fatalf(
					"expected length %d, got %d: %q",
					tt.wantLength,
					len(got),
					got,
				)
			}

			if tt.detailType == "account_number" {
				if !isDigits(got) {
					t.Errorf("account number contains non-digits: %s", got)
				}
			}

			if tt.detailType == "iban" {
				if got[:2] != strings.ToUpper(tt.countryCode) {
					t.Errorf("unexpected country code: %s", got[:2])
				}

				if !isDigits(got[2:4]) {
					t.Errorf("IBAN check digits are not numeric: %s", got)
				}

				if mod97(got[4:]+got[:4]) != 1 {
					t.Errorf("invalid IBAN checksum: %s", got)
				}
			}
		})
	}
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
