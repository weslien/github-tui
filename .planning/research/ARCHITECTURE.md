# Architecture Research

**Domain:** GitHub TUI - Actions & Projects V2 Integration
**Researched:** 2026-02-16
**Confidence:** HIGH

## Standard Architecture

### System Overview

The existing GitHub TUI follows a clean 3-layer MVC architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                        UI Layer (tview)                      │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │FilterUI │  │SelectUI │  │ ViewUI  │  │SearchUI │        │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘        │
│       │            │            │            │              │
│       └────────────┴────────────┴────────────┘              │
│                      │                                       │
│                 Updater Channel (async rendering)           │
│                      │                                       │
├──────────────────────┴───────────────────────────────────────┤
│                   GitHub Client Layer                        │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────┐  │
│  │           GraphQL Client (githubv4)                   │  │
│  │  • Query functions (query_*.go)                       │  │
│  │  • Mutation functions (mutation_*.go)                 │  │
│  │  • Type mappings to domain                            │  │
│  └───────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      Domain Layer                            │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │  Issue   │  │ Comment  │  │  Label   │                   │
│  └──────────┘  └──────────┘  └──────────┘                   │
│  All implement Item interface for polymorphic rendering     │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **ui/** | tview components, keybindings, user interaction, async updates | SelectUI (table list), ViewUI (preview), FilterUI (search bar) |
| **github/** | API client layer, GraphQL queries/mutations, type conversion | Package-level client singleton, query functions, domain mappers |
| **domain/** | Business entities, Item interface implementation, field rendering | Structs implementing Item interface with Key() and Fields() |
| **config/** | Configuration loading, token management, repository context | Global config state, YAML parsing |
| **utils/** | Cross-cutting utilities (editor, browser, string formatting) | Helper functions without state |

## Recommended Project Structure for New Features

```
github-tui/
├── cmd/ght/
│   └── main.go                 # Entry point - unchanged
├── config/
│   └── config.go               # Config - may need REST API URL if custom
├── domain/
│   ├── item.go                 # Item interface - unchanged
│   ├── issue.go                # Existing entities
│   ├── workflow_run.go         # NEW: Actions workflow run entity
│   ├── job.go                  # NEW: Actions job entity
│   ├── step.go                 # NEW: Actions step entity
│   ├── project_v2.go           # NEW: Projects V2 project entity
│   └── project_v2_item.go      # NEW: Projects V2 item entity
├── github/
│   ├── client.go               # GraphQL client - unchanged
│   ├── rest_client.go          # NEW: REST client for Actions API
│   ├── query_*.go              # Existing GraphQL queries
│   ├── query_actions.go        # NEW: Actions REST API calls (not GraphQL)
│   ├── query_projects_v2.go    # NEW: Projects V2 GraphQL queries
│   └── mutation_projects_v2.go # NEW: Projects V2 mutations
├── ui/
│   ├── ui.go                   # Main UI orchestrator - needs new primitives
│   ├── select.go               # Reusable SelectUI - unchanged
│   ├── view.go                 # Reusable ViewUI - unchanged
│   ├── actions.go              # NEW: Actions workflow runs UI
│   ├── jobs.go                 # NEW: Jobs UI (sub-view of workflow)
│   ├── logs.go                 # NEW: Log viewer UI (streaming capability)
│   └── projects_v2.go          # NEW: Projects V2 board UI
└── utils/
    └── utils.go                # Utilities - may need log streaming helpers
```

### Structure Rationale

- **domain/**: New entities follow existing pattern (WorkflowRun, Job, Step all implement Item interface)
- **github/**: REST client coexists with GraphQL client; Actions requires REST API exclusively
- **ui/**: New UI components follow SelectUI/ViewUI patterns; tab-based navigation for feature switching
- **Parallel structure**: Actions and Projects V2 are independent feature branches that can be built separately

## Architectural Patterns

### Pattern 1: Item Interface Polymorphism

**What:** All domain entities implement the Item interface for uniform rendering in SelectUI tables.

**When to use:** Any listable entity that appears in a table view (Issues, Comments, Labels, WorkflowRuns, Jobs, ProjectV2Items).

**Trade-offs:**
- ✅ Single SelectUI component handles all list types
- ✅ Easy to add new entity types without UI changes
- ❌ Fields() method must flatten rich data into table columns
- ❌ Color-coding logic embedded in domain layer

**Example:**
```go
// domain/workflow_run.go
type WorkflowRun struct {
    ID         string
    Name       string
    Status     string // queued, in_progress, completed
    Conclusion string // success, failure, cancelled, skipped
    HeadBranch string
    CreatedAt  string
    UpdatedAt  string
    URL        string
    Jobs       []Item // nested jobs
}

func (w *WorkflowRun) Key() string {
    return w.ID
}

func (w *WorkflowRun) Fields() []Field {
    statusColor := tcell.ColorYellow
    if w.Status == "completed" {
        statusColor = tcell.ColorGreen
        if w.Conclusion == "failure" {
            statusColor = tcell.ColorRed
        }
    }

    return []Field{
        {Text: w.Name, Color: tcell.ColorWhite},
        {Text: w.Status, Color: statusColor},
        {Text: w.HeadBranch, Color: tcell.ColorBlue},
        {Text: w.CreatedAt, Color: tcell.ColorGray},
    }
}
```

### Pattern 2: Async UI Updates via Channel

**What:** UI updates are queued through a buffered channel and applied via QueueUpdateDraw to avoid race conditions.

**When to use:** Any background operation that modifies UI state (API calls, polling, batch operations).

**Trade-offs:**
- ✅ Thread-safe UI updates
- ✅ Decouples API layer from UI threading
- ❌ Adds latency (100-item buffer in updater channel)
- ❌ Must remember to use updater channel, not direct UI calls

**Example:**
```go
// ui/actions.go
func (ui *ActionsUI) UpdateView() {
    UI.updater <- func() {
        ui.Clear()
        for i, run := range ui.workflowRuns {
            for j, field := range run.Fields() {
                ui.SetCell(i, j, tview.NewTableCell(field.Text).SetTextColor(field.Color))
            }
        }
    }
}
```

### Pattern 3: Dual-Client Architecture (GraphQL + REST)

**What:** Maintain separate client instances for GraphQL and REST APIs with unified error handling.

**When to use:** When GitHub feature requires REST API (Actions logs) alongside GraphQL queries (Projects V2).

**Trade-offs:**
- ✅ Use best API for each feature (Actions has no GraphQL support)
- ✅ REST client can handle log streaming (chunked transfers)
- ❌ Two authentication mechanisms to maintain
- ❌ Different error response formats to normalize

**Example:**
```go
// github/rest_client.go
package github

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

var restClient *http.Client
var restBaseURL = "https://api.github.com"

func NewRESTClient(token string) {
    restClient = &http.Client{
        Transport: &authTransport{
            token: token,
            base:  http.DefaultTransport,
        },
    }
}

type authTransport struct {
    token string
    base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+t.token)
    req.Header.Set("Accept", "application/vnd.github.v3+json")
    return t.base.RoundTrip(req)
}

// github/query_actions.go
func GetWorkflowRuns(owner, repo string, cursor *string) ([]WorkflowRun, *PageInfo, error) {
    url := fmt.Sprintf("%s/repos/%s/%s/actions/runs?per_page=30", restBaseURL, owner, repo)
    if cursor != nil && *cursor != "" {
        url += "&page=" + *cursor
    }

    req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
    if err != nil {
        return nil, nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := restClient.Do(req)
    if err != nil {
        return nil, nil, fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, nil, fmt.Errorf("API error %d: %s", resp.StatusCode, body)
    }

    var result struct {
        WorkflowRuns []struct {
            ID         int64  `json:"id"`
            Name       string `json:"name"`
            Status     string `json:"status"`
            Conclusion string `json:"conclusion"`
            HeadBranch string `json:"head_branch"`
            CreatedAt  string `json:"created_at"`
            UpdatedAt  string `json:"updated_at"`
            HTMLURL    string `json:"html_url"`
        } `json:"workflow_runs"`
        TotalCount int `json:"total_count"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, nil, fmt.Errorf("decode response: %w", err)
    }

    runs := make([]WorkflowRun, len(result.WorkflowRuns))
    for i, r := range result.WorkflowRuns {
        runs[i] = WorkflowRun{
            ID:         fmt.Sprintf("%d", r.ID),
            Name:       r.Name,
            Status:     r.Status,
            Conclusion: r.Conclusion,
            HeadBranch: r.HeadBranch,
            CreatedAt:  r.CreatedAt,
            UpdatedAt:  r.UpdatedAt,
            URL:        r.HTMLURL,
        }
    }

    // REST API uses Link header for pagination, simplified here
    pageInfo := &PageInfo{
        HasNextPage: len(runs) == 30, // Simplified
    }

    return runs, pageInfo, nil
}
```

### Pattern 4: Nested UI Components for Hierarchical Data

**What:** Workflow runs contain jobs, jobs contain steps. Use SelectUI selection changes to populate child SelectUI instances.

**When to use:** When entities have parent-child relationships (WorkflowRun → Job → Step, ProjectV2 → ProjectV2Item).

**Trade-offs:**
- ✅ Reuses SelectUI component at multiple levels
- ✅ Familiar navigation pattern (like Issue → Comment)
- ❌ Requires careful state management for nested data fetching
- ❌ Deep nesting (3 levels) may clutter UI grid layout

**Example:**
```go
// ui/actions.go
func NewActionsUI() {
    // Similar to IssueUI pattern
    ui := NewSelectListUI(UIKindWorkflowRuns, tcell.ColorBlue, func(ui *SelectUI) {
        ui.getList = func(cursor *string) ([]domain.Item, *github.PageInfo) {
            runs, pageInfo, err := github.GetWorkflowRuns(
                config.GitHub.Owner,
                config.GitHub.Repo,
                cursor,
            )
            if err != nil {
                log.Println(err)
                return nil, nil
            }
            items := make([]domain.Item, len(runs))
            for i, run := range runs {
                items[i] = &run
            }
            return items, pageInfo
        }

        ui.header = []string{"", "Workflow", "Status", "Branch", "Updated"}
        ui.hasHeader = true
    })

    ui.SetSelectionChangedFunc(func(row, col int) {
        // Load jobs for selected workflow run
        if run := ui.GetSelect(); run != nil {
            workflowRun := run.(*domain.WorkflowRun)
            jobs, err := github.GetWorkflowRunJobs(workflowRun.ID)
            if err != nil {
                log.Println(err)
                return
            }
            JobsUI.SetList(jobs) // Populate nested JobsUI
        }
    })

    ActionsUI = ui
}
```

## Data Flow

### Request Flow: Actions Feature

```
[User presses 'Ctrl-A' to switch to Actions tab]
    ↓
[ui.go switches focus to ActionsUI]
    ↓
[ActionsUI.GetList() called] → [github.GetWorkflowRuns()] → [REST API request]
    ↓                                                              ↓
[Response] ← [JSON decode] ← [HTTP 200 with workflow_runs array]
    ↓
[Convert to domain.WorkflowRun items]
    ↓
[ActionsUI.UpdateView() via updater channel]
    ↓
[tview renders table with workflow runs]

[User selects a workflow run]
    ↓
[SelectionChangedFunc triggers] → [github.GetWorkflowRunJobs(runID)]
    ↓                                                    ↓
[Jobs returned] ← [REST API /repos/{owner}/{repo}/actions/runs/{run_id}/jobs]
    ↓
[JobsUI.SetList(jobs)] → [UI.updater channel] → [Jobs table rendered]

[User presses 'l' for logs]
    ↓
[github.GetWorkflowRunLogs(runID)] → [REST API returns 302 redirect]
    ↓
[Follow redirect to log archive URL] → [Download logs]
    ↓
[Display in ViewUI or external viewer]
```

### Request Flow: Projects V2 Feature

```
[User presses 'Ctrl-V' to switch to Projects tab]
    ↓
[ui.go switches focus to ProjectsV2UI]
    ↓
[ProjectsV2UI.GetList() called] → [github.GetProjectsV2()] → [GraphQL query]
    ↓                                                              ↓
[Response] ← [GraphQL decode] ← [{organization {projectsV2 {nodes}}}]
    ↓
[Convert to domain.ProjectV2 items]
    ↓
[ProjectsV2UI.UpdateView() via updater channel]
    ↓
[tview renders table with projects]

[User selects a project]
    ↓
[SelectionChangedFunc triggers] → [github.GetProjectV2Items(projectID)]
    ↓                                                        ↓
[Items returned] ← [GraphQL query with project items and field values]
    ↓
[ProjectV2ItemsUI.SetList(items)] → [UI.updater channel] → [Items table rendered]

[User presses 'e' to edit item field]
    ↓
[Show edit form with field types (text, number, date, select, iteration)]
    ↓
[github.UpdateProjectV2ItemFieldValue(mutation)] → [GraphQL mutation]
    ↓
[Refresh ProjectV2ItemsUI]
```

### State Management

The application uses **package-level globals** for UI components and client state:

```
ui.UI           *ui.ui (global TUI app state)
github.client   *githubv4.Client (GraphQL client)
github.restClient *http.Client (REST client for Actions)
config.GitHub   github (owner, repo, token)

ui.IssueUI      *SelectUI
ui.ActionsUI    *SelectUI (NEW)
ui.JobsUI       *SelectUI (NEW)
ui.ProjectsV2UI *SelectUI (NEW)
```

No external state management library needed; channel-based updater handles async coordination.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| GitHub GraphQL API (v4) | githubv4 library with oauth2 | Existing - used for Issues, Projects (v1), Comments, Labels |
| GitHub REST API (v3) | net/http with Bearer token auth | **NEW** - required for Actions (no GraphQL support) |
| GitHub Actions Logs | REST API redirect to archive URL | Logs returned as 302 redirect to .zip file |
| System Editor ($EDITOR) | os/exec suspension of TUI | Existing - used for editing issue bodies |
| System Browser | utils.Open() for URL launching | Existing - used for opening issues in browser |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| **ui/ ↔ github/** | Direct function calls | UI calls github.GetWorkflowRuns(), github.GetProjectsV2() |
| **github/ ↔ domain/** | Type conversion (ToDomain methods) | github.WorkflowRun → domain.WorkflowRun |
| **ui/ ↔ domain/** | Item interface | SelectUI operates on []domain.Item, calls Fields() for rendering |
| **ui/ → ui/ (async)** | updater channel | UI.updater <- func() for thread-safe updates |
| **REST client ↔ GraphQL client** | Independent, no direct calls | Both use config.GitHub.Token, different request formats |

### API Client Coexistence

```go
// cmd/ght/main.go
func main() {
    config.Init()
    getRepoInfo()

    // Initialize both clients with same token
    github.NewClient(config.GitHub.Token)       // GraphQL client
    github.NewRESTClient(config.GitHub.Token)   // REST client

    if err := ui.New().Start(); err != nil {
        log.Fatal(err)
    }
}
```

Both clients authenticate with `config.GitHub.Token`, but:
- GraphQL client uses oauth2.StaticTokenSource
- REST client uses custom http.RoundTripper with Authorization header

## Build Order & Dependencies

### Component Dependencies

```
Phase 1: Actions Foundation (no dependencies on existing UI patterns)
  ├── domain/workflow_run.go (implements Item interface)
  ├── domain/job.go (implements Item interface)
  ├── domain/step.go (implements Item interface)
  └── github/rest_client.go (net/http wrapper)

Phase 2: Actions API Layer (depends on Phase 1)
  ├── github/query_actions.go (REST calls using rest_client)
  └── Tests for API layer (integration tests optional)

Phase 3: Actions UI (depends on Phase 1 + 2)
  ├── ui/actions.go (SelectUI for workflow runs)
  ├── ui/jobs.go (SelectUI for jobs)
  ├── ui/logs.go (ViewUI or custom log viewer)
  └── ui/ui.go (integrate Actions tab into grid layout)

Phase 4: Projects V2 Foundation (independent of Actions)
  ├── domain/project_v2.go (implements Item interface)
  ├── domain/project_v2_item.go (implements Item interface)
  └── domain/project_v2_field.go (field type definitions)

Phase 5: Projects V2 API Layer (depends on Phase 4)
  ├── github/query_projects_v2.go (GraphQL queries)
  ├── github/mutation_projects_v2.go (GraphQL mutations)
  └── Tests for API layer

Phase 6: Projects V2 UI (depends on Phase 4 + 5)
  ├── ui/projects_v2.go (SelectUI for projects and items)
  ├── ui/project_v2_board.go (optional: kanban-style board view)
  └── ui/ui.go (integrate Projects V2 tab into grid layout)
```

### Suggested Build Order

**Recommendation: Build Actions first, then Projects V2**

Rationale:
1. **Actions is simpler**: Read-only workflow/job listing with log viewing (no mutations initially)
2. **REST client pattern**: Establishes dual-client architecture early for validation
3. **Immediate value**: Developers frequently check CI status; high-utility feature
4. **Projects V2 is complex**: Field types (text, number, date, select, iteration), mutations, board layout

**Minimal Viable Features (MVF) per Phase:**

**Actions MVF:**
- List workflow runs (status, conclusion, branch)
- Select run → view jobs
- Select job → view logs (download .zip, display in ViewUI)
- Keybinding 'r' to re-run failed jobs

**Projects V2 MVF:**
- List organization/user projects
- Select project → view items (issues, PRs, drafts)
- Display item field values (text, number, select)
- Edit single-select and text fields
- Add existing issue/PR to project

**Defer to later:**
- Actions: Workflow approval, cancel runs, artifacts download
- Projects V2: Kanban board view, iteration fields, bulk operations, custom field creation

## REST API Client Implementation Considerations

### Authentication

Both GraphQL and REST APIs accept the same personal access token. Minimum required scopes:

**Actions API:**
- `repo` (for private repositories)
- `actions:read` (to list workflows and logs)
- `actions:write` (to re-run workflows)

**Projects V2 API:**
- `project` (for mutations: add items, update fields)
- `read:project` (for queries: list projects, items)

**Recommendation:** Update README.md to specify required token scopes for new features.

### Error Handling

REST API returns errors differently than GraphQL:

**GraphQL error:**
```json
{
  "data": null,
  "errors": [{"message": "Field 'xyz' doesn't exist", "type": "INVALID_QUERY"}]
}
```

**REST API error:**
```json
{
  "message": "Not Found",
  "documentation_url": "https://docs.github.com/rest/actions/workflow-runs"
}
```

Normalize errors in github/ layer:

```go
type APIError struct {
    StatusCode int
    Message    string
    Source     string // "graphql" or "rest"
}

func (e *APIError) Error() string {
    return fmt.Sprintf("[%s] %d: %s", e.Source, e.StatusCode, e.Message)
}
```

### Rate Limiting

GitHub REST API: 5000 requests/hour for authenticated requests
GitHub GraphQL API: 5000 points/hour (queries cost different points)

**Strategy:**
- Cache workflow runs for 30 seconds (most won't update that frequently)
- Paginate with reasonable page sizes (30 items per page)
- Show rate limit status in UI footer (optional enhancement)

### Streaming Logs

Actions logs can be large (10MB+). Options:

1. **Download to temp file, open in ViewUI** (simple, works for most cases)
2. **Stream line-by-line to ViewUI** (better UX, requires chunked HTTP reads)
3. **External viewer** (fallback: download .zip, extract, open in $PAGER)

**Recommendation:** Start with option 1 (download to temp file), enhance with streaming in later iteration.

```go
// github/query_actions.go
func DownloadWorkflowRunLogs(runID string) (string, error) {
    // GET returns 302 redirect
    url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%s/logs",
        restBaseURL, config.GitHub.Owner, config.GitHub.Repo, runID)

    req, _ := http.NewRequest("GET", url, nil)
    resp, err := restClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Follow redirect
    if resp.StatusCode == http.StatusFound {
        redirectURL := resp.Header.Get("Location")
        resp, err = http.Get(redirectURL)
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
    }

    // Save to temp file
    tmpFile, err := os.CreateTemp("", "ght-logs-*.zip")
    if err != nil {
        return "", err
    }
    defer tmpFile.Close()

    if _, err := io.Copy(tmpFile, resp.Body); err != nil {
        return "", err
    }

    return tmpFile.Name(), nil
}
```

## UI Layout Integration

### Current Grid Layout (9 rows × 7 cols)

```
Row 0: IssueFilterUI (search bar)
Row 1-4: IssueUI (left col 1-3) + Assignees/Labels/Milestones/Projects (col 0) + IssueViewUI (col 4-6)
Row 5-8: CommentUI (left col 0-3) + CommentViewUI (right col 4-6)
Row 9: SearchUI (footer search bar)
```

### Proposed Layout with Tabs

**Option A: Side-by-side tabs (split left column)**

```
Row 0: TabBar [Issues | Actions | Projects]
Row 1-4: [Active tab content] + [Preview pane]
Row 5-8: [Active tab details] + [Detail preview]
Row 9: SearchUI
```

**Option B: Full-screen tab switching**

Switch entire layout based on active tab. Issues tab shows current layout, Actions tab shows:

```
Row 0: IssueFilterUI (shared search bar, queries adapted per tab)
Row 1-4: ActionsUI (workflow runs) + WorkflowViewUI (run details)
Row 5-8: JobsUI (jobs list) + LogViewUI (log preview)
Row 9: SearchUI
```

**Recommendation:** Option B (full-screen tab switching) reuses UI primitives without cramping layout.

### Keybinding Additions

```go
// ui/ui.go - SetInputCapture additions
case tcell.KeyCtrlA:
    // Switch to Actions tab
    ui.switchTab(TabActions)
case tcell.KeyCtrlV:
    // Switch to Projects V2 tab (Ctrl-P already used)
    ui.switchTab(TabProjectsV2)
```

## Projects V2 Field Type Handling

Projects V2 supports multiple field types with different input patterns:

| Field Type | GraphQL Type | Input Method | Example |
|-----------|--------------|--------------|---------|
| Text | `text` | InputField | "In progress review notes" |
| Number | `number` | InputField with validation | 42 |
| Date | `date` | InputField (YYYY-MM-DD) | 2026-03-15 |
| Single Select | `singleSelect` | Dropdown | "High" (from options) |
| Iteration | `iteration` | Dropdown | "Sprint 5" (from iterations) |

### Form Handling for Field Updates

```go
// ui/projects_v2.go
func editProjectItemField(item *domain.ProjectV2Item, field *domain.ProjectV2Field) {
    switch field.Type {
    case "singleSelect":
        // Show dropdown with field.Options
        dropdown := tview.NewDropDown().
            SetLabel(field.Name).
            SetOptions(field.Options, func(text string, index int) {
                // Update field with selected option ID
                optionID := field.OptionIDs[index]
                github.UpdateProjectV2ItemField(item.ID, field.ID, optionID)
            })
        // ... show form with dropdown

    case "text":
        // Show input field
        input := tview.NewInputField().SetLabel(field.Name).SetText(item.FieldValues[field.ID])
        // ... handle submit

    case "number":
        // Show input field with number validation
        input := tview.NewInputField().SetLabel(field.Name).
            SetAcceptanceFunc(tview.InputFieldInteger)
        // ... handle submit

    case "date":
        // Show input field with date validation (YYYY-MM-DD)
        input := tview.NewInputField().SetLabel(field.Name).
            SetPlaceholder("YYYY-MM-DD")
        // ... handle submit with date parsing

    case "iteration":
        // Show dropdown with iteration titles
        dropdown := tview.NewDropDown().
            SetLabel(field.Name).
            SetOptions(field.IterationTitles, func(text string, index int) {
                iterationID := field.IterationIDs[index]
                github.UpdateProjectV2ItemField(item.ID, field.ID, iterationID)
            })
        // ... show form
    }
}
```

Field metadata must be fetched when loading project:

```graphql
query($projectId: ID!) {
  node(id: $projectId) {
    ... on ProjectV2 {
      fields(first: 20) {
        nodes {
          ... on ProjectV2Field {
            id
            name
            dataType
          }
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
          ... on ProjectV2IterationField {
            id
            name
            configuration {
              iterations {
                id
                title
                startDate
              }
            }
          }
        }
      }
    }
  }
}
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Mixing GraphQL and REST in Same Function

**What people do:**
```go
func GetWorkflowWithIssues(runID string) (*domain.WorkflowRun, error) {
    run, _ := github.GetWorkflowRun(runID) // REST API
    issue, _ := github.GetIssue(...)       // GraphQL API (from PR linked to workflow)
    // ... combine data
}
```

**Why it's wrong:** Creates tight coupling between APIs, hard to test, error handling becomes complex.

**Do this instead:**
```go
// Keep API calls separate in UI layer
run, err := github.GetWorkflowRun(runID)      // REST
if err != nil { /* handle */ }
ActionsUI.UpdateView()

issue, err := github.GetIssue(...)             // GraphQL
if err != nil { /* handle */ }
IssueViewUI.updateView(issue.Body)
```

### Anti-Pattern 2: Polling Without Backoff

**What people do:**
```go
// Poll workflow status every second
ticker := time.NewTicker(1 * time.Second)
for range ticker.C {
    runs, _ := github.GetWorkflowRuns(...)
    ActionsUI.SetList(runs)
}
```

**Why it's wrong:** Wastes API rate limit, poor UX (constant UI flicker), doesn't account for rate limit errors.

**Do this instead:**
```go
// Manual refresh with 'f' key, or smart polling with exponential backoff
ui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
    switch event.Rune() {
    case 'f':
        go ui.FetchList() // Manual refresh
    case 'w':
        go ui.WatchWorkflow() // Start smart polling (10s initially, back off to 60s)
    }
    return event
})
```

### Anti-Pattern 3: Large Log Files in Memory

**What people do:**
```go
logs, _ := github.GetWorkflowRunLogs(runID) // Download entire 50MB log
ViewUI.SetText(string(logs)) // OOM on large logs
```

**Why it's wrong:** TUIs can crash on large strings (50MB+ logs), poor memory usage, slow rendering.

**Do this instead:**
```go
// Option 1: Stream to temp file, tail last 10K lines
logFile, _ := github.DownloadWorkflowRunLogs(runID)
tailOutput, _ := exec.Command("tail", "-n", "10000", logFile).Output()
ViewUI.SetText(string(tailOutput))

// Option 2: External viewer
utils.Open(logFile) // Open in system pager
```

### Anti-Pattern 4: Hardcoded PageInfo Logic

**What people do:**
```go
func GetProjectItems(projectID string) []domain.Item {
    // Fetch first page only, ignore pagination
    resp, _ := github.query(...)
    return resp.Items
}
```

**Why it's wrong:** Users won't see items beyond first page, inconsistent with Issues pagination pattern.

**Do this instead:**
```go
// Use existing SelectUI pagination pattern
ui.getList = func(cursor *string) ([]domain.Item, *github.PageInfo) {
    items, pageInfo, err := github.GetProjectV2Items(projectID, cursor)
    if err != nil {
        log.Println(err)
        return nil, nil
    }
    return items, pageInfo // SelectUI handles 'f' key for "fetch more"
}
```

## Testing Strategy

### Unit Tests

**Priority 1:** Domain entities
```go
// domain/workflow_run_test.go
func TestWorkflowRun_Fields(t *testing.T) {
    run := &WorkflowRun{
        Name: "CI",
        Status: "completed",
        Conclusion: "success",
    }
    fields := run.Fields()
    assert.Equal(t, "CI", fields[0].Text)
    assert.Equal(t, tcell.ColorGreen, fields[1].Color) // Success = green
}
```

**Priority 2:** REST client error handling
```go
// github/rest_client_test.go
func TestGetWorkflowRuns_APIError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        w.Write([]byte(`{"message": "Not Found"}`))
    }))
    defer server.Close()

    restBaseURL = server.URL
    _, _, err := GetWorkflowRuns("owner", "repo", nil)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "Not Found")
}
```

**Priority 3:** GraphQL query construction (Projects V2)
```go
// github/query_projects_v2_test.go
func TestGetProjectV2_QueryStructure(t *testing.T) {
    // Test that query includes required fields: id, title, items, fields
    // Use github.com/shurcooL/githubv4's testing utilities
}
```

### Integration Tests

**Optional:** Real API tests with test repository (requires GITHUB_TOKEN env var)

```go
// github/integration_test.go
// +build integration

func TestActionsAPI_RealRepository(t *testing.T) {
    if os.Getenv("GITHUB_TOKEN") == "" {
        t.Skip("GITHUB_TOKEN not set")
    }
    NewRESTClient(os.Getenv("GITHUB_TOKEN"))
    runs, pageInfo, err := GetWorkflowRuns("skanehira", "github-tui", nil)
    assert.NoError(t, err)
    assert.NotNil(t, pageInfo)
    assert.Greater(t, len(runs), 0)
}
```

Run with: `go test -tags=integration ./github/...`

### Manual Testing Checklist

**Actions:**
- [ ] List workflow runs for repository with multiple workflows
- [ ] Select run → jobs populate correctly
- [ ] View logs (download works, display readable)
- [ ] Re-run failed workflow (requires write:actions scope)
- [ ] Pagination works ('f' key loads more runs)

**Projects V2:**
- [ ] List organization projects
- [ ] Select project → items populate
- [ ] View item field values (text, number, date, select)
- [ ] Edit text field → mutation succeeds
- [ ] Edit single-select field → dropdown shows options
- [ ] Add existing issue to project

## Sources

**GitHub Actions API:**
- [REST API endpoints for workflow runs](https://docs.github.com/en/rest/actions/workflow-runs) - HIGH confidence
- [REST API endpoints for workflow jobs](https://docs.github.com/en/rest/actions/workflow-jobs) - HIGH confidence
- [GitHub community discussion: GraphQL Actions API](https://github.com/orgs/community/discussions/24493) - MEDIUM confidence (confirms GraphQL not available)

**GitHub Projects V2 API:**
- [Using the API to manage Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) - HIGH confidence
- [Examples for calling GitHub GraphQL API (ProjectsV2)](https://devopsjournal.io/blog/2022/11/28/github-graphql-queries) - MEDIUM confidence

**Existing Codebase:**
- Analyzed ui/, github/, domain/ layers directly from repository
- Confirmed tview components (SelectUI, ViewUI, FilterUI patterns)
- Verified channel-based updater architecture (UI.updater buffered channel)
- Identified Item interface polymorphism pattern

---

*Architecture research for: GitHub TUI - Actions & Projects V2 Integration*
*Researched: 2026-02-16*
