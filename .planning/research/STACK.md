# Stack Research

**Domain:** GitHub TUI - Actions & Projects V2 Integration
**Researched:** 2026-02-16
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| shurcooL/githubv4 | v0.0.0-20240727 | GraphQL v4 client for Projects V2 | Already in use; handles all ProjectsV2 queries via GraphQL. Last updated July 2024 but stable and functional for GitHub's GraphQL API. Only option for Projects V2 (no REST API exists). |
| google/go-github/v83 | v83.0.0 | REST API client for GitHub Actions | Required for Actions workflows, runs, and logs. GitHub Actions has NO GraphQL API - REST only. Latest version (Feb 2025) with native pagination iterators. |
| rivo/tview | latest (Aug 2025) | Terminal UI framework | Already in use. TextView widget perfect for displaying logs with scrolling. No version needed - use latest commit. |
| gdamore/tcell/v2 | v2.9.0 | Low-level terminal manipulation | Already in use as tview's foundation. Version 2.9.0 (Aug 2025) is latest stable v2. Note: v3 exists but has breaking changes. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| acarl005/stripansi | latest | Strip ANSI escape codes from logs | Required for Actions logs - GitHub returns logs with ANSI color codes. Use Strip() function before displaying in tview or for search/filter operations. |
| fatih/color | v1.7.0+ | ANSI color rendering in TUI | Optional: If rendering colored logs in tview. Set `color.NoColor = false` for non-TTY handling in GitHub Actions context. Already in go.mod as indirect dependency. |
| golang.org/x/oauth2 | v0.0.0-20200902+ | OAuth token authentication | Already in use. Required for both githubv4 and go-github authentication. Same token works for both GraphQL and REST. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| GitHub Personal Access Token | API authentication | Requires scopes: `repo` (Actions), `read:project` or `project` (Projects V2), `actions:read` (workflow logs) |
| GitHub GraphQL Explorer | Test Projects V2 queries | https://docs.github.com/en/graphql/overview/explorer - validate ProjectsV2 schema queries before coding |
| GitHub REST API Docs | Test Actions endpoints | https://docs.github.com/en/rest/actions - verify workflow run structures |

## Installation

```bash
# New REST API client for Actions
go get github.com/google/go-github/v83@v83.0.0

# ANSI escape code stripper for logs
go get github.com/acarl005/stripansi@latest

# Already in go.mod (keep existing versions):
# - github.com/shurcooL/githubv4@v0.0.0-20200928013246-d292edc3691b
# - github.com/rivo/tview@v0.0.0-20210312174852-ae9464cc3598
# - github.com/gdamore/tcell/v2@v2.2.0
# - golang.org/x/oauth2@v0.0.0-20200902213428-5d25da1a8d43
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| google/go-github/v83 | Manual http.Client + REST calls | Never - go-github handles pagination, rate limiting, and type safety. Rolling your own is error-prone. |
| shurcooL/githubv4 | google/go-github GraphQL support | Never for Projects V2 - go-github has minimal GraphQL support and focuses on REST. githubv4 is purpose-built for GraphQL. |
| acarl005/stripansi | Manual regex ANSI stripping | Use stripansi - reliable, tested, handles edge cases. Manual approach: `regexp.MustCompile("\x1b\\[[0-9;]*m")` misses advanced codes. |
| rivo/tview TextView | Custom scroll implementation | Never - tview's TextView handles scrolling, word wrap, ANSI rendering, and efficient updates. Mature and battle-tested. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| GraphQL for GitHub Actions | GitHub Actions has NO GraphQL API as of 2026. Multiple community discussions confirm REST-only. Attempting GraphQL queries for workflows/runs will fail. | google/go-github REST client |
| REST API for Projects V2 | Projects V2 has NO REST API - GraphQL only. Confirmed in official docs. Old "Projects" (v1/Classic) had REST but are deprecated. | shurcooL/githubv4 GraphQL |
| go-github versions < v60 | Pre-v60 has different API structure. v83 introduced native iterators (Feb 2025) replacing manual pagination. Older versions lack Actions endpoints added in v50+. | go-github/v83 (latest) |
| tcell v3 | Breaking changes from v2. tview currently depends on v2. Upgrading requires tview update first. | tcell/v2 (v2.9.0) |
| shurcooL/githubv4 for Actions | Can technically make custom GraphQL queries but Actions objects aren't in GitHub's GraphQL schema. Would get empty results. | go-github REST client |

## Stack Patterns by Variant

**For Projects V2 Queries:**
- Use shurcooL/githubv4 with `projectsV2` field
- Structure: `organization(login).projectV2(number)` or `user(login).projectV2(number)`
- Items have `content` field with unions: `Issue`, `PullRequest`, `DraftIssue`
- Field values are type-specific: `ProjectV2ItemFieldTextValue`, `ProjectV2ItemFieldDateValue`, etc.

**For GitHub Actions Workflows:**
- Use google/go-github Actions service: `client.Actions.ListWorkflows(ctx, owner, repo, opts)`
- Returns `*github.Workflows` with pagination via `opts.Page`
- V83 supports native iterators: `client.Actions.ListWorkflowsAll(ctx, owner, repo)`

**For Actions Workflow Runs:**
- Use `client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, opts)`
- Supports filtering: `opts.Status = "failure"`, `opts.Branch = "main"`
- Re-run: `client.Actions.RerunWorkflow(ctx, owner, repo, runID)`
- Re-run failed: `client.Actions.RerunFailedJobs(ctx, owner, repo, runID)`

**For Actions Logs:**
- Use `client.Actions.GetWorkflowRunLogs(ctx, owner, repo, runID, true)` for redirect URL
- Returns HTTP 302 redirect with `Location` header (expires in 1 minute)
- Download ZIP archive, extract, parse text files
- CRITICAL: Strip ANSI codes with stripansi.Strip() before displaying in tview

**For Log Display in TUI:**
- Use tview.TextView for scrollable log viewer
- Set `textView.SetScrollable(true)`, `SetDynamicColors(false)` (since ANSI stripped)
- For large logs: stream line-by-line, append with `fmt.Fprintf(textView, "%s\n", line)`
- tview handles efficient rendering via tcell's double-buffering

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| google/go-github/v83 | Go 1.22+ | Requires Go version per N-1 policy (as of Go 1.23 release). Check go.mod for actual minimum. |
| shurcooL/githubv4 | Go 1.8+ | Minimal requirements, compatible with project's Go 1.22. |
| rivo/tview (latest) | tcell/v2.9.0 | tview built on tcell v2. Both must use v2 (not v3). |
| gdamore/tcell/v2 | Go 1.12+ | Stable v2 branch. v3 is available but incompatible with tview's current version. |
| acarl005/stripansi | Go 1.11+ | No dependencies, works with any modern Go version. |

## Critical Integration Notes

### Dual Client Pattern
You'll need **two separate clients** in the codebase:

```go
// GraphQL client (existing - for Projects V2)
import "github.com/shurcooL/githubv4"
graphqlClient := githubv4.NewClient(oauth2Client)

// REST client (new - for Actions)
import "github.com/google/go-github/v83/github"
restClient := github.NewClient(oauth2Client)
```

Both use the **same** `oauth2.NewClient(ctx, tokenSource)` for authentication.

### Authentication Scopes
Personal Access Token needs:
- `repo` - Access private repositories (includes Actions read)
- `read:project` - Read Projects V2 (existing requirement)
- `workflow` - Trigger workflow re-runs (write operation)

Classic tokens work. Fine-grained tokens need:
- Actions: read + write (for re-run)
- Projects: read
- Contents: read (for repo access)

### Projects V2 GraphQL Schema Notes

**Key difference from existing "Projects" code:**
- Old Projects (v1/Classic): `repository.projects(...)` - deprecated
- New Projects V2: `organization.projectsV2(...)` or `user.projectsV2(...)`
- Projects V2 are **organization-scoped** or **user-scoped**, NOT repository-scoped
- Project items link back to repos via `content { ... on Issue { repository { name } } }`

**Existing code impact:**
Check `/Users/gustav/src/github-tui/github/query_project.go` - likely uses old Projects API. New Projects V2 implementation needs separate query structures.

### Actions Logs: ZIP Archive Handling

**Log download workflow:**
1. Call `GetWorkflowRunLogs()` - returns HTTP 302 redirect
2. Follow redirect URL (expires in 1 minute!) - downloads ZIP
3. Extract ZIP in memory (use `archive/zip` package)
4. Parse log files (plain text with ANSI codes)
5. Strip ANSI codes for display/search
6. Option: Keep color codes if rendering with tview's ANSI support

**Memory considerations:**
Large workflow logs (>100MB) can cause memory issues. Stream extraction or implement log pagination (download specific jobs, not entire run).

### TUI Display Patterns

**For Projects V2 navigation:**
- List view: Project titles (Table widget)
- Detail view: Project items (Table with columns from custom fields)
- Navigate to linked issue/PR: Extract URL from `content.url`, open or fetch details

**For Actions workflows:**
- List view: Workflow names + status badges
- Run history: Table with run ID, status, branch, timestamp
- Log viewer: Full-screen TextView with scrolling

## Sources

- [GitHub Actions REST API - Workflow Runs](https://docs.github.com/en/rest/actions/workflow-runs) — Actions endpoints (HIGH confidence)
- [GitHub Projects V2 GraphQL API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) — ProjectsV2 schema (HIGH confidence)
- [google/go-github v83.0.0 Release](https://github.com/google/go-github/releases) — Latest version (Feb 2025) (HIGH confidence)
- [shurcooL/githubv4 Repository](https://github.com/shurcooL/githubv4) — GraphQL client status (MEDIUM confidence - stable but low recent activity)
- [rivo/tview Package](https://pkg.go.dev/github.com/rivo/tview) — Latest version Aug 2025 (HIGH confidence)
- [gdamore/tcell Releases](https://github.com/gdamore/tcell/releases) — tcell v2.9.0 (HIGH confidence)
- [acarl005/stripansi Repository](https://github.com/acarl005/stripansi) — ANSI stripping utility (MEDIUM confidence - simple, focused library)
- [GitHub Actions GraphQL API Discussion](https://github.com/orgs/community/discussions/24493) — Confirms no GraphQL for Actions (HIGH confidence - official community discussion)

---
*Stack research for: GitHub TUI - Actions & Projects V2 Integration*
*Researched: 2026-02-16*
