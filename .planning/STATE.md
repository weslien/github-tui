# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-16)

**Core value:** Developers can interact with their GitHub repositories without leaving the terminal — fast, keyboard-driven, and distraction-free
**Current focus:** Phase 2: GitHub Actions Integration

## Current Position

Phase: 2 of 3 (GitHub Actions Integration)
Plan: 1 of 3 complete
Status: In Progress
Last activity: 2026-02-17 — Completed 02-01-PLAN.md (Actions domain types + API layer)

Progress: [███░░░░░░░] 33%

## Performance Metrics

**Velocity:**
- Total plans completed: 3
- Average duration: 4min
- Total execution time: 0.18 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 2 | 8min | 4min |
| 02-actions | 1 | 3min | 3min |

**Recent Trend:**
- Last 5 plans: 01-01 (5min), 01-02 (3min), 02-01 (3min)
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
- 02-01: Shared statusDisplay() helper in domain package for WorkflowRun and WorkflowJob color mapping
- 02-01: CleanLog applied automatically inside GetWorkflowJobLog (callers get clean text)
- 02-01: ListWorkflows uses full pagination; other list functions delegate pagination to caller
- 02-01: WorkflowJob duration uses StartedAt/CompletedAt (job-level, not run-level timestamps)

### Pending Todos

None yet.

### Blockers/Concerns

**Phase 2 (Actions):**
- ~~Actions logs can be 100MB+, need chunked download or streaming to avoid OOM~~ Resolved: GetWorkflowJobLog uses io.LimitReader with 10MB cap
- GitHub API has 10-page pagination limit (1000 items max), need date-based workaround for older runs

**Phase 3 (Projects V2):**
- shurcooL/githubv4 may have schema lag with Projects V2 custom field types (single-select, iteration)
- GraphQL node limit can explode with nested queries, must keep `first: 10-20` for nested queries
- Projects V2 field type unions may cause unmarshaling issues, test early with real projects

## Session Continuity

Last session: 2026-02-17 (plan execution)
Stopped at: Completed 02-01-PLAN.md — Actions domain types and API layer done, ready for 02-02
Resume file: None

---
*State initialized: 2026-02-16*
*Last updated: 2026-02-17*
