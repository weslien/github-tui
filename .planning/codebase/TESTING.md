# Testing Patterns

**Analysis Date:** 2026-02-16

## Test Framework

**Runner:**
- `go test` - Standard Go test framework
- Config: None (uses default Go test configuration)

**Assertion Library:**
- Standard `testing` package (no assertion library used)
- Manual assertion patterns: `if condition != expected { t.Fatalf(...) }`

**Run Commands:**
```bash
go test ./...                    # Run all tests
go test -v ./...                # Run with verbose output
go test -race ./...             # Run with race detector
go test -cover ./...            # Run with coverage
go test -coverprofile=cov.out ./...  # Generate coverage profile
```

## Test File Organization

**Location:**
- No test files found in project
- No `*_test.go` files present

**Naming:**
- Convention: `{module}_test.go` (Go standard, not used here)

**Structure:**
- Not applicable - no tests implemented

## Test Structure

**Suite Organization:**
- Not implemented - no test files found

**Patterns:**
- Setup: Not applicable
- Teardown: Not applicable
- Assertion: Not applicable

## Mocking

**Framework:**
- No mocking framework detected
- No test doubles or mocks found

**Patterns:**
- Not applicable

**What to Mock:**
- GitHub API client (`github/client.go`) - would benefit from mock implementation
- TUI components (`ui/` package) - event handling and rendering difficult to test

**What NOT to Mock:**
- Domain models (`domain/` package) - simple data structures
- Utility functions (`utils/` package) - pure functions or system operations

## Fixtures and Factories

**Test Data:**
- Not found in codebase

**Location:**
- Not applicable

## Coverage

**Requirements:**
- No coverage targets enforced
- No coverage configuration found

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=cov.out ./...
go tool cover -html=cov.out
```

## Test Types

**Unit Tests:**
- Not implemented
- Should test: domain models, utility functions, GraphQL query/mutation building

**Integration Tests:**
- Not implemented
- Would require GitHub API testing (likely via VCR or similar cassette recording)

**E2E Tests:**
- Not implemented
- Framework not used

## Common Patterns

**Async Testing:**
- Not applicable - no tests present
- Note: Codebase uses goroutines extensively in UI operations (`go func() { ... }()`)

**Error Testing:**
- Not applicable - no tests present
- Note: Error handling should be tested for:
  - Config file not found: `config/config.go` line 47
  - GitHub API failures: throughout `github/` package
  - Invalid remote URLs: `cmd/ght/main.go` parseRemote function

## Testing Gaps and Recommendations

**Critical Untested Areas:**
- `cmd/ght/main.go` - Remote URL parsing logic (`parseRemote()`, `getOwnerRepo()`)
  - Should test SSH, HTTPS, Git protocols
  - Edge cases: invalid URLs, missing owner/repo

- `config/config.go` - Configuration loading
  - Missing config file handling
  - Invalid YAML parsing
  - Missing token validation

- `github/` - All GraphQL operations
  - Query building and execution
  - Error handling on API failures
  - Pagination logic (cursor handling)

- `ui/select.go` - SelectUI filtering and state management
  - Search word filtering in `UpdateView()`
  - Selection toggling in `toggleSelected()`
  - Pagination in `FetchList()`

- `domain/` - Item implementations
  - Key() method uniqueness
  - Fields() method formatting
  - Domain model transformations

**Suggested Test Coverage Order:**
1. Utility functions: `utils/strings.go` Replace function (highest ROI, pure functions)
2. Domain models: `domain/issue.go`, `domain/comment.go` Fields() output
3. Config loading: `config/config.go` error cases
4. URL parsing: `cmd/ght/main.go` parseRemote() variants
5. GitHub client: `github/client.go` query/mutation execution

**Testing Infrastructure Needed:**
- Test data fixtures for domain models
- GitHub API mock/stub for integration testing
- Table-driven test patterns for URL parsing validation
- Goroutine synchronization helpers for concurrent operation testing

---

*Testing analysis: 2026-02-16*
