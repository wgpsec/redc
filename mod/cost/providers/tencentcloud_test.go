package providers

import (
	"os"
	"testing"
)

func TestGetTencentcloudPricing(t *testing.T) {
	// Skip if credentials are not available
	secretId := os.Getenv("TENCENTCLOUD_SECRET_ID")
	secretKey := os.Getenv("TENCENTCLOUD_SECRET_KEY")
	
	if secretId == "" || secretKey == "" {
		t.Skip("Skipping Tencent Cloud pricing test: TENCENTCLOUD_SECRET_ID or TENCENTCLOUD_SECRET_KEY not set")
	}
	
	tests := []struct {
		name         string
		region       string
		resourceType string
		wantErr      bool
	}{
		{
			name:         "Valid instance type in Guangzhou",
			region:       "ap-guangzhou",
			resourceType: "S5.MEDIUM4",
			wantErr:      false,
		},
		{
			name:         "Valid instance type in Beijing",
			region:       "ap-beijing",
			resourceType: "S5.LARGE8",
			wantErr:      false,
		},
		{
			name:         "Empty region uses default",
			region:       "",
			resourceType: "S5.MEDIUM4",
			wantErr:      false,
		},
		{
			name:         "Empty resource type",
			region:       "ap-guangzhou",
			resourceType: "",
			wantErr:      true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := GetTencentcloudPricing(tt.region, tt.resourceType, secretId, secretKey)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTencentcloudPricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Verify pricing data structure
				if pricing == nil {
					t.Error("GetTencentcloudPricing() returned nil pricing data")
					return
				}
				
				if pricing.Provider != "tencentcloud" {
					t.Errorf("Expected provider 'tencentcloud', got '%s'", pricing.Provider)
				}
				
				if pricing.ResourceType != tt.resourceType {
					t.Errorf("Expected resource type '%s', got '%s'", tt.resourceType, pricing.ResourceType)
				}
				
				if pricing.HourlyPrice <= 0 {
					t.Errorf("Expected positive hourly price, got %f", pricing.HourlyPrice)
				}
				
				if pricing.MonthlyPrice <= 0 {
					t.Errorf("Expected positive monthly price, got %f", pricing.MonthlyPrice)
				}
				
				// Verify monthly price is hourly * 720
				expectedMonthly := pricing.HourlyPrice * 720
				if pricing.MonthlyPrice != expectedMonthly {
					t.Errorf("Expected monthly price %f (hourly * 720), got %f", expectedMonthly, pricing.MonthlyPrice)
				}
				
				if pricing.Currency == "" {
					t.Error("Expected non-empty currency")
				}
				
				if pricing.Region == "" {
					t.Error("Expected non-empty region")
				}
				
				t.Logf("Pricing for %s in %s: %s %.4f/hour, %s %.2f/month",
					pricing.ResourceType, pricing.Region,
					pricing.Currency, pricing.HourlyPrice,
					pricing.Currency, pricing.MonthlyPrice)
			}
		})
	}
}

func TestGetTencentcloudPricing_MissingCredentials(t *testing.T) {
	tests := []struct {
		name      string
		secretId  string
		secretKey string
		wantErr   bool
	}{
		{
			name:      "Missing secret ID",
			secretId:  "",
			secretKey: "test-key",
			wantErr:   true,
		},
		{
			name:      "Missing secret key",
			secretId:  "test-id",
			secretKey: "",
			wantErr:   true,
		},
		{
			name:      "Missing both credentials",
			secretId:  "",
			secretKey: "",
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetTencentcloudPricing("ap-guangzhou", "S5.MEDIUM4", tt.secretId, tt.secretKey)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTencentcloudPricing() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTencentcloudPricing_RegionSpecific(t *testing.T) {
	// Skip if credentials are not available
	secretId := os.Getenv("TENCENTCLOUD_SECRET_ID")
	secretKey := os.Getenv("TENCENTCLOUD_SECRET_KEY")
	
	if secretId == "" || secretKey == "" {
		t.Skip("Skipping Tencent Cloud pricing test: TENCENTCLOUD_SECRET_ID or TENCENTCLOUD_SECRET_KEY not set")
	}
	
	// Test that different regions can return different pricing
	// (though they might be the same in practice)
	regions := []string{"ap-guangzhou", "ap-beijing", "ap-shanghai"}
	resourceType := "S5.MEDIUM4"
	
	pricingByRegion := make(map[string]*PricingData)
	
	for _, region := range regions {
		pricing, err := GetTencentcloudPricing(region, resourceType, secretId, secretKey)
		if err != nil {
			t.Logf("Warning: Failed to get pricing for region %s: %v", region, err)
			continue
		}
		
		pricingByRegion[region] = pricing
		
		// Verify region is set correctly
		if pricing.Region != region {
			t.Errorf("Expected region '%s', got '%s'", region, pricing.Region)
		}
		
		t.Logf("Region %s: %s %.4f/hour", region, pricing.Currency, pricing.HourlyPrice)
	}
	
	if len(pricingByRegion) < 2 {
		t.Skip("Not enough regions succeeded to compare pricing")
	}
}
