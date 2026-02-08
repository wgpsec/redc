package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// PricingData represents pricing information for a cloud resource
// This is a local copy to avoid circular imports
type PricingData struct {
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	ResourceType string            `json:"resource_type"`
	Currency     string            `json:"currency"`
	HourlyPrice  float64           `json:"hourly_price"`
	MonthlyPrice float64           `json:"monthly_price"`
	PricingTiers []PricingTier     `json:"pricing_tiers,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PricingTier represents tiered pricing structure
type PricingTier struct {
	MinUnits     int     `json:"min_units"`
	MaxUnits     int     `json:"max_units"`
	PricePerUnit float64 `json:"price_per_unit"`
}

// FallbackDatabase represents the structure of the fallback pricing JSON file
type FallbackDatabase struct {
	Version     string                                       `json:"version"`
	LastUpdated string                                       `json:"last_updated"`
	Pricing     map[string]map[string]map[string]*PricingData `json:"pricing"`
}

var (
	fallbackDB   *FallbackDatabase
	fallbackOnce sync.Once
	fallbackErr  error
)

// GetFallbackPricing retrieves pricing data from the fallback database
func GetFallbackPricing(provider, region, resourceType string) (*PricingData, error) {
	if fallbackDB == nil {
		return nil, fmt.Errorf("fallback database not loaded")
	}

	// Navigate through the nested map structure
	providerData, ok := fallbackDB.Pricing[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found in fallback database", provider)
	}

	regionData, ok := providerData[region]
	if !ok {
		return nil, fmt.Errorf("region %s not found for provider %s in fallback database", region, provider)
	}

	pricingData, ok := regionData[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource type %s not found for provider %s, region %s in fallback database", resourceType, provider, region)
	}

	return pricingData, nil
}

// LoadFallbackData loads the fallback pricing database from JSON file
// This function uses sync.Once to ensure the data is loaded only once
func LoadFallbackData(filePath string) error {
	fallbackOnce.Do(func() {
		// Read the JSON file
		data, err := os.ReadFile(filePath)
		if err != nil {
			fallbackErr = fmt.Errorf("failed to read fallback pricing file: %w", err)
			return
		}

		// Parse the JSON data
		var db FallbackDatabase
		if err := json.Unmarshal(data, &db); err != nil {
			fallbackErr = fmt.Errorf("failed to parse fallback pricing JSON: %w", err)
			return
		}

		fallbackDB = &db
		fallbackErr = nil
	})

	return fallbackErr
}

// ResetFallbackData resets the fallback database (useful for testing)
func ResetFallbackData() {
	fallbackDB = nil
	fallbackOnce = sync.Once{}
	fallbackErr = nil
}
