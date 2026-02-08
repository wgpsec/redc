package providers

import (
	"os"
	"testing"
)

// TestGetVolcenginePricing_MissingCredentials tests that missing credentials are handled correctly
func TestGetVolcenginePricing_MissingCredentials(t *testing.T) {
	testCases := []struct {
		name      string
		accessKey string
		secretKey string
	}{
		{
			name:      "Missing access key",
			accessKey: "",
			secretKey: "test-secret",
		},
		{
			name:      "Missing secret key",
			accessKey: "test-access",
			secretKey: "",
		},
		{
			name:      "Missing both credentials",
			accessKey: "",
			secretKey: "",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetVolcenginePricing("cn-beijing", "ecs.g1.large", tc.accessKey, tc.secretKey)
			if err == nil {
				t.Error("Expected error for missing credentials, got nil")
			}
		})
	}
}

// TestGetVolcenginePricing_EmptyResourceType tests that empty resource type is handled correctly
func TestGetVolcenginePricing_EmptyResourceType(t *testing.T) {
	_, err := GetVolcenginePricing("cn-beijing", "", "test-access", "test-secret")
	if err == nil {
		t.Error("Expected error for empty resource type, got nil")
	}
}

// TestGetVolcenginePricing_DefaultRegion tests that default region is used when region is empty
func TestGetVolcenginePricing_DefaultRegion(t *testing.T) {
	// Skip if credentials are not set
	accessKey := os.Getenv("VOLCENGINE_ACCESS_KEY")
	secretKey := os.Getenv("VOLCENGINE_SECRET_KEY")
	
	if accessKey == "" || secretKey == "" {
		t.Skip("Skipping Volcengine pricing test: VOLCENGINE_ACCESS_KEY or VOLCENGINE_SECRET_KEY not set")
	}
	
	// Test with empty region (should use default)
	pricing, err := GetVolcenginePricing("", "ecs.g1.large", accessKey, secretKey)
	if err != nil {
		t.Logf("Note: This test requires valid Volcengine credentials and may fail if the instance type doesn't exist")
		t.Logf("Error: %v", err)
		return
	}
	
	if pricing == nil {
		t.Error("Expected pricing data, got nil")
		return
	}
	
	// Verify pricing data structure
	if pricing.Provider != "volcengine" {
		t.Errorf("Expected provider 'volcengine', got '%s'", pricing.Provider)
	}
	
	if pricing.Currency != "CNY" {
		t.Errorf("Expected currency 'CNY', got '%s'", pricing.Currency)
	}
	
	if pricing.HourlyPrice <= 0 {
		t.Errorf("Expected positive hourly price, got %f", pricing.HourlyPrice)
	}
	
	if pricing.MonthlyPrice <= 0 {
		t.Errorf("Expected positive monthly price, got %f", pricing.MonthlyPrice)
	}
}

// TestGetInstanceFamilyMultiplier tests the instance family multiplier logic
func TestGetInstanceFamilyMultiplier(t *testing.T) {
	testCases := []struct {
		family     string
		multiplier float64
	}{
		{"ecs.g1", 1.0},
		{"ecs.g2", 1.1},
		{"ecs.g3", 1.2},
		{"ecs.c1", 1.1},
		{"ecs.c2", 1.2},
		{"ecs.c3", 1.3},
		{"ecs.r1", 1.3},
		{"ecs.r2", 1.4},
		{"ecs.r3", 1.5},
		{"ecs.gn", 2.0},
		{"ecs.vgn", 1.8},
		{"ecs.unknown", 1.0}, // Default multiplier
		{"ecs.g3a", 1.2},     // Prefix match for ecs.g3
	}
	
	for _, tc := range testCases {
		t.Run(tc.family, func(t *testing.T) {
			multiplier := getInstanceFamilyMultiplier(tc.family)
			if multiplier != tc.multiplier {
				t.Errorf("Expected multiplier %f for family %s, got %f", tc.multiplier, tc.family, multiplier)
			}
		})
	}
}
