package cost

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
// It allows bursts up to the bucket capacity and refills tokens at a constant rate
type RateLimiter struct {
	mu            sync.Mutex
	tokens        float64   // Current number of tokens in the bucket
	capacity      float64   // Maximum number of tokens (bucket capacity)
	refillRate    float64   // Tokens added per second
	lastRefillTime time.Time // Last time tokens were refilled
}

// NewRateLimiter creates a new rate limiter with the specified capacity and refill rate
// capacity: maximum number of tokens (allows bursts)
// refillRate: tokens added per second
func NewRateLimiter(capacity float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:        capacity, // Start with full bucket
		capacity:      capacity,
		refillRate:    refillRate,
		lastRefillTime: time.Now(),
	}
}

// Wait blocks until a token is available, then consumes it
// Returns immediately if a token is available
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for {
		// Refill tokens based on time elapsed
		rl.refill()

		// If we have at least one token, consume it and return
		if rl.tokens >= 1.0 {
			rl.tokens -= 1.0
			return
		}

		// Calculate how long to wait for the next token
		tokensNeeded := 1.0 - rl.tokens
		waitDuration := time.Duration(tokensNeeded/rl.refillRate*float64(time.Second))

		// Unlock, wait, and relock
		rl.mu.Unlock()
		time.Sleep(waitDuration)
		rl.mu.Lock()
	}
}

// TryAcquire attempts to acquire a token without blocking
// Returns true if a token was acquired, false otherwise
func (rl *RateLimiter) TryAcquire() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	rl.refill()

	// If we have at least one token, consume it and return true
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}

// refill adds tokens to the bucket based on time elapsed since last refill
// Must be called with lock held
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefillTime)
	
	// Calculate tokens to add based on elapsed time
	tokensToAdd := elapsed.Seconds() * rl.refillRate
	
	// Add tokens, but don't exceed capacity
	rl.tokens = min(rl.tokens+tokensToAdd, rl.capacity)
	
	// Update last refill time
	rl.lastRefillTime = now
}

// AvailableTokens returns the current number of available tokens
func (rl *RateLimiter) AvailableTokens() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.refill()
	return rl.tokens
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ProviderRateLimiters manages rate limiters for different cloud providers
type ProviderRateLimiters struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// NewProviderRateLimiters creates rate limiters for all supported providers
// with their respective rate limits:
// - Alibaba Cloud: 500 requests/minute (8.33 req/sec)
// - Tencent Cloud: 20 requests/second
// - AWS: 10 requests/second
// - Volcengine: 20 requests/second
func NewProviderRateLimiters() *ProviderRateLimiters {
	prl := &ProviderRateLimiters{
		limiters: make(map[string]*RateLimiter),
	}
	
	// Alibaba Cloud: 500 requests/minute = 8.33 requests/second
	// Use capacity of 50 to allow bursts
	prl.limiters["alicloud"] = NewRateLimiter(50, 500.0/60.0)
	
	// Tencent Cloud: 20 requests/second
	// Use capacity of 20 to allow bursts
	prl.limiters["tencentcloud"] = NewRateLimiter(20, 20.0)
	
	// AWS: 10 requests/second
	// Use capacity of 10 to allow bursts
	prl.limiters["aws"] = NewRateLimiter(10, 10.0)
	
	// Volcengine: 20 requests/second (estimated, similar to Tencent Cloud)
	// Use capacity of 20 to allow bursts
	prl.limiters["volcengine"] = NewRateLimiter(20, 20.0)
	
	return prl
}

// Wait blocks until a token is available for the specified provider
func (prl *ProviderRateLimiters) Wait(provider string) error {
	prl.mu.RLock()
	limiter, ok := prl.limiters[provider]
	prl.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("no rate limiter configured for provider: %s", provider)
	}
	
	limiter.Wait()
	return nil
}

// TryAcquire attempts to acquire a token for the specified provider without blocking
func (prl *ProviderRateLimiters) TryAcquire(provider string) (bool, error) {
	prl.mu.RLock()
	limiter, ok := prl.limiters[provider]
	prl.mu.RUnlock()
	
	if !ok {
		return false, fmt.Errorf("no rate limiter configured for provider: %s", provider)
	}
	
	return limiter.TryAcquire(), nil
}

// GetLimiter returns the rate limiter for a specific provider
func (prl *ProviderRateLimiters) GetLimiter(provider string) (*RateLimiter, error) {
	prl.mu.RLock()
	defer prl.mu.RUnlock()
	
	limiter, ok := prl.limiters[provider]
	if !ok {
		return nil, fmt.Errorf("no rate limiter configured for provider: %s", provider)
	}
	
	return limiter, nil
}
