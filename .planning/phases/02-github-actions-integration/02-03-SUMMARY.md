---
phase: 02-github-actions-integration
plan: 03
subsystem: ui
tags: [tview, actions-tab, workflow-jobs, job-logs, drill-down, full-screen-preview, context-cancellation]

# Dependency graph
requires:
  - phase: 02-github-actions-integration
    plan: 01
    provides: "WorkflowJob domain type, ConvertWorkflowJob, ListWorkflowJobs, GetWorkflowJobLog, CleanLog"
  - phase: 02-github-actions-integration
    plan: 02
    provides: "Actions tab UI with WorkflowRunsUI, actionsStatusLine, tab switching, tview.Pages pattern"
provides:
  - "Jobs drill-down from workflow runs via Enter key with tview.Pages inner navigation"
  - "Full-screen job log viewer with ANSI/timestamp stripping and search support"
  - "10MB log download cap with explicit truncation detection and user-facing message"
  - "Context-aware log downloads with 30s timeout and cancellation on navigation"
  - "Complete runs -> jobs -> logs -> back navigation within Actions tab"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [inner tview.Pages for sub-navigation, context.WithTimeout for log downloads, explicit truncation bool from IO-limited reads]

key-files:
  created: []
  modified:
    - ui/actions.go
    - ui/ui.go
    - ui/view.go
    - github/actions.go

key-decisions:
  - "Used inner tview.Pages (actionsPages) to switch between runs-view and jobs-view within the Actions grid"
  - "Changed GetWorkflowJobLog to return (string, bool, error) for reliable truncation detection before CleanLog reduces size"
  - "Fixed FullScreenPreview and Message to use activePage instead of hardcoded 'main' for correct page routing from Actions tab"
  - "Added returnPage field to ViewUI so CommonViewUI 'o' key returns to correct page (main or actions)"
  - "Distinguish context.Canceled (user navigated away, silent) from context.DeadlineExceeded (timeout, show message)"

patterns-established:
  - "Inner Pages pattern: use tview.Pages within a tab grid for sub-navigation (runs vs jobs)"
  - "Log download pattern: context.WithTimeout + cancellation tracking via package-level CancelFunc"
  - "Truncation detection: check raw body len against limit before cleaning, return bool alongside content"

# Metrics
duration: 4min
completed: 2026-02-17
---

# Phase 2 Plan 3: Jobs Drill-Down and Log Viewing Summary

**Jobs list drill-down from workflow runs with full-screen log viewer, ANSI stripping, 10MB cap, and complete runs->jobs->logs->back navigation**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-17T10:04:51Z
- **Completed:** 2026-02-17T10:09:05Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Complete drill-down navigation: Enter on run shows jobs, Enter on job shows log, 'o' closes log, Escape returns to runs
- WorkflowJobsUI SelectUI with status/job/duration columns and colored status display
- Full-screen log viewer via existing FullScreenPreview with search (/) and navigation (n/N)
- Log downloads capped at 10MB with explicit truncation detection and user-facing message
- Context-aware downloads with 30s timeout, cancellation on navigation, and differentiated error messages (404, timeout, general)
- Fixed FullScreenPreview and Message page routing to work correctly from both main and Actions pages

## Task Commits

Each task was committed atomically:

1. **Task 1: Add jobs drill-down and log viewing** - `3522f64` (feat)
2. **Task 2: Fix log truncation detection** - `033da7d` (feat)

## Files Created/Modified
- `ui/actions.go` - Added WorkflowJobsUI, jobs getList/capture, inner actionsPages, switchToRunsView, fetchAndDisplayJobLog, isNotFoundError
- `ui/ui.go` - Fixed FullScreenPreview to use activePage, fixed Message to use activePage instead of hardcoded "main"
- `ui/view.go` - Added returnPage field to ViewUI, updated 'o' handler to use returnPage for correct navigation
- `github/actions.go` - Changed GetWorkflowJobLog signature to return truncation bool alongside content

## Decisions Made
- Used inner `tview.Pages` (actionsPages) for runs/jobs sub-navigation rather than grid item replacement, keeping layout clean
- Changed `GetWorkflowJobLog` from `(string, error)` to `(string, bool, error)` because CleanLog reduces content size, making post-clean length comparison unreliable for truncation detection
- Fixed `FullScreenPreview` and `Message` to use `ui.activePage` instead of hardcoded `"main"` -- required for any overlay (modals, full-screen preview) to work correctly when called from the Actions tab
- Added `returnPage` string field to `ViewUI` so the CommonViewUI `'o'` close handler returns to the correct page
- Distinguished `context.Canceled` (user navigated away, silent return) from `context.DeadlineExceeded` (show timeout message to user)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed FullScreenPreview page routing for Actions tab**
- **Found during:** Task 1
- **Issue:** FullScreenPreview hardcodes ShowPage("main") and CommonViewUI 'o' handler hardcodes SwitchToPage("main"), breaking full-screen log viewing from Actions tab
- **Fix:** Added returnPage field to ViewUI, FullScreenPreview sets it from activePage, 'o' handler uses returnPage. Also fixed Message to use activePage.
- **Files modified:** ui/ui.go, ui/view.go
- **Verification:** go build ./... passes
- **Committed in:** 3522f64 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed truncation detection using raw body size instead of cleaned content length**
- **Found during:** Task 2
- **Issue:** Plan's truncation check compared cleaned log length against 10MB, but CleanLog strips ANSI codes and timestamps reducing size significantly, causing truncation to go undetected
- **Fix:** Changed GetWorkflowJobLog to return (string, bool, error) with truncation flag checked against raw body before cleaning
- **Files modified:** github/actions.go, ui/actions.go
- **Verification:** go build ./... && go vet ./... passes
- **Committed in:** 033da7d (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for correct behavior. Page routing fix ensures overlays work from Actions tab. Truncation fix ensures users are informed when logs are capped. No scope creep.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 2 (GitHub Actions Integration) is now complete: all 3 plans executed
- Complete workflow: Issues -> (Ctrl+A) -> Runs -> (Enter) -> Jobs -> (Enter) -> Log -> (o) -> Jobs -> (Esc) -> Runs -> (Ctrl+I) -> Issues
- Ready for Phase 3 (Projects V2 integration) when scheduled

## Self-Check: PASSED

- All 4 key files exist on disk (ui/actions.go, ui/ui.go, ui/view.go, github/actions.go)
- Commit 3522f64 (Task 1) found in git log
- Commit 033da7d (Task 2) found in git log
- go build ./... passes
- go vet ./... passes

---
*Phase: 02-github-actions-integration*
*Completed: 2026-02-17*
