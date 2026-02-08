package cost

import (
	"fmt"
	"testing"
)

// TestCalculateCost_SingleResource tests cost calculation for a single resource
func TestCalculateCost_SingleResource(t *testing.T) {
	// Create a mock pricing service
	ps := NewPricingService(":memory:")
	defer ps.Close()

	// Set up fallback provider with known pricing
	ps.SetFallbackProvider(func(provider, region, resourceType string) (*PricingData, error) {
		return &PricingData{
			Provider:     provider,
			Region:       region,
			ResourceType: resourceType,
			Currency:     "USD",
			HourlyPrice:  0.5,
			MonthlyPrice: 360.0, // This should be ignored, we calculate from hourly
		}, nil
	})

	// Create calculator
	calc := NewCostCalculator()

	// Create test resources
	resources := &TemplateResources{
		Provider: "aws",
		Region:   "us-east-1",
		Resources: []ResourceSpec{
			{
				Type:     "t2.micro",
				Name:     "test-instance",
				Count:    1,
				Provider: "aws",
				Region:   "us-east-1",
			},
		},
	}

	// Calculate cost
	estimate, err := calc.CalculateCost(resources, ps)
	if err != nil {
		t.Fatalf("CalculateCost failed: %v", err)
	}

	// Verify results
	if estimate == nil {
		t.Fatal("Expected non-nil estimate")
	}

	// Check hourly cost
	expectedHourly := 0.5
	if estimate.TotalHourlyCost != expectedHourly {
		t.Errorf("Expected hourly cost %f, got %f", expectedHourly, estimate.TotalHourlyCost)
	}

	// Check monthly cost (hourly × 720)
	expectedMonthly := 0.5 * 720
	if estimate.TotalMonthlyCost != expectedMonthly {
		t.Errorf("Expected monthly cost %f, got %f", expectedMonthly, estimate.TotalMonthlyCost)
	}

	// Check currency
	if estimate.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", estimate.Currency)
	}

	// Check breakdown
	if len(estimate.Breakdown) != 1 {
		t.Fatalf("Expected 1 breakdown entry, got %d", len(estimate.Breakdown))
	}

	breakdown := estimate.Breakdown[0]
	if !breakdown.Available {
		t.Error("Expected breakdown to be available")
	}
	if breakdown.UnitHourly != 0.5 {
		t.Errorf("Expected unit hourly %f, got %f", 0.5, breakdown.UnitHourly)
	}
	if breakdown.UnitMonthly != 360.0 {
		t.Errorf("Expected unit monthly %f, got %f", 360.0, breakdown.UnitMonthly)
	}
}

// TestCalculateCost_MultipleResources tests cost calculation with multiple resources
func TestCalculateCost_MultipleResources(t *testing.T) {
	ps := NewPricingService(":memory:")
	defer ps.Close()

	// Set up fallback provider with different pricing for different types
	ps.SetFallbackProvider(func(provider, region, resourceType string) (*PricingData, error) {
		prices := map[string]float64{
			"t2.micro":  0.5,
			"t2.small":  1.0,
			"t2.medium": 2.0,
		}
		return &PricingData{
			Provider:     provider,
			Region:       region,
			ResourceType: resourceType,
			Currency:     "USD",
			HourlyPrice:  prices[resourceType],
		}, nil
	})

	calc := NewCostCalculator()

	resources := &TemplateResources{
		Provider: "aws",
		Region:   "us-east-1",
		Resources: []ResourceSpec{
			{Type: "t2.micro", Name: "instance-1", Count: 1, Provider: "aws", Region: "us-east-1"},
			{Type: "t2.small", Name: "instance-2", Count: 1, Provider: "aws", Region: "us-east-1"},
			{Type: "t2.medium", Name: "instance-3", Count: 1, Provider: "aws", Region: "us-east-1"},
		},
	}

	estimate, err := calc.CalculateCost(resources, ps)
	if err != nil {
		t.Fatalf("CalculateCost failed: %v", err)
	}

	// Total hourly should be 0.5 + 1.0 + 2.0 = 3.5
	expectedHourly := 3.5
	if estimate.TotalHourlyCost != expectedHourly {
		t.Errorf("Expected hourly cost %f, got %f", expectedHourly, estimate.TotalHourlyCost)
	}

	// Total monthly should be 3.5 × 720 = 2520
	expectedMonthly := 3.5 * 720
	if estimate.TotalMonthlyCost != expectedMonthly {
		t.Errorf("Expected monthly cost %f, got %f", expectedMonthly, estimate.TotalMonthlyCost)
	}

	// Check breakdown count
	if len(estimate.Breakdown) != 3 {
		t.Fatalf("Expected 3 breakdown entries, got %d", len(estimate.Breakdown))
	}
}

// TestCalculateCost_ResourceCount tests cost multiplication by resource count
func TestCalculateCost_ResourceCount(t *testing.T) {
	ps := NewPricingService(":memory:")
	defer ps.Close()

	ps.SetFallbackProvider(func(provider, region, resourceType string) (*PricingData, error) {
		return &PricingData{
			Provider:     provider,
			Region:       region,
			ResourceType: resourceType,
			Currency:     "USD",
			HourlyPrice:  1.0,
		}, nil
	})

	calc := NewCostCalculator()

	// Create resource with count = 5
	resources := &TemplateResources{
		Provider: "aws",
		Region:   "us-east-1",
		Resources: []ResourceSpec{
			{
				Type:     "t2.micro",
				Name:     "test-instances",
				Count:    5, // 5 instances
				Provider: "aws",
				Region:   "us-east-1",
			},
		},
	}

	estimate, err := calc.CalculateCost(resources, ps)
	if err != nil {
		t.Fatalf("CalculateCost failed: %v", err)
	}

	// Hourly cost should be 1.0 × 5 = 5.0
	expectedHourly := 5.0
	if estimate.TotalHourlyCost != expectedHourly {
		t.Errorf("Expected hourly cost %f, got %f", expectedHourly, estimate.TotalHourlyCost)
	}

	// Monthly cost should be 5.0 × 720 = 3600
	expectedMonthly := 5.0 * 720
	if estimate.TotalMonthlyCost != expectedMonthly {
		t.Errorf("Expected monthly cost %f, got %f", expectedMonthly, estimate.TotalMonthlyCost)
	}

	// Check breakdown
	breakdown := estimate.Breakdown[0]
	if breakdown.Count != 5 {
		t.Errorf("Expected count 5, got %d", breakdown.Count)
	}
	if breakdown.TotalHourly != 5.0 {
		t.Errorf("Expected total hourly %f, got %f", 5.0, breakdown.TotalHourly)
	}
}

// TestCalculateCost_UnavailablePricing tests handling of unavailable pricing
func TestCalculateCost_UnavailablePricing(t *testing.T) {
	ps := NewPricingService(":memory:")
	defer ps.Close()

	// Don't set fallback provider - pricing will be unavailable

	calc := NewCostCalculator()

	resources := &TemplateResources{
		Provider: "aws",
		Region:   "us-east-1",
		Resources: []ResourceSpec{
			{
				Type:     "unknown-type",
				Name:     "test-instance",
				Count:    1,
				Provider: "aws",
				Region:   "us-east-1",
			},
		},
	}

	estimate, err := calc.CalculateCost(resources, ps)
	if err != nil {
		t.Fatalf("CalculateCost failed: %v", err)
	}

	// Total costs should be 0 when pricing unavailable
	if estimate.TotalHourlyCost != 0 {
		t.Errorf("Expected hourly cost 0, got %f", estimate.TotalHourlyCost)
	}
	if estimate.TotalMonthlyCost != 0 {
		t.Errorf("Expected monthly cost 0, got %f", estimate.TotalMonthlyCost)
	}

	// Check breakdown
	if len(estimate.Breakdown) != 1 {
		t.Fatalf("Expected 1 breakdown entry, got %d", len(estimate.Breakdown))
	}

	breakdown := estimate.Breakdown[0]
	if breakdown.Available {
		t.Error("Expected breakdown to be unavailable")
	}

	// Check warnings
	if len(estimate.Warnings) == 0 {
		t.Error("Expected warnings for unavailable pricing")
	}
}

// TestCalculateCost_MixedAvailability tests partial pricing availability
func TestCalculateCost_MixedAvailability(t *testing.T) {
	ps := NewPricingService(":memory:")
	defer ps.Close()

	// Set up fallback provider that only has pricing for t2.micro
	ps.SetFallbackProvider(func(provider, region, resourceType string) (*PricingData, error) {
		if resourceType == "t2.micro" {
			return &PricingData{
				Provider:     provider,
				Region:       region,
				ResourceType: resourceType,
				Currency:     "USD",
				HourlyPrice:  0.5,
			}, nil
		}
		return nil, fmt.Errorf("pricing not available")
	})

	calc := NewCostCalculator()

	resources := &TemplateResources{
		Provider: "aws",
		Region:   "us-east-1",
		Resources: []ResourceSpec{
			{Type: "t2.micro", Name: "instance-1", Count: 1, Provider: "aws", Region: "us-east-1"},
			{Type: "unknown-type", Name: "instance-2", Count: 1, Provider: "aws", Region: "us-east-1"},
		},
	}

	estimate, err := calc.CalculateCost(resources, ps)
	if err != nil {
		t.Fatalf("CalculateCost failed: %v", err)
	}

	// Total should only include available pricing (0.5)
	expectedHourly := 0.5
	if estimate.TotalHourlyCost != expectedHourly {
		t.Errorf("Expected hourly cost %f, got %f", expectedHourly, estimate.TotalHourlyCost)
	}

	// Check that we have 2 breakdown entries
	if len(estimate.Breakdown) != 2 {
		t.Fatalf("Expected 2 breakdown entries, got %d", len(estimate.Breakdown))
	}

	// First should be available, second unavailable
	if !estimate.Breakdown[0].Available {
		t.Error("Expected first breakdown to be available")
	}
	if estimate.Breakdown[1].Available {
		t.Error("Expected second breakdown to be unavailable")
	}

	// Check warnings
	if len(estimate.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(estimate.Warnings))
	}
}

// TestCalculateCost_EmptyResources tests handling of empty resource list
func TestCalculateCost_EmptyResources(t *testing.T) {
	ps := NewPricingService(":memory:")
	defer ps.Close()

	calc := NewCostCalculator()

	resources := &TemplateResources{
		Provider:  "aws",
		Region:    "us-east-1",
		Resources: []ResourceSpec{},
	}

	estimate, err := calc.CalculateCost(resources, ps)
	if err != nil {
		t.Fatalf("CalculateCost failed: %v", err)
	}

	// All costs should be 0
	if estimate.TotalHourlyCost != 0 {
		t.Errorf("Expected hourly cost 0, got %f", estimate.TotalHourlyCost)
	}
	if estimate.TotalMonthlyCost != 0 {
		t.Errorf("Expected monthly cost 0, got %f", estimate.TotalMonthlyCost)
	}

	// Breakdown should be empty
	if len(estimate.Breakdown) != 0 {
		t.Errorf("Expected 0 breakdown entries, got %d", len(estimate.Breakdown))
	}
}

