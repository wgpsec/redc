package cost

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPricingService_InitDB tests database initialization
func TestPricingService_InitDB(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	// Create pricing service
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Verify database was created
	if ps.db == nil {
		t.Fatal("Database connection should not be nil")
	}
	
	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file should exist at %s", dbPath)
	}
	
	// Verify table exists by querying it
	var count int
	err := ps.db.QueryRow("SELECT COUNT(*) FROM pricing_cache").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query pricing_cache table: %v", err)
	}
	
	if count != 0 {
		t.Errorf("Expected empty table, got %d rows", count)
	}
}

// TestPricingService_CachePricing tests caching pricing data
func TestPricingService_CachePricing(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Create test pricing data
	testData := &PricingData{
		Provider:     "alicloud",
		Region:       "cn-hangzhou",
		ResourceType: "ecs.g6.large",
		Currency:     "CNY",
		HourlyPrice:  1.5,
		MonthlyPrice: 1080.0,
		Metadata: map[string]string{
			"cpu":    "4",
			"memory": "16GB",
		},
	}
	
	// Cache the pricing data
	err := ps.cachePricing(testData)
	if err != nil {
		t.Fatalf("Failed to cache pricing data: %v", err)
	}
	
	// Verify data was cached by querying directly
	var count int
	err = ps.db.QueryRow("SELECT COUNT(*) FROM pricing_cache WHERE provider = ? AND region = ? AND resource_type = ?",
		testData.Provider, testData.Region, testData.ResourceType).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query cached data: %v", err)
	}
	
	if count != 1 {
		t.Errorf("Expected 1 cached entry, got %d", count)
	}
}

// TestPricingService_GetCachedPricing tests retrieving cached pricing data
func TestPricingService_GetCachedPricing(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Create and cache test pricing data
	testData := &PricingData{
		Provider:     "aws",
		Region:       "us-east-1",
		ResourceType: "t2.micro",
		Currency:     "USD",
		HourlyPrice:  0.0116,
		MonthlyPrice: 8.352,
		PricingTiers: []PricingTier{
			{MinUnits: 0, MaxUnits: 100, PricePerUnit: 0.0116},
			{MinUnits: 101, MaxUnits: 1000, PricePerUnit: 0.01},
		},
	}
	
	err := ps.cachePricing(testData)
	if err != nil {
		t.Fatalf("Failed to cache pricing data: %v", err)
	}
	
	// Retrieve cached data
	cachedData, err := ps.getCachedPricing(testData.Provider, testData.Region, testData.ResourceType)
	if err != nil {
		t.Fatalf("Failed to get cached pricing: %v", err)
	}
	
	if cachedData == nil {
		t.Fatal("Expected cached data, got nil")
	}
	
	// Verify data matches
	if cachedData.Provider != testData.Provider {
		t.Errorf("Provider mismatch: expected %s, got %s", testData.Provider, cachedData.Provider)
	}
	if cachedData.Region != testData.Region {
		t.Errorf("Region mismatch: expected %s, got %s", testData.Region, cachedData.Region)
	}
	if cachedData.ResourceType != testData.ResourceType {
		t.Errorf("ResourceType mismatch: expected %s, got %s", testData.ResourceType, cachedData.ResourceType)
	}
	if cachedData.Currency != testData.Currency {
		t.Errorf("Currency mismatch: expected %s, got %s", testData.Currency, cachedData.Currency)
	}
	if cachedData.HourlyPrice != testData.HourlyPrice {
		t.Errorf("HourlyPrice mismatch: expected %f, got %f", testData.HourlyPrice, cachedData.HourlyPrice)
	}
	if cachedData.MonthlyPrice != testData.MonthlyPrice {
		t.Errorf("MonthlyPrice mismatch: expected %f, got %f", testData.MonthlyPrice, cachedData.MonthlyPrice)
	}
	if len(cachedData.PricingTiers) != len(testData.PricingTiers) {
		t.Errorf("PricingTiers length mismatch: expected %d, got %d", len(testData.PricingTiers), len(cachedData.PricingTiers))
	}
}

// TestPricingService_CacheNotFound tests retrieving non-existent cached data
func TestPricingService_CacheNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Try to get non-existent data
	cachedData, err := ps.getCachedPricing("nonexistent", "region", "type")
	if err != nil {
		t.Fatalf("Expected no error for non-existent data, got: %v", err)
	}
	
	if cachedData != nil {
		t.Error("Expected nil for non-existent data, got data")
	}
}

// TestPricingService_CacheUpdate tests updating existing cached data
func TestPricingService_CacheUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Cache initial data
	initialData := &PricingData{
		Provider:     "tencentcloud",
		Region:       "ap-guangzhou",
		ResourceType: "S2.MEDIUM4",
		Currency:     "CNY",
		HourlyPrice:  0.5,
		MonthlyPrice: 360.0,
	}
	
	err := ps.cachePricing(initialData)
	if err != nil {
		t.Fatalf("Failed to cache initial data: %v", err)
	}
	
	// Update with new pricing
	updatedData := &PricingData{
		Provider:     "tencentcloud",
		Region:       "ap-guangzhou",
		ResourceType: "S2.MEDIUM4",
		Currency:     "CNY",
		HourlyPrice:  0.6,
		MonthlyPrice: 432.0,
	}
	
	err = ps.cachePricing(updatedData)
	if err != nil {
		t.Fatalf("Failed to update cached data: %v", err)
	}
	
	// Verify only one entry exists
	var count int
	err = ps.db.QueryRow("SELECT COUNT(*) FROM pricing_cache WHERE provider = ? AND region = ? AND resource_type = ?",
		updatedData.Provider, updatedData.Region, updatedData.ResourceType).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query cached data: %v", err)
	}
	
	if count != 1 {
		t.Errorf("Expected 1 cached entry after update, got %d", count)
	}
	
	// Verify data was updated
	cachedData, err := ps.getCachedPricing(updatedData.Provider, updatedData.Region, updatedData.ResourceType)
	if err != nil {
		t.Fatalf("Failed to get cached pricing: %v", err)
	}
	
	if cachedData.HourlyPrice != updatedData.HourlyPrice {
		t.Errorf("Expected updated hourly price %f, got %f", updatedData.HourlyPrice, cachedData.HourlyPrice)
	}
}

// TestPricingService_CacheExpiration tests that expired cache entries are not returned
func TestPricingService_CacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Insert expired data directly into database
	expiredData := &PricingData{
		Provider:     "alicloud",
		Region:       "cn-beijing",
		ResourceType: "ecs.t5-lc1m1.small",
		Currency:     "CNY",
		HourlyPrice:  0.1,
		MonthlyPrice: 72.0,
	}
	
	// Insert with expired timestamp
	now := time.Now()
	expiredTime := now.Add(-25 * time.Hour) // 25 hours ago
	
	_, err := ps.db.Exec(`
		INSERT INTO pricing_cache (provider, region, resource_type, pricing_data, currency, cached_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, expiredData.Provider, expiredData.Region, expiredData.ResourceType, `{"provider":"alicloud"}`, expiredData.Currency, expiredTime, expiredTime.Add(24*time.Hour))
	
	if err != nil {
		t.Fatalf("Failed to insert expired data: %v", err)
	}
	
	// Try to retrieve expired data
	cachedData, err := ps.getCachedPricing(expiredData.Provider, expiredData.Region, expiredData.ResourceType)
	if err != nil {
		t.Fatalf("Expected no error for expired data, got: %v", err)
	}
	
	if cachedData != nil {
		t.Error("Expected nil for expired data, got data")
	}
}

// TestPricingService_CleanExpiredCache tests cleaning expired cache entries
func TestPricingService_CleanExpiredCache(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	now := time.Now()
	
	// Insert valid data
	validData := &PricingData{
		Provider:     "aws",
		Region:       "us-west-2",
		ResourceType: "t3.small",
		Currency:     "USD",
		HourlyPrice:  0.0208,
		MonthlyPrice: 14.976,
	}
	err := ps.cachePricing(validData)
	if err != nil {
		t.Fatalf("Failed to cache valid data: %v", err)
	}
	
	// Insert expired data directly
	expiredTime := now.Add(-25 * time.Hour)
	_, err = ps.db.Exec(`
		INSERT INTO pricing_cache (provider, region, resource_type, pricing_data, currency, cached_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "alicloud", "cn-shanghai", "ecs.expired", `{"provider":"alicloud"}`, "CNY", expiredTime, expiredTime.Add(24*time.Hour))
	
	if err != nil {
		t.Fatalf("Failed to insert expired data: %v", err)
	}
	
	// Verify we have 2 entries
	var countBefore int
	err = ps.db.QueryRow("SELECT COUNT(*) FROM pricing_cache").Scan(&countBefore)
	if err != nil {
		t.Fatalf("Failed to count entries: %v", err)
	}
	if countBefore != 2 {
		t.Errorf("Expected 2 entries before cleanup, got %d", countBefore)
	}
	
	// Clean expired cache
	err = ps.CleanExpiredCache()
	if err != nil {
		t.Fatalf("Failed to clean expired cache: %v", err)
	}
	
	// Verify only valid entry remains
	var countAfter int
	err = ps.db.QueryRow("SELECT COUNT(*) FROM pricing_cache").Scan(&countAfter)
	if err != nil {
		t.Fatalf("Failed to count entries: %v", err)
	}
	if countAfter != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", countAfter)
	}
	
	// Verify the valid data is still accessible
	cachedData, err := ps.getCachedPricing(validData.Provider, validData.Region, validData.ResourceType)
	if err != nil {
		t.Fatalf("Failed to get valid cached data: %v", err)
	}
	if cachedData == nil {
		t.Error("Valid data should still be accessible after cleanup")
	}
}

// TestPricingService_GetPricing tests the main GetPricing method
func TestPricingService_GetPricing(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Cache some data
	testData := &PricingData{
		Provider:     "alicloud",
		Region:       "cn-hangzhou",
		ResourceType: "ecs.g6.xlarge",
		Currency:     "CNY",
		HourlyPrice:  2.5,
		MonthlyPrice: 1800.0,
	}
	
	err := ps.cachePricing(testData)
	if err != nil {
		t.Fatalf("Failed to cache data: %v", err)
	}
	
	// Get pricing (should return cached data)
	pricingData, err := ps.GetPricing(testData.Provider, testData.Region, testData.ResourceType)
	if err != nil {
		t.Fatalf("Failed to get pricing: %v", err)
	}
	
	if pricingData == nil {
		t.Fatal("Expected pricing data, got nil")
	}
	
	if pricingData.HourlyPrice != testData.HourlyPrice {
		t.Errorf("Expected hourly price %f, got %f", testData.HourlyPrice, pricingData.HourlyPrice)
	}
}

// TestPricingService_GetPricing_NotInCache tests GetPricing when data is not cached
func TestPricingService_GetPricing_NotInCache(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	defer ps.Close()
	
	// Try to get pricing for non-cached data
	_, err := ps.GetPricing("aws", "eu-west-1", "t2.large")
	if err == nil {
		t.Error("Expected error for non-cached data, got nil")
	}
}

// TestPricingService_Close tests closing the database connection
func TestPricingService_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pricing.db")
	
	ps := NewPricingService(dbPath)
	
	// Close the connection
	err := ps.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	
	// Verify we can't query after closing
	_, err = ps.getCachedPricing("test", "test", "test")
	if err == nil {
		t.Error("Expected error when querying closed database, got nil")
	}
}
