package providers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// GetTencentcloudPricing retrieves pricing data for Tencent Cloud resources
// It uses the InquiryPriceRunInstances API to fetch real-time pricing information
// The region parameter can be either a region (e.g., "ap-guangzhou") or a zone (e.g., "ap-beijing-7")
func GetTencentcloudPricing(region, resourceType, secretId, secretKey string) (*PricingData, error) {
	if secretId == "" || secretKey == "" {
		return nil, fmt.Errorf("missing Tencent Cloud secret ID or secret key")
	}
	
	if region == "" {
		region = "ap-guangzhou" // Default region
	}
	
	if resourceType == "" {
		return nil, fmt.Errorf("resource type cannot be empty")
	}
	
	// Extract region and zone from the input
	// If region contains a zone suffix (e.g., "ap-beijing-7"), extract both
	// Otherwise, use the region and default to zone-1
	actualRegion := region
	actualZone := region
	
	// Check if region contains a zone suffix (e.g., "ap-beijing-7")
	// Zone format: region + "-" + number
	parts := strings.Split(region, "-")
	if len(parts) >= 3 {
		// This looks like a zone (e.g., "ap-beijing-7")
		// Extract region by removing the last part
		actualRegion = strings.Join(parts[:len(parts)-1], "-")
		actualZone = region
	} else {
		// This is just a region, use default zone
		actualZone = region + "-1"
	}
	
	// Create credential
	credential := common.NewCredential(secretId, secretKey)
	
	// Create client profile
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	
	// Create CVM client with the actual region
	client, err := cvm.NewClient(credential, actualRegion, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent Cloud CVM client: %w", err)
	}
	
	// Create InquiryPriceRunInstances request
	request := cvm.NewInquiryPriceRunInstancesRequest()
	
	// Set instance type
	request.InstanceType = common.StringPtr(resourceType)
	
	// Set instance count to 1 for pricing inquiry
	request.InstanceCount = common.Int64Ptr(1)
	
	// Set instance charge type to POSTPAID_BY_HOUR for hourly pricing
	request.InstanceChargeType = common.StringPtr("POSTPAID_BY_HOUR")
	
	// Set placement with the actual zone
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(actualZone),
	}
	
	// Set image ID (use a common Ubuntu image as default)
	// This is required but doesn't affect pricing for the instance type
	request.ImageId = common.StringPtr("img-pi0ii46r") // Ubuntu Server 20.04 LTS
	
	// Call InquiryPriceRunInstances API
	response, err := client.InquiryPriceRunInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call InquiryPriceRunInstances API: %w", err)
	}
	
	if response == nil || response.Response == nil || response.Response.Price == nil {
		return nil, fmt.Errorf("empty or invalid pricing response from Tencent Cloud")
	}
	
	// Parse the pricing data
	price := response.Response.Price
	
	// Get instance price (hourly)
	var hourlyPrice float64
	if price.InstancePrice != nil && price.InstancePrice.UnitPrice != nil {
		hourlyPrice = *price.InstancePrice.UnitPrice
	} else {
		return nil, fmt.Errorf("instance price not found in response")
	}
	
	// Calculate monthly price (720 hours per month)
	monthlyPrice := hourlyPrice * 720
	
	// Get currency from response, default to CNY if not specified
	currency := "CNY"
	if price.InstancePrice.ChargeUnit != nil {
		// ChargeUnit is typically "HOUR" for hourly pricing
		// Currency is CNY for Tencent Cloud China regions
		currency = "CNY"
	}
	
	// Create PricingData structure
	pricingData := &PricingData{
		Provider:     "tencentcloud",
		Region:       actualRegion,
		ResourceType: resourceType,
		Currency:     currency,
		HourlyPrice:  hourlyPrice,
		MonthlyPrice: monthlyPrice,
		Metadata:     make(map[string]string),
	}
	
	// Add zone to metadata
	pricingData.Metadata["zone"] = actualZone
	
	// Add metadata if available
	if price.InstancePrice.OriginalPrice != nil {
		pricingData.Metadata["original_price"] = strconv.FormatFloat(*price.InstancePrice.OriginalPrice, 'f', -1, 64)
	}
	if price.InstancePrice.DiscountPrice != nil {
		pricingData.Metadata["discount_price"] = strconv.FormatFloat(*price.InstancePrice.DiscountPrice, 'f', -1, 64)
	}
	if price.InstancePrice.Discount != nil {
		pricingData.Metadata["discount"] = strconv.FormatFloat(*price.InstancePrice.Discount, 'f', -1, 64)
	}
	if price.InstancePrice.ChargeUnit != nil {
		pricingData.Metadata["charge_unit"] = *price.InstancePrice.ChargeUnit
	}
	
	return pricingData, nil
}
