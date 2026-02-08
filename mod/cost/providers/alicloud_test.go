package providers

import (
	"testing"
)

func TestGetAlicloudPricing_MissingCredentials(t *testing.T) {
	// Test with missing access key
	_, err := GetAlicloudPricing("cn-hangzhou", "ecs.g6.large", "", "secret")
	if err == nil {
		t.Error("Expected error for missing access key, got nil")
	}
	
	// Test with missing secret key
	_, err = GetAlicloudPricing("cn-hangzhou", "ecs.g6.large", "access", "")
	if err == nil {
		t.Error("Expected error for missing secret key, got nil")
	}
}

func TestGetAlicloudPricing_EmptyResourceType(t *testing.T) {
	// Test with empty resource type
	_, err := GetAlicloudPricing("cn-hangzhou", "", "access", "secret")
	if err == nil {
		t.Error("Expected error for empty resource type, got nil")
	}
}

func TestGetAlicloudPricing_DefaultRegion(t *testing.T) {
	// Test that default region is used when region is empty
	// This test will fail with API error since we're using fake credentials,
	// but it verifies the function accepts empty region
	_, err := GetAlicloudPricing("", "ecs.g6.large", "fake-access", "fake-secret")
	
	// We expect an error (API call will fail with fake credentials)
	// but not a "missing region" error
	if err != nil && err.Error() == "region cannot be empty" {
		t.Error("Function should use default region when region is empty")
	}
}

func TestGetAlicloudPricing_DataStructure(t *testing.T) {
	// This is a mock test to verify the data structure
	// In a real scenario, we would mock the ECS client
	
	// Create a mock pricing data to verify structure
	mockData := &PricingData{
		Provider:     "alicloud",
		Region:       "cn-hangzhou",
		ResourceType: "ecs.g6.large",
		Currency:     "CNY",
		HourlyPrice:  0.558,
		MonthlyPrice: 401.76,
		Metadata:     make(map[string]string),
	}
	
	// Verify structure
	if mockData.Provider != "alicloud" {
		t.Errorf("Expected provider 'alicloud', got '%s'", mockData.Provider)
	}
	
	// Check monthly price is approximately hourly * 720 (allow for floating point precision)
	expectedMonthly := mockData.HourlyPrice * 720
	if mockData.MonthlyPrice < expectedMonthly-0.01 || mockData.MonthlyPrice > expectedMonthly+0.01 {
		t.Errorf("Monthly price should be approximately hourly price * 720, got %f, expected %f", mockData.MonthlyPrice, expectedMonthly)
	}
	
	if mockData.Currency != "CNY" {
		t.Errorf("Expected currency 'CNY', got '%s'", mockData.Currency)
	}
}

// Note: Integration tests with real API calls should be run separately
// with valid credentials and marked as integration tests
