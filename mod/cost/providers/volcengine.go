package providers

import (
	"fmt"
	"strconv"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// GetVolcenginePricing retrieves pricing data for Volcengine resources
// It uses the DescribeInstanceTypes API to fetch instance specifications and estimates pricing
func GetVolcenginePricing(region, resourceType, accessKey, secretKey string) (*PricingData, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing Volcengine access key or secret key")
	}
	
	if region == "" {
		region = "cn-beijing" // Default region
	}
	
	if resourceType == "" {
		return nil, fmt.Errorf("resource type cannot be empty")
	}
	
	// Create Volcengine session
	config := volcengine.NewConfig().
		WithRegion(region).
		WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, ""))
	
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Volcengine session: %w", err)
	}
	
	// Create ECS client
	client := ecs.New(sess)
	
	// Create DescribeInstanceTypes request
	request := &ecs.DescribeInstanceTypesInput{
		InstanceTypes: []*string{volcengine.String(resourceType)},
	}
	
	// Call DescribeInstanceTypes API
	response, err := client.DescribeInstanceTypes(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call DescribeInstanceTypes API: %w", err)
	}
	
	if response == nil || response.InstanceTypes == nil || len(response.InstanceTypes) == 0 {
		return nil, fmt.Errorf("instance type not found: %s", resourceType)
	}
	
	// Get instance type information
	instanceType := response.InstanceTypes[0]
	
	// Estimate pricing based on instance specifications
	// Note: Volcengine doesn't provide a direct pricing API like Alibaba Cloud or Tencent Cloud
	// We estimate based on CPU and memory specifications
	hourlyPrice := estimateVolcenginePricing(instanceType)
	monthlyPrice := hourlyPrice * 720 // 720 hours per month (30 days * 24 hours)
	
	// Create PricingData structure
	pricingData := &PricingData{
		Provider:     "volcengine",
		Region:       region,
		ResourceType: resourceType,
		Currency:     "CNY",
		HourlyPrice:  hourlyPrice,
		MonthlyPrice: monthlyPrice,
		Metadata:     make(map[string]string),
	}
	
	// Add metadata if available
	if instanceType.Processor != nil && instanceType.Processor.Model != nil {
		pricingData.Metadata["processor_model"] = *instanceType.Processor.Model
	}
	if instanceType.Processor != nil && instanceType.Processor.BaseFrequency != nil {
		pricingData.Metadata["base_frequency"] = fmt.Sprintf("%.2f", *instanceType.Processor.BaseFrequency)
	}
	if instanceType.Processor != nil && instanceType.Processor.Cpus != nil {
		pricingData.Metadata["cpus"] = strconv.FormatInt(int64(*instanceType.Processor.Cpus), 10)
	}
	if instanceType.Memory != nil && instanceType.Memory.Size != nil {
		pricingData.Metadata["memory_size"] = strconv.FormatInt(int64(*instanceType.Memory.Size), 10)
	}
	if instanceType.InstanceTypeFamily != nil {
		pricingData.Metadata["instance_family"] = *instanceType.InstanceTypeFamily
	}
	
	return pricingData, nil
}

// estimateVolcenginePricing estimates pricing based on instance specifications
// This is a simplified estimation model based on CPU and memory
func estimateVolcenginePricing(instanceType *ecs.InstanceTypeForDescribeInstanceTypesOutput) float64 {
	// Base pricing model (estimated):
	// - CPU: ~0.05 CNY per vCPU per hour
	// - Memory: ~0.02 CNY per GB per hour
	
	var cpus int32 = 1
	var memoryGB float64 = 1.0
	
	if instanceType.Processor != nil && instanceType.Processor.Cpus != nil {
		cpus = *instanceType.Processor.Cpus
	}
	
	if instanceType.Memory != nil && instanceType.Memory.Size != nil {
		// Memory size is in MB, convert to GB
		memoryGB = float64(*instanceType.Memory.Size) / 1024.0
	}
	
	// Estimate hourly price
	cpuPrice := float64(cpus) * 0.05
	memoryPrice := memoryGB * 0.02
	
	totalPrice := cpuPrice + memoryPrice
	
	// Apply instance family multiplier for different performance tiers
	if instanceType.InstanceTypeFamily != nil {
		family := *instanceType.InstanceTypeFamily
		multiplier := getInstanceFamilyMultiplier(family)
		totalPrice *= multiplier
	}
	
	return totalPrice
}

// getInstanceFamilyMultiplier returns a pricing multiplier based on instance family
func getInstanceFamilyMultiplier(family string) float64 {
	// Different instance families have different pricing
	// These are estimated multipliers based on typical cloud pricing patterns
	multipliers := map[string]float64{
		"ecs.g1":  1.0,  // General purpose (baseline)
		"ecs.g2":  1.1,  // General purpose (newer generation)
		"ecs.g3":  1.2,  // General purpose (latest generation)
		"ecs.c1":  1.1,  // Compute optimized
		"ecs.c2":  1.2,  // Compute optimized (newer)
		"ecs.c3":  1.3,  // Compute optimized (latest)
		"ecs.r1":  1.3,  // Memory optimized
		"ecs.r2":  1.4,  // Memory optimized (newer)
		"ecs.r3":  1.5,  // Memory optimized (latest)
		"ecs.i1":  1.4,  // Storage optimized
		"ecs.i2":  1.5,  // Storage optimized (newer)
		"ecs.gn":  2.0,  // GPU instances
		"ecs.vgn": 1.8,  // GPU instances (virtualized)
	}
	
	// Check for exact match
	if multiplier, ok := multipliers[family]; ok {
		return multiplier
	}
	
	// Check for prefix match (e.g., "ecs.g3a" matches "ecs.g3")
	for prefix, multiplier := range multipliers {
		if len(family) >= len(prefix) && family[:len(prefix)] == prefix {
			return multiplier
		}
	}
	
	// Default multiplier
	return 1.0
}
