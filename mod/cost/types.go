package cost

// PricingData represents pricing information for a cloud resource
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
