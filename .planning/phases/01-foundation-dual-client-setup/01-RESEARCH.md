# Phase 1: Foundation & Dual-Client Setup - Research

**Researched:** 2026-02-16
**Domain:** Dual REST+GraphQL GitHub client architecture with unified rate limiting
**Confidence:** HIGH

## Summary

Phase 1 establishes foundational infrastructure for both GitHub Actions (Phase 2) and Projects V2 (Phase 3): dual REST+GraphQL client architecture, unified rate limiting, token scope validation, and async UI update patterns. This is pure plumbing with no user-facing features.

The research confirms that google/go-github/v83 (latest as of 2026-02-16) is the standard choice for GitHub REST API access, providing type-safe clients with built-in pagination and error handling. Rate limiting requires tracking two independent pools: REST (5000 req/hr via X-RateLimit headers) and GraphQL (5000 pts/hr via response body rateLimit field), both sharing a 100 concurrent request limit. Token scope validation uses the X-OAuth-Scopes response header from any API call.

The existing codebase already implements the critical async update pattern: a buffered channel (`UI.updater`) receives update functions from worker goroutines, which a dedicated goroutine forwards to `app.QueueUpdateDraw()`. This prevents the well-documented tview deadlock when calling `QueueUpdateDraw()` directly from event handlers or when the update queue saturates.

**Primary recommendation:** Extend existing github/client.go to initialize both REST (go-github) and GraphQL (githubv4) clients with the same OAuth2 token, implement a unified RateLimiter type that wraps both clients with http.RoundTripper middleware, validate token scopes on startup using a test API call, and leverage the existing UI.updater channel pattern for all async updates.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Phase Boundary
Establish dual REST+GraphQL client architecture with unified rate limiting and async UI update patterns. This is shared infrastructure that both Actions (Phase 2) and Projects V2 (Phase 3) depend on. No user-facing features — only foundational plumbing.

### Claude's Discretion (All Implementation Details)
- Rate limit visibility approach (status bar, log, warning modal)
- Missing token scope behavior (hard fail vs degraded mode)
- REST client error presentation in TUI
- Rate limiter implementation pattern (middleware, wrapper, or standalone)
- Async update channel buffering strategy
- REST client library integration approach (alongside existing GraphQL client)

User deferred all implementation decisions to Claude — this is pure infrastructure with no UX preferences.

### Specific Ideas
No specific requirements — open to standard approaches. Follow existing codebase patterns where possible.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google/go-github/v83 | v83.0.0 (2026-02-16) | REST API client for GitHub | Official go-github client maintained by Google, type-safe API coverage, built-in pagination, handles edge cases. Latest version includes native iterators for List* methods. |
| shurcooL/githubv4 | (existing) | GraphQL API client | Already in codebase for issue management, mature library with strong typing for GraphQL queries. |
| golang.org/x/oauth2 | (existing) | OAuth2 token handling | Standard library extension, already used for GraphQL client authentication. |
| golang.org/x/time/rate | Latest | Token bucket rate limiter | Official rate limiting implementation, thread-safe, supports burst and per-second limits. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| context | stdlib | Request cancellation, timeouts | Every API call should accept context.Context for timeout control. |
| net/http | stdlib | HTTP client customization | Custom http.RoundTripper for rate limiting middleware. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| google/go-github | Manual http.Client + REST calls | Never — go-github handles pagination, rate limiting headers, type safety. Custom implementations are error-prone and miss edge cases. |
| golang.org/x/time/rate | Custom ticker-based limiter | Could work for simple cases, but x/time/rate provides burst handling, token bucket algorithm, and is battle-tested. |
| http.RoundTripper middleware | Separate limiter called before API methods | Middleware is cleaner (single responsibility), less invasive, works with both clients uniformly. |

**Installation:**

```bash
go get github.com/google/go-github/v83@v83.0.0
go get golang.org/x/time/rate@latest
```

## Architecture Patterns

### Recommended Project Structure

```
github/
├── client.go              # Existing GraphQL client init
├── rest_client.go         # NEW: REST client init (google/go-github)
├── rate_limiter.go        # NEW: Unified rate limiter (RoundTripper)
├── token_validator.go     # NEW: Token scope validation
└── [existing query/mutation files]

ui/
├── ui.go                  # Existing updater channel (REUSE)
└── [existing UI components]
```

### Pattern 1: Dual Client Initialization with Shared OAuth2 Token

**What:** Initialize both REST and GraphQL clients using the same PAT token from config, sharing the OAuth2 http.Client.

**When to use:** During app startup in github.NewClient(), immediately after config.Init().

**Example:**

```go
// github/client.go (extend existing)
package github

import (
    "context"
    "github.com/google/go-github/v83/github"
    "github.com/shurcooL/githubv4"
    "golang.org/x/oauth2"
)

var (
    graphQLClient *githubv4.Client  // existing
    restClient    *github.Client    // NEW
    rateLimiter   *RateLimiter      // NEW
)

func NewClient(token string) {
    ctx := context.Background()

    // Create OAuth2 token source
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )

    // Create base HTTP client with OAuth2
    httpClient := oauth2.NewClient(ctx, ts)

    // Wrap with rate limiting transport (unified across both clients)
    rateLimiter = NewRateLimiter()
    httpClient.Transport = rateLimiter.WrapTransport(httpClient.Transport)

    // Initialize both clients with the same HTTP client
    graphQLClient = githubv4.NewClient(httpClient)
    restClient = github.NewClient(httpClient)
}

func GetRESTClient() *github.Client {
    return restClient
}
```

**Source:** Verified pattern from existing codebase (/Users/gustav/src/github-tui/github/client.go) and official go-github documentation.

### Pattern 2: Unified Rate Limiter via http.RoundTripper Middleware

**What:** Implement rate limiter as http.RoundTripper middleware to track both REST and GraphQL rate limits independently, enforce concurrent request limit, and parse rate limit headers/response bodies.

**When to use:** Wrap the OAuth2 transport during client initialization, before passing to githubv4/go-github constructors.

**Example:**

```go
// github/rate_limiter.go
package github

import (
    "context"
    "fmt"
    "net/http"
    "strconv"
    "sync"
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    restLimiter     *rate.Limiter  // 5000 req/hr
    graphQLLimiter  *rate.Limiter  // 5000 pts/hr
    concurrentSem   chan struct{}  // 100 concurrent limit

    mu sync.RWMutex
    restRemaining    int
    restLimit        int
    graphQLRemaining int
    graphQLLimit     int
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        // 5000 req/hr = ~1.39 req/sec, allow burst of 10
        restLimiter: rate.NewLimiter(rate.Limit(1.39), 10),
        graphQLLimiter: rate.NewLimiter(rate.Limit(1.39), 10),
        concurrentSem: make(chan struct{}, 90), // Conservative: 90 instead of 100
    }
}

type rateLimitTransport struct {
    base http.RoundTripper
    rl   *RateLimiter
}

func (rl *RateLimiter) WrapTransport(base http.RoundTripper) http.RoundTripper {
    return &rateLimitTransport{base: base, rl: rl}
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    // Acquire concurrent slot (blocks if 90 requests in flight)
    t.rl.concurrentSem <- struct{}{}
    defer func() { <-t.rl.concurrentSem }()

    // Determine if REST or GraphQL (GraphQL uses POST to /graphql)
    isGraphQL := req.URL.Path == "/graphql"

    // Wait for rate limit token
    ctx := req.Context()
    if isGraphQL {
        if err := t.rl.graphQLLimiter.Wait(ctx); err != nil {
            return nil, fmt.Errorf("rate limit wait canceled: %w", err)
        }
    } else {
        if err := t.rl.restLimiter.Wait(ctx); err != nil {
            return nil, fmt.Errorf("rate limit wait canceled: %w", err)
        }
    }

    // Execute request
    resp, err := t.base.RoundTrip(req)
    if err != nil {
        return resp, err
    }

    // Parse rate limit info from response
    if isGraphQL {
        t.rl.updateGraphQLLimits(resp)
    } else {
        t.rl.updateRESTLimits(resp)
    }

    return resp, nil
}

func (rl *RateLimiter) updateRESTLimits(resp *http.Response) {
    if limitStr := resp.Header.Get("X-RateLimit-Limit"); limitStr != "" {
        if limit, err := strconv.Atoi(limitStr); err == nil {
            rl.mu.Lock()
            rl.restLimit = limit
            rl.mu.Unlock()
        }
    }
    if remainingStr := resp.Header.Get("X-RateLimit-Remaining"); remainingStr != "" {
        if remaining, err := strconv.Atoi(remainingStr); err == nil {
            rl.mu.Lock()
            rl.restRemaining = remaining
            rl.mu.Unlock()
        }
    }
}

func (rl *RateLimiter) updateGraphQLLimits(resp *http.Response) {
    // Parse response body for rateLimit.cost and rateLimit.remaining
    // This requires reading the body and parsing JSON
    // Implementation detail: store in rl.graphQLRemaining, rl.graphQLLimit
}

func (rl *RateLimiter) GetRESTStats() (remaining, limit int) {
    rl.mu.RLock()
    defer rl.mu.RUnlock()
    return rl.restRemaining, rl.restLimit
}

func (rl *RateLimiter) GetGraphQLStats() (remaining, limit int) {
    rl.mu.RLock()
    defer rl.mu.RUnlock()
    return rl.graphQLRemaining, rl.graphQLLimit
}
```

**Sources:**
- http.RoundTripper pattern: [Writing HTTP client middleware in Go](https://echorand.me/posts/go-http-client-middleware/)
- golang.org/x/time/rate usage: [rate package documentation](https://pkg.go.dev/golang.org/x/time/rate)
- GitHub rate limit headers: [GitHub REST API Rate Limiting](https://docs.github.com/en/rest/rate-limit)

### Pattern 3: Token Scope Validation on Startup

**What:** Make a test API call on startup to retrieve X-OAuth-Scopes header and validate required scopes are present.

**When to use:** After client initialization, before ui.New().Start() in main.go.

**Example:**

```go
// github/token_validator.go
package github

import (
    "context"
    "fmt"
    "net/http"
    "strings"
)

type TokenScopes struct {
    HasRepo    bool
    HasActions bool
    HasProject bool
}

func ValidateTokenScopes(token string) (*TokenScopes, error) {
    // Make a simple API call to get X-OAuth-Scopes header
    req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Authorization", "Bearer "+token)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("api call failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("token validation failed: status %d", resp.StatusCode)
    }

    // Parse X-OAuth-Scopes header (classic PAT)
    // Note: Fine-grained PATs don't return this header - need different approach
    scopesHeader := resp.Header.Get("X-OAuth-Scopes")
    scopes := strings.Split(scopesHeader, ", ")

    ts := &TokenScopes{}
    for _, scope := range scopes {
        scope = strings.TrimSpace(scope)
        if scope == "repo" {
            ts.HasRepo = true
        }
        if scope == "actions" || scope == "repo" { // repo includes actions
            ts.HasActions = true
        }
        if scope == "project" || scope == "read:org" {
            ts.HasProject = true
        }
    }

    return ts, nil
}

func (ts *TokenScopes) Validate() error {
    var missing []string
    if !ts.HasRepo {
        missing = append(missing, "repo")
    }
    if !ts.HasActions {
        missing = append(missing, "actions (or repo)")
    }
    if !ts.HasProject {
        missing = append(missing, "project or read:org")
    }

    if len(missing) > 0 {
        return fmt.Errorf("missing required token scopes: %s", strings.Join(missing, ", "))
    }
    return nil
}
```

**Usage in main.go:**

```go
func main() {
    config.Init()
    github.NewClient(config.GitHub.Token)

    // Validate token scopes
    scopes, err := github.ValidateTokenScopes(config.GitHub.Token)
    if err != nil {
        log.Fatalf("Token validation failed: %v", err)
    }
    if err := scopes.Validate(); err != nil {
        log.Fatalf("Token missing required scopes: %v\nSee: https://github.com/skanehira/github-tui#token-scopes", err)
    }

    if err := ui.New().Start(); err != nil {
        log.Fatal(err)
    }
}
```

**Sources:**
- Token scope checking: [API for determining Personal Access Token scopes](https://github.com/orgs/community/discussions/24345)
- Required scopes: [GitHub Permissions for Fine-Grained PATs](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens)

### Pattern 4: Async UI Updates via Channel (EXISTING - REUSE)

**What:** Worker goroutines send update functions to a buffered channel (`UI.updater`), which a dedicated goroutine forwards to `app.QueueUpdateDraw()` to avoid deadlocks.

**When to use:** Any time a goroutine (API calls, background tasks) needs to update the TUI.

**Example (EXISTING PATTERN):**

```go
// ui/ui.go (lines 24, 32, 188-192)
type ui struct {
    app          *tview.Application
    pages        *tview.Pages
    current      int
    primitives   []Primitive
    primitiveLen int
    updater      chan func()  // Buffered channel for async updates
}

func New() *ui {
    ui := &ui{
        app: tview.NewApplication(),
    }

    ui.updater = make(chan func(), 100)  // Buffer size: 100

    UI = ui
    return ui
}

// Start the update forwarder goroutine
func (ui *ui) Start() error {
    // ... UI initialization ...

    go func() {
        for f := range UI.updater {
            go ui.app.QueueUpdateDraw(f)  // Forward to tview
        }
    }()

    if err := ui.app.Run(); err != nil {
        ui.app.Stop()
        return err
    }
    return nil
}

// Usage from worker goroutines (ui/select.go line 104)
func (ui *SelectUI) UpdateView() {
    UI.updater <- func() {
        ui.Clear()
        for i, h := range ui.header {
            ui.SetCell(0, i, &tview.TableCell{
                Text:            h,
                NotSelectable:   true,
                Align:           tview.AlignLeft,
                Color:           tcell.ColorWhite,
                BackgroundColor: tcell.ColorDefault,
                Attributes:      tcell.AttrBold | tcell.AttrUnderline,
            })
        }
        // ... render items ...
    }
}
```

**Why this works:**
1. Worker goroutines never call `QueueUpdateDraw()` directly (prevents deadlock)
2. Buffered channel (size 100) absorbs burst updates
3. Dedicated forwarder goroutine handles backpressure
4. Each update function runs in its own goroutine from QueueUpdateDraw (concurrent execution)

**Source:** Existing implementation in /Users/gustav/src/github-tui/ui/ui.go and verified against [tview issue #690](https://github.com/rivo/tview/issues/690) and [tview issue #199](https://github.com/rivo/tview/issues/199).

### Anti-Patterns to Avoid

- **Calling QueueUpdateDraw from event handlers:** Event handlers run on the main goroutine. Calling QueueUpdateDraw from SetInputCapture or similar can deadlock if the update queue is full. Solution: Use UI.updater channel or modify UI directly in handlers.
- **Using separate http.Client instances for REST and GraphQL:** Leads to independent rate limit state, no unified tracking. Solution: Share the same wrapped http.Client.
- **Ignoring GraphQL rateLimit.cost in responses:** GraphQL cost varies per query. Not tracking it causes unexpected throttling. Solution: Parse response body rateLimit field.
- **Hard-coding rate limits without parsing headers:** GitHub can adjust limits dynamically. Solution: Always parse X-RateLimit-* headers and update limiter state.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GitHub REST API client | Custom HTTP client with manual JSON parsing | google/go-github/v83 | Library handles pagination (LinkHeader parsing), rate limit headers, retry logic, type-safe structs for all endpoints, and edge cases (conditional requests, ETags). Latest v83 adds native iterators for List* methods. |
| Token bucket rate limiter | Custom time.Ticker with counters | golang.org/x/time/rate.Limiter | Handles burst capacity, fractional rates, context cancellation, thread-safe token management, and Reserve/Wait/Allow semantics correctly. Custom implementations often fail under concurrent load. |
| OAuth2 token refresh | Manual token expiry tracking | golang.org/x/oauth2 | Already in use, handles token refresh, transport wrapping, and expiry logic. GitHub PATs don't expire but pattern is established. |

**Key insight:** GitHub API client libraries (go-github, githubv4) are maintained by the community and handle thousands of edge cases discovered over years of production use. Rate limiting is deceptively complex with burst handling, concurrent requests, and context cancellation. Established libraries prevent subtle bugs that only appear under load.

## Common Pitfalls

### Pitfall 1: tview QueueUpdateDraw Deadlock from Event Handlers

**What goes wrong:** Calling `app.QueueUpdateDraw()` from within `SetInputCapture()` callbacks or other event handlers causes deadlock if the update queue buffer is full. The main goroutine blocks waiting to send to the channel while also being responsible for processing the channel.

**Why it happens:** Event handlers execute on the main goroutine (the same one running the tview event loop). When they try to queue an update via QueueUpdateDraw and the buffer is full, they block waiting for space. But the only goroutine that can free space is the main goroutine, creating a circular wait.

**How to avoid:**
1. NEVER call `QueueUpdateDraw()` from event handlers (SetInputCapture, SetSelectedFunc, SetInputCapture, etc.)
2. Modify UI directly in event handlers — they already run on the main goroutine
3. For long-running operations triggered by events, spawn a goroutine and use the UI.updater channel

**Warning signs:**
- App becomes unresponsive after certain key presses
- Deadlock occurs consistently with specific UI interactions
- Stack trace shows blocking on channel send in event handler

**Sources:**
- [tview issue #199: SetInputCapture deadlock](https://github.com/rivo/tview/issues/199)
- [tview issue #690: QueueUpdateDraw deadlock](https://github.com/rivo/tview/issues/690)

### Pitfall 2: REST vs GraphQL Rate Limit Confusion

**What goes wrong:** Developers assume REST and GraphQL share the same rate limit pool, or only track one while ignoring the other. This leads to unexpected 429 (rate limit) errors and request failures.

**Why it happens:** GitHub uses separate rate limiting for REST (5000 requests/hour) and GraphQL (5000 points/hour). REST is simple counting, GraphQL uses a complexity-based cost system. Both share a 100 concurrent request limit, which is easy to miss.

**How to avoid:**
1. Track both pools independently with separate rate.Limiter instances
2. Parse REST headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
3. Parse GraphQL response body: `data.rateLimit.cost`, `data.rateLimit.remaining`
4. Enforce concurrent request limit with semaphore (90 conservative vs 100 actual limit)
5. Display both limits in TUI (e.g., status bar: "API: REST 4200/5000 | GraphQL 3800/5000")

**Warning signs:**
- 429 rate limit errors when you expect quota remaining
- One API works but the other gets throttled
- Concurrent request failures with 503 or timeout errors

**Source:** [GitHub GraphQL Rate Limits](https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api)

### Pitfall 3: Token Scope Validation Only with Fine-Grained PATs

**What goes wrong:** Using `X-OAuth-Scopes` header to validate token scopes works for classic PATs but returns empty for fine-grained PATs. App incorrectly reports missing scopes when token is actually valid.

**Why it happens:** Fine-grained PATs use a different permission model (resource-based permissions) and don't include `X-OAuth-Scopes` in response headers. Classic PATs use scope-based permissions and return scopes in the header.

**How to avoid:**
1. Try X-OAuth-Scopes header first (classic PAT detection)
2. If header is empty or missing, attempt actual API calls for required resources
3. Catch 403/404 errors and parse error message for permission issues
4. Provide clear error message: "Token validation failed. Ensure token has: repo, actions, project (or read:org) permissions."
5. Consider degraded mode: skip scope validation and fail fast on first permission error

**Warning signs:**
- Token validation fails but API calls succeed
- Empty X-OAuth-Scopes header in responses
- Users report "missing scopes" error with valid fine-grained PATs

**Source:** [Fine-grained vs Classic PAT differences](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)

### Pitfall 4: Concurrent Request Limit (100) Ignored

**What goes wrong:** App spawns many goroutines for parallel API calls (e.g., fetching workflow runs + jobs + logs concurrently), exceeding GitHub's 100 concurrent request limit. Results in 503 errors, timeouts, or temporary bans.

**Why it happens:** Both REST and GraphQL share a single 100 concurrent request limit across all requests. Developers focus on per-hour rate limits and miss the concurrency constraint documented separately.

**How to avoid:**
1. Use semaphore (buffered channel) with capacity 90 (conservative) to limit concurrent requests
2. Acquire semaphore before each API call, release after response
3. Implement in http.RoundTripper middleware so it applies to both REST and GraphQL
4. Consider user-configurable limit for users with higher GitHub Enterprise quotas

**Warning signs:**
- 503 Service Unavailable errors during parallel fetches
- Intermittent timeout errors with fast retry recovery
- Rate limit headers show quota remaining but requests still fail

**Source:** [GitHub GraphQL concurrent request limit](https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api)

## Code Examples

Verified patterns from official sources:

### Creating go-github Client with OAuth2

```go
import (
    "context"
    "github.com/google/go-github/v83/github"
    "golang.org/x/oauth2"
)

func createClient(token string) *github.Client {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )
    tc := oauth2.NewClient(ctx, ts)

    return github.NewClient(tc)
}
```

**Source:** [google/go-github README](https://github.com/google/go-github) - verified pattern from official documentation.

### Checking Rate Limit Status

```go
// REST rate limit via go-github
func checkRESTRateLimit(client *github.Client) {
    ctx := context.Background()
    limits, _, err := client.RateLimits(ctx)
    if err != nil {
        log.Printf("Rate limit check failed: %v", err)
        return
    }

    core := limits.Core
    fmt.Printf("REST: %d/%d remaining, resets at %v\n",
        core.Remaining, core.Limit, core.Reset.Time)
}
```

**Source:** go-github provides `Client.RateLimits()` method — [pkg.go.dev documentation](https://pkg.go.dev/github.com/google/go-github/v83/github)

### GraphQL Query with Cost Tracking

```go
// Existing githubv4 pattern - extend to track rateLimit
func queryWithRateLimit(client *githubv4.Client) {
    var q struct {
        Repository struct {
            Issues struct {
                Nodes []struct {
                    Title string
                }
            } `graphql:"issues(first: 10)"`
        } `graphql:"repository(owner: $owner, name: $name)"`
        RateLimit struct {
            Cost      int
            Remaining int
            Limit     int
        }
    }

    variables := map[string]interface{}{
        "owner": githubv4.String("owner"),
        "name":  githubv4.String("repo"),
    }

    err := client.Query(context.Background(), &q, variables)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("GraphQL: Cost=%d, Remaining=%d/%d\n",
        q.RateLimit.Cost, q.RateLimit.Remaining, q.RateLimit.Limit)
}
```

**Source:** Existing codebase pattern + [GitHub GraphQL rate limit documentation](https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api)

### Rate Limiter with golang.org/x/time/rate

```go
import (
    "context"
    "golang.org/x/time/rate"
    "time"
)

// Create limiter: 5000 req/hr = 1.39 req/sec, allow burst of 10
limiter := rate.NewLimiter(rate.Limit(1.39), 10)

// Wait for token (blocks until available or context canceled)
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := limiter.Wait(ctx)
if err != nil {
    // Context canceled or deadline exceeded
    return err
}

// Make API call
resp, err := client.Do(req)
```

**Source:** [golang.org/x/time/rate documentation](https://pkg.go.dev/golang.org/x/time/rate)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual rate limit tracking with counters | golang.org/x/time/rate token bucket | Always standard practice | More accurate burst handling, context-aware waiting, thread-safe |
| Separate HTTP clients for REST/GraphQL | Shared http.Client with unified middleware | Recommended pattern since oauth2 library stabilized (2015+) | Single rate limit state, consistent error handling, shared auth |
| Direct QueueUpdateDraw calls | Channel-based update queue | Best practice since tview introduced (to avoid deadlocks) | Prevents deadlock from event handlers, allows buffering |
| Classic PAT scope validation via X-OAuth-Scopes | Try-and-catch with fine-grained PAT support | Fine-grained PATs introduced 2022 | Must handle both token types |
| go-github v32 (older in codebase) | go-github v83 with native iterators | v83 released 2026-02-16 | Simplified pagination, better API coverage |

**Deprecated/outdated:**
- **go-github < v48**: Pre-v48 versions don't support newer GitHub APIs (Actions workflow dispatch, Projects V2 fields). Minimum v60+ recommended for Actions, v70+ for Projects V2.
- **Manual pagination with LinkHeader parsing**: go-github v83 introduces native iterators (`ListAll*` methods) that handle pagination automatically.
- **Hard-coded rate limits**: Always parse X-RateLimit headers and GraphQL rateLimit fields. GitHub adjusts limits dynamically for Enterprise users.

## Open Questions

1. **GraphQL response body parsing in RoundTripper**
   - What we know: Must parse JSON response body to extract rateLimit.cost and rateLimit.remaining
   - What's unclear: http.RoundTripper should not typically consume response body (breaks io.Reader for client). Need to peek/copy body without consuming it.
   - Recommendation: Use io.TeeReader to duplicate body stream, parse copy for rateLimit, leave original for client. Alternative: Parse rateLimit in githubv4 client wrapper instead of RoundTripper.

2. **Fine-grained PAT scope validation strategy**
   - What we know: X-OAuth-Scopes header empty for fine-grained PATs
   - What's unclear: Best approach — fail fast on startup, attempt validation calls, or degrade gracefully?
   - Recommendation: Implement degraded mode — skip scope validation if X-OAuth-Scopes is empty, let first API call fail with clear error. Provide config flag for strict validation (fail on startup).

3. **Rate limit display location in TUI**
   - What we know: User has full discretion on visibility approach
   - What's unclear: Status bar, log-only, warning modal, or combination?
   - Recommendation: Start with log-only (least invasive), add status bar display as enhancement if requested. Warning modal only on approaching limit (< 10% remaining).

4. **Channel buffer size tuning**
   - What we know: Current buffer is 100 (ui.updater channel)
   - What's unclear: Optimal size for concurrent API calls from Phase 2/3?
   - Recommendation: Keep 100 for Phase 1. Monitor in Phase 2 (Actions can fetch many logs concurrently). Increase to 200 if warnings about dropped updates appear.

## Sources

### Primary (HIGH confidence)

- [google/go-github v83.0.0 release](https://github.com/google/go-github/releases) - Latest version verification (2026-02-16)
- [golang.org/x/time/rate package](https://pkg.go.dev/golang.org/x/time/rate) - Rate limiter API and usage
- [GitHub REST API Rate Limiting](https://docs.github.com/en/rest/rate-limit) - REST rate limit headers and endpoints
- [GitHub GraphQL Rate Limits](https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api) - GraphQL points, cost calculation, concurrent limits
- [GitHub Token Permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens) - Required scopes for Actions, Projects V2, repo access
- Existing codebase: /Users/gustav/src/github-tui/github/client.go, /Users/gustav/src/github-tui/ui/ui.go - Verified current architecture

### Secondary (MEDIUM confidence)

- [Writing HTTP client middleware in Go](https://echorand.me/posts/go-http-client-middleware/) - http.RoundTripper pattern with code examples
- [tview issue #199](https://github.com/rivo/tview/issues/199) - QueueUpdateDraw deadlock from SetInputCapture, maintainer guidance
- [tview issue #690](https://github.com/rivo/tview/issues/690) - QueueUpdateDraw usage clarification, multiple Application instances issue
- [API for determining Personal Access Token scopes](https://github.com/orgs/community/discussions/24345) - X-OAuth-Scopes header usage
- [Go by Example: Rate Limiting](https://gobyexample.com/rate-limiting) - Rate limiting patterns with channels
- [How to rate limit HTTP requests in Go](https://www.alexedwards.net/blog/how-to-rate-limit-http-requests) - HTTP rate limiting with x/time/rate

### Tertiary (LOW confidence)

- WebSearch results for "golang oauth2 custom transport" - General patterns confirmed by official docs
- WebSearch results for "tview deadlock goroutine" - Community discussions confirming pitfalls

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - google/go-github v83 verified from official releases (2026-02-16), x/time/rate is stdlib extension, existing libs confirmed in go.mod
- Architecture: HIGH - Patterns verified from existing codebase (github/client.go, ui/ui.go), official library documentation, and maintainer responses in GitHub issues
- Pitfalls: HIGH - Deadlock issues documented in official tview issues with maintainer responses, rate limit confusion from official GitHub docs, concurrent limit in official GraphQL docs

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (30 days) - Rate limiting and API patterns are stable. Monitor for google/go-github v84+ releases and tview updates.
