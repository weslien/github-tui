# Project Research Summary

**Project:** GitHub TUI - Actions & Projects V2 Integration
**Domain:** Terminal UI for GitHub API (Actions REST API + Projects V2 GraphQL)
**Researched:** 2026-02-16
**Confidence:** HIGH

## Executive Summary

GitHub TUI needs to integrate two major features: GitHub Actions monitoring and Projects V2 support. The research reveals a clear architectural pattern: a dual-client approach using REST API for Actions (no GraphQL support exists) and GraphQL API for Projects V2 (no REST API exists). This integration follows the existing TUI's clean MVC pattern with tview components, async channel-based updates, and domain entities implementing a polymorphic Item interface.

The recommended approach is to build Actions first (simpler, read-only, high utility) followed by Projects V2 (complex custom fields, mutations, org-scoped). Both features require separate API clients sharing the same OAuth token, unified rate limit tracking (REST uses 5000 req/hour, GraphQL uses 5000 points/hour with different calculations), and careful UI update orchestration to avoid deadlocks. The existing codebase provides strong patterns to follow: SelectUI for tables, ViewUI for previews, channel-based async updates, and Item interface for polymorphic rendering.

Key risks center on rate limit management across dual APIs, tview QueueUpdateDraw deadlocks with concurrent goroutines, and Projects V2 schema lag with shurcooL/githubv4 library. Mitigation strategies include unified rate limit tracking from Phase 1, establishing async update patterns early, and potentially using raw GraphQL queries for Projects V2 to avoid schema dependency issues. Actions' 10-page pagination limit and Projects V2's node ID resolution issues require specific workarounds during implementation.

## Key Findings

### Recommended Stack

The existing GitHub TUI has a solid foundation with tview, githubv4, and tcell. The new features require two key additions: google/go-github/v83 REST client for Actions API (GitHub Actions has NO GraphQL support) and acarl005/stripansi for cleaning ANSI codes from Actions logs. The architecture follows a dual-client pattern where both REST and GraphQL clients use the same OAuth token but maintain separate rate limit pools.

**Core technologies:**
- **google/go-github/v83**: REST API client for Actions workflows, runs, and logs — required because Actions has no GraphQL API, latest version with native pagination iterators
- **shurcooL/githubv4**: GraphQL v4 client for Projects V2 queries — already in use, only option for Projects V2 (no REST API exists), may have schema lag issues
- **acarl005/stripansi**: ANSI escape code stripper for Actions logs — GitHub returns logs with color codes, must strip before displaying in tview
- **rivo/tview**: Terminal UI framework — already in use, TextView widget perfect for log display with scrolling, SelectUI handles tables
- **gdamore/tcell/v2**: Low-level terminal manipulation — foundation for tview, stick with v2 (v3 has breaking changes)

**Critical integration notes:**
- Both APIs share 100 concurrent request limit (enforce with semaphore: max 90)
- REST rate limit: 5000 requests/hour (check `X-RateLimit-*` headers)
- GraphQL rate limit: 5000 points/hour (parse `data.rateLimit.cost` from response body)
- Actions logs are ZIP archives via 302 redirect (download, extract, strip ANSI)
- Projects V2 field types are dynamic unions (text, number, date, single-select, iteration)

### Expected Features

Research confirms that both Actions and Projects V2 have clear table stakes and differentiators. The competitive landscape (gh CLI, lazyactions TUI, gh-projects extensions) shows that read-only monitoring with excellent keyboard-driven navigation is sufficient for MVP, with write operations deferred to Phase 2.

**Must have (table stakes):**
- **Actions**: List workflow runs with status/branch/conclusion, view job logs with search, filter by status and workflow name, view run metadata
- **Projects V2**: List org/user projects, view project items with custom fields displayed, filter by status column, navigate to linked issues/PRs

**Should have (competitive):**
- **Actions**: Re-run failed workflows (write permission), cancel running workflows, live log streaming (Phase 2), workflow dispatch with inputs
- **Projects V2**: Custom field sorting, project-item navigation to existing issue UI, read-only optimization (faster than web UI)

**Defer (v2+):**
- **Actions**: Multi-run comparison (HIGH complexity), bulk operations (checkboxes + batch API), workflow analytics (web UI has this)
- **Projects V2**: Full write access with drag-drop (web UI better for editing), view switching to Board/Roadmap layouts, cross-project search

**Anti-features identified:**
- Local workflow execution (different tool: `act`)
- Custom workflow YAML builder (IDE extensions better)
- Real-time notifications (GitHub already sends these)
- Full Projects V2 mutations (state sync complexity, keep read-only with "open in browser")

### Architecture Approach

The existing GitHub TUI follows clean 3-layer MVC with ui/ (tview components), github/ (API client), and domain/ (business entities). New features extend this pattern with dual REST+GraphQL clients, nested SelectUI for hierarchical data (WorkflowRun → Job → Step), and channel-based async updates to avoid deadlocks. The Item interface enables polymorphic rendering where all domain entities expose Fields() for table display.

**Major components:**
1. **REST Client Layer** (github/rest_client.go) — HTTP client with Bearer auth for Actions API, separate from GraphQL client but shares token
2. **Domain Entities** (domain/workflow_run.go, domain/project_v2.go) — Implement Item interface with Fields() for table rendering, color-coded status
3. **UI Components** (ui/actions.go, ui/projects_v2.go) — SelectUI for tables, ViewUI for previews, nested SelectUI for hierarchical data (runs → jobs → logs)
4. **Async Update Queue** (UI.updater channel) — Buffered channel for thread-safe UI updates from API goroutines, prevents tview deadlocks
5. **Unified Rate Limiter** — Tracks both REST (5000 req/hr) and GraphQL (5000 pts/hr) pools, enforces 90 concurrent request limit

**Key patterns:**
- **Item Interface Polymorphism**: All entities (WorkflowRun, Job, ProjectV2Item) implement Item.Fields() for uniform SelectUI rendering
- **Async UI Updates**: API goroutines queue updates via `UI.updater <- func()` to avoid QueueUpdateDraw deadlocks
- **Dual-Client Architecture**: GraphQL and REST clients coexist with unified error handling and rate limit state
- **Nested UI Components**: SelectUI selection changes populate child SelectUI instances (WorkflowRun → Jobs, Project → Items)

### Critical Pitfalls

Research identified 7 critical pitfalls that must be addressed during implementation. The top 5 priorities are rate limit tracking, tview deadlocks, schema lag, pagination limits, and token scopes.

1. **shurcooL/githubv4 Schema Lag** — Library may lack Projects V2 custom field types (single-select, iteration), causing runtime panics. Mitigation: Test Projects V2 queries early, consider raw GraphQL with encoding/json, use map[string]interface{} for custom fields.

2. **GraphQL Node Limit Explosion** — Nested queries multiply node consumption exponentially (parent × child × grandchild). Keep `first: 10-20` for nested queries, never 30+. Projects V2 items have many field connections (text, number, date, select, assignees, labels) that explode node count.

3. **tview QueueUpdateDraw Deadlock** — Calling QueueUpdate() from event handlers deadlocks UI. NEVER call from SetInputCapture callbacks. Use App.Draw() directly on main goroutine, only QueueUpdateDraw() from worker goroutines. Establish channel-based update pattern in Phase 1.

4. **REST vs GraphQL Rate Limit Confusion** — Separate pools with different calculation methods (requests vs points). Both share 100 concurrent request limit. Must track both pools, parse GraphQL cost from response body AND REST X-RateLimit headers, implement semaphore for concurrent limit.

5. **Actions Pagination 10 Page Limit** — GitHub hard-limits pagination to 10 pages (1000 items). Use `created:<YYYY-MM-DD` filter with date ranges for older runs. Show warning: "Displaying last 1000 runs (GitHub API limit)".

6. **Projects V2 Node ID vs Database ID Confusion** — Webhooks return database IDs, GraphQL requires node IDs. Direct node() lookups fail for field IDs. Always query through parent paths (org → project → field), never direct node lookups.

7. **Insufficient PAT Scopes** — Classic PAT needs `project` + `read:org` + `repo`. Fine-grained PATs don't work with user-owned projects. Implement token validation on startup, check X-OAuth-Scopes header, fail fast with clear message.

## Implications for Roadmap

Based on research, the roadmap should proceed in three phases: Foundation (dual-client + rate limiter), Actions Integration (read-only monitoring), and Projects V2 Integration (read-only with navigation). This order is driven by increasing complexity (Actions is simpler than Projects V2), dependency resolution (both need Foundation patterns), and risk mitigation (establish async patterns before complex GraphQL queries).

### Phase 1: Foundation & Dual-Client Setup
**Rationale:** Both Actions and Projects V2 require foundational infrastructure: dual REST+GraphQL clients, unified rate limiter, async UI update patterns, and token scope validation. Building this shared foundation first prevents rework and catches critical pitfalls (deadlocks, rate limits) early.

**Delivers:**
- REST client (github/rest_client.go) with Bearer auth and error handling
- GraphQL client enhancement with rate limit cost tracking
- Unified rate limiter tracking both REST (5000 req/hr) and GraphQL (5000 pts/hr) pools
- Semaphore for 90 concurrent request limit across both APIs
- Token scope validation on startup (check X-OAuth-Scopes header)
- Async UI update queue pattern with channel (UI.updater)
- GraphQL complexity calculator (prevent node limit explosion)

**Addresses:**
- **Features**: Infrastructure for both Actions and Projects V2
- **Stack**: Integrate google/go-github/v83, establish dual-client pattern
- **Architecture**: Implement unified rate limiter, async update queue
- **Pitfalls**: #3 (tview deadlocks), #4 (rate limit confusion), #7 (token scopes)

**Avoids:**
- tview QueueUpdateDraw deadlocks by establishing channel pattern early
- Rate limit confusion by implementing unified tracking from start
- Token scope issues by validating on startup before any API calls

### Phase 2: Actions Integration (Read-Only)
**Rationale:** Actions is simpler than Projects V2 (no custom field complexity), provides immediate value (developers check CI status frequently), and validates the dual-client architecture with real REST API usage. Read-only monitoring covers 80% use case.

**Delivers:**
- WorkflowRun, Job, Step domain entities implementing Item interface
- ActionsUI (SelectUI) listing workflow runs with filtering
- JobsUI (SelectUI) for nested job hierarchy
- LogViewUI for displaying downloaded logs (ANSI stripped)
- Filter by status (success/failure/in_progress) and workflow name
- Navigation to GitHub (open run/commit/PR in browser)
- Pagination with date-based workaround for 10-page limit

**Uses:**
- **Stack**: google/go-github/v83 REST client, acarl005/stripansi for logs
- **Architecture**: Nested SelectUI pattern (WorkflowRun → Job), Item interface
- **Foundation**: REST client and unified rate limiter from Phase 1

**Implements:**
- **Features**: List runs, view logs, filter by status/workflow, navigate to GitHub
- **Table Stakes**: All "must have" Actions features (read-only MVP)

**Avoids:**
- **Pitfall #5**: Actions pagination limit (implement date-based workaround)
- **Pitfall #3**: Deadlocks (use established async update pattern)

### Phase 3: Projects V2 Integration (Read-Only)
**Rationale:** Projects V2 is more complex (custom fields, field type unions, org-scoped) but provides high value for project tracking. Read-only view is faster than web UI and integrates with existing issue navigation. Defer mutations to avoid state sync complexity.

**Delivers:**
- ProjectV2, ProjectV2Item, ProjectV2Field domain entities
- ProjectsUI (SelectUI) listing org/user projects
- ProjectItemsUI (SelectUI) with custom field columns (text, number, date, single-select)
- Filter by status column values
- Navigation to linked issues/PRs (reuse existing IssueUI)
- Custom field display with type-aware rendering

**Uses:**
- **Stack**: shurcooL/githubv4 GraphQL client (existing), fallback to raw queries if schema lag
- **Architecture**: SelectUI with dynamic columns, nested SelectUI (Project → Items)
- **Foundation**: GraphQL rate limiter, complexity calculator from Phase 1

**Implements:**
- **Features**: List projects, view items, display custom fields, filter by status, navigate to issue/PR
- **Table Stakes**: All "must have" Projects V2 features (read-only MVP)

**Avoids:**
- **Pitfall #1**: Schema lag (test custom field types early, use interface{} fallback)
- **Pitfall #2**: Node limit explosion (use first: 10-20, separate queries for fields)
- **Pitfall #6**: Node ID issues (query through parent paths, cache full objects)

### Phase 4: Actions Write Operations (Optional Enhancement)
**Rationale:** Once read-only monitoring is validated, add write operations (re-run, cancel, workflow dispatch) for full CI/CD workflow support. Requires `workflow` token scope (write permission).

**Delivers:**
- Re-run failed workflows (keybinding `r`)
- Cancel running workflows (keybinding `c`)
- Workflow dispatch with input form UI
- Live log streaming with auto-refresh (Phase 2 polling + websocket)

**Uses:**
- **Stack**: google/go-github/v83 Actions.RerunWorkflow(), Actions.CancelWorkflow()
- **Architecture**: Form UI for workflow_dispatch inputs (tview.InputField, tview.DropDown)

**Implements:**
- **Features**: Re-run, cancel, dispatch (all "should have" write operations)

### Phase 5: Projects V2 Advanced Features (Optional Enhancement)
**Rationale:** After read-only validation, add advanced features like custom field sorting and view switching. Defer full write operations (drag-drop, field editing) as web UI is better suited.

**Delivers:**
- Custom field sorting (sort by Priority, Status, Sprint)
- View switching (Table/Board/Roadmap layouts)
- Saved views (load predefined filters from web UI)
- Cross-project search

**Uses:**
- **Stack**: shurcooL/githubv4 with ProjectV2View queries
- **Architecture**: Multiple SelectUI rendering modes (table vs kanban)

**Implements:**
- **Features**: Sorting, view switching (all "nice to have" enhancements)

### Phase Ordering Rationale

- **Foundation First**: Both features depend on dual-client architecture, rate limiting, and async patterns. Building these first prevents rework and catches critical deadlock/rate limit pitfalls early.
- **Actions Before Projects V2**: Actions is simpler (no custom fields), provides immediate value (CI monitoring), and validates REST client with real usage. Projects V2 has higher complexity (field type unions, schema lag risks).
- **Read-Only Before Write**: Read-only monitoring covers 80% use cases for both features. Write operations add auth complexity (token scopes), mutation handling, and state sync issues. Validate read patterns first.
- **Incremental Risk**: Phase 1 addresses critical pitfalls (#3, #4, #7), Phase 2 adds moderate risks (#5), Phase 3 adds complex risks (#1, #2, #6). This prevents compounding unknowns.

### Research Flags

Phases likely needing deeper research during planning:

- **Phase 3 (Projects V2)**: Complex custom field types (iteration, milestone), schema lag with shurcooL/githubv4 may require raw GraphQL approach. Test early with real Projects V2 API.
- **Phase 4 (Actions Write)**: Workflow dispatch input validation, form UI patterns for dynamic inputs. Research how gh CLI handles --field flags.

Phases with standard patterns (skip research-phase):

- **Phase 1 (Foundation)**: Standard REST client, rate limiter, channel-based async patterns. Well-documented in Go.
- **Phase 2 (Actions Read)**: Straightforward REST API calls, SelectUI tables, log display. Clear patterns from existing code.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official GitHub API docs confirm Actions=REST, Projects V2=GraphQL. Library versions verified from releases. ANSI stripping requirement confirmed from Actions log format. |
| Features | HIGH | Analyzed gh CLI, lazyactions, gh-projects extensions. MVP features validated against competitor offerings. Table stakes confirmed by user expectations and tool analysis. |
| Architecture | HIGH | Existing codebase provides clear patterns (SelectUI, Item interface, async updates). Dual-client approach is industry standard. TUI patterns verified from k9s, lazydocker. |
| Pitfalls | HIGH | Critical pitfalls backed by GitHub community discussions, official docs (rate limits, pagination), and library issues (tview deadlocks, schema lag). Multiple sources confirm each pitfall. |

**Overall confidence:** HIGH

Research is backed by official documentation, multiple community sources, and existing codebase analysis. The dual-client architectural pattern is well-established, and pitfall mitigation strategies are specific and actionable.

### Gaps to Address

While research confidence is high, several areas need validation during implementation:

- **shurcooL/githubv4 Projects V2 Support**: Library's last schema update is 2024-07-27, but Projects V2 evolves quarterly. Test custom field types (single-select, iteration) early in Phase 3. If schema lag confirmed, implement raw GraphQL fallback with encoding/json.

- **Actions Log Size Limits**: Research indicates logs can be 100MB+. Phase 2 must implement chunked download or temp file streaming to avoid OOM. Validate TextView performance with 10k+ line logs during implementation.

- **Projects V2 Field Type Unions**: GraphQL unions for ProjectV2ItemFieldValue (text, number, date, single-select, iteration) may cause unmarshaling issues. Phase 3 should test with real projects having all field types before committing to struct-based approach.

- **Rate Limit Thresholds**: Research shows 5000 req/hr (REST) and 5000 pts/hr (GraphQL), but real-world behavior with concurrent requests needs validation. Phase 1 should log rate limit headers and adjust semaphore limit (90) if needed.

- **Pagination Cursor Consistency**: Actions uses Link headers (page numbers), Projects V2 uses GraphQL cursors (opaque strings). Phase 2/3 must test pagination end-of-data detection to avoid infinite loops on empty results.

## Sources

### Primary (HIGH confidence)
- [GitHub REST API - Actions Workflow Runs](https://docs.github.com/en/rest/actions/workflow-runs) — Actions endpoints, pagination, log download
- [GitHub REST API - Actions Workflow Jobs](https://docs.github.com/en/rest/actions/workflow-jobs) — Job listing, re-run operations
- [GitHub Projects V2 GraphQL API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) — ProjectsV2 schema, custom fields, mutations
- [GitHub GraphQL Rate Limits](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api) — 5000 points/hour, 500k node limit, complexity calculations
- [GitHub REST Rate Limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api) — 5000 req/hour, 100 concurrent limit
- [GitHub GraphQL Breaking Changes](https://docs.github.com/en/graphql/overview/breaking-changes) — Quarterly releases, 3-month deprecation notice
- [google/go-github v83.0.0 Release](https://github.com/google/go-github/releases) — Latest version, native pagination iterators
- [rivo/tview Package](https://pkg.go.dev/github.com/rivo/tview) — TextView, SelectUI patterns

### Secondary (MEDIUM confidence)
- [lazyactions - GitHub Actions TUI](https://github.com/nnnkkk7/lazyactions) — Reference for Actions TUI features, log streaming UX
- [gh-projects extension](https://github.com/heaths/gh-projects) — Projects V2 CLI patterns, field management
- [shurcooL/githubv4 Repository](https://github.com/shurcooL/githubv4) — GraphQL client status, last schema update
- [GitHub Community Discussion - Actions GraphQL API](https://github.com/orgs/community/discussions/24493) — Confirms no GraphQL for Actions
- [GitHub Community Discussion - Projects V2 Limitations](https://github.com/orgs/community/discussions/44265) — Status field management issues
- [GitHub Community Discussion - Actions Pagination Limit](https://github.com/orgs/community/discussions/26782) — 10 page limit confirmation
- [tview QueueUpdate Deadlock Issue](https://github.com/rivo/tview/issues/690) — Deadlock patterns and prevention

### Tertiary (LOW confidence)
- [Show HN: Lazyactions](https://news.ycombinator.com/item?id=46885757) — User feedback on Actions TUI
- [Terminal UI: BubbleTea vs Ratatui](https://www.glukhov.org/post/2026/02/tui-frameworks-bubbletea-go-vs-ratatui-rust/) — TUI framework patterns
- [Making a TUI with Go](https://taranveerbains.ca/blog/13-making-a-tui-with-go) — Concurrent update patterns

---
*Research completed: 2026-02-16*
*Ready for roadmap: yes*
