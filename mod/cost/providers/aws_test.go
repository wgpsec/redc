package providers

import (
	"testing"
)

func TestGetAWSPricing_MissingCredentials(t *testing.T) {
	// Test with missing access key
	_, err := GetAWSPricing("us-east-1", "t2.micro", "", "secret")
	if err == nil {
		t.Error("Expected error for missing access key, got nil")
	}
	
	// Test with missing secret key
	_, err = GetAWSPricing("us-east-1", "t2.micro", "access", "")
	if err == nil {
		t.Error("Expected error for missing secret key, got nil")
	}
}

func TestGetAWSPricing_EmptyResourceType(t *testing.T) {
	// Test with empty resource type
	_, err := GetAWSPricing("us-east-1", "", "access", "secret")
	if err == nil {
		t.Error("Expected error for empty resource type, got nil")
	}
}

func TestGetAWSPricing_DefaultRegion(t *testing.T) {
	// Test that default region is used when region is empty
	// This test will fail with API error since we're using fake credentials,
	// but it verifies the function accepts empty region
	_, err := GetAWSPricing("", "t2.micro", "fake-access", "fake-secret")
	
	// We expect an error (API call will fail with fake credentials)
	// but not a "missing region" error
	if err != nil && err.Error() == "region cannot be empty" {
		t.Error("Function should use default region when region is empty")
	}
}

func TestGetAWSPricing_DataStructure(t *testing.T) {
	// This is a mock test to verify the data structure
	// In a real scenario, we would mock the pricing client
	
	// Create a mock pricing data to verify structure
	mockData := &PricingData{
		Provider:     "aws",
		Region:       "us-east-1",
		ResourceType: "t2.micro",
		Currency:     "USD",
		HourlyPrice:  0.0116,
		MonthlyPrice: 8.352,
		Metadata:     make(map[string]string),
	}
	
	// Verify structure
	if mockData.Provider != "aws" {
		t.Errorf("Expected provider 'aws', got '%s'", mockData.Provider)
	}
	
	// Check monthly price is approximately hourly * 720 (allow for floating point precision)
	expectedMonthly := mockData.HourlyPrice * 720
	if mockData.MonthlyPrice < expectedMonthly-0.01 || mockData.MonthlyPrice > expectedMonthly+0.01 {
		t.Errorf("Monthly price should be approximately hourly price * 720, got %f, expected %f", mockData.MonthlyPrice, expectedMonthly)
	}
	
	if mockData.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", mockData.Currency)
	}
}

func TestGetAWSLocationName(t *testing.T) {
	tests := []struct {
		region       string
		expectedName string
	}{
		{"us-east-1", "US East (N. Virginia)"},
		{"us-west-2", "US West (Oregon)"},
		{"eu-west-1", "EU (Ireland)"},
		{"ap-southeast-1", "Asia Pacific (Singapore)"},
		{"unknown-region", "unknown-region"}, // Should return the region code itself
	}
	
	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := getAWSLocationName(tt.region)
			if result != tt.expectedName {
				t.Errorf("getAWSLocationName(%s) = %s, want %s", tt.region, result, tt.expectedName)
			}
		})
	}
}

// Note: Integration tests with real API calls should be run separately
// with valid credentials and marked as integration tests
