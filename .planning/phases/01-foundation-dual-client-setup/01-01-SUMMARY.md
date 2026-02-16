---
phase: 01-foundation-dual-client-setup
plan: 01
subsystem: api
tags: [go-github, graphql, rest, rate-limiting, oauth2, http-roundtripper]

# Dependency graph
requires: []
provides:
  - "Dual REST+GraphQL client initialization with shared OAuth2 transport"
  - "RateLimiter type with http.RoundTripper middleware"
  - "Independent REST/GraphQL rate limit tracking"
  - "Concurrent request semaphore (90 max)"
  - "GetRESTClient() and GetGraphQLClient() accessor functions"
affects: [01-02-PLAN, 02-actions, 03-projects-v2]

# Tech tracking
tech-stack:
  added: [google/go-github/v68, golang.org/x/time/rate]
  patterns: [http.RoundTripper middleware, channel-based semaphore, independent rate trackers]

key-files:
  created:
    - github/rate_limiter.go
    - github/rate_limiter_test.go
  modified:
    - github/client.go
    - go.mod
    - go.sum

key-decisions:
  - "Used channel-based semaphore (chan struct{}) for concurrent request limiting instead of sync.Mutex"
  - "GraphQL rate limit body parsing deferred to query level in Phase 2/3 to avoid consuming response body in RoundTripper"
  - "REST rate limit headers parsed in RoundTripper, GraphQL uses pre-request rate.Limiter only"
  - "Used aliased import (gogithub) to avoid name collision with package name"

patterns-established:
  - "RoundTripper middleware pattern: wrap base transport with rate limiting, header parsing"
  - "Dual client initialization: both REST and GraphQL share single OAuth2 http.Client"
  - "Thread-safe accessor pattern: RWMutex-protected getters for rate limit stats"

# Metrics
duration: 5min
completed: 2026-02-16
---

# Phase 1 Plan 1: Dual Client & Rate Limiter Summary

**google/go-github REST client alongside existing GraphQL client with shared OAuth2 transport and unified RoundTripper rate limiter tracking REST/GraphQL independently**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-16T21:23:48Z
- **Completed:** 2026-02-16T21:29:40Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Dual REST+GraphQL client initialization sharing a single OAuth2 HTTP client
- RateLimiter with http.RoundTripper middleware enforcing per-API-type rate limits and 90-concurrent-request semaphore
- REST X-RateLimit-* header parsing with thread-safe stat accessors
- 5 table-driven tests passing with -race flag covering detection, header parsing, concurrency, and threshold logic

## Task Commits

Each task was committed atomically:

1. **Task 1: Add go-github dependency and refactor client.go for dual client initialization** - `0d87bba` (feat)
2. **Task 2: Implement unified rate limiter with http.RoundTripper middleware and tests** - `4980512` (feat)

## Files Created/Modified
- `github/client.go` - Dual client init (REST + GraphQL), renamed client var, accessor functions
- `github/rate_limiter.go` - RateLimiter struct, rateLimitTransport RoundTripper, header parsing, semaphore
- `github/rate_limiter_test.go` - Table-driven tests for detection, header parsing, concurrency, threshold
- `go.mod` - Added google/go-github/v68, golang.org/x/time dependencies
- `go.sum` - Updated checksums

## Decisions Made
- Used `gogithub` import alias for `github.com/google/go-github/v68/github` to avoid collision with the `github` package name
- Deferred GraphQL response body parsing to query-level code in later phases (avoids consuming response body in RoundTripper)
- Channel-based semaphore (`chan struct{}` with capacity 90) for concurrent request limiting, simpler than semaphore libraries
- Rate limiter var declared in client.go but left nil -- wiring into transport deferred to Plan 02

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created RateLimiter stub for Task 1 compilation**
- **Found during:** Task 1 (client.go refactor)
- **Issue:** client.go declares `var rateLimiter *RateLimiter` but RateLimiter type does not exist until Task 2
- **Fix:** Created minimal stub `type RateLimiter struct{}` in rate_limiter.go to allow Task 1 to compile independently
- **Files modified:** github/rate_limiter.go
- **Verification:** `go build ./...` passed
- **Committed in:** 0d87bba (Task 1 commit, replaced in Task 2)

**2. [Rule 3 - Blocking] Fixed missing go.sum entry for go-querystring**
- **Found during:** Task 1 (build verification)
- **Issue:** `go get` for go-github did not pull transitive dependency go-querystring into go.sum
- **Fix:** Ran `go get github.com/google/go-github/v68/github@v68.0.0` and `go mod tidy`
- **Files modified:** go.sum
- **Verification:** `go build ./...` passed
- **Committed in:** 0d87bba (Task 1 commit)

**3. [Rule 3 - Blocking] Installed Go toolchain via Homebrew**
- **Found during:** Task 1 (pre-execution)
- **Issue:** Go was not installed on the system (not found in PATH)
- **Fix:** Ran `brew install go` to install Go 1.26.0
- **Files modified:** None (system-level)
- **Verification:** `go version` returns go1.26.0

---

**Total deviations:** 3 auto-fixed (3 blocking)
**Impact on plan:** All auto-fixes necessary for compilation and execution. No scope creep.

## Issues Encountered
- Go toolchain was not installed on the system; installed via Homebrew before execution could begin
- `go get` for go-github v68 did not automatically resolve transitive dependency go-querystring; required explicit fetch

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- REST and GraphQL clients are initialized but rate limiter is NOT yet wired into the transport (rateLimiter var is nil)
- Plan 01-02 will wire the rate limiter into the shared OAuth2 transport via WrapTransport()
- All existing GraphQL functions continue to work unchanged
- Phase 2 (Actions) can use GetRESTClient() once transport wiring is complete

## Self-Check: PASSED

- All 6 key files exist on disk
- Commit 0d87bba (Task 1) found in git log
- Commit 4980512 (Task 2) found in git log

---
*Phase: 01-foundation-dual-client-setup*
*Completed: 2026-02-16*
