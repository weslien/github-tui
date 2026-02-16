# Feature Research: GitHub Actions & Projects V2 for TUI

**Domain:** GitHub TUI/CLI tools (GitHub Actions + Projects V2 integration)
**Researched:** 2026-02-16
**Confidence:** HIGH

## Feature Landscape

### GitHub Actions - Table Stakes (Users Expect These)

Features users assume exist when GitHub Actions support is added. Missing these = feature feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| List workflow runs | Core discovery - users need to see what's running/ran | LOW | Table view with status, workflow name, branch, commit, trigger, conclusion, duration |
| View run status | Users need to know if runs succeeded/failed | LOW | Color-coded status: queued, in_progress, completed (success/failure/cancelled) |
| View job logs | Essential for debugging failures | MEDIUM | Streaming logs with ANSI color support, auto-scroll, search within logs |
| Re-run failed workflows | Common recovery action after fixing issues | LOW | API call to re-run, requires write permissions |
| Filter runs by status | Users want to focus on failures or specific states | LOW | Filter by: success, failure, in_progress, queued, cancelled |
| Filter runs by workflow | Large repos have many workflows, need to focus | LOW | Dropdown/list of workflow names from .github/workflows/ |
| View run details | Context needed: commit SHA, author, timing, trigger event | LOW | Metadata panel showing run context |
| Navigate to related resources | Users need to jump to the commit/PR that triggered run | LOW | Open browser to GitHub run/commit/PR pages |

### GitHub Actions - Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable. These go beyond gh CLI's basic commands.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Live log streaming | Watch runs complete in real-time without browser refresh | MEDIUM | WebSocket or polling-based updates, lazyactions has this |
| Run/job tree view | Hierarchical display: Workflow → Runs → Jobs → Steps | MEDIUM | Expandable tree showing job dependencies and status |
| Workflow file preview | Quick view of .github/workflows/*.yml without leaving TUI | LOW | Read from repo, syntax highlighting optional |
| Multi-run comparison | Compare logs/timing across multiple runs of same workflow | HIGH | Side-by-side view, useful for intermittent failures |
| Keyboard-driven log navigation | vim-style: gg/G for top/bottom, / for search, n/N for next/prev | LOW | Matches existing TUI vim navigation patterns |
| Cancel running workflows | Stop workflows that are stuck or unnecessary | LOW | API call, useful for saving Actions minutes |
| Bulk operations | Re-run multiple failed runs at once | MEDIUM | Checkboxes + batch API calls |
| Workflow dispatch with inputs | Trigger manual workflows with custom parameters | MEDIUM | Form UI for workflow_dispatch inputs, gh CLI has this via --field flags |

### GitHub Projects V2 - Table Stakes (Users Expect These)

Features users assume exist when Projects V2 support is added.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| List projects | Discovery - see what projects exist | LOW | Org and repo-level projects, table view with name, description |
| View project items | Core functionality - see issues/PRs in project | MEDIUM | GraphQL API returns items with custom fields |
| Display custom fields | Projects V2 value = custom fields (status, priority, etc) | MEDIUM | Dynamic columns based on field types: text, number, date, single select |
| Filter by status | Users organize by Status column (Todo/In Progress/Done) | LOW | Filter dropdown for single-select fields |
| Navigate to issue/PR | Projects are index, users need to jump to actual item | LOW | Open issue/PR in existing issue view or browser |
| View project metadata | Title, description, owner, item count | LOW | Header/info panel |
| Multi-project support | Repos often have multiple projects | LOW | List + select pattern like existing issues UI |

### GitHub Projects V2 - Differentiators (Competitive Advantage)

Features that provide value beyond basic read-only access. Note: gh CLI Projects support is limited, extensions (gh-projects, gh-pm) fill gaps.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Read-only view optimization | Fast, focused view for tracking - no write complexity | LOW | Deliberately simpler than web UI, faster to load |
| Project-item linking | Click project item → jump to issue/PR view in TUI | LOW | Reuse existing issue UI, seamless navigation |
| Custom field sorting | Sort by Priority, Status, Sprint, etc | MEDIUM | Server-side via GraphQL orderBy |
| View switching | Support Table/Board/Roadmap layouts from API | HIGH | Projects V2 API supports views, complex UI work |
| Saved views | Access predefined filters/layouts from web UI | MEDIUM | GraphQL ProjectV2View queries |
| Cross-project item search | Find issues across all projects | MEDIUM | Aggregate search across projects |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems or scope creep.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full Projects V2 write access (drag-drop, field editing) | Web UI parity | Projects V2 GraphQL mutations are complex, maintaining state sync is difficult, web UI is better for editing | Keep read-only with "open in browser" for edits |
| Local workflow execution (like `act`) | Test Actions locally | Different tool with different scope, requires Docker, heavy implementation | Recommend separate `act` tool usage |
| Custom workflow builder/editor | Visual workflow creation | YAML editing is well-served by IDEs with GitHub Actions extensions | Recommend VS Code + GitHub Actions extension |
| Real-time notifications | Alert on workflow completion | Requires persistent connection, adds complexity, GitHub already sends notifications | Use GitHub's native notifications |
| Workflow analytics/metrics | Track success rates, timing trends | Reporting is complex, web UI has Insights | Refer users to Actions Insights page |
| Matrix strategy visualization | Show matrix builds in grid | Complex display for limited value, logs already show matrix params | Show matrix params in log metadata |

## Feature Dependencies

```
GitHub Actions:
List workflow runs
    └──requires──> GitHub API read:actions scope

View job logs
    └──requires──> List workflow runs (get run ID)
    └──enhances──> Live log streaming

Re-run failed workflows
    └──requires──> View run details (identify failure)
    └──requires──> GitHub API write:actions scope

Workflow dispatch with inputs
    └──requires──> Workflow file preview (know which inputs exist)

Live log streaming
    └──requires──> View job logs (base functionality)

GitHub Projects V2:
List projects
    └──requires──> GitHub GraphQL API + project scope

View project items
    └──requires──> List projects (get project ID)

Display custom fields
    └──requires──> View project items (fields are per-item)

Navigate to issue/PR
    └──requires──> Existing issue/PR view UI (reuse)
    └──requires──> View project items (get item node ID)

Custom field sorting
    └──requires──> Display custom fields (know field types)

View switching
    └──requires──> GraphQL ProjectV2View API
    └──conflicts──> Simple table view (adds UI complexity)
```

### Dependency Notes

- **View job logs requires List workflow runs:** Must select a run before viewing logs, run ID needed
- **Re-run requires View run details:** User needs to identify which run failed before re-running
- **Live log streaming enhances View job logs:** Streaming is progressive enhancement, base logs work without it
- **Navigate to issue/PR requires existing UI:** Projects V2 links to issues/PRs, reuse existing issue view for seamless UX
- **View switching conflicts with Simple table view:** Supporting multiple view layouts (Table/Board/Roadmap) adds significant UI complexity, conflicts with "simple read-only" positioning

## MVP Definition

### GitHub Actions - Launch With (Phase 1)

Minimum viable Actions support — what's needed to validate the feature.

- [x] List workflow runs — table view with status, workflow name, branch, conclusion
- [x] View run details — metadata panel (commit, author, trigger, timing)
- [x] View job logs — read-only log viewer with search and navigation
- [x] Filter runs by status — dropdown for success/failure/in_progress/queued
- [x] Filter runs by workflow — dropdown of workflow names
- [x] Navigate to GitHub — open run/commit/PR in browser

Rationale: Read-only monitoring covers 80% use case. Users can view status, debug via logs, jump to browser for actions (re-run).

### GitHub Actions - Add After Validation (Phase 2)

Features to add once core is working and validated.

- [ ] Re-run failed workflows — requires write permissions, adds value for CI/CD workflows
- [ ] Cancel running workflows — stop stuck/unnecessary runs
- [ ] Live log streaming — real-time updates without manual refresh
- [ ] Workflow dispatch with inputs — trigger manual workflows from TUI
- [ ] Workflow file preview — quick view of .yml without browser

Rationale: Write operations (re-run, cancel, dispatch) require additional auth setup. Live streaming is enhancement, not blocker.

### GitHub Projects V2 - Launch With (Phase 1)

Minimum viable Projects support — read-only view for tracking.

- [x] List projects — org + repo level, table view
- [x] View project items — list issues/PRs in project
- [x] Display custom fields — Status, Priority, etc as table columns
- [x] Filter by status — filter by Status field values
- [x] Navigate to issue/PR — open item in existing issue view or browser
- [x] View project metadata — title, description, item count

Rationale: Read-only project tracking. Users can view project state, navigate to items for details/edits in issue view or web UI.

### GitHub Projects V2 - Future Consideration (Phase 3+)

Features to defer until core is validated.

- [ ] Custom field sorting — sort by Priority, Status, custom fields
- [ ] View switching — support Board/Roadmap layouts (currently Table only)
- [ ] Saved views — load predefined filters from web UI
- [ ] Cross-project search — find items across multiple projects

Rationale: Sorting and advanced views add complexity. Start simple, validate read-only UX first.

## Feature Prioritization Matrix

### GitHub Actions

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| List workflow runs | HIGH | LOW | P1 |
| View job logs | HIGH | MEDIUM | P1 |
| View run details | HIGH | LOW | P1 |
| Filter by status | HIGH | LOW | P1 |
| Filter by workflow | HIGH | LOW | P1 |
| Navigate to GitHub | HIGH | LOW | P1 |
| Re-run failed workflows | MEDIUM | LOW | P2 |
| Cancel running workflows | MEDIUM | LOW | P2 |
| Live log streaming | MEDIUM | MEDIUM | P2 |
| Workflow dispatch | MEDIUM | MEDIUM | P2 |
| Workflow file preview | LOW | LOW | P2 |
| Multi-run comparison | LOW | HIGH | P3 |
| Bulk operations | LOW | MEDIUM | P3 |

### GitHub Projects V2

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| List projects | HIGH | LOW | P1 |
| View project items | HIGH | MEDIUM | P1 |
| Display custom fields | HIGH | MEDIUM | P1 |
| Filter by status | HIGH | LOW | P1 |
| Navigate to issue/PR | HIGH | LOW | P1 |
| View project metadata | MEDIUM | LOW | P1 |
| Custom field sorting | MEDIUM | MEDIUM | P2 |
| Project-item linking | HIGH | LOW | P1 |
| Cross-project search | LOW | MEDIUM | P3 |
| View switching | LOW | HIGH | P3 |
| Saved views | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for launch (Phase 1)
- P2: Should have, add when possible (Phase 2)
- P3: Nice to have, future consideration (Phase 3+)

## Competitor Feature Analysis

### GitHub Actions

| Feature | gh CLI | lazyactions TUI | Our Approach |
|---------|--------|-----------------|--------------|
| List runs | `gh run list` (JSON/table) | Table view with live updates | Table view (SelectUI pattern), match existing UI |
| View logs | `gh run view --log` (stream to stdout) | TUI with auto-scroll, search | TextView with vim navigation, search |
| Re-run | `gh run rerun` | Keybinding `r` | Phase 2: keybinding + modal confirm |
| Cancel | `gh run cancel` | Keybinding `c` | Phase 2: keybinding + modal confirm |
| Watch run | `gh run watch` (blocking CLI) | Live streaming | Phase 2: auto-refresh option |
| Workflow dispatch | `gh workflow run -f key=value` | Form UI with inputs | Phase 2: form with text inputs |
| Navigate to GitHub | Manual copy URL | Keybinding `y` (copy URL) | Keybinding to open browser |

### GitHub Projects V2

| Feature | gh CLI | gh-projects extension | gh-pm extension | Our Approach |
|---------|--------|----------------------|----------------|--------------|
| List projects | `gh project list` | Enhanced list with filters | List with fuzzy search | Table view (SelectUI) |
| View items | `gh project item-list` | Table with custom fields | Kanban-style view | Table with custom field columns |
| Custom fields | `gh project field-list` | View + create fields | Field management | Display only (read-only) |
| Navigate to item | Manual (print URL) | Not built-in | Jump to issue | Reuse existing issue UI |
| Filter | Limited | By assignee, status | Advanced filters | Status filter (dropdown) |
| Write operations | Create/edit/delete via flags | Full CRUD | Full CRUD | Deliberately excluded (anti-feature) |

**Key differentiators:**
- **gh CLI:** Scriptable but not interactive, requires chaining commands
- **lazyactions:** Actions-only, great reference for log streaming UX
- **gh-projects/gh-pm:** Full write access, more complex
- **Our approach:** Integrated into existing TUI, vim-style navigation, read-optimized, reuse issue view for seamless experience

## UX Patterns from Research

### Log Viewing (k9s, lazydocker, lazyactions)

**Common patterns:**
1. **Auto-scroll with toggle:** Logs auto-scroll to bottom, keybinding to disable (k9s: Ctrl+S, lazydocker: auto)
2. **Search:** `/` to enter search, `n`/`N` for next/prev match (vim-style)
3. **Vim navigation:** `gg` top, `G` bottom, `j`/`k` line by line, Ctrl+D/U for half-page
4. **ANSI color support:** Preserve terminal colors from logs
5. **Timestamps:** Toggle timestamp display on/off
6. **Line numbers:** Optional line numbers for reference
7. **Maximize view:** Fullscreen log view (hide sidebars)

### Streaming/Live Updates (k9s, lazydocker)

**Common patterns:**
1. **Auto-refresh indicators:** Visual cue when data is stale or refreshing
2. **Manual refresh:** `Ctrl+R` or `r` to force refresh
3. **Refresh interval config:** User-configurable polling interval
4. **Connection status:** Indicate when streaming is active/paused/disconnected

### Table/List Views (all TUIs)

**Common patterns:**
1. **Status color coding:** Green (success), Red (failure), Yellow (in-progress), Gray (queued)
2. **Relative timestamps:** "2m ago", "1h ago" vs absolute "2026-02-16 14:30"
3. **Sortable columns:** Click or keybinding to sort by column
4. **Multi-select:** Checkboxes or visual indicators for bulk operations
5. **Quick filters:** Top row with filter inputs or dropdowns

## Implementation Recommendations

### GitHub Actions Architecture

**New UI Components:**
1. **WorkflowRunsUI** (SelectUI) — table of runs with filtering
2. **WorkflowRunViewUI** (ViewUI) — run details metadata
3. **WorkflowLogUI** (ViewUI) — job log viewer with search
4. **WorkflowFilterUI** (FilterUI) — filter by workflow name, status, branch

**Data Layer:**
- GitHub REST API v3: `/repos/{owner}/{repo}/actions/runs` (list runs)
- GitHub REST API v3: `/repos/{owner}/{repo}/actions/runs/{run_id}/jobs` (list jobs)
- GitHub REST API v3: `/repos/{owner}/{repo}/actions/jobs/{job_id}/logs` (get logs)
- Polling for live updates (Phase 2): 5-10 second intervals for in_progress runs

**Navigation Flow:**
```
Actions Tab (top-level)
  → WorkflowRunsUI (table) [select run]
    → WorkflowLogUI (logs) [default to first failed job or first job]
    → WorkflowRunViewUI (metadata) [toggle with keybinding]
```

### GitHub Projects V2 Architecture

**New UI Components:**
1. **ProjectsUI** (SelectUI) — table of projects (reuse existing projects.go pattern)
2. **ProjectItemsUI** (SelectUI) — table of project items with custom fields
3. **ProjectViewUI** (ViewUI) — project metadata (title, description, stats)

**Data Layer:**
- GitHub GraphQL API: `projectsV2` query (list projects)
- GitHub GraphQL API: `projectV2` query with `items` (list items + custom fields)
- Field types to support: SingleSelectField, TextField, NumberField, DateField

**Navigation Flow:**
```
Projects Tab (existing, enhance)
  → ProjectsUI (list) [select project]
    → ProjectItemsUI (items table) [select item]
      → Navigate to IssueUI (reuse existing) OR open in browser
```

## API Scope Requirements

### GitHub Actions
- **Read-only (Phase 1):** No additional scopes (public repo), `repo` for private repos
- **Write operations (Phase 2):** `workflow` scope for re-run, cancel, dispatch

### GitHub Projects V2
- **Read-only:** `project:read` scope (or `project` for full access)
- GraphQL API required (not REST)

## Sources

### Official Documentation
- [GitHub CLI Manual - gh workflow run](https://cli.github.com/manual/gh_workflow_run) — HIGH confidence
- [GitHub CLI Manual - gh project](https://cli.github.com/manual/gh_project) — HIGH confidence
- [GitHub Docs - Using the API to manage Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) — HIGH confidence

### Tools Analyzed
- [lazyactions - GitHub Actions TUI](https://github.com/nnnkkk7/lazyactions) — PRIMARY reference for Actions TUI features, MEDIUM confidence
- [gh-projects extension](https://github.com/heaths/gh-projects) — Projects V2 extension analysis, MEDIUM confidence
- [gh-pm extension](https://github.com/yahsan2/gh-pm) — Projects V2 project management extension, MEDIUM confidence
- [k9s - Kubernetes TUI](https://k9scli.io/) — Log streaming patterns, HIGH confidence
- [lazydocker - Docker TUI](https://lazydocker.com/) — Container log patterns, HIGH confidence

### Community Resources
- [Show HN: Lazyactions](https://news.ycombinator.com/item?id=46885757) — User feedback on Actions TUI, LOW confidence
- [GitHub Discussions - ProjectV2 API](https://github.com/orgs/community/discussions/153532) — API limitations research, MEDIUM confidence
- [Terminal UI: BubbleTea vs Ratatui](https://www.glukhov.org/post/2026/02/tui-frameworks-bubbletea-go-vs-ratatui-rust/) — TUI framework patterns, MEDIUM confidence

---
*Feature research for: GitHub TUI (Actions + Projects V2 milestone)*
*Researched: 2026-02-16*
*Confidence: HIGH (official docs + multiple tool analysis)*
