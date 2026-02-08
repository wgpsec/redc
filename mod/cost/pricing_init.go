package cost

import (
	"fmt"
	"path/filepath"

	"red-cloud/mod/cost/providers"
)

// convertProviderPricingData converts providers.PricingData to cost.PricingData
func convertProviderPricingData(src *providers.PricingData) *PricingData {
	if src == nil {
		return nil
	}

	dst := &PricingData{
		Provider:     src.Provider,
		Region:       src.Region,
		ResourceType: src.ResourceType,
		Currency:     src.Currency,
		HourlyPrice:  src.HourlyPrice,
		MonthlyPrice: src.MonthlyPrice,
		Metadata:     src.Metadata,
	}

	// Convert pricing tiers if present
	if len(src.PricingTiers) > 0 {
		dst.PricingTiers = make([]PricingTier, len(src.PricingTiers))
		for i, tier := range src.PricingTiers {
			dst.PricingTiers[i] = PricingTier{
				MinUnits:     tier.MinUnits,
				MaxUnits:     tier.MaxUnits,
				PricePerUnit: tier.PricePerUnit,
			}
		}
	}

	return dst
}

// InitializeFallbackPricing loads the fallback pricing database and sets up the fallback provider
func InitializeFallbackPricing(ps *PricingService, fallbackFilePath string) error {
	// Load the fallback data from JSON file
	if err := providers.LoadFallbackData(fallbackFilePath); err != nil {
		return fmt.Errorf("failed to load fallback pricing data: %w", err)
	}

	// Set the fallback provider function with type conversion
	ps.SetFallbackProvider(func(provider, region, resourceType string) (*PricingData, error) {
		providerData, err := providers.GetFallbackPricing(provider, region, resourceType)
		if err != nil {
			return nil, err
		}
		return convertProviderPricingData(providerData), nil
	})

	return nil
}

// NewPricingServiceWithFallback creates a new pricing service with fallback pricing enabled
func NewPricingServiceWithFallback(dbPath string, fallbackFilePath string) (*PricingService, error) {
	ps := NewPricingService(dbPath)

	// If fallbackFilePath is empty, try to find it relative to the database path
	if fallbackFilePath == "" {
		dbDir := filepath.Dir(dbPath)
		fallbackFilePath = filepath.Join(dbDir, "..", "mod", "cost", "providers", "pricing_fallback.json")
	}

	// Initialize fallback pricing
	if err := InitializeFallbackPricing(ps, fallbackFilePath); err != nil {
		// Log warning but don't fail - service can work without fallback
		fmt.Printf("Warning: Failed to initialize fallback pricing: %v\n", err)
	}

	return ps, nil
}
