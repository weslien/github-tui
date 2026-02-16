# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-16)

**Core value:** Developers can interact with their GitHub repositories without leaving the terminal — fast, keyboard-driven, and distraction-free
**Current focus:** Phase 1: Foundation & Dual-Client Setup

## Current Position

Phase: 1 of 3 (Foundation & Dual-Client Setup)
Plan: 2 of 2 complete
Status: Phase Complete
Last activity: 2026-02-16 — Completed 01-02-PLAN.md (token validation + rate limiter wiring)

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 2
- Average duration: 4min
- Total execution time: 0.13 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 2 | 8min | 4min |

**Recent Trend:**
- Last 5 plans: 01-01 (5min), 01-02 (3min)
- Trend: Improving

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Phase 1 (Foundation): Dual REST+GraphQL client pattern required because Actions only has REST API and Projects V2 only has GraphQL API
- Phase 1 (Foundation): Unified rate limiter tracks both REST (5000 req/hr) and GraphQL (5000 pts/hr) pools independently
- Phase ordering: Foundation → Actions → Projects V2 (increasing complexity, Actions simpler than Projects V2)
- 01-01: Used channel-based semaphore for concurrent request limiting (simpler than semaphore libraries)
- 01-01: Deferred GraphQL response body parsing to query-level code in later phases
- 01-01: Used gogithub import alias to avoid package name collision
- 01-02: ValidateTokenScopes uses direct HTTP (not rate-limited client) for header access
- 01-02: Fine-grained PATs degrade gracefully (warn, don't block) since X-OAuth-Scopes is empty
- 01-02: admin:org scope implies project access (same as read:org)

### Pending Todos

None yet.

### Blockers/Concerns

**Phase 2 (Actions):**
- Actions logs can be 100MB+, need chunked download or streaming to avoid OOM
- GitHub API has 10-page pagination limit (1000 items max), need date-based workaround for older runs

**Phase 3 (Projects V2):**
- shurcooL/githubv4 may have schema lag with Projects V2 custom field types (single-select, iteration)
- GraphQL node limit can explode with nested queries, must keep `first: 10-20` for nested queries
- Projects V2 field type unions may cause unmarshaling issues, test early with real projects

## Session Continuity

Last session: 2026-02-16 (plan execution)
Stopped at: Completed 01-02-PLAN.md — Phase 1 complete, ready for Phase 2
Resume file: None

---
*State initialized: 2026-02-16*
*Last updated: 2026-02-16*
