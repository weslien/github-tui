# Coding Conventions

**Analysis Date:** 2026-02-16

## Naming Patterns

**Files:**
- Lowercase with underscores for compound names: `query_issue.go`, `mutation_comment.go`
- Files match their primary package: `config.go` in `config/` package
- UI component files named by feature: `issues.go`, `comments.go`, `labels.go`
- Test files not found (no `_test.go` convention used)

**Functions:**
- CamelCase with capital first letter for exported functions: `NewClient()`, `GetIssues()`, `UpdateIssue()`
- Lowercase with capital first letter for package functions: `NewSelectListUI()`, `NewFilterUI()`
- Constructor functions start with `New`: `NewIssueUI()`, `NewClient()`
- Query/mutation functions start with action verb: `GetIssues()`, `CreateIssue()`, `ReopenIssue()`, `CloseIssue()`
- Helper functions use descriptive verbs: `parseRemote()`, `getOwnerRepo()`, `yankIssueURLs()`, `openBrowser()`

**Variables:**
- CamelCase for local and exported variables
- Global package variables capitalized: `UI *ui`, `IssueUI *SelectUI`, `IssueFilterUI *FilterUI`
- Private package-level variables lowercase: `client *githubv4.Client`
- Constants UPPERCASE with underscores: `UIKindIssue`, `UIKindAssignee`, `UIKindComment`, `unselected`, `selected`
- Map variables follow pattern: `userMap`, `labelMap`, `projectMap`, `milestoneID`, `issueBody`

**Types:**
- Struct names PascalCase: `Issue`, `Comment`, `Label`, `FilterUI`, `SelectUI`
- Private structs lowercase: `ui`, `github`, `app`
- Interface names PascalCase: `Item`, `Primitive`
- Type aliases PascalCase: `UIKind`, `GetListFunc`, `CaptureFunc`, `SetSelectUIOpt`

## Code Style

**Formatting:**
- Standard Go formatting (implied gofmt usage)
- Indentation: tabs (Go standard)
- Line breaks after imports organized by groups
- Function declarations formatted on single line when possible

**Linting:**
- No explicit linter config files found
- Code follows Go conventions and idioms
- Error handling uses standard `if err != nil` pattern

## Import Organization

**Order:**
1. Standard library imports (`io`, `os`, `context`, `fmt`, etc.)
2. Third-party packages (`github.com/...`)
3. Local package imports (relative to module: `github.com/skanehira/ght/...`)

**Example from `/Users/gustav/src/github-tui/ui/issues.go`:**
```go
import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shurcooL/githubv4"
	"github.com/skanehira/ght/config"
	"github.com/skanehira/ght/domain"
	"github.com/skanehira/ght/github"
	"github.com/skanehira/ght/utils"
)
```

**Path Aliases:**
- No path aliases used in project
- Full import paths from module root: `github.com/skanehira/ght/...`

## Error Handling

**Patterns:**
- Errors returned directly from functions: `GetIssues()` returns `(*Issues, error)`
- Error checking with `if err != nil` followed by immediate return or log
- Logging on error: `log.Println(err)` for non-fatal errors, `log.Fatal(err)` or `log.Fatalf()` for initialization errors
- Custom error variables defined in `domain/error.go`:
  ```go
  var (
      ErrCommentBodyIsEmpty = errors.New("comment body is empty")
      ErrNotFoundComment    = errors.New("not found comment")
      ErrNotFoundIssue      = errors.New("not found issue")
  )
  ```
- No error wrapping with context (no `fmt.Errorf("... %w", err)` pattern)
- UI errors displayed via modal: `UI.Message(err.Error(), focusFunc)` in `/Users/gustav/src/github-tui/ui/issues.go`

**Example from `/Users/gustav/src/github-tui/ui/issues.go`:**
```go
v := map[string]interface{}{
    "owner": githubv4.String(owner),
    "name":  githubv4.String(name),
}
resp, err := github.GetRepo(v)
if err != nil {
    UI.Message(err.Error(), func() {
        UI.app.SetFocus(IssueUI)
    })
    return
}
```

## Logging

**Framework:** `log` standard library package

**Patterns:**
- `log.Fatal(err)` - for startup/initialization failures (config loading, client setup)
- `log.Fatalf(msg, err)` - for formatted fatal messages with context
- `log.Println(err)` - for runtime errors in goroutines and UI operations
- Logging configured in `config/config.go` to write to both stderr and file (`$CONFIG_DIR/ght/debug.log`)

**Example from `/Users/gustav/src/github-tui/config/config.go`:**
```go
logFile := filepath.Join(configDir, "ght", "debug.log")
output, err := os.Create(logFile)
log.SetOutput(io.MultiWriter(output, os.Stderr))
```

## Comments

**When to Comment:**
- Rarely used in codebase
- Generally implicit from function names and structure
- Commented-out code preserved for reference: `// log.Println(err)` in `ui/labels.go`
- URL references in comments: `// Replace is customized for this project` with GitHub link in `utils/strings.go`

**JSDoc/TSDoc:**
- Not used (Go convention)
- No function documentation comments found

## Function Design

**Size:** Functions typically 20-60 lines, with UI handling functions being longer (50-200+ lines for complex forms)

**Parameters:**
- Functional options pattern used: `SetSelectUIOpt func(ui *SelectUI)` in `/Users/gustav/src/github-tui/ui/select.go`
- Example:
  ```go
  func NewSelectListUI(uiKind UIKind, boxColor tcell.Color, setOpt SetSelectUIOpt) *SelectUI {
      ui := &SelectUI{...}
      setOpt(ui)  // Apply options
      return ui
  }
  ```
- Callback functions as parameters: `focusFunc func()`, `searchFunc func(text string)`
- Maps for variable arguments: `variables map[string]interface{}`

**Return Values:**
- Functions return either result or error, not both in most cases
- Some functions return result with additional data: `(*Issues, *github.PageInfo)`
- Single return value for void operations with error: `error`

## Module Design

**Exports:**
- Exported symbols use PascalCase: `NewClient`, `GetIssues`, `UpdateIssue`
- Package-level exported variables: `GitHub` (config struct), `App` (app struct), `UI` (global ui instance)
- Exported interfaces: `Item`, `Primitive`

**Barrel Files:**
- Not used in project (no index files to re-export)
- Each package handles its own exports

**Package Structure:**
- `config/` - Configuration loading and management
- `domain/` - Domain models and interfaces (Item, Issue, Comment, Label, etc.)
- `github/` - GitHub API client and GraphQL queries/mutations
- `ui/` - TUI components and rendering
- `utils/` - Utility functions (editing, string operations)
- `cmd/ght/` - Application entry point

---

*Convention analysis: 2026-02-16*
