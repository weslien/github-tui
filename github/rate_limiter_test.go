package github

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockTransport is a configurable http.RoundTripper for testing.
type mockTransport struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func TestRateLimitTransport_DetectsGraphQL(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantGQL  bool
	}{
		{
			name:    "POST to /graphql is GraphQL",
			method:  http.MethodPost,
			path:    "/graphql",
			wantGQL: true,
		},
		{
			name:    "GET to /graphql is not GraphQL",
			method:  http.MethodGet,
			path:    "/graphql",
			wantGQL: false,
		},
		{
			name:    "GET to /repos/owner/name is REST",
			method:  http.MethodGet,
			path:    "/repos/owner/name",
			wantGQL: false,
		},
		{
			name:    "POST to /repos/owner/name/issues is REST",
			method:  http.MethodPost,
			path:    "/repos/owner/name/issues",
			wantGQL: false,
		},
		{
			name:    "PUT to /graphql is not GraphQL",
			method:  http.MethodPut,
			path:    "/graphql",
			wantGQL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "https://api.github.com"+tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			got := isGraphQLRequest(req)
			if got != tt.wantGQL {
				t.Errorf("isGraphQLRequest() = %v, want %v", got, tt.wantGQL)
			}
		})
	}
}

func TestRateLimitTransport_ParsesRESTHeaders(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		remaining     int
		resetEpoch    int64
		wantLimit     int
		wantRemaining int
	}{
		{
			name:          "standard rate limit headers",
			limit:         5000,
			remaining:     4999,
			resetEpoch:    1700000000,
			wantLimit:     5000,
			wantRemaining: 4999,
		},
		{
			name:          "approaching limit",
			limit:         5000,
			remaining:     100,
			resetEpoch:    1700001000,
			wantLimit:     5000,
			wantRemaining: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTransport{
				handler: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
					}
					resp.Header.Set("X-RateLimit-Limit", strconv.Itoa(tt.limit))
					resp.Header.Set("X-RateLimit-Remaining", strconv.Itoa(tt.remaining))
					resp.Header.Set("X-RateLimit-Reset", strconv.FormatInt(tt.resetEpoch, 10))
					return resp, nil
				},
			}

			rl := NewRateLimiter()
			transport := rl.WrapTransport(mock)

			req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/owner/name", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			_, err = transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip() error: %v", err)
			}

			remaining, limit, resetAt := rl.GetRESTStats()
			if remaining != tt.wantRemaining {
				t.Errorf("remaining = %d, want %d", remaining, tt.wantRemaining)
			}
			if limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tt.wantLimit)
			}
			wantReset := time.Unix(tt.resetEpoch, 0)
			if !resetAt.Equal(wantReset) {
				t.Errorf("resetAt = %v, want %v", resetAt, wantReset)
			}
		})
	}
}

func TestRateLimitTransport_GraphQLDoesNotParseRESTHeaders(t *testing.T) {
	mock := &mockTransport{
		handler: func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			}
			// Even if headers are present, they should not be parsed for GraphQL
			resp.Header.Set("X-RateLimit-Limit", "1000")
			resp.Header.Set("X-RateLimit-Remaining", "999")
			return resp, nil
		},
	}

	rl := NewRateLimiter()
	transport := rl.WrapTransport(mock)

	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error: %v", err)
	}

	// REST stats should remain at defaults (5000/5000), not be overwritten
	remaining, limit, _ := rl.GetRESTStats()
	if remaining != 5000 {
		t.Errorf("REST remaining = %d, want 5000 (should not be affected by GraphQL request)", remaining)
	}
	if limit != 5000 {
		t.Errorf("REST limit = %d, want 5000 (should not be affected by GraphQL request)", limit)
	}
}

func TestRateLimitTransport_ConcurrentSemaphore(t *testing.T) {
	const totalRequests = 95
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32

	mock := &mockTransport{
		handler: func(req *http.Request) (*http.Response, error) {
			current := inFlight.Add(1)
			// Track maximum concurrent in-flight requests.
			for {
				old := maxInFlight.Load()
				if current <= old || maxInFlight.CompareAndSwap(old, current) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			inFlight.Add(-1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			}, nil
		},
	}

	rl := NewRateLimiter()
	// Use unlimited rate limiter for this test so only the semaphore matters.
	rl.restLimiter.SetLimit(10000)
	rl.restLimiter.SetBurst(totalRequests + 10)
	transport := rl.WrapTransport(mock)

	var wg sync.WaitGroup
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/test/test", nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
				return
			}
			_, err = transport.RoundTrip(req)
			if err != nil {
				t.Errorf("RoundTrip() error: %v", err)
			}
		}()
	}

	wg.Wait()

	observed := maxInFlight.Load()
	if observed > maxConcurrentRequests {
		t.Errorf("max concurrent requests = %d, want <= %d", observed, maxConcurrentRequests)
	}
	t.Logf("max observed concurrent requests: %d (limit: %d)", observed, maxConcurrentRequests)
}

func TestIsApproachingLimit(t *testing.T) {
	tests := []struct {
		name          string
		restRemaining int
		restLimit     int
		gqlRemaining  int
		gqlLimit      int
		threshold     float64
		wantREST      bool
		wantGraphQL   bool
	}{
		{
			name:          "both well above threshold",
			restRemaining: 4500,
			restLimit:     5000,
			gqlRemaining:  4500,
			gqlLimit:      5000,
			threshold:     0.1,
			wantREST:      false,
			wantGraphQL:   false,
		},
		{
			name:          "REST below threshold",
			restRemaining: 400,
			restLimit:     5000,
			gqlRemaining:  4500,
			gqlLimit:      5000,
			threshold:     0.1,
			wantREST:      true,
			wantGraphQL:   false,
		},
		{
			name:          "GraphQL below threshold",
			restRemaining: 4500,
			restLimit:     5000,
			gqlRemaining:  300,
			gqlLimit:      5000,
			threshold:     0.1,
			wantREST:      false,
			wantGraphQL:   true,
		},
		{
			name:          "both below threshold",
			restRemaining: 100,
			restLimit:     5000,
			gqlRemaining:  200,
			gqlLimit:      5000,
			threshold:     0.1,
			wantREST:      true,
			wantGraphQL:   true,
		},
		{
			name:          "exactly at threshold boundary",
			restRemaining: 500,
			restLimit:     5000,
			gqlRemaining:  500,
			gqlLimit:      5000,
			threshold:     0.1,
			wantREST:      false,
			wantGraphQL:   false,
		},
		{
			name:          "just below threshold",
			restRemaining: 499,
			restLimit:     5000,
			gqlRemaining:  499,
			gqlLimit:      5000,
			threshold:     0.1,
			wantREST:      true,
			wantGraphQL:   true,
		},
		{
			name:          "zero limits returns false",
			restRemaining: 0,
			restLimit:     0,
			gqlRemaining:  0,
			gqlLimit:      0,
			threshold:     0.1,
			wantREST:      false,
			wantGraphQL:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter()
			rl.mu.Lock()
			rl.restRemaining = tt.restRemaining
			rl.restLimit = tt.restLimit
			rl.graphQLRemaining = tt.gqlRemaining
			rl.graphQLLimit = tt.gqlLimit
			rl.mu.Unlock()

			gotREST, gotGraphQL := rl.IsApproachingLimit(tt.threshold)
			if gotREST != tt.wantREST {
				t.Errorf("IsApproachingLimit() REST = %v, want %v", gotREST, tt.wantREST)
			}
			if gotGraphQL != tt.wantGraphQL {
				t.Errorf("IsApproachingLimit() GraphQL = %v, want %v", gotGraphQL, tt.wantGraphQL)
			}
		})
	}
}
