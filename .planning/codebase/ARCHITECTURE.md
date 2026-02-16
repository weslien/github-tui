# Architecture

**Analysis Date:** 2026-02-16

## Pattern Overview

**Overall:** Layered MVC with Domain-Driven Design

**Key Characteristics:**
- Three-layer separation: presentation (UI), application (GitHub API client), and domain (business entities)
- TUI-based presentation using tview for terminal UI rendering
- GraphQL-based GitHub API integration with strongly-typed queries and mutations
- Event-driven updates through a channel-based updater pattern
- Centralized configuration management with YAML-based settings

## Layers

**Presentation Layer (UI):**
- Purpose: Renders terminal UI components and handles user interactions
- Location: `ui/`
- Contains: SelectUI, ViewUI, FilterUI, and specialized UIs (IssueUI, CommentUI, etc.)
- Depends on: Domain entities, GitHub API client, tview library
- Used by: Main application entry point in `cmd/ght/main.go`

**Application/Client Layer (GitHub):**
- Purpose: Provides GraphQL API abstraction for GitHub operations
- Location: `github/`
- Contains: Client initialization, query functions, mutation functions, and strongly-typed GraphQL structures
- Depends on: shurcooL/githubv4, oauth2, domain types for marshalling
- Used by: UI layer for data fetching and mutations

**Domain Layer:**
- Purpose: Defines core business entities and interfaces
- Location: `domain/`
- Contains: Issue, Comment, Label, Milestone, Assignee, Project types, and Item interface
- Depends on: tcell for color constants
- Used by: Both UI and GitHub API layers

**Configuration Layer:**
- Purpose: Manages application configuration and logging setup
- Location: `config/`
- Contains: GitHub token and repository configuration, logging initialization
- Depends on: go-yaml for parsing YAML config files
- Used by: Main entry point and GitHub client initialization

**Utilities Layer:**
- Purpose: Shared helper functions
- Location: `utils/`
- Contains: Edit function for launching external editor
- Used by: UI layer for content editing

## Data Flow

**User Input → UI Update:**

1. User presses key in terminal
2. UI component (SelectUI, ViewUI, etc.) captures event via `InputCapture`
3. Event handler executes action (fetch data, mutate, navigate)
4. UI calls GitHub API client function
5. API client returns domain entities
6. UI updates internal state and view
7. Updates sent through `UI.updater` channel to redraw display

**Initialization Flow:**

1. `main()` calls `config.Init()` - reads YAML config, sets up logging
2. `main()` calls `github.NewClient(token)` - creates OAuth2-authenticated GraphQL client
3. `main()` calls `ui.New().Start()` - initializes all UI components and starts event loop
4. UI.Start() creates all SelectUI, ViewUI, FilterUI instances
5. UI components call `getList` callbacks to populate initial data
6. Main event loop begins, goroutine manages UI updates from `updater` channel

**Issue List Display:**

1. IssueUI's `getList` function is called
2. Query string is validated and constructed from filter input
3. `github.GetIssues()` executes GraphQL query with pagination
4. Response nodes are converted to domain.Issue via `ToDomain()` method
5. SelectUI renders as table with columns: repo, number, state, author, title
6. User selection and interactions trigger mutations or preview updates

## Key Abstractions

**Primitive Interface:**
- Purpose: Unified abstraction for all UI components
- Location: `ui/ui.go` (lines 12-16)
- Implementations: FilterUI, SelectUI, ViewUI
- Pattern: Defines focus/blur methods for navigation and tview.Primitive embedding

**Item Interface:**
- Purpose: Represents displayable domain entities with key/field rendering
- Location: `domain/item.go` (lines 5-7)
- Implementations: Issue, Comment, Label, Assignee, Milestone, Project
- Pattern: Polymorphic rendering allowing different entity types in same SelectUI

**GetListFunc:**
- Purpose: Callback pattern for loading paginated data from GitHub
- Location: `ui/select.go` (line 33)
- Usage: Each SelectUI-based component defines custom data loading logic
- Pattern: Closures capture UI-specific query parameters and filtering

**UpdaterChannel:**
- Purpose: Thread-safe UI updates from goroutines
- Location: `ui/ui.go` (line 32)
- Pattern: Sends update functions to be executed on main thread via `app.QueueUpdateDraw()`

## Entry Points

**Command Entry:**
- Location: `cmd/ght/main.go`
- Triggers: User runs `ght` or `ght owner/repo` command
- Responsibilities: Parse arguments, initialize config, setup GitHub client, launch UI

**UI Entry:**
- Location: `ui/ui.go` - `Start()` method (line 125)
- Triggers: Called from main after client setup
- Responsibilities: Create all UI components, build grid layout, setup event routing, start event loop

**Config Initialization:**
- Location: `config/config.go` - `Init()` function (line 29)
- Triggers: Called first in main
- Responsibilities: Read YAML config file, validate GitHub token, setup logging to file and stderr

## Error Handling

**Strategy:** Immediate logging and user feedback via modal dialogs

**Patterns:**

- **Config Errors:** `log.Fatal()` on startup (missing config, invalid token) - prevents app launch
- **API Errors:** Logged with `log.Println()`, error message shown in `ui.Message()` modal dialog
- **Goroutine Errors:** Logged and shown to user without crashing (e.g., in issue closing, comment deletion)
- **Edit Command Errors:** Return error to caller for display via modal dialog

**Examples:**
- `github/client.go` - Query/mutation errors logged, let caller handle via modal
- `ui/issues.go` - GitHub API errors logged and displayed to user with `ui.Message(err.Error(), ...)`
- `config/config.go` - Fatal errors on missing token, config file, or parse failure

## Cross-Cutting Concerns

**Logging:**
- Initialized in `config.Init()` to write to both `~/.config/ght/debug.log` and stderr
- Used throughout codebase via standard `log` package
- No structured logging; simple text format for debugging

**Validation:**
- GitHub token presence validated at startup in `config.Init()`
- Repo owner/name extracted via `main.go` helper functions (Git remote parsing or command-line args)
- Query strings validated in UI components before API calls (e.g., ensuring "is:issue" in IssueUI)

**Authentication:**
- Personal access token from config file provided to `github.NewClient()`
- OAuth2 flow wraps token in `oauth2.StaticTokenSource` for GraphQL client
- Token passed to `githubv4.NewClient()` via HTTP client

**Pagination:**
- Each list query specifies `first: 30` and optional `after: cursor`
- PageInfo returned with `HasNextPage` and `EndCursor`
- SelectUI manages cursor and `hasNext` flag for fetch-more functionality

**UI Navigation:**
- Global `UI` variable holds main ui instance (`ui/ui.go` line 8)
- SelectUI tracks current focus via `current` index in primitives array
- Ctrl+N/Ctrl+P cycle through: filters, assignees, labels, milestones, projects, issues, views
- Ctrl+G jumps to issues, Ctrl+T jumps to filters

