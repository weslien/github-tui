# Pitfalls Research

**Domain:** GitHub Actions API + Projects V2 GraphQL Integration in Go TUI
**Researched:** 2026-02-16
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: shurcooL/githubv4 Schema Lag with Projects V2

**What goes wrong:**
The shurcooL/githubv4 library uses code generation from GitHub's GraphQL schema, but Projects V2 is a rapidly evolving API with frequent schema changes. The library may not have up-to-date type definitions for newer Projects V2 fields, custom field types, or mutations. This causes compilation errors or runtime panics when trying to access fields that exist in GitHub's API but not in the generated Go types.

**Why it happens:**
Projects V2 launched after the library was stable, and GitHub announces breaking changes only 3 months in advance with quarterly releases (Jan 1, Apr 1, Jul 1, Oct 1). The library maintainer may lag behind GitHub's schema updates. Custom fields in Projects V2 are particularly problematic because they're dynamically created by users and have varying types (single-select, iteration, milestone, etc.) that require union type support.

**How to avoid:**
1. Check the library's last schema update date against GitHub's GraphQL changelog
2. Test against GitHub's GraphQL API directly using `curl` before implementing in Go
3. Consider using raw GraphQL queries with `encoding/json` for Projects V2 to avoid schema dependency
4. Implement fallback to `map[string]interface{}` for custom fields rather than rigid struct types
5. Use GitHub's GraphQL Explorer to verify field availability before coding

**Warning signs:**
- Compilation errors about missing struct fields that exist in GitHub docs
- Runtime errors: "json: cannot unmarshal object into Go value of type X"
- Fields returning `null` despite being populated in GitHub UI
- Errors with ProjectV2ItemFieldValue unions (text, number, date, single-select, iteration)
- Field ordering bugs where API returns partial results based on custom field positions

**Phase to address:**
Phase 1 (Foundation) - Evaluate library compatibility and establish schema update monitoring. May require wrapper layer for Projects V2 queries.

**Sources:**
- [Projects V2 API limitations discussion](https://github.com/orgs/community/discussions/44265)
- [shurcooL/githubv4 GitHub repo](https://github.com/shurcooL/githubv4)
- [GitHub GraphQL Schema Breaking Changes](https://docs.github.com/en/graphql/overview/breaking-changes)
- [Field ordering bug in Projects V2](https://github.com/orgs/community/discussions/164519)

---

### Pitfall 2: GraphQL Node Limit Explosion with Nested Queries

**What goes wrong:**
GitHub's GraphQL API has a 500,000 node limit per query. Nested queries multiply node consumption exponentially, not additively. Requesting 50 repositories with 20 pull requests each and 10 comments per PR consumes 22,060 nodes, rapidly approaching the limit. When combined with Projects V2 items that have multiple custom fields, a seemingly innocent pagination request can exceed limits and fail with cryptic error messages.

**Why it happens:**
Developers think in terms of "items per page" (e.g., `first: 30`) without realizing nested connections multiply. The formula is: parent_nodes + (parent_nodes × child_limit) + (parent_nodes × child_limit × grandchild_limit). Projects V2 exacerbates this because each item has fields (text, number, date, single-select, iteration, assignees, labels, milestone) that are separate GraphQL connections.

**How to avoid:**
1. Keep `first`/`last` arguments low (10-20) for nested queries, not 30
2. Split complex queries into multiple simpler queries
3. Use GraphQL fragments to avoid repeating expensive nested structures
4. Calculate node consumption: sum all `(parent_count × child_limit)` across nesting levels
5. For Projects V2: fetch items first, then fetch field values in separate query
6. Monitor rate limit headers: `X-RateLimit-Remaining` and `X-RateLimit-Cost`

**Warning signs:**
- GraphQL errors: "Your query has X nodes which exceeds the maximum of 500000"
- Rate limit exhaustion faster than expected (5000 points/hour for users)
- Queries that worked with 5 repos fail with 50
- High complexity scores (>1000) in rate limit headers
- Slow response times (>3s) indicating backend strain

**Phase to address:**
Phase 1 (Foundation) - Establish query complexity calculator and pagination strategy. Critical for Projects V2 which has deeply nested field structures.

**Sources:**
- [GitHub GraphQL Rate Limits](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api)
- [Using Pagination in GraphQL API](https://docs.github.com/en/graphql/guides/using-pagination-in-the-graphql-api)

---

### Pitfall 3: tview Deadlock with QueueUpdateDraw from API Goroutines

**What goes wrong:**
When long-running API calls complete in goroutines and try to update the TUI using `App.QueueUpdateDraw()`, the application deadlocks if the update queue buffer is full or if called from the wrong context. This is especially problematic with Actions log streaming and Projects V2 real-time updates where multiple goroutines are fetching data concurrently. The TUI becomes completely unresponsive and requires force-kill.

**Why it happens:**
tview's `QueueUpdate()` uses a buffered channel that can fill up. Calling it from within an event handler (which runs on the main goroutine) causes deadlock because the main loop is blocked waiting for itself. The issue is amplified when polling Actions runs every 5-10 seconds or streaming logs, creating many concurrent update attempts.

**How to avoid:**
1. **NEVER** call `QueueUpdate()` from event handlers (SetInputCapture, button press callbacks)
2. Use `App.Draw()` directly when already on the main goroutine
3. Only use `QueueUpdateDraw()` from worker goroutines spawned outside event loop
4. Implement update coalescing: collect updates in a channel, have single goroutine drain and update UI
5. For TextView updates (logs), use `io.Writer` interface which is goroutine-safe
6. Use `select` with timeout when calling `QueueUpdateDraw()` to detect deadlock early

**Warning signs:**
- UI freezes completely, no key input accepted
- CPU usage drops to 0% while app is running
- goroutine count increasing in debug builds
- No response to Ctrl+C (SIGINT)
- Happens after API calls complete, not during initial render

**Phase to address:**
Phase 1 (Foundation) - Establish UI update architecture pattern. Must be correct from start as retrofitting is painful.

**Sources:**
- [tview QueueUpdate deadlock issue](https://github.com/rivo/tview/issues/690)
- [SetInputCapture deadlock issue](https://github.com/rivo/tview/issues/199)
- [Go TUI with concurrent updates](https://taranveerbains.ca/blog/13-making-a-tui-with-go)

---

### Pitfall 4: REST vs GraphQL Rate Limit Confusion

**What goes wrong:**
GitHub Actions API requires REST while Projects V2 requires GraphQL. Both share the same authentication token but have separate rate limit pools with different calculation methods. Developers assume they have 5000 requests available but Actions uses REST (5000 requests/hour) while Projects V2 uses GraphQL (5000 points/hour with complex calculations). The app exhausts one pool thinking it's fine because the other shows capacity. More critically, the 100 concurrent request limit is shared across both APIs, causing throttling when mixing calls.

**Why it happens:**
The rate limit response headers differ between APIs. REST returns `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`. GraphQL returns cost and remaining points in response JSON body, not headers. Developers check one but not the other, or implement separate clients that don't communicate rate limit state. The shared 100 concurrent request limit is often ignored because it's buried in documentation.

**How to avoid:**
1. Implement unified rate limit tracker that monitors both REST and GraphQL pools
2. Parse GraphQL cost from response body: `data.rateLimit.cost` and `data.rateLimit.remaining`
3. Parse REST headers: `X-RateLimit-*` after each request
4. Track concurrent request count globally, limit to 90 (safety margin)
5. Implement exponential backoff on 429 (rate limit) and 403 (secondary limit) responses
6. Use semaphore pattern: `make(chan struct{}, 90)` to enforce concurrent limit
7. Show both rate limits in TUI status bar for visibility

**Warning signs:**
- 429 Too Many Requests from GitHub API
- 403 Forbidden with message about secondary rate limits
- Inconsistent errors: works sometimes, fails randomly
- GraphQL returns cost=0 but REST is exhausted (or vice versa)
- Exponential slowdown as app hits limits repeatedly

**Phase to address:**
Phase 1 (Foundation) - Critical for reliability. Rate limiting must be built into API client layer from start.

**Sources:**
- [Understanding GitHub API Rate Limits](https://github.com/orgs/community/discussions/163553)
- [Rate Limits for GraphQL API](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api)
- [Rate Limits for REST API](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)

---

### Pitfall 5: Actions Pagination Hardcoded 10 Page Limit

**What goes wrong:**
GitHub's REST API for listing workflow runs (`/repos/{owner}/{repo}/actions/runs`) has an undocumented hard limit of 10 pages. Even if the `Link` header shows `rel="next"`, you can only paginate through 1000 items (100 per page × 10 pages max). Older workflow runs become inaccessible via standard pagination. This breaks "view history" features where users expect to see all runs, not just the most recent 1000.

**Why it happens:**
GitHub intentionally limits deep pagination for performance reasons but doesn't document it clearly. The API continues to return `Link: <...>; rel="next"` beyond page 10, giving the false impression more data is available. GitHub expects clients to use the `created` filter with date ranges to access older data, but this requires a different pagination strategy.

**How to avoid:**
1. Use `created` filter with date ranges instead of pure offset pagination
2. Implement "load older" that fetches `created:<YYYY-MM-DD` instead of incrementing page
3. Cache workflow runs locally in SQLite for unlimited history
4. Show warning in UI: "Displaying last 1000 runs (GitHub API limit)"
5. For complete history: fetch page 1-10, then use oldest run's `created_at` date to fetch earlier batches
6. Consider using GraphQL API if it has better pagination (verify Projects V2 approach)

**Warning signs:**
- Pagination stops returning results after page 10
- `Link` header exists but returns empty results
- Older workflow runs visible in GitHub UI but not in your app
- Page 10 returns data, page 11 returns empty array
- Users report "missing workflow runs from last month"

**Phase to address:**
Phase 2 (Actions Integration) - Must be addressed during Actions API implementation to avoid user confusion.

**Sources:**
- [GitHub Actions pagination 10 page limit](https://github.com/orgs/community/discussions/26782)
- [REST API workflow runs endpoint](https://docs.github.com/en/rest/actions/workflow-runs)

---

### Pitfall 6: Projects V2 Node ID vs Database ID Confusion

**What goes wrong:**
GitHub Projects V2 uses two ID types: global node IDs (e.g., `PVTI_lADOAR...`) and legacy database IDs (integers). The GraphQL API requires node IDs, but webhooks and some REST API endpoints return database IDs. Direct node lookups via `node(id: "...")` fail for Projects V2 field IDs and option IDs with "Could not resolve to a node with the global id", even though the IDs are valid when queried through parent resources. This makes webhook-driven updates nearly impossible.

**Why it happens:**
GitHub is migrating from integer IDs to global node IDs. Projects V2 was built during this transition, resulting in inconsistent ID formats. Field IDs and option IDs from webhooks aren't valid for direct `node()` queries - they only work when traversed through the organization → project → field path. The `X-Github-Next-Global-ID: 1` header forces new ID format, but legacy endpoints still return old format.

**How to avoid:**
1. Always query Projects V2 data through parent paths, never direct `node()` lookups
2. Store full object paths in cache, not just IDs: `{orgId, projectId, fieldId}`
3. Use field names as lookup keys, not field IDs from webhooks
4. Implement ID translation layer: map webhook IDs to GraphQL node IDs via full query
5. For updates: query project → find field by name → get correct node ID → perform mutation
6. Add `X-Github-Next-Global-ID: 1` header to future-proof against ID migration

**Warning signs:**
- Error: "Could not resolve to a node with the global id of '...'"
- Field IDs work in one query, fail in another
- Webhook payloads have different ID format than GraphQL responses
- Direct node lookups fail but parent queries succeed with same ID
- Inconsistent ID formats: some start with `PVTI_`, others are integers

**Phase to address:**
Phase 3 (Projects V2 Integration) - Must be solved before webhook support or real-time updates.

**Sources:**
- [Using Global Node IDs](https://docs.github.com/en/graphql/guides/using-global-node-ids)
- [Migrating GraphQL Global Node IDs](https://docs.github.com/en/graphql/guides/migrating-graphql-global-node-ids)
- [Projects V2 field ID node resolution failure](https://github.com/orgs/community/discussions/50253)
- [Projects V2 API status field management limitation](https://github.com/orgs/community/discussions/44265)

---

### Pitfall 7: PAT Token Scope Insufficient for Projects V2

**What goes wrong:**
Classic Personal Access Tokens (PATs) with standard `repo` scope can read Actions data but cannot access Projects V2 API. Projects V2 requires either classic PAT with `project` scope (and `read:org` for org-owned projects) or fine-grained PAT with "Projects: Read+Write" permission. Worse, fine-grained PATs don't work with user-owned projects at all. The app fails with 403 or returns null for Projects V2 queries despite valid authentication for other APIs.

**Why it happens:**
Projects V2 was launched after GitHub introduced fine-grained PATs, creating scope fragmentation. GitHub's default `GITHUB_TOKEN` in Actions also cannot access Projects V2 API. The distinction between user-owned and org-owned projects isn't obvious, and GitHub recommends fine-grained PATs (which don't work for user projects) over classic PATs (which do work but are being deprecated).

**How to avoid:**
1. Document required scopes prominently: classic PAT with `project` + `read:org` + `repo`
2. Implement token validation on startup: test Projects V2 query before app runs
3. Provide clear error message: "Token missing 'project' scope. Visit GitHub Settings > Tokens"
4. For user projects: enforce classic PAT, reject fine-grained tokens early
5. For org projects: accept both classic and fine-grained with proper permissions
6. Check `X-OAuth-Scopes` response header to detect missing scopes programmatically
7. Build scope checker: query `/user` endpoint, parse `X-OAuth-Scopes` header

**Warning signs:**
- 403 Forbidden specifically on Projects V2 queries, not Actions queries
- GraphQL returns `"data": { "viewer": { "projectV2": null } }`
- Error message: "Resource not accessible by personal access token"
- Actions API works, Projects V2 fails with same token
- Works for some projects (org) but not others (user)

**Phase to address:**
Phase 1 (Foundation) - Must be caught during initial authentication setup to prevent user frustration.

**Sources:**
- [Managing Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
- [Projects V2 PAT permissions discussion](https://github.com/orgs/community/discussions/46681)
- [GitHub Agentic Workflows Authorization](https://github.github.com/gh-aw/reference/auth/)

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Polling Actions runs every 5s | Simple to implement | Rate limit exhaustion, poor performance | Never - use 30s minimum or webhooks |
| Storing only item IDs in cache | Small memory footprint | Requires full query to retrieve data, node ID resolution issues | Never for Projects V2 - store full objects |
| Using `first: 100` for all queries | Fewer pagination requests | Hits node limit quickly with nested data | Never - use 10-20 for nested queries, 50-100 for flat |
| Single goroutine for all API calls | No concurrency issues | Slow UI, blocks on rate limits | Only for MVP, refactor before Actions integration |
| Ignoring GraphQL cost in responses | Simpler code | No rate limit awareness, unexpected throttling | Never - cost tracking is essential |
| Using `interface{}` for Projects V2 fields | Works with any custom field type | Type safety lost, runtime panics | Acceptable for MVP, add type assertions later |
| Hardcoding workflow file names | Fast lookup | Breaks when users rename workflows | Only for demos - use API to enumerate workflows |

---

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| GitHub Actions Logs | Downloading entire log file (100MB+) blocking UI | Stream logs in chunks, download to temp file in background, display tail |
| Projects V2 Status Fields | Trying to create/modify status columns via API | Accept read-only limitation, show warning: "Status columns managed in GitHub UI only" |
| GraphQL Pagination | Reusing same `after` cursor across queries | Each nested connection has separate cursor, must track per-level |
| Mixed REST/GraphQL | Using separate HTTP clients with different configs | Unified client with shared rate limit state, timeout config, retry logic |
| Actions Workflow Status | Polling status immediately after trigger | Runs stay "queued" for 5-30s, implement 10s delay before first poll |
| Projects V2 Custom Fields | Assuming field type from name (e.g., "Priority" = number) | Query field metadata, cache `__typename` to determine single-select/number/date/text |
| GraphQL Errors | Checking only HTTP status | GraphQL returns 200 with errors in JSON body, must parse `errors` array |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Full Project Item Scan | Slow load times, high rate limit cost | Fetch items with `first: 20`, implement virtual scrolling | >100 items per project |
| Polling All Active Workflows | Rate limit exhaustion | Poll only visible workflow, implement priority queue | >5 concurrent runs |
| Rendering Full Log in TextView | 100MB+ log freezes TUI | Virtual scrolling, render visible lines only (500 line buffer) | Logs >10k lines |
| Synchronous API Calls | UI blocks for 2-5s per call | All API calls in goroutines, show loading indicators | Always noticeable |
| No Query Deduplication | Multiple identical queries in-flight | Request deduplication: cache in-flight queries by key | >10 concurrent API calls |
| Fetching All Custom Fields | High node consumption (50+ fields per project) | Fetch only fields visible in current view | Projects with >10 custom fields |
| Naive GraphQL Nesting | Query cost >1000, slow responses | Flatten queries: separate passes for items vs fields | >3 levels of nesting |

---

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing PAT in plaintext config file | Token leaked in dotfiles repo, full GitHub access compromised | Use OS keychain (keyring library), never commit tokens |
| Logging API responses | Sensitive data (private repo content, user emails) in logs | Redact sensitive fields, log only IDs and status codes |
| Insufficient PAT scope validation | User grants `repo` but needs `project`, app fails cryptically | Validate scopes on startup, fail fast with clear message |
| No token expiration handling | Fine-grained PATs expire, app stops working silently | Check expiration metadata, warn 7 days before expiry |
| Displaying raw webhook payloads | Exposes internal org structure, user data | Sanitize and whitelist displayed fields |
| No rate limit error distinction | 403 secondary rate limit treated as auth error | Parse error message: "secondary rate limit" vs "forbidden" |
| Token in error messages | Exception logged with full API request including auth header | Strip `Authorization` header from error context |

---

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No visual rate limit indicator | Mysterious slowdowns, failed operations | Status bar shows "API: 4200/5000 reqs, resets 23:45" |
| Blocking UI during log download | TUI frozen for 30s-2min | Background download with progress bar, allow cancel |
| No workflow run status color coding | Can't distinguish failed vs success vs running at a glance | Red (failed), Green (success), Yellow (running), Gray (queued) |
| Flat list of all Actions runs | Overwhelming, hard to find specific workflow | Group by workflow name, collapsible sections |
| No "why did this fail" context | Users see "API error" with no actionable info | Show specific error: "Rate limit exceeded, retry at 15:30" or "Missing 'project' token scope" |
| Auto-refresh without indicator | UI updates unpredictably, cursor jumps | Visual flash or status "Updated 5s ago" |
| No keyboard shortcut discoverability | Users don't know `r` refreshes or `l` shows logs | Footer bar: `r:refresh l:logs q:quit` |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Actions Integration:** Workflow run status refreshing - verify handles `queued` → `in_progress` → `completed` transitions, not just final state
- [ ] **Actions Logs:** Log viewer implemented - verify handles multi-job workflows (separate log per job), not just single job
- [ ] **Projects V2 Items:** Item list displaying - verify custom fields render correctly for all types (text, number, date, single-select, iteration, milestone)
- [ ] **Projects V2 Status:** Status column shown - verify read-only message displayed, mutation attempts blocked with explanation
- [ ] **Pagination:** "Load more" button works - verify handles end-of-data (empty results), not just hasNextPage flag
- [ ] **Rate Limits:** API calls succeed - verify both REST and GraphQL limits tracked separately, not just one
- [ ] **Error Handling:** Errors displayed to user - verify distinguishes network errors, auth errors, rate limits, and API errors with specific messages
- [ ] **Token Validation:** App starts successfully - verify token scope check happens before any API calls, not after first failure
- [ ] **Concurrent Updates:** Multiple API calls in progress - verify no race conditions, deadlocks, or duplicate requests
- [ ] **GraphQL Complexity:** Complex queries work - verify node consumption calculated and limited, not just trusting GitHub to reject

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| shurcooL/githubv4 Schema Lag | HIGH | Replace with raw GraphQL client using `net/http` and `encoding/json`, rewrite all Projects V2 queries, maintain query templates |
| GraphQL Node Limit Explosion | MEDIUM | Split query into multiple smaller queries, reduce pagination size, cache results to minimize re-fetching |
| tview QueueUpdateDraw Deadlock | HIGH | Redesign update architecture: channel-based update queue with single UI goroutine, refactor all API completion handlers |
| Rate Limit Confusion | LOW | Add unified rate limit middleware, extract into separate package, inject into both REST and GraphQL clients |
| Actions Pagination 10 Page Limit | MEDIUM | Implement date-based pagination with `created` filter, may require local caching layer for full history |
| Projects V2 Node ID Issues | MEDIUM | Add ID translation cache: webhook ID → full GraphQL path → node ID, query via parent relationships instead of direct lookup |
| Insufficient PAT Scopes | LOW | Detect at startup, show clear error with link to token settings, block operation until user updates token |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| shurcooL/githubv4 Schema Lag | Phase 1: Foundation | Test Projects V2 custom fields query during library evaluation, check last schema update date |
| GraphQL Node Limit Explosion | Phase 1: Foundation | Create complexity calculator, test with realistic nested queries, verify cost stays <1000 |
| tview QueueUpdateDraw Deadlock | Phase 1: Foundation | Implement update queue pattern, stress test with 10 concurrent API goroutines |
| Rate Limit Confusion | Phase 1: Foundation | Implement unified rate limiter, verify tracks both pools, test rapid-fire requests |
| Actions Pagination 10 Page Limit | Phase 2: Actions Integration | Test with repo having 1500+ workflow runs, verify date-based pagination |
| Projects V2 Node ID Issues | Phase 3: Projects V2 Integration | Test direct node lookups vs parent queries, verify field ID resolution |
| Insufficient PAT Scopes | Phase 1: Foundation | Implement scope validation, test with token missing `project` scope |

---

## Sources

### GitHub API Documentation
- [GraphQL Rate Limits and Query Limits](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api)
- [REST API Rate Limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
- [Using Pagination in GraphQL API](https://docs.github.com/en/graphql/guides/using-pagination-in-the-graphql-api)
- [Using Global Node IDs](https://docs.github.com/en/graphql/guides/using-global-node-ids)
- [Migrating GraphQL Global Node IDs](https://docs.github.com/en/graphql/guides/migrating-graphql-global-node-ids)
- [GraphQL Breaking Changes Changelog](https://docs.github.com/en/graphql/overview/breaking-changes)
- [REST API Workflow Runs Endpoint](https://docs.github.com/en/rest/actions/workflow-runs)
- [Managing Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
- [Using the API to Manage Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects)

### Community Discussions & Issues
- [ProjectsV2 API: Can't manage issue status (columns)](https://github.com/orgs/community/discussions/44265)
- [ProjectsV2 GraphQL returns partial results based on field ordering](https://github.com/orgs/community/discussions/164519)
- [Understanding GitHub API Rate Limits: REST, GraphQL, and Beyond](https://github.com/orgs/community/discussions/163553)
- [Projects V2 Field ID node resolution failure](https://github.com/orgs/community/discussions/50253)
- [GitHub Actions pagination 10 page limit](https://github.com/orgs/community/discussions/26782)
- [How To Use Pagination With GitHub's API](https://github.com/orgs/community/discussions/69826)
- [Projects V2 PAT permissions discussion](https://github.com/orgs/community/discussions/46681)

### Go Libraries & TUI
- [shurcooL/githubv4 GitHub Repository](https://github.com/shurcooL/githubv4)
- [shurcooL/githubv4 Pagination Issue](https://github.com/shurcooL/githubv4/issues/20)
- [tview QueueUpdate Deadlock Issue](https://github.com/rivo/tview/issues/690)
- [tview SetInputCapture Deadlock Issue](https://github.com/rivo/tview/issues/199)
- [rivo/tview GitHub Repository](https://github.com/rivo/tview)

### Technical Articles
- [Intro to GraphQL using custom fields in GitHub Projects](https://some-natalie.dev/blog/graphql-intro/)
- [Examples for calling the GitHub GraphQL API (with ProjectsV2)](https://devopsjournal.io/blog/2022/11/28/github-graphql-queries)
- [Making a TUI with Go](https://taranveerbains.ca/blog/13-making-a-tui-with-go)
- [Building A Terminal User Interface With Golang](https://earthly.dev/blog/tui-app-with-go/)
- [A Developer's Guide: Managing Rate Limits for the GitHub API](https://www.lunar.dev/post/a-developers-guide-managing-rate-limits-for-the-github-api)

---

*Pitfalls research for: GitHub TUI (Actions + Projects V2 Integration)*
*Researched: 2026-02-16*
