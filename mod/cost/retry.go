package cost

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// NonRetryableError wraps an error that should not be retried
// This is used for errors that are permanent and won't be fixed by retrying
// Examples: invalid resource type, unsupported region, invalid credentials format
type NonRetryableError struct {
	Err error
}

func (e *NonRetryableError) Error() string {
	return fmt.Sprintf("non-retryable error: %v", e.Err)
}

func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// IsNonRetryable checks if an error is non-retryable
func IsNonRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	// Check if error is wrapped as NonRetryableError
	var nonRetryable *NonRetryableError
	if errors, ok := err.(*NonRetryableError); ok {
		nonRetryable = errors
		_ = nonRetryable // Use the variable to avoid unused warning
		return true
	}
	
	// Check error message for known non-retryable patterns
	errMsg := strings.ToLower(err.Error())
	nonRetryablePatterns := []string{
		"invalidinstancetype",
		"invalid instance type",
		"instance type does not exist",
		"valuenotsupported",
		"value not supported",
		"unsupported resource type",
		"invalid resource type",
		"resource type not found",
		"invalid credentials format",
		"malformed credentials",
		"invalidsystemdiskcategory",
		"invalid system disk category",
		"systemdisk.category is not valid",
		"notsupportdiskcategory",
		"not support disk category",
		"pricing not available for provider",
		"unsupported provider",
	}
	
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	
	return false
}

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	Multiplier     float64       // Backoff multiplier for exponential backoff
}

// DefaultRetryConfig returns the default retry configuration
// - 3 retries (4 total attempts including the initial one)
// - Initial backoff of 1 second
// - Max backoff of 30 seconds
// - Exponential multiplier of 2
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// WithRetry executes a function with exponential backoff retry logic
// It will retry up to MaxRetries times if the function returns an error
// The backoff duration increases exponentially with each retry
func WithRetry(config RetryConfig, fn RetryableFunc, operationName string) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			// Success - log if this was a retry
			if attempt > 0 {
				log.Printf("[Retry] %s succeeded after %d retries", operationName, attempt)
			}
			return nil
		}
		
		lastErr = err
		
		// If this was the last attempt, don't wait
		if attempt == config.MaxRetries {
			log.Printf("[Retry] %s failed after %d retries: %v", operationName, config.MaxRetries, err)
			break
		}
		
		// Calculate backoff duration with exponential increase
		backoff := calculateBackoff(config, attempt)
		
		log.Printf("[Retry] %s failed (attempt %d/%d): %v. Retrying in %v...",
			operationName, attempt+1, config.MaxRetries+1, err, backoff)
		
		// Wait before retrying
		time.Sleep(backoff)
	}
	
	return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

// calculateBackoff calculates the backoff duration for a given attempt
// Uses exponential backoff: initialBackoff * (multiplier ^ attempt)
// Capped at maxBackoff to prevent excessively long waits
func calculateBackoff(config RetryConfig, attempt int) time.Duration {
	// Calculate exponential backoff
	backoff := float64(config.InitialBackoff) * math.Pow(config.Multiplier, float64(attempt))
	
	// Cap at max backoff
	if backoff > float64(config.MaxBackoff) {
		backoff = float64(config.MaxBackoff)
	}
	
	return time.Duration(backoff)
}

// RetryableFuncWithResult is a function that returns a result and can be retried
type RetryableFuncWithResult[T any] func() (T, error)

// WithRetryAndResult executes a function with exponential backoff retry logic and returns a result
// It will retry up to MaxRetries times if the function returns an error
// The backoff duration increases exponentially with each retry
// Non-retryable errors will not be retried
func WithRetryAndResult[T any](config RetryConfig, fn RetryableFuncWithResult[T], operationName string) (T, error) {
	var lastErr error
	var result T
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		res, err := fn()
		if err == nil {
			// Success - log if this was a retry
			if attempt > 0 {
				log.Printf("[Retry] %s succeeded after %d retries", operationName, attempt)
			}
			return res, nil
		}
		
		lastErr = err
		
		// Check if error is non-retryable
		if IsNonRetryable(err) {
			log.Printf("[Retry] %s failed with non-retryable error: %v. Skipping retries.", operationName, err)
			return result, fmt.Errorf("non-retryable error: %w", err)
		}
		
		// If this was the last attempt, don't wait
		if attempt == config.MaxRetries {
			log.Printf("[Retry] %s failed after %d retries: %v", operationName, config.MaxRetries, err)
			break
		}
		
		// Calculate backoff duration with exponential increase
		backoff := calculateBackoff(config, attempt)
		
		log.Printf("[Retry] %s failed (attempt %d/%d): %v. Retrying in %v...",
			operationName, attempt+1, config.MaxRetries+1, err, backoff)
		
		// Wait before retrying
		time.Sleep(backoff)
	}
	
	return result, fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}
