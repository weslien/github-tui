# Roadmap: github-tui (ght)

## Overview

This milestone adds GitHub Actions monitoring and Projects V2 support to the existing terminal UI. The journey proceeds in three phases: establishing dual REST+GraphQL client architecture with unified rate limiting, integrating read-only Actions workflow monitoring with log viewing, and adding Projects V2 project/item browsing with custom field display. Each phase builds on the previous, with Foundation establishing critical patterns that both Actions and Projects V2 depend on.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & Dual-Client Setup** - REST client, unified rate limiter, token validation, async patterns (completed 2026-02-16)
- [ ] **Phase 2: GitHub Actions Integration** - Workflow runs, jobs, logs, filtering, navigation
- [ ] **Phase 3: Projects V2 Integration** - Project listing, items view, custom fields, navigation

## Phase Details

### Phase 1: Foundation & Dual-Client Setup
**Goal**: Establish dual REST+GraphQL client architecture with unified rate limiting and async UI update patterns
**Depends on**: Nothing (first phase)
**Requirements**: FOUND-01, FOUND-02, FOUND-03
**Success Criteria** (what must be TRUE):
  1. App successfully initializes both REST and GraphQL clients using the same PAT token from config
  2. App tracks REST (5000 req/hr) and GraphQL (5000 pts/hr) rate limits independently and displays current usage
  3. App validates PAT token scopes on startup and fails fast with clear message if required scopes missing
  4. UI updates from async API calls happen without deadlocks using established channel-based pattern
**Plans**: 2 plans

Plans:
- [ ] 01-01-PLAN.md -- REST client + rate limiter middleware (Wave 1)
- [ ] 01-02-PLAN.md -- Token scope validation + wiring (Wave 2)

### Phase 2: GitHub Actions Integration
**Goal**: Users can monitor GitHub Actions workflow runs, view job logs, and navigate to GitHub
**Depends on**: Phase 1
**Requirements**: ACT-01, ACT-02, ACT-03, ACT-04, ACT-05, ACT-06, ACT-07
**Success Criteria** (what must be TRUE):
  1. User can view a list of workflow runs showing status, workflow name, branch, event, and timing
  2. User can filter workflow runs by status (success, failure, in_progress) and by workflow name
  3. User can select a workflow run, view its jobs, and view logs for specific jobs with ANSI codes properly stripped
  4. User can open a workflow run in browser from the TUI
  5. Actions tab is accessible via keybinding from any view in the TUI
**Plans**: 3 plans

Plans:
- [ ] 02-01-PLAN.md -- Domain types (WorkflowRun, WorkflowJob) + API layer + log cleaning (Wave 1, TDD)
- [ ] 02-02-PLAN.md -- Actions tab UI with workflow runs list, tab switching, filtering (Wave 2)
- [ ] 02-03-PLAN.md -- Jobs drill-down + log viewing with ANSI stripping (Wave 3)

### Phase 3: Projects V2 Integration
**Goal**: Users can browse GitHub Projects V2 projects and items with custom field display
**Depends on**: Phase 2
**Requirements**: PROJ-01, PROJ-02, PROJ-03, PROJ-04
**Success Criteria** (what must be TRUE):
  1. User can view a list of their Projects V2 (both user-owned and org-scoped projects)
  2. User can select a project and view its items (issues, PRs, draft items) in a table
  3. User can see custom field values (status, text, date, number, single-select) displayed as columns in the items table
  4. User can open a project item in browser from the TUI
**Plans**: TBD

Plans:
(Plans will be added during plan-phase execution)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & Dual-Client Setup | 0/2 | Complete    | 2026-02-16 |
| 2. GitHub Actions Integration | 0/3 | Not started | - |
| 3. Projects V2 Integration | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-16*
*Last updated: 2026-02-16 (Phase 1 planned)*
