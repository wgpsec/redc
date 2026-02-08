package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// GetAWSPricing retrieves pricing data for AWS resources
// It uses the AWS Price List API to fetch real-time pricing information
func GetAWSPricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing AWS access key or secret key")
	}
	
	if region == "" {
		region = "us-east-1" // Default region
	}
	
	if resourceType == "" {
		return nil, fmt.Errorf("resource type cannot be empty")
	}
	
	// Create AWS config with credentials
	// Note: AWS Pricing API is only available in us-east-1 and ap-south-1
	// We use us-east-1 for the pricing API endpoint
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"), // Pricing API region
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Create pricing client
	client := pricing.NewFromConfig(cfg)
	
	// Build filters for the pricing query
	// For EC2 instances, we need to filter by instance type, region, and other attributes
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("ServiceCode"),
			Value: aws.String("AmazonEC2"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("instanceType"),
			Value: aws.String(resourceType),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("location"),
			Value: aws.String(getAWSLocationName(region)),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("tenancy"),
			Value: aws.String("Shared"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("operatingSystem"),
			Value: aws.String("Linux"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("preInstalledSw"),
			Value: aws.String("NA"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("capacitystatus"),
			Value: aws.String("Used"),
		},
	}
	
	// Call GetProducts API
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
		MaxResults:  aws.Int32(1), // We only need one result
	}
	
	response, err := client.GetProducts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to call AWS Pricing API: %w", err)
	}
	
	if response == nil || len(response.PriceList) == 0 {
		return nil, fmt.Errorf("no pricing data found for instance type %s in region %s", resourceType, region)
	}
	
	// Parse the pricing data from the first result
	priceListJSON := response.PriceList[0]
	
	// Parse the JSON response
	var priceData map[string]interface{}
	if err := json.Unmarshal([]byte(priceListJSON), &priceData); err != nil {
		return nil, fmt.Errorf("failed to parse pricing JSON: %w", err)
	}
	
	// Extract pricing information from the nested structure
	// AWS pricing JSON has a complex structure: product -> terms -> OnDemand -> priceDimensions
	terms, ok := priceData["terms"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid pricing data structure: missing terms")
	}
	
	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid pricing data structure: missing OnDemand terms")
	}
	
	// Get the first (and usually only) offer term
	var hourlyPrice float64
	var currency string
	found := false
	
	for _, offerTerm := range onDemand {
		offerTermMap, ok := offerTerm.(map[string]interface{})
		if !ok {
			continue
		}
		
		priceDimensions, ok := offerTermMap["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}
		
		// Get the first price dimension
		for _, priceDim := range priceDimensions {
			priceDimMap, ok := priceDim.(map[string]interface{})
			if !ok {
				continue
			}
			
			pricePerUnit, ok := priceDimMap["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}
			
			// Extract the price and currency
			for curr, priceStr := range pricePerUnit {
				currency = curr
				priceStrVal, ok := priceStr.(string)
				if !ok {
					continue
				}
				
				price, err := strconv.ParseFloat(priceStrVal, 64)
				if err != nil {
					continue
				}
				
				hourlyPrice = price
				found = true
				break
			}
			
			if found {
				break
			}
		}
		
		if found {
			break
		}
	}
	
	if !found {
		return nil, fmt.Errorf("could not extract pricing information from AWS response")
	}
	
	// Calculate monthly price (720 hours per month)
	monthlyPrice := hourlyPrice * 720
	
	// Create PricingData structure
	pricingData := &PricingData{
		Provider:     "aws",
		Region:       region,
		ResourceType: resourceType,
		Currency:     currency,
		HourlyPrice:  hourlyPrice,
		MonthlyPrice: monthlyPrice,
		Metadata:     make(map[string]string),
	}
	
	// Add metadata from product attributes if available
	if product, ok := priceData["product"].(map[string]interface{}); ok {
		if attributes, ok := product["attributes"].(map[string]interface{}); ok {
			if vcpu, ok := attributes["vcpu"].(string); ok {
				pricingData.Metadata["vcpu"] = vcpu
			}
			if memory, ok := attributes["memory"].(string); ok {
				pricingData.Metadata["memory"] = memory
			}
			if storage, ok := attributes["storage"].(string); ok {
				pricingData.Metadata["storage"] = storage
			}
			if networkPerformance, ok := attributes["networkPerformance"].(string); ok {
				pricingData.Metadata["network_performance"] = networkPerformance
			}
		}
	}
	
	return pricingData, nil
}

// getAWSLocationName converts AWS region code to location name used in pricing API
// The AWS Pricing API uses location names instead of region codes
func getAWSLocationName(region string) string {
	locationMap := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"ca-central-1":   "Canada (Central)",
		"eu-central-1":   "EU (Frankfurt)",
		"eu-west-1":      "EU (Ireland)",
		"eu-west-2":      "EU (London)",
		"eu-west-3":      "EU (Paris)",
		"eu-north-1":     "EU (Stockholm)",
		"eu-south-1":     "EU (Milan)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-northeast-2": "Asia Pacific (Seoul)",
		"ap-northeast-3": "Asia Pacific (Osaka)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
		"sa-east-1":      "South America (Sao Paulo)",
		"me-south-1":     "Middle East (Bahrain)",
		"af-south-1":     "Africa (Cape Town)",
	}
	
	if locationName, ok := locationMap[region]; ok {
		return locationName
	}
	
	// If region not found in map, return the region code itself
	// This might not work but it's better than failing
	return region
}
