---
phase: 02-github-actions-integration
plan: 01
subsystem: api
tags: [go-github, actions, workflow-runs, workflow-jobs, log-cleaning, ansi-stripping, domain-types, tdd]

# Dependency graph
requires:
  - phase: 01-foundation-dual-client-setup
    provides: "Dual REST+GraphQL client with GetRESTClient() accessor and gogithub import alias"
provides:
  - "WorkflowRun domain type implementing Item interface with status-colored fields"
  - "WorkflowJob domain type implementing Item interface with status-colored fields"
  - "ConvertWorkflowRun and ConvertWorkflowJob mapping go-github types to domain"
  - "CleanLog utility for stripping ANSI codes and GitHub timestamp prefixes"
  - "ListWorkflowRuns, ListWorkflows, ListWorkflowJobs, GetWorkflowJobLog API wrappers"
  - "ListWorkflowRunsByWorkflowID for workflow-specific filtering"
affects: [02-02-PLAN, 02-03-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns: [status-to-color mapping via statusDisplay helper, io.LimitReader for size-capped downloads, compiled regex for log cleaning]

key-files:
  created:
    - domain/workflow_run.go
    - domain/workflow_job.go
    - github/actions.go
    - github/actions_test.go
  modified: []

key-decisions:
  - "Shared statusDisplay() helper in domain package used by both WorkflowRun and WorkflowJob"
  - "CleanLog applied automatically inside GetWorkflowJobLog so callers always get clean text"
  - "ListWorkflows uses full pagination (all pages), other list functions use caller-provided options"
  - "WorkflowJob duration computed from StartedAt to CompletedAt (not RunStartedAt/UpdatedAt like runs)"

patterns-established:
  - "Status/conclusion color mapping: completed+success=green, completed+failure=red, completed+other=gray, in_progress=yellow, default=gray"
  - "Domain type Fields() pattern: return slice of Field with status as first field for consistent list display"
  - "API wrapper pattern: check GetRESTClient() != nil, wrap errors with context using fmt.Errorf %w"

# Metrics
duration: 3min
completed: 2026-02-17
---

# Phase 2 Plan 1: Actions Domain Types and API Layer Summary

**WorkflowRun and WorkflowJob domain types with status-colored fields, go-github conversion functions, ANSI/timestamp log cleaning, and paginated API wrappers**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-17T09:53:23Z
- **Completed:** 2026-02-17T09:56:06Z
- **Tasks:** 2 (TDD: RED + GREEN)
- **Files modified:** 4

## Accomplishments
- WorkflowRun and WorkflowJob domain types implementing Item interface with status-dependent color mapping (green/red/yellow/gray)
- ConvertWorkflowRun and ConvertWorkflowJob converting go-github types to domain types with duration formatting and time display
- CleanLog utility stripping ANSI escape codes and GitHub timestamp prefixes via compiled regexes
- 21 table-driven tests passing with -race flag covering conversion, color mapping, duration formatting, and log cleaning
- Full API wrapper layer: ListWorkflowRuns, ListWorkflows (paginated), ListWorkflowJobs, GetWorkflowJobLog (10MB cap), ListWorkflowRunsByWorkflowID

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing tests (RED)** - `c74c3e9` (test)
2. **Task 2: Implement domain types and API layer (GREEN)** - `a695545` (feat)

## Files Created/Modified
- `domain/workflow_run.go` - WorkflowRun struct with Key(), Fields(), and statusDisplay() helper
- `domain/workflow_job.go` - WorkflowJob struct with Key(), Fields() using shared statusDisplay()
- `github/actions.go` - CleanLog, ConvertWorkflowRun, ConvertWorkflowJob, formatDuration, formatTime, and 6 API wrapper functions
- `github/actions_test.go` - 21 table-driven tests for conversion, color mapping, duration formatting, and log cleaning

## Decisions Made
- Placed statusDisplay() in domain/workflow_run.go as an unexported helper shared by both WorkflowRun and WorkflowJob (same package, avoids duplication)
- CleanLog is applied automatically inside GetWorkflowJobLog so all callers receive clean text without needing to call CleanLog separately
- ListWorkflows implements full pagination (fetches all pages) since workflow count is typically small; other list functions delegate pagination to the caller via options
- WorkflowJob duration uses StartedAt/CompletedAt (job-level timestamps) rather than RunStartedAt/UpdatedAt (run-level timestamps)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Domain types and API wrappers are ready for Plan 02-02 (Actions Tab UI) to consume
- WorkflowRun and WorkflowJob implement domain.Item, ready for use in tview list components
- All API functions use GetRESTClient() from Phase 1 foundation
- CleanLog is ready for log viewer integration in Plan 02-03

## Self-Check: PASSED

- All 5 key files exist on disk
- Commit c74c3e9 (Task 1 - RED) found in git log
- Commit a695545 (Task 2 - GREEN) found in git log

---
*Phase: 02-github-actions-integration*
*Completed: 2026-02-17*
