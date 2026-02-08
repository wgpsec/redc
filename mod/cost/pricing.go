package cost

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// CredentialProvider is a function type that provides credentials for a given provider
type CredentialProvider func(provider string) (accessKey, secretKey, region string, err error)

// inFlightRequest represents an in-flight pricing request
type inFlightRequest struct {
	wg     sync.WaitGroup
	result *PricingData
	err    error
}

// PricingService manages pricing data retrieval and caching
type PricingService struct {
	dbPath             string
	db                 *sql.DB
	fallbackProvider   func(provider, region, resourceType string) (*PricingData, error)
	credentialProvider CredentialProvider
	rateLimiters       *ProviderRateLimiters
	inFlightRequests   sync.Map // map[string]*inFlightRequest for request deduplication
}

// NewPricingService creates a new pricing service instance
func NewPricingService(dbPath string) *PricingService {
	ps := &PricingService{
		dbPath:             dbPath,
		fallbackProvider:   nil,
		credentialProvider: nil,
		rateLimiters:       NewProviderRateLimiters(),
	}
	
	// Initialize database
	if err := ps.initDB(); err != nil {
		// Log error but don't fail - service can work without cache
		fmt.Printf("Warning: Failed to initialize pricing cache database: %v\n", err)
	}
	
	return ps
}

// SetCredentialProvider sets the credential provider function
func (ps *PricingService) SetCredentialProvider(provider CredentialProvider) {
	ps.credentialProvider = provider
}

// SetFallbackProvider sets the fallback pricing provider function
func (ps *PricingService) SetFallbackProvider(provider func(provider, region, resourceType string) (*PricingData, error)) {
	ps.fallbackProvider = provider
}

// initDB initializes the SQLite database and creates the schema
func (ps *PricingService) initDB() error {
	// Open database connection
	db, err := sql.Open("sqlite3", ps.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	ps.db = db
	
	// Create pricing_cache table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS pricing_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		region TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		pricing_data TEXT NOT NULL,
		currency TEXT NOT NULL,
		cached_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		UNIQUE(provider, region, resource_type)
	);
	`
	
	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create pricing_cache table: %w", err)
	}
	
	// Create indexes for efficient lookups
	createIndexSQL := []string{
		`CREATE INDEX IF NOT EXISTS idx_pricing_lookup ON pricing_cache(provider, region, resource_type);`,
		`CREATE INDEX IF NOT EXISTS idx_pricing_expiry ON pricing_cache(expires_at);`,
	}
	
	for _, indexSQL := range createIndexSQL {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	
	return nil
}

// Close closes the database connection
func (ps *PricingService) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}

// GetPricing retrieves pricing for a resource, using cache if available
func (ps *PricingService) GetPricing(provider, region, resourceType string) (*PricingData, error) {
	// Try to get from cache first
	cachedData, err := ps.getCachedPricing(provider, region, resourceType)
	if err == nil && cachedData != nil {
		return cachedData, nil
	}
	
	// Cache miss - fetch from provider API with request deduplication
	return ps.fetchPricingWithDeduplication(provider, region, resourceType)
}

// fetchPricingWithDeduplication ensures only one request is made for identical concurrent requests
func (ps *PricingService) fetchPricingWithDeduplication(provider, region, resourceType string) (*PricingData, error) {
	// Create a unique key for this request
	key := fmt.Sprintf("%s:%s:%s", provider, region, resourceType)
	
	// Try to load existing in-flight request
	if existing, loaded := ps.inFlightRequests.Load(key); loaded {
		// Another goroutine is already fetching this data, wait for it
		req := existing.(*inFlightRequest)
		req.wg.Wait()
		return req.result, req.err
	}
	
	// Create a new in-flight request
	req := &inFlightRequest{}
	req.wg.Add(1)
	
	// Try to store our request - if another goroutine beat us, use theirs
	actual, loaded := ps.inFlightRequests.LoadOrStore(key, req)
	if loaded {
		// Another goroutine stored their request first, wait for it
		actualReq := actual.(*inFlightRequest)
		actualReq.wg.Wait()
		return actualReq.result, actualReq.err
	}
	
	// We won the race - fetch the data
	defer func() {
		// Clean up the in-flight request when done
		ps.inFlightRequests.Delete(key)
		req.wg.Done()
	}()
	
	// Fetch pricing with retry logic
	req.result, req.err = ps.fetchPricingWithRetry(provider, region, resourceType)
	return req.result, req.err
}

// fetchPricingWithRetry fetches pricing from provider API with exponential backoff retry
func (ps *PricingService) fetchPricingWithRetry(provider, region, resourceType string) (*PricingData, error) {
	// If fallback provider is set and no credential provider is configured,
	// use fallback directly (useful for testing)
	if ps.fallbackProvider != nil && ps.credentialProvider == nil {
		fallbackData, fallbackErr := ps.fallbackProvider(provider, region, resourceType)
		if fallbackErr == nil {
			// Cache the fallback data
			if cacheErr := ps.cachePricing(fallbackData); cacheErr != nil {
				fmt.Printf("Warning: Failed to cache fallback pricing data: %v\n", cacheErr)
			}
			return fallbackData, nil
		}
		// If fallback fails and no credentials, return the fallback error directly
		// Don't try real API without credentials
		return nil, fmt.Errorf("fallback pricing failed: %w", fallbackErr)
	}
	
	config := DefaultRetryConfig()
	operationName := fmt.Sprintf("FetchPricing(%s, %s, %s)", provider, region, resourceType)
	
	// Use WithRetryAndResult to fetch pricing data with retries
	pricingData, err := WithRetryAndResult(config, func() (*PricingData, error) {
		return ps.fetchPricingFromProvider(provider, region, resourceType)
	}, operationName)
	
	if err != nil {
		// All retries failed - try fallback pricing if not already tried
		if ps.fallbackProvider != nil && ps.credentialProvider != nil {
			fallbackData, fallbackErr := ps.fallbackProvider(provider, region, resourceType)
			if fallbackErr == nil {
				fmt.Printf("Using fallback pricing for %s/%s/%s\n", provider, region, resourceType)
				// Cache the fallback data
				if cacheErr := ps.cachePricing(fallbackData); cacheErr != nil {
					fmt.Printf("Warning: Failed to cache fallback pricing data: %v\n", cacheErr)
				}
				return fallbackData, nil
			}
			// Log fallback error but return original API error
			fmt.Printf("Warning: Fallback pricing also failed: %v\n", fallbackErr)
		}
		return nil, fmt.Errorf("failed to fetch pricing after retries: %w", err)
	}
	
	// Cache the successfully fetched pricing data
	if cacheErr := ps.cachePricing(pricingData); cacheErr != nil {
		// Log cache error but don't fail the request
		fmt.Printf("Warning: Failed to cache pricing data: %v\n", cacheErr)
	}
	
	return pricingData, nil
}

// fetchPricingFromProvider fetches pricing from the appropriate provider API
func (ps *PricingService) fetchPricingFromProvider(provider, region, resourceType string) (*PricingData, error) {
	// Check if this is a supported cloud provider
	supportedProviders := map[string]bool{
		"alicloud":     true,
		"tencentcloud": true,
		"aws":          true,
		"volcengine":   true,
	}
	
	if !supportedProviders[provider] {
		// For non-cloud providers (tls, local, etc.), pricing is not available
		return nil, fmt.Errorf("pricing not available for provider: %s", provider)
	}
	
	// Apply rate limiting before making API call
	if err := ps.rateLimiters.Wait(provider); err != nil {
		return nil, fmt.Errorf("rate limiter error for provider %s: %w", provider, err)
	}
	
	// Get credentials from credential provider if available
	var accessKey, secretKey, defaultRegion string
	var err error
	
	if ps.credentialProvider != nil {
		accessKey, secretKey, defaultRegion, err = ps.credentialProvider(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to get credentials for provider %s: %w", provider, err)
		}
		
		// Use default region from config if region parameter is empty
		if region == "" && defaultRegion != "" {
			region = defaultRegion
		}
	}
	
	// Call provider-specific pricing function
	switch provider {
	case "alicloud":
		return ps.getAlicloudPricing(region, resourceType, accessKey, secretKey)
	case "tencentcloud":
		return ps.getTencentcloudPricing(region, resourceType, accessKey, secretKey)
	case "aws":
		return ps.getAWSPricing(region, resourceType, accessKey, secretKey)
	case "volcengine":
		return ps.getVolcenginePricing(region, resourceType, accessKey, secretKey)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getCachedPricing retrieves pricing data from the cache
func (ps *PricingService) getCachedPricing(provider, region, resourceType string) (*PricingData, error) {
	if ps.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	query := `
		SELECT pricing_data, currency, cached_at, expires_at
		FROM pricing_cache
		WHERE provider = ? AND region = ? AND resource_type = ?
		AND expires_at > ?
	`
	
	now := time.Now()
	var pricingDataJSON string
	var currency string
	var cachedAt, expiresAt time.Time
	
	err := ps.db.QueryRow(query, provider, region, resourceType, now).Scan(
		&pricingDataJSON, &currency, &cachedAt, &expiresAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // No cached data found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query cache: %w", err)
	}
	
	// Parse the JSON pricing data
	var pricingData PricingData
	if err := json.Unmarshal([]byte(pricingDataJSON), &pricingData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pricing data: %w", err)
	}
	
	return &pricingData, nil
}

// cachePricing stores pricing data in the cache
func (ps *PricingService) cachePricing(data *PricingData) error {
	if ps.db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	// Marshal pricing data to JSON
	pricingDataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal pricing data: %w", err)
	}
	
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	
	// Insert or replace the pricing data
	query := `
		INSERT INTO pricing_cache (provider, region, resource_type, pricing_data, currency, cached_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, region, resource_type)
		DO UPDATE SET
			pricing_data = excluded.pricing_data,
			currency = excluded.currency,
			cached_at = excluded.cached_at,
			expires_at = excluded.expires_at
	`
	
	_, err = ps.db.Exec(query,
		data.Provider,
		data.Region,
		data.ResourceType,
		string(pricingDataJSON),
		data.Currency,
		now,
		expiresAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to cache pricing data: %w", err)
	}
	
	return nil
}

// RefreshCache updates pricing data from provider APIs
func (ps *PricingService) RefreshCache(provider, region string) error {
	// TODO: Implement in task 3
	return nil
}

// CleanExpiredCache removes expired cache entries
func (ps *PricingService) CleanExpiredCache() error {
	if ps.db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	query := `DELETE FROM pricing_cache WHERE expires_at <= ?`
	
	result, err := ps.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to clean expired cache: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cleaned %d expired cache entries\n", rowsAffected)
	}
	
	return nil
}

// StartCacheCleanup starts a background goroutine to clean expired cache entries
func (ps *PricingService) StartCacheCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for range ticker.C {
			if err := ps.CleanExpiredCache(); err != nil {
				fmt.Printf("Error cleaning expired cache: %v\n", err)
			}
		}
	}()
}
