package cost

import (
	"fmt"
	"strings"
	"time"
)

// CostEstimate represents the complete cost breakdown
type CostEstimate struct {
	TotalHourlyCost    float64                         `json:"total_hourly_cost"`
	TotalMonthlyCost   float64                         `json:"total_monthly_cost"`
	Currency           string                          `json:"currency"`
	Breakdown          []ResourceCostBreakdown         `json:"breakdown"`
	ProviderBreakdown  map[string]*ProviderCostSummary `json:"provider_breakdown,omitempty"`
	UnavailableCount   int                             `json:"unavailable_count"`
	Timestamp          time.Time                       `json:"timestamp"`
	Disclaimer         string                          `json:"disclaimer"`
	Warnings           []string                        `json:"warnings,omitempty"`
}

// ProviderCostSummary represents aggregated costs for a single provider
type ProviderCostSummary struct {
	Provider         string  `json:"provider"`
	TotalHourlyCost  float64 `json:"total_hourly_cost"`
	TotalMonthlyCost float64 `json:"total_monthly_cost"`
	Currency         string  `json:"currency"`
	ResourceCount    int     `json:"resource_count"`
}

// ResourceCostBreakdown represents cost breakdown for a single resource
type ResourceCostBreakdown struct {
	ResourceType string  `json:"resource_type"`
	ResourceName string  `json:"resource_name"`
	Provider     string  `json:"provider"`
	Count        int     `json:"count"`
	UnitHourly   float64 `json:"unit_hourly"`
	UnitMonthly  float64 `json:"unit_monthly"`
	TotalHourly  float64 `json:"total_hourly"`
	TotalMonthly float64 `json:"total_monthly"`
	Currency     string  `json:"currency"`
	Available    bool    `json:"available"` // false if pricing unavailable
}

// CostCalculator computes cost estimates from resource specifications
type CostCalculator struct {
	// TODO: Add fields in task 6
}

// NewCostCalculator creates a new cost calculator instance
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{}
}

// CalculateCost computes cost estimates for a set of resources
func (cc *CostCalculator) CalculateCost(resources *TemplateResources, pricingService *PricingService) (*CostEstimate, error) {
	estimate := &CostEstimate{
		Timestamp:         time.Now(),
		Disclaimer:        "This is an estimate only. Actual costs may vary based on usage, region, and pricing changes.",
		Warnings:          []string{},
		Breakdown:         []ResourceCostBreakdown{},
		ProviderBreakdown: make(map[string]*ProviderCostSummary),
		UnavailableCount:  0,
	}

	// Process each resource
	for _, resource := range resources.Resources {
		breakdown := cc.calculateResourceCost(resource, pricingService)
		estimate.Breakdown = append(estimate.Breakdown, breakdown)

		// Add to totals if pricing is available
		if breakdown.Available {
			estimate.TotalHourlyCost += breakdown.TotalHourly
			estimate.TotalMonthlyCost += breakdown.TotalMonthly

			// Set currency from first available resource
			if estimate.Currency == "" {
				estimate.Currency = breakdown.Currency
			}

			// Aggregate by provider
			provider := breakdown.Provider
			if provider == "" {
				provider = "unknown"
			}

			if _, exists := estimate.ProviderBreakdown[provider]; !exists {
				estimate.ProviderBreakdown[provider] = &ProviderCostSummary{
					Provider:         provider,
					TotalHourlyCost:  0,
					TotalMonthlyCost: 0,
					Currency:         breakdown.Currency,
					ResourceCount:    0,
				}
			}

			estimate.ProviderBreakdown[provider].TotalHourlyCost += breakdown.TotalHourly
			estimate.ProviderBreakdown[provider].TotalMonthlyCost += breakdown.TotalMonthly
			estimate.ProviderBreakdown[provider].ResourceCount++
		} else {
			// Track unavailable pricing
			estimate.UnavailableCount++

			// Add warning for unavailable pricing
			estimate.Warnings = append(estimate.Warnings,
				fmt.Sprintf("Pricing unavailable for %s (%s)", breakdown.ResourceName, breakdown.ResourceType))
		}
	}

	// Set default currency if none found
	if estimate.Currency == "" {
		estimate.Currency = "USD"
	}

	return estimate, nil
}

// calculateResourceCost calculates cost for a single resource
func (cc *CostCalculator) calculateResourceCost(resource ResourceSpec, pricingService *PricingService) ResourceCostBreakdown {
	breakdown := ResourceCostBreakdown{
		ResourceType: resource.Type,
		ResourceName: resource.Name,
		Provider:     resource.Provider,
		Count:        resource.Count,
		Available:    false,
	}

	// Determine the resource type to use for pricing lookup
	// For compute instances, we need to extract the actual instance type from attributes
	pricingResourceType := resource.Type
	
	// Extract instance type from attributes for compute resources
	if resource.Type == "alicloud_instance" || resource.Type == "aws_instance" || resource.Type == "tencentcloud_instance" || resource.Type == "volcengine_ecs_instance" {
		if instanceType, ok := resource.Attributes["instance_type"].(string); ok && instanceType != "" {
			// Check if instance_type contains unresolved expressions
			// Unresolved expressions typically contain "${", "data.", or other Terraform syntax
			if strings.Contains(instanceType, "${") || 
			   strings.Contains(instanceType, "data.") ||
			   strings.Contains(instanceType, "local.") ||
			   strings.Contains(instanceType, "module.") {
				// Instance type contains unresolved expression - pricing unavailable
				return breakdown
			}
			pricingResourceType = instanceType
		} else {
			// Instance type not found in attributes - pricing unavailable
			return breakdown
		}
	} else {
		// For non-compute resources (security groups, networks, key pairs, etc.), 
		// pricing is typically not available through instance pricing APIs
		// Return early to avoid unnecessary API calls
		nonComputeResources := []string{
			// Alibaba Cloud non-compute resources
			"alicloud_security_group", "alicloud_security_group_rule",
			"alicloud_vpc", "alicloud_vswitch",
			// AWS non-compute resources
			"aws_security_group", "aws_security_group_rule",
			"aws_vpc", "aws_subnet",
			"aws_key_pair", "aws_eip",
			// Tencent Cloud non-compute resources
			"tencentcloud_security_group", "tencentcloud_security_group_rule",
			"tencentcloud_vpc", "tencentcloud_subnet",
			// Volcengine non-compute resources
			"volcengine_eip_address", "volcengine_eip_associate",
			"volcengine_security_group", "volcengine_security_group_rule",
			"volcengine_vpc", "volcengine_subnet",
			// Non-cloud providers (no pricing available)
			"tls_private_key", "tls_cert_request", "tls_locally_signed_cert", "tls_self_signed_cert",
			"local_file", "local_sensitive_file",
			"random_id", "random_string", "random_password",
			"null_resource",
		}
		
		for _, nonCompute := range nonComputeResources {
			if resource.Type == nonCompute {
				// Pricing not available for this resource type
				return breakdown
			}
		}
	}
	
	// Get pricing data for this resource
	// For Tencent Cloud, use availability_zone if available
	regionOrZone := resource.Region
	if resource.Provider == "tencentcloud" {
		if zone, ok := resource.Attributes["availability_zone"].(string); ok && zone != "" {
			regionOrZone = zone
		}
	}
	
	pricing, err := pricingService.GetPricing(resource.Provider, regionOrZone, pricingResourceType)
	if err != nil || pricing == nil {
		// Pricing unavailable
		return breakdown
	}

	// Mark as available
	breakdown.Available = true
	breakdown.Currency = pricing.Currency

	// Check if tiered pricing is available
	if len(pricing.PricingTiers) > 0 {
		// Use tiered pricing calculation
		breakdown.UnitHourly = cc.calculateTieredPrice(resource.Count, pricing.PricingTiers)
		breakdown.UnitMonthly = breakdown.UnitHourly * 720 // 720 hours per month (30 days * 24 hours)
		
		// For tiered pricing, total is already calculated per unit based on quantity
		breakdown.TotalHourly = breakdown.UnitHourly * float64(resource.Count)
		breakdown.TotalMonthly = breakdown.UnitMonthly * float64(resource.Count)
	} else {
		// Use simple flat pricing
		breakdown.UnitHourly = pricing.HourlyPrice
		breakdown.UnitMonthly = pricing.HourlyPrice * 720 // 720 hours per month (30 days * 24 hours)

		// Calculate total costs (multiply by count)
		breakdown.TotalHourly = breakdown.UnitHourly * float64(resource.Count)
		breakdown.TotalMonthly = breakdown.UnitMonthly * float64(resource.Count)
	}

	return breakdown
}

// calculateTieredPrice calculates the effective price per unit based on tiered pricing
// For a given quantity, it finds the appropriate tier and returns the price per unit for that tier
func (cc *CostCalculator) calculateTieredPrice(quantity int, tiers []PricingTier) float64 {
	if len(tiers) == 0 {
		return 0
	}

	// Find the appropriate tier for the given quantity
	for _, tier := range tiers {
		// Check if quantity falls within this tier
		// MaxUnits of 0 or -1 typically means unlimited/no upper bound
		if quantity >= tier.MinUnits && (tier.MaxUnits == 0 || tier.MaxUnits == -1 || quantity <= tier.MaxUnits) {
			return tier.PricePerUnit
		}
	}

	// If no tier matches, use the last tier (highest tier)
	// This handles cases where quantity exceeds all defined tiers
	return tiers[len(tiers)-1].PricePerUnit
}
