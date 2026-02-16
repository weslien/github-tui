package github

// RateLimiter provides unified rate limiting for both REST and GraphQL GitHub
// API clients. It implements an http.RoundTripper middleware that tracks rate
// limits independently for each API type and enforces a concurrent request
// semaphore.
//
// Full implementation follows in Task 2.
type RateLimiter struct{}
