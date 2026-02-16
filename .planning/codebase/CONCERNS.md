# Codebase Concerns

**Analysis Date:** 2026-02-16

## Tech Debt

**Missing Test Coverage:**
- Issue: Zero test coverage across entire codebase - no `*_test.go` or `*_spec.go` files exist
- Files: All files in `ui/`, `github/`, `config/`, `domain/`
- Impact: Changes cannot be verified automatically, regressions go undetected, refactoring is risky
- Fix approach: Implement table-driven test suite using Go's standard `testing` package, start with critical paths in `github/client.go`, `config/config.go`, and `ui/issues.go`

**Global State in UI Module:**
- Issue: Widespread use of package-level variables as singletons (`IssueUI`, `CommentUI`, `UI`, `IssueFilterUI`, etc.)
- Files: `ui/issues.go`, `ui/comments.go`, `ui/ui.go`, `ui/select.go`, `ui/labels.go`, `ui/milestones.go`, `ui/projects.go`, `ui/assignees.go`, `ui/search.go`
- Impact: Hard to test components in isolation, difficult to create multiple UI instances, race conditions possible in concurrent operations
- Fix approach: Refactor UI module to use dependency injection pattern, pass UI context through function parameters rather than global access

**Uncontrolled Goroutines in Form Setup:**
- Issue: Multiple unsupervised goroutines spawned in `createIssueForm()` for async data loading (lines 288-431 in `ui/issues.go`)
- Files: `ui/issues.go` lines 288-431
- Impact: No error propagation from background goroutines, no timeout control, potential resource leaks if form is closed before goroutines complete
- Fix approach: Use `context.Context` with timeout, use `sync.WaitGroup` or `errgroup.Group` to coordinate goroutines, handle errors from background tasks

**Inconsistent Error Handling:**
- Issue: Mix of error handling approaches - some use `log.Println()`, others use `UI.Message()`, some return errors
- Files: `ui/issues.go` (lines 58, 134, 179), `ui/comments.go` (lines 147, 291, 320, 349, 379, 411), `github/client.go`
- Impact: Errors silently logged to stdout/stderr instead of shown to user, inconsistent user feedback
- Fix approach: Standardize on `error` returns from functions, let caller decide how to display (log vs UI message)

**Index Variable Shadowing in Loops:**
- Issue: Loop variable shadowing in lines 57-59, 62-65, 68-72, 78-81 of `github/query_issue.go`
- Files: `github/query_issue.go` lines 57-59, 62-65, 68-72, 78-81
- Impact: Incorrect loop variable references, subtle bugs when accessing the wrong variable
- Fix approach: Rename loop variables to avoid shadowing (e.g., `for idx, label := range i.Labels.Nodes { labels[idx] = ... }`)

**Mutable Global Configuration:**
- Issue: `config.GitHub` and `config.App` are mutable package-level variables modified directly in `config.Init()`
- Files: `config/config.go` lines 25-26, 67
- Impact: Configuration can be unexpectedly modified, no thread safety for concurrent access, hard to test with different configs
- Fix approach: Make config immutable after initialization, use getter functions instead of direct variable access

## Known Bugs

**Nil Pointer in getSelectedComments() on Empty Selection:**
- Symptoms: Panic when calling `getSelectedComments()` with no selection and empty CommentUI
- Files: `ui/comments.go` line 211
- Trigger: Try to open comment with no comments loaded, or call comment operations on empty state
- Workaround: Check for nil before using selected comment
- Code: Line 211 appends `data.(*domain.Comment)` without nil check when `CommentUI.selected` is empty

**Array Index Out of Bounds in toggleSelected():**
- Symptoms: Potential panic on accessing `ui.items[0]` when `ui.items` is empty
- Files: `ui/select.go` line 225
- Trigger: Call `toggleSelected()` when UI has no items
- Workaround: Prevent toggling when items list is empty
- Code: Line 225 accesses `ui.items[0]` without checking if items exist

**parseRemote() Bounds Error:**
- Symptoms: Panic on SSH remotes with path separators
- Files: `cmd/ght/main.go` lines 69, 73, 79
- Trigger: Use SSH remote URL with unexpected path structure
- Workaround: Normalize remote parsing logic
- Code: Lines 69, 73, 79 access array indices without checking bounds before slicing

**Temporary File Cleanup Race Condition:**
- Symptoms: Race condition between `os.Remove()` and file content read
- Files: `utils/utils.go` line 16
- Trigger: Editor process still writing to file when cleanup attempts removal
- Workaround: Use defer cleanup after reading file content
- Code: Line 16 removes temp file before ensuring editor has fully written content (line 37 reads it)

## Security Considerations

**GitHub Token in CLI Arguments:**
- Risk: User can pass GitHub token directly via environment variables, but no validation that token is actually configured
- Files: `config/config.go`, `cmd/ght/main.go`
- Current mitigation: Token read from config file (not CLI args)
- Recommendations: Add token validation (non-empty), consider supporting OAuth device flow instead of personal access tokens

**Unvalidated Editor Execution:**
- Risk: `$EDITOR` environment variable executed without validation
- Files: `utils/utils.go` line 25-28, `cmd/ght/main.go` (config path depends on user env)
- Current mitigation: Falls back to vim if EDITOR not set
- Recommendations: Whitelist allowed editors or validate editor path, use `os/exec.LookPath` to verify editor exists

**No Input Validation on GraphQL Variables:**
- Risk: User input passed directly to GraphQL queries without escaping
- Files: `ui/issues.go` lines 26-27, 51, 273-278
- Current mitigation: GraphQL library handles some escaping
- Recommendations: Validate repo owner/name format before querying, validate query input for injection attempts

**Debug Log File Creation Without Restriction:**
- Risk: Debug logs written to `~/.config/ght/debug.log` (or Windows/Mac equivalent) without log rotation
- Files: `config/config.go` lines 35-36, 41
- Current mitigation: None
- Recommendations: Implement log rotation, limit log file size, consider sensitive data in logs

## Performance Bottlenecks

**N+1 Query Pattern in Issue Details:**
- Problem: Every issue selection triggers separate GraphQL query to fetch full issue with all related data
- Files: `ui/select.go` lines 156-162 calls `updateUIRelatedIssue()` which doesn't batch queries
- Cause: Selection change handler makes individual queries instead of loading in bulk
- Improvement path: Batch load issue details when fetching initial list, cache related data

**Inefficient List Filtering in Memory:**
- Problem: Full list filtering on every keystroke in search, no debouncing
- Files: `ui/select.go` lines 130-141
- Cause: Search updates filter on every keystroke, re-renders full table
- Improvement path: Add debouncing to search input (100-200ms), implement virtual scrolling for large lists

**Uncontrolled Goroutine Spawning in Form:**
- Problem: Form creation in `createIssueForm()` spawns 5+ independent goroutines without coordination
- Files: `ui/issues.go` lines 288-431
- Cause: Each autocomplete data source loads independently without progress tracking
- Improvement path: Use single goroutine with concurrent fetches, show loading indicators for async operations

**Synchronous Wait in Close/Open Operations:**
- Problem: `closeIssues()` and `openIssues()` spawn goroutines in loop with `sync.WaitGroup.Wait()` blocking main thread
- Files: `ui/issues.go` lines 158-174, 140-156
- Cause: Operations block UI while waiting for all API calls to complete serially
- Improvement path: Implement async operation queue with progress indicator, allow cancellation

## Fragile Areas

**UI Form State Management:**
- Files: `ui/issues.go` lines 187-520 (createIssueForm function)
- Why fragile: Complex form with manual state management across multiple input fields, dynamic form item addition/removal based on async results
- Safe modification: Use form state struct instead of scattered variables, implement form validation before submission
- Test coverage: No tests for form state transitions, validation logic, or async data loading

**GraphQL Response Parsing:**
- Files: `github/query_issue.go`, `github/query_*.go` (all query files)
- Why fragile: Nested struct definitions with `graphql` tags, no validation of response structure
- Safe modification: Add response validation functions, use typed wrappers for GraphQL queries
- Test coverage: No tests for parsing different response shapes, missing fields, or malformed responses

**Type Assertions Without Guards:**
- Files: `ui/issues.go` lines 64, 116, 120, 527; `ui/comments.go` lines 88, 211, 214; `ui/select.go` lines 221, 249
- Why fragile: Direct type assertions without checking if cast succeeds
- Safe modification: Add nil checks and type assertions with ok flag (`data.(*domain.Issue)` should be `d, ok := data.(*domain.Issue); if !ok ...`)
- Test coverage: No tests exercising type assertion failures

**Selection Index Management:**
- Files: `ui/select.go` lines 157, 177-201, 218-233, 244-256
- Why fragile: Complex row/column index calculations with header offset, no bounds checking before array access
- Safe modification: Create helper methods for index calculation and validation, add bounds checks
- Test coverage: No tests for edge cases (empty list, single item, header offset)

## Scaling Limits

**Fixed GraphQL Page Size:**
- Current capacity: Hard-coded to 30 items per page for issues, 10 items for comments, 100 for templates
- Limit: Large repositories may require more efficient pagination or search filters
- Scaling path: Make page size configurable, implement smarter pagination (load more on scroll), consider GitHub's search API limits

**Comment Loading Limit:**
- Current capacity: Hard-coded to first 100 comments per issue
- Limit: Issues with >100 comments will not show all comments
- Scaling path: Implement pagination for comments, add "load more comments" functionality

**Single-threaded UI Event Loop:**
- Current capacity: All operations queue through single `ui.updater` channel with buffer size 100
- Limit: 100 concurrent UI updates will block, rapid operations may lose updates
- Scaling path: Increase channel buffer or implement priority queue for UI updates

## Dependencies at Risk

**Outdated GitHub GraphQL Client:**
- Risk: `github.com/shurcooL/githubv4` dependency appears unmaintained (last releases 2020)
- Impact: No support for new GitHub API features, potential security issues in dependencies
- Migration plan: Consider migrating to official GitHub Go SDK or `hasura/go-graphql-client`

**Outdated TUI Framework:**
- Risk: `github.com/rivo/tview` v0.0.0-20210312174852-ae9464cc3598 is very old (March 2021)
- Impact: Potential terminal rendering bugs, no new features, possible security issues
- Migration plan: Update to latest tview version, test terminal rendering edge cases

**Unmaintained OAuth2 Package Version:**
- Risk: `golang.org/x/oauth2` locked to old version
- Impact: Security fixes not applied, potential vulnerability to token interception
- Migration plan: Update to latest version, verify token storage security

## Missing Critical Features

**No Offline Mode:**
- Problem: Application requires live GitHub API connection for all operations
- Blocks: Cannot browse issues when network is unavailable, no cache of previously viewed issues
- Gap: Consider implementing basic local cache of issue/comment data

**No Configuration UI:**
- Problem: Configuration must be edited manually in YAML file
- Blocks: Users cannot easily change settings without restarting app
- Gap: Implement in-app configuration editor for common settings (editor, filters)

**No Markdown Preview:**
- Problem: Issue/comment bodies are plain text in preview, no markdown rendering
- Blocks: Cannot see formatted text, links, code blocks in preview pane
- Gap: Implement markdown renderer for preview (consider `charmbracelet/glamour` or similar)

**No Issue Search History:**
- Problem: Search queries are not saved between sessions
- Blocks: Users must re-enter complex search queries
- Gap: Store recent searches in config file, provide search history dropdown

## Test Coverage Gaps

**Core GitHub Client Functions:**
- What's not tested: All functions in `github/client.go` (CreateIssue, GetRepos, GetIssues, etc.)
- Files: `github/client.go`
- Risk: API call failures, response parsing issues, authentication errors undetected
- Priority: High - these are critical paths

**Configuration Loading:**
- What's not tested: Config initialization, file parsing, validation logic, missing file handling
- Files: `config/config.go`
- Risk: Config file corruption, permission issues, or missing credentials silently fail
- Priority: High - app cannot function without correct config

**UI Form Operations:**
- What's not tested: Form state transitions, async data loading, validation, form submission
- Files: `ui/issues.go` (createIssueForm, createComment, editIssue functions)
- Risk: Form bugs affect core user workflows (create/edit issues)
- Priority: High - user-facing features

**Selection and Filtering Logic:**
- What's not tested: List filtering, selection tracking, index calculations, edge cases
- Files: `ui/select.go` (UpdateView, toggleSelected, GetSelect, FetchList)
- Risk: Silent data corruption, lost selections, rendering bugs
- Priority: Medium - affects data consistency

**Remote URL Parsing:**
- What's not tested: Various Git remote URL formats, edge cases
- Files: `cmd/ght/main.go` (parseRemote function)
- Risk: Panic on unexpected remote formats
- Priority: Medium - affects app initialization

---

*Concerns audit: 2026-02-16*
