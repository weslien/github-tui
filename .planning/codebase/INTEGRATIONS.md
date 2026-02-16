# External Integrations

**Analysis Date:** 2026-02-16

## APIs & External Services

**GitHub:**
- GitHub GraphQL API v4 - Primary service for all repository, issue, comment, label, milestone, project, and assignee operations
  - SDK/Client: `github.com/shurcooL/githubv4` (GraphQL client)
  - Auth: `GITHUB_TOKEN` or configured in `config.yaml` under `github.token`
  - Implementation: `github/client.go` creates oauth2 HTTP client authenticated with personal access token

## Data Storage

**Databases:**
- None - Stateless TUI application (no persistent database)
- All data fetched on-demand from GitHub API

**File Storage:**
- Local filesystem only
- Configuration files stored in platform-specific config directory
- Temporary files: Created in system temp directory for editor content (see `utils/utils.go`)

**Caching:**
- None detected - Each UI operation fetches fresh data from GitHub API

## Authentication & Identity

**Auth Provider:**
- GitHub Personal Access Token (custom token-based)
- Implementation: `config/config.go` reads token from `config.yaml`
- Flow:
  1. Token stored in `config.yaml` under `github.token`
  2. Config initialized in `cmd/ght/main.go` via `config.Init()`
  3. Token passed to `github.NewClient(config.GitHub.Token)`
  4. OAuth2 static token source wraps token: `oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})`
  5. HTTP client created with token authentication
  6. GraphQL client uses authenticated HTTP client for all API calls

## Monitoring & Observability

**Error Tracking:**
- None (no external service integration)

**Logs:**
- Standard Go logging to stderr and local debug.log file
- Log file location: `{UserConfigDir}/ght/debug.log`
- Log output setup: `config/config.go` lines 35-41 configures multi-writer to both stderr and log file

## CI/CD & Deployment

**Hosting:**
- No hosting required - Distributed as compiled binary
- Installation: `go install ./cmd/ght`

**CI Pipeline:**
- None detected in codebase

## Environment Configuration

**Required env vars:**
- `EDITOR` (optional) - Text editor for editing issue/comment bodies; defaults to `vim` if unset
- GitHub token must be in `config.yaml` (not as env var)

**Secrets location:**
- Configuration file: `config.yaml` in platform-specific config directory
- File contains `github.token` field with Personal Access Token
- File is user-owned and should have restricted permissions

## Webhooks & Callbacks

**Incoming:**
- None - TUI client pulls data from GitHub API

**Outgoing:**
- None - TUI client does not send webhooks

## GitHub API Integration Details

**Query Operations:** (via `github/query.go`)
- `GetIssues()` - Fetch issues with pagination
- `GetIssue()` - Fetch single issue details
- `GetRepos()` - Fetch user's repositories
- `GetRepo()` - Fetch repository details
- `GetRepoLabels()` - Fetch repository labels with pagination
- `GetRepoMilestones()` - Fetch repository milestones with pagination
- `GetRepoProjects()` - Fetch repository projects with pagination
- `GetRepoAssignableUsers()` - Fetch users that can be assigned
- `GetIssueTemplates()` - Fetch issue templates

**Mutation Operations:** (via `github/mutation_*.go`)
- `CreateIssue()` - Create new issue
- `CloseIssue()` - Close issue by ID
- `ReopenIssue()` - Reopen closed issue
- `UpdateIssue()` - Update issue properties
- `AddIssueComment()` - Add comment to issue
- `UpdateIssueComment()` - Update existing comment
- `DeleteIssueComment()` - Delete comment by ID

---

*Integration audit: 2026-02-16*
