package github

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// maxConcurrentRequests is a conservative limit below GitHub's 100
	// concurrent request cap.
	maxConcurrentRequests = 90

	// defaultRequestsPerSecond approximates 5000 requests per hour.
	defaultRequestsPerSecond = 1.39

	// defaultBurst allows short bursts of requests.
	defaultBurst = 10
)

// RateLimiter provides unified rate limiting for both REST and GraphQL GitHub
// API clients. It tracks REST and GraphQL rate limits independently and
// enforces a concurrent request semaphore to stay below GitHub's limits.
type RateLimiter struct {
	restLimiter    *rate.Limiter
	graphQLLimiter *rate.Limiter
	concurrentSem  chan struct{}

	mu               sync.RWMutex
	restRemaining    int
	restLimit        int
	graphQLRemaining int
	graphQLLimit     int
	restResetAt      time.Time
}

// NewRateLimiter creates a RateLimiter with conservative defaults for both
// REST (5000 req/hr) and GraphQL (5000 pts/hr) rate limits.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		restLimiter:    rate.NewLimiter(rate.Limit(defaultRequestsPerSecond), defaultBurst),
		graphQLLimiter: rate.NewLimiter(rate.Limit(defaultRequestsPerSecond), defaultBurst),
		concurrentSem:  make(chan struct{}, maxConcurrentRequests),
		restRemaining:  5000,
		restLimit:      5000,
		graphQLRemaining: 5000,
		graphQLLimit:     5000,
	}
}

// WrapTransport wraps a base http.RoundTripper with rate limiting middleware.
// The returned RoundTripper enforces concurrent request limits, per-API-type
// rate limiting, and parses REST rate limit headers from responses.
func (rl *RateLimiter) WrapTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &rateLimitTransport{
		base: base,
		rl:   rl,
	}
}

// GetRESTStats returns the current REST API rate limit state.
func (rl *RateLimiter) GetRESTStats() (remaining, limit int, resetAt time.Time) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.restRemaining, rl.restLimit, rl.restResetAt
}

// GetGraphQLStats returns the current GraphQL API rate limit state.
func (rl *RateLimiter) GetGraphQLStats() (remaining, limit int) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.graphQLRemaining, rl.graphQLLimit
}

// IsApproachingLimit returns true for each API type where the remaining
// requests are below the given threshold fraction of the limit. For example,
// threshold=0.1 means "below 10% remaining."
func (rl *RateLimiter) IsApproachingLimit(threshold float64) (rest bool, graphql bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rl.restLimit > 0 {
		rest = float64(rl.restRemaining) < threshold*float64(rl.restLimit)
	}
	if rl.graphQLLimit > 0 {
		graphql = float64(rl.graphQLRemaining) < threshold*float64(rl.graphQLLimit)
	}
	return rest, graphql
}

// isGraphQLRequest returns true if the request targets GitHub's GraphQL
// endpoint.
func isGraphQLRequest(req *http.Request) bool {
	return req.Method == http.MethodPost && req.URL.Path == "/graphql"
}

// rateLimitTransport is an http.RoundTripper that applies rate limiting and
// concurrent request control before delegating to the base transport.
type rateLimitTransport struct {
	base http.RoundTripper
	rl   *RateLimiter
}

// RoundTrip implements http.RoundTripper. It acquires a concurrency slot,
// waits for the appropriate rate limiter, performs the request, parses REST
// rate limit headers from the response, and then releases the slot.
func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Acquire concurrent semaphore slot.
	t.rl.concurrentSem <- struct{}{}
	defer func() { <-t.rl.concurrentSem }()

	// Determine which rate limiter to use.
	limiter := t.rl.restLimiter
	if isGraphQLRequest(req) {
		limiter = t.rl.graphQLLimiter
	}

	// Wait for rate limiter, respecting the request context.
	if err := limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	// Perform the actual request.
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Parse REST rate limit headers (only for non-GraphQL responses).
	if !isGraphQLRequest(req) {
		t.parseRESTHeaders(resp)
	}

	return resp, nil
}

// parseRESTHeaders extracts rate limit information from standard GitHub REST
// API response headers and updates the RateLimiter state.
func (t *rateLimitTransport) parseRESTHeaders(resp *http.Response) {
	t.rl.mu.Lock()
	defer t.rl.mu.Unlock()

	if v := resp.Header.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.rl.restLimit = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.rl.restRemaining = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			t.rl.restResetAt = time.Unix(n, 0)
		}
	}
}
