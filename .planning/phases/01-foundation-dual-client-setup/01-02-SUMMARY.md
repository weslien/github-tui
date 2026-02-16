---
phase: 01-foundation-dual-client-setup
plan: 02
subsystem: api
tags: [token-validation, oauth-scopes, rate-limiting, transport-wiring, startup-checks]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Dual REST+GraphQL client, RateLimiter type with WrapTransport"
provides:
  - "Token scope validation (classic PAT via X-OAuth-Scopes, fine-grained PAT graceful degradation)"
  - "Rate limiter wired into shared HTTP transport for all API calls"
  - "GetRateLimiter() accessor for UI rate limit display"
  - "Startup validation sequence: config -> client -> token check -> UI"
affects: [02-actions, 03-projects-v2]

# Tech tracking
tech-stack:
  added: []
  patterns: [internal test helper for URL injection, startup fail-fast validation]

key-files:
  created:
    - github/token_validator.go
    - github/token_validator_test.go
  modified:
    - github/client.go
    - cmd/ght/main.go
    - .gitignore

key-decisions:
  - "ValidateTokenScopes uses direct HTTP request (not rate-limited client) for header access"
  - "Fine-grained PATs degrade gracefully (warn, don't block) since X-OAuth-Scopes is empty"
  - "validateTokenScopesInternal pattern for URL injection in tests (httptest servers)"
  - "admin:org scope implies project access (same as read:org)"

patterns-established:
  - "Startup fail-fast: validate token scopes before launching UI to prevent cryptic 403s"
  - "Internal test helper pattern: unexported function with URL param, exported wrapper for production"
  - "Graceful degradation: fine-grained PATs cannot be scope-checked, so warn and continue"

# Metrics
duration: 3min
completed: 2026-02-16
---

# Phase 1 Plan 2: Token Validation & Rate Limiter Wiring Summary

**Token scope validation on startup via X-OAuth-Scopes header with fine-grained PAT graceful degradation, plus rate limiter wired into shared OAuth2 transport for all API calls**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-16T21:32:04Z
- **Completed:** 2026-02-16T21:35:37Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Token scope validator that checks required scopes (repo, project/read:org) via GET /user X-OAuth-Scopes header
- Fine-grained PAT detection and graceful degradation (warn but allow app to continue)
- 9 tests covering all scenarios: all scopes present, missing scopes, fine-grained PAT, invalid token, scope implications, bearer token sending
- Rate limiter wired into shared OAuth2 HTTP transport so all REST and GraphQL calls flow through it
- Startup validation sequence: config -> client init -> token validation -> UI launch
- GetRateLimiter() accessor for future rate limit display in UI

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement token scope validator with tests** - `dd8a59d` (feat)
2. **Task 2: Wire rate limiter into client transport and add token validation to startup** - `06d94a8` (feat)

## Files Created/Modified
- `github/token_validator.go` - TokenScopes struct, ValidateTokenScopes, MissingScopes, Validate, scope parsing helpers
- `github/token_validator_test.go` - 9 table-driven tests with httptest servers covering all validation scenarios
- `github/client.go` - Rate limiter creation and WrapTransport wiring in NewClient, GetRateLimiter accessor
- `cmd/ght/main.go` - Token validation with 10s timeout in startup sequence before UI launch
- `.gitignore` - Fixed overly broad `ght` pattern to `/ght` (root-only binary ignore)

## Decisions Made
- ValidateTokenScopes makes its own HTTP request (not through the rate-limited client) because it needs direct header access and runs before rate limiter state matters
- Fine-grained PATs degrade gracefully: since X-OAuth-Scopes header is empty/missing for fine-grained PATs, scope validation is skipped with a warning log rather than blocking the user
- Used internal function pattern (validateTokenScopesInternal) for URL injection in tests, keeping the exported API clean
- admin:org scope implies project access, same as read:org (GitHub scope hierarchy)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed overly broad .gitignore pattern blocking cmd/ght/ staging**
- **Found during:** Task 2 (staging files for commit)
- **Issue:** `.gitignore` had bare `ght` pattern which matched both the root binary and `cmd/ght/` directory, preventing `git add cmd/ght/main.go`
- **Fix:** Changed `ght` to `/ght` to only match the root-level binary
- **Files modified:** .gitignore
- **Verification:** `git add cmd/ght/main.go` succeeded
- **Committed in:** 06d94a8 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary for staging modified file. No scope creep.

## Issues Encountered
None beyond the .gitignore fix documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 1 (Foundation) is complete: dual REST+GraphQL client with rate limiter wired into transport
- Token validation runs on startup, giving users immediate feedback on missing scopes
- Phase 2 (Actions) can use GetRESTClient() for Actions API calls, all rate-limited automatically
- Phase 3 (Projects V2) can use GetGraphQLClient() for Projects V2 queries, all rate-limited
- GetRateLimiter() is ready for Phase 2/3 UI rate limit display

## Self-Check: PASSED

- All 5 key files exist on disk
- Commit dd8a59d (Task 1) found in git log
- Commit 06d94a8 (Task 2) found in git log

---
*Phase: 01-foundation-dual-client-setup*
*Completed: 2026-02-16*
