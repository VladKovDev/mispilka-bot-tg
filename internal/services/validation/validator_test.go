package validation

import (
	"testing"
)

func TestValidator_ValidateScenarioName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Test Scenario", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 201)), true},
		{"valid with spaces", "My Premium Course 2024", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScenarioName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScenarioName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateProductPrice(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid price", "500", false},
		{"valid price 1000", "1000", false},
		{"valid large price", "50000", false},
		{"empty", "", true},
		{"not a number", "abc", true},
		{"has letters", "500rub", true},
		{"negative", "-100", true},
		{"zero", "0", true},
		{"decimal", "99.99", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProductPrice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProductPrice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidatePrivateGroupID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid group ID", "-1001234567890", false},
		{"valid shorter", "-100123456", false},
		{"empty", "", true},
		{"missing minus", "1001234567890", true},
		{"missing prefix", "-1234567890", true},
		{"has letters", "-100abcdef", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrivateGroupID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrivateGroupID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
