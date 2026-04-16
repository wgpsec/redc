package ai

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"red-cloud/mod/gologger"
)

// ProviderConfig represents a single AI provider configuration
type ProviderConfig struct {
	Name    string `json:"name"`
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
	// Provider type: "openai" or "anthropic"
	Provider string `json:"provider"`
}

// ProviderManager manages multiple AI providers with automatic failover and retry.
type ProviderManager struct {
	mu             sync.RWMutex
	providers      []ProviderConfig
	currentIdx     int
	failoverCount  int
	lastError      string
	lastErrorTime  time.Time
}

// permanentErrorPatterns are errors that indicate the provider is definitively unusable (billing, auth).
// These trigger immediate failover without retry.
var permanentErrorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)insufficient.*(balance|funds|credit)`),
	regexp.MustCompile(`(?i)authentication.error`),
	regexp.MustCompile(`(?i)invalid.*api.*key`),
	regexp.MustCompile(`(?i)\b401\b`),
	regexp.MustCompile(`(?i)\b403\b.*(?:forbidden|denied)`),
	regexp.MustCompile(`(?i)\b404\b`),
	regexp.MustCompile(`(?i)not.found`),
	regexp.MustCompile(`(?i)billing`),
	regexp.MustCompile(`(?i)quota.*exceeded`),
}

// transientErrorPatterns are errors worth retrying on the same provider before failover.
var transientErrorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)rate.limit`),
	regexp.MustCompile(`(?i)\b429\b`),
	regexp.MustCompile(`(?i)overloaded`),
	regexp.MustCompile(`(?i)\b529\b`),
	regexp.MustCompile(`(?i)timeout|timed.out`),
	regexp.MustCompile(`(?i)connection.*(refused|reset)`),
	regexp.MustCompile(`(?i)server.*error`),
	regexp.MustCompile(`(?i)\b50[0-4]\b`),
	regexp.MustCompile(`(?i)temporarily.*unavailable`),
	regexp.MustCompile(`(?i)broken.pipe`),
	regexp.MustCompile(`(?i)i/o.timeout`),
}

// NewProviderManager creates a manager from a primary provider config and optional fallbacks.
func NewProviderManager(primary ProviderConfig, fallbacks []ProviderConfig) *ProviderManager {
	providers := make([]ProviderConfig, 0, 1+len(fallbacks))
	providers = append(providers, primary)
	providers = append(providers, fallbacks...)
	return &ProviderManager{
		providers: providers,
	}
}

// Current returns the currently active provider.
func (pm *ProviderManager) Current() ProviderConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.providers[pm.currentIdx]
}

// CurrentClient creates an AI Client from the current provider.
func (pm *ProviderManager) CurrentClient() *Client {
	p := pm.Current()
	return NewClient(p.Provider, p.APIKey, p.BaseURL, p.Model)
}

// Count returns the total number of configured providers.
func (pm *ProviderManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.providers)
}

// IsPermanentError checks if the error is definitively unrecoverable (auth, billing).
func IsPermanentError(errMsg string) bool {
	for _, p := range permanentErrorPatterns {
		if p.MatchString(errMsg) {
			return true
		}
	}
	return false
}

// IsTransientError checks if the error is temporary and worth retrying.
func IsTransientError(errMsg string) bool {
	if IsPermanentError(errMsg) {
		return false
	}
	for _, p := range transientErrorPatterns {
		if p.MatchString(errMsg) {
			return true
		}
	}
	return false
}

// ShouldFailover returns true if the error warrants trying a different provider.
func ShouldFailover(errMsg string) bool {
	return IsPermanentError(errMsg) || IsTransientError(errMsg)
}

// RetryDelay returns the backoff delay for a given retry attempt.
func RetryDelay(attempt int) time.Duration {
	delays := []time.Duration{3 * time.Second, 6 * time.Second, 10 * time.Second}
	if attempt >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempt]
}

// Failover switches to the next available provider. Returns true if successful.
func (pm *ProviderManager) Failover(errMsg string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.providers) <= 1 {
		return false
	}

	old := pm.providers[pm.currentIdx]
	pm.currentIdx = (pm.currentIdx + 1) % len(pm.providers)
	pm.failoverCount++

	if pm.failoverCount >= len(pm.providers) {
		gologger.Error().Msgf("ai: all %d providers exhausted", len(pm.providers))
		return false
	}

	pm.lastError = errMsg
	pm.lastErrorTime = time.Now()
	gologger.Info().Msgf("ai: failover %s → %s (reason: %s)", old.Name, pm.providers[pm.currentIdx].Name, truncate(errMsg, 80))
	return true
}

// ResetFailover resets the failover counter (call at the start of each new operation).
func (pm *ProviderManager) ResetFailover() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.failoverCount = 0
}

// ExecuteWithRetry runs fn with automatic retry + failover. Returns the result or final error.
// maxRetries is per-provider. fn receives the current Client.
func (pm *ProviderManager) ExecuteWithRetry(maxRetries int, fn func(client *Client) error) error {
	pm.ResetFailover()

	for {
		client := pm.CurrentClient()
		var lastErr error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			lastErr = fn(client)
			if lastErr == nil {
				return nil
			}

			errMsg := lastErr.Error()

			// Permanent error → skip retry, go straight to failover
			if IsPermanentError(errMsg) {
				gologger.Warning().Msgf("ai: permanent error on %s: %s", pm.Current().Name, truncate(errMsg, 100))
				break
			}

			// Transient error → retry with backoff
			if IsTransientError(errMsg) && attempt < maxRetries {
				delay := RetryDelay(attempt)
				gologger.Info().Msgf("ai: transient error, retry %d/%d in %v: %s", attempt+1, maxRetries, delay, truncate(errMsg, 80))
				time.Sleep(delay)
				continue
			}

			// Unknown error on last attempt → try failover
			break
		}

		// Try failover
		if lastErr != nil && ShouldFailover(lastErr.Error()) {
			if pm.Failover(lastErr.Error()) {
				continue
			}
		}

		return fmt.Errorf("all providers failed: %w", lastErr)
	}
}

// Status returns a human-readable status string.
func (pm *ProviderManager) Status() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	names := make([]string, len(pm.providers))
	for i, p := range pm.providers {
		marker := "  "
		if i == pm.currentIdx {
			marker = "→ "
		}
		names[i] = fmt.Sprintf("%s%s (%s)", marker, p.Name, p.BaseURL)
	}
	return strings.Join(names, "\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
