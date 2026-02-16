# Phase 1: Foundation & Dual-Client Setup - Context

**Gathered:** 2026-02-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish dual REST+GraphQL client architecture with unified rate limiting and async UI update patterns. This is shared infrastructure that both Actions (Phase 2) and Projects V2 (Phase 3) depend on. No user-facing features — only foundational plumbing.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
- Rate limit visibility approach (status bar, log, warning modal)
- Missing token scope behavior (hard fail vs degraded mode)
- REST client error presentation in TUI
- Rate limiter implementation pattern (middleware, wrapper, or standalone)
- Async update channel buffering strategy
- REST client library integration approach (alongside existing GraphQL client)

User deferred all implementation decisions to Claude — this is pure infrastructure with no UX preferences.

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Follow existing codebase patterns where possible.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-foundation-dual-client-setup*
*Context gathered: 2026-02-16*
