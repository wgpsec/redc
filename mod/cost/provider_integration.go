package cost

import (
	"fmt"
	
	"red-cloud/mod/cost/providers"
)

// getAlicloudPricing fetches Alibaba Cloud pricing using the providers package
func (ps *PricingService) getAlicloudPricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	// Call the providers package function
	providerData, err := providers.GetAlicloudPricing(region, resourceType, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	
	if providerData == nil {
		return nil, fmt.Errorf("received nil pricing data from Alibaba Cloud provider")
	}
	
	// Convert providers.PricingData to cost.PricingData
	// They have the same structure, so we can copy fields
	return &PricingData{
		Provider:     providerData.Provider,
		Region:       providerData.Region,
		ResourceType: providerData.ResourceType,
		Currency:     providerData.Currency,
		HourlyPrice:  providerData.HourlyPrice,
		MonthlyPrice: providerData.MonthlyPrice,
		PricingTiers: convertPricingTiers(providerData.PricingTiers),
		Metadata:     providerData.Metadata,
	}, nil
}

// getTencentcloudPricing fetches Tencent Cloud pricing using the providers package
func (ps *PricingService) getTencentcloudPricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	// Call the providers package function
	providerData, err := providers.GetTencentcloudPricing(region, resourceType, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	
	if providerData == nil {
		return nil, fmt.Errorf("received nil pricing data from Tencent Cloud provider")
	}
	
	// Convert providers.PricingData to cost.PricingData
	return &PricingData{
		Provider:     providerData.Provider,
		Region:       providerData.Region,
		ResourceType: providerData.ResourceType,
		Currency:     providerData.Currency,
		HourlyPrice:  providerData.HourlyPrice,
		MonthlyPrice: providerData.MonthlyPrice,
		PricingTiers: convertPricingTiers(providerData.PricingTiers),
		Metadata:     providerData.Metadata,
	}, nil
}

// getAWSPricing fetches AWS pricing using the providers package
func (ps *PricingService) getAWSPricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	// Call the providers package function
	providerData, err := providers.GetAWSPricing(region, resourceType, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	
	if providerData == nil {
		return nil, fmt.Errorf("received nil pricing data from AWS provider")
	}
	
	// Convert providers.PricingData to cost.PricingData
	return &PricingData{
		Provider:     providerData.Provider,
		Region:       providerData.Region,
		ResourceType: providerData.ResourceType,
		Currency:     providerData.Currency,
		HourlyPrice:  providerData.HourlyPrice,
		MonthlyPrice: providerData.MonthlyPrice,
		PricingTiers: convertPricingTiers(providerData.PricingTiers),
		Metadata:     providerData.Metadata,
	}, nil
}

// getVolcenginePricing fetches Volcengine pricing using the providers package
func (ps *PricingService) getVolcenginePricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	// Call the providers package function
	providerData, err := providers.GetVolcenginePricing(region, resourceType, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	
	if providerData == nil {
		return nil, fmt.Errorf("received nil pricing data from Volcengine provider")
	}
	
	// Convert providers.PricingData to cost.PricingData
	return &PricingData{
		Provider:     providerData.Provider,
		Region:       providerData.Region,
		ResourceType: providerData.ResourceType,
		Currency:     providerData.Currency,
		HourlyPrice:  providerData.HourlyPrice,
		MonthlyPrice: providerData.MonthlyPrice,
		PricingTiers: convertPricingTiers(providerData.PricingTiers),
		Metadata:     providerData.Metadata,
	}, nil
}

// convertPricingTiers converts providers.PricingTier to cost.PricingTier
func convertPricingTiers(providerTiers []providers.PricingTier) []PricingTier {
	if providerTiers == nil {
		return nil
	}
	
	tiers := make([]PricingTier, len(providerTiers))
	for i, pt := range providerTiers {
		tiers[i] = PricingTier{
			MinUnits:     pt.MinUnits,
			MaxUnits:     pt.MaxUnits,
			PricePerUnit: pt.PricePerUnit,
		}
	}
	return tiers
}
