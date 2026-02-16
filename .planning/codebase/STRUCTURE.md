# Codebase Structure

**Analysis Date:** 2026-02-16

## Directory Layout

```
github-tui/
├── cmd/ght/              # Application entry point
│   └── main.go           # CLI bootstrapping and repo detection
├── config/               # Configuration management
│   └── config.go         # YAML config loading, logging setup
├── domain/               # Domain entities and interfaces
│   ├── item.go           # Item interface, Field struct
│   ├── issue.go          # Issue entity
│   ├── comment.go        # Comment entity
│   ├── label.go          # Label entity
│   ├── assignees.go      # Assignee entity
│   ├── milestone.go      # Milestone entity
│   ├── project.go        # Project entity
│   └── error.go          # Domain error types
├── github/               # GitHub API client and types
│   ├── client.go         # GraphQL client initialization, high-level API functions
│   ├── query.go          # Base query/mutation structures
│   ├── query_issue.go    # Issue-related GraphQL types and converters
│   ├── query_comment.go  # Comment-related GraphQL types
│   ├── query_label.go    # Label-related GraphQL types
│   ├── query_assignees.go # Assignee query types
│   ├── query_milestone.go # Milestone query types
│   ├── query_project.go  # Project query types
│   ├── query_repository.go # Repository query types
│   ├── mutation_issue.go # Issue mutation structures
│   └── mutation_comment.go # Comment mutation structures
├── ui/                   # Terminal UI components
│   ├── ui.go             # Main UI orchestrator, Primitive interface, grid layout
│   ├── select.go         # SelectUI for list selection with filtering
│   ├── view.go           # ViewUI for content preview with search
│   ├── filter.go         # FilterUI for query string input
│   ├── issues.go         # IssueUI specialized SelectUI with issue-specific actions
│   ├── comments.go       # CommentUI for issue comments
│   ├── labels.go         # LabelsUI for label selection
│   ├── assignees.go      # AssignableUI for assignee selection
│   ├── milestones.go     # MilestoneUI for milestone selection
│   ├── projects.go       # ProjectUI for project selection
│   ├── search.go         # SearchUI for text search in preview
│   └── select.go         # SelectUI base class (reused for multiple entity types)
├── utils/                # Utility functions
│   └── utils.go          # Edit function for external editor integration
├── go.mod                # Module declaration and dependencies
└── go.sum                # Dependency checksums
```

## Directory Purposes

**cmd/ght/:**
- Purpose: Application entry point and repository detection
- Contains: main() function, CLI argument parsing, Git remote URL parsing
- Key files: `main.go`

**config/:**
- Purpose: Centralized configuration management
- Contains: YAML configuration parsing, GitHub token validation, logging initialization
- Key files: `config.go`
- Public exports: `GitHub` (struct with Owner, Repo, Token), `App` (struct with File path)

**domain/:**
- Purpose: Define business entities and core abstractions
- Contains: Domain models (Issue, Comment, Label, etc.), Item interface for polymorphism
- Key files: `item.go` (interface definition), `issue.go` (primary entity type)
- Design: Minimal dependencies, no framework coupling, focuses on data representation

**github/:**
- Purpose: Abstract GitHub GraphQL API operations
- Contains: GraphQL client setup, query functions, mutation functions, strongly-typed query/mutation structures
- Key files: `client.go` (public API), query/mutation files (internal structures)
- Design: GraphQL queries/mutations are typed structures matching GitHub API schema; functions in client.go provide high-level interface

**ui/:**
- Purpose: Terminal user interface rendering and interaction
- Contains: tview-based components, event handling, layout management
- Key files: `ui.go` (orchestrator), `select.go` (reusable list UI), `view.go` (text preview), `issues.go` (issue-specific functionality)
- Design: SelectUI and ViewUI are reused across multiple domains; specialized UIs extend base behavior via function options

**utils/:**
- Purpose: Shared utility functions
- Contains: Editor launch helper
- Key files: `utils.go`
- Design: Minimal, no state, pure functions

## Key File Locations

**Entry Points:**
- `cmd/ght/main.go`: Application startup, argument parsing, component initialization
- `ui/ui.go`: UI Start() method (line 125) - initializes all components and event loop

**Configuration:**
- `config/config.go`: Reads from `~/.config/ght/config.yaml` on Linux/macOS, generates debug.log
- `config/config.go`: Variables GitHub and App are globally accessible

**Core Logic:**
- `ui/ui.go`: Main UI loop, component navigation (lines 51-79), event routing
- `ui/select.go`: SelectUI class (lines 37-51) - base for issues, comments, labels, etc.
- `ui/issues.go`: Issue-specific operations (create, open, close, edit)
- `github/client.go`: High-level API functions like GetIssues(), CreateIssue(), UpdateIssueComment()

**Testing:**
- No test files present; testing infrastructure not yet established

## Naming Conventions

**Files:**
- Lowercase with underscores: `main.go`, `config.go`, `select.go`
- Feature/entity-specific files named after entity: `issues.go`, `comments.go`, `labels.go`
- Prefixes for query/mutation files: `query_*.go`, `mutation_*.go`

**Directories:**
- Lowercase plural for packages: `cmd`, `config`, `domain`, `github`, `ui`, `utils`
- Package matching directory name

**Functions:**
- PascalCase for public functions: `NewClient()`, `GetIssues()`, `CreateIssue()`, `NewIssueUI()`
- camelCase for private functions: `getRepoInfo()`, `parseRemote()`, `yankIssueURLs()`

**Variables & Constants:**
- SCREAMING_SNAKE_CASE for constants: `UIKindIssue`, `UIKindComment`
- camelCase for variables: `client`, `ui`, `items`, `selected`
- Global UI singletons prefixed with operation: `IssueUI`, `CommentUI`, `IssueFilterUI`, `IssueViewUI`

**Types:**
- PascalCase for types: `Issue`, `SelectUI`, `ViewUI`, `GetListFunc`
- Struct fields PascalCase: `ID`, `Title`, `State`, `Author`, `URL`
- Private fields lowercase: `cursor`, `hasNext`, `items`, `selected`

## Where to Add New Code

**New Feature (e.g., Pull Requests):**
- Domain model: `domain/pullrequest.go` - define PR entity with Key() and Fields() methods
- GitHub types: `github/query_pr.go` - add GraphQL types, `github/mutation_pr.go` for mutations
- GitHub client: `github/client.go` - add GetPRs(), GetPR(), CreatePR(), etc. functions
- UI component: `ui/prs.go` - create NewPRUI() using NewSelectListUI() with PR-specific actions
- Main layout: `ui/ui.go` - add new PR UI to Start() primitives array and grid layout

**New Domain Entity:**
- File: `domain/newentity.go`
- Must implement Item interface (Key() and Fields() methods)
- Use tcell colors for Fields() display
- No logic beyond data representation

**New UI Component (using SelectUI pattern):**
- File: `ui/newui.go`
- Function: NewXxxUI() creates SelectUI via NewSelectListUI() with configuration option
- Set up getList callback returning domain.Item slice
- Set up capture callback for keyboard shortcuts
- Initialize global variable (e.g., `var MyUI *SelectUI`)

**New GitHub API Call:**
- Add query structure to appropriate `github/query_*.go` file
- Add public function to `github/client.go`
- Use `client.Query()` or `client.Mutate()` with context.Background()
- Return domain types or GitHub-internal types for conversion

**Shared Utilities:**
- File: `utils/utils.go` (add function)
- Keep functions pure and stateless
- Import only standard library and minimal dependencies

**New Keyboard Shortcut:**
- Global shortcuts: `ui/ui.go` - SetInputCapture() method (lines 162-182)
- Component-specific: Add to component's SetInputCapture() callback
- Follow naming: 'j'/'k' for navigation, 'o' for open, 'c' for close, 'e' for edit, 'n' for new

## Special Directories

**cmd/ght/:**
- Purpose: Package for executable binary
- Generated: No
- Committed: Yes
- Single main() function; rest of logic in packages

**.git/, .gitignore:**
- Purpose: Version control
- Generated: git system
- Committed: Yes

**go.mod, go.sum:**
- Purpose: Dependency management
- Generated: Go tooling
- Committed: Yes

**.planning/codebase/:**
- Purpose: Architecture and structure documentation
- Generated: Created during codebase mapping
- Committed: Yes (referenced by CI/analysis tools)

