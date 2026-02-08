package providers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

// GetAlicloudPricing retrieves pricing data for Alibaba Cloud resources
// It uses the DescribePrice API to fetch real-time pricing information
func GetAlicloudPricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing Alibaba Cloud access key or secret key")
	}
	
	if region == "" {
		region = "cn-hangzhou" // Default region
	}
	
	if resourceType == "" {
		return nil, fmt.Errorf("resource type cannot be empty")
	}
	
	// Create ECS client
	client, err := ecs.NewClientWithAccessKey(region, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba Cloud ECS client: %w", err)
	}
	
	// Try different disk categories in order of preference
	// Different instance types support different disk types:
	// - New generation instances (ecs.c7a, ecs.g7, etc.) support cloud_essd
	// - Old generation instances (ecs.n1, ecs.t5, etc.) support cloud_efficiency, cloud_ssd
	diskCategories := []string{"cloud_essd", "cloud_efficiency", "cloud_ssd"}
	
	var lastErr error
	for _, diskCategory := range diskCategories {
		// Create DescribePrice request
		request := ecs.CreateDescribePriceRequest()
		request.Scheme = "https"
		request.ResourceType = "instance"
		request.InstanceType = resourceType
		request.PriceUnit = "Hour" // Get hourly pricing
		request.SystemDiskCategory = diskCategory
		
		// Call DescribePrice API
		response, err := client.DescribePrice(request)
		if err != nil {
			lastErr = err
			// Check if this is a disk category compatibility error
			if strings.Contains(strings.ToLower(err.Error()), "invalidsystemdiskcategory") ||
			   strings.Contains(strings.ToLower(err.Error()), "notsupportdiskcategory") {
				// Try next disk category
				continue
			}
			// For other errors, return immediately
			return nil, fmt.Errorf("failed to call DescribePrice API: %w", err)
		}
		
		if response == nil || response.PriceInfo.Price.OriginalPrice == 0 {
			lastErr = fmt.Errorf("empty or invalid pricing response from Alibaba Cloud")
			continue
		}
		
		// Success - we found a compatible disk category
		return buildPricingData(response, region, resourceType, diskCategory)
	}
	
	// All disk categories failed
	if lastErr != nil {
		return nil, fmt.Errorf("failed to get pricing with any disk category: %w", lastErr)
	}
	return nil, fmt.Errorf("failed to get pricing: no compatible disk category found")
}

// buildPricingData constructs PricingData from the API response
func buildPricingData(response *ecs.DescribePriceResponse, region, resourceType, diskCategory string) (*PricingData, error) {
	if response == nil || response.PriceInfo.Price.OriginalPrice == 0 {
		return nil, fmt.Errorf("empty or invalid pricing response from Alibaba Cloud")
	}
	// Parse the pricing data
	hourlyPrice := response.PriceInfo.Price.OriginalPrice
	monthlyPrice := hourlyPrice * 720 // 720 hours per month (30 days * 24 hours)
	
	// Get currency from response, default to CNY if not specified
	currency := response.PriceInfo.Price.Currency
	if currency == "" {
		currency = "CNY"
	}
	
	// Create PricingData structure
	pricingData := &PricingData{
		Provider:     "alicloud",
		Region:       region,
		ResourceType: resourceType,
		Currency:     currency,
		HourlyPrice:  hourlyPrice,
		MonthlyPrice: monthlyPrice,
		Metadata:     make(map[string]string),
	}
	
	// Add metadata if available
	if response.PriceInfo.Price.DiscountPrice > 0 {
		pricingData.Metadata["discount_price"] = strconv.FormatFloat(response.PriceInfo.Price.DiscountPrice, 'f', -1, 64)
	}
	if response.PriceInfo.Price.TradePrice > 0 {
		pricingData.Metadata["trade_price"] = strconv.FormatFloat(response.PriceInfo.Price.TradePrice, 'f', -1, 64)
	}
	
	// Record which disk category was used
	pricingData.Metadata["disk_category"] = diskCategory
	
	return pricingData, nil
}
