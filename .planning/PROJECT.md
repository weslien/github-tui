# github-tui (ght)

## What This Is

A terminal UI client for GitHub, built in Go with tview. It lets developers browse and manage GitHub issues, comments, and repository data directly from the terminal using vim-style keybindings. Currently supports full issue and comment lifecycle; expanding to cover GitHub Actions and Projects V2.

## Core Value

Developers can interact with their GitHub repositories without leaving the terminal — fast, keyboard-driven, and distraction-free.

## Requirements

### Validated

<!-- Shipped and confirmed working. -->

- ✓ Issue listing with pagination and filtering — existing
- ✓ Issue create, close, open, edit, preview — existing
- ✓ Issue open in browser — existing
- ✓ Issue comment list, preview, add, edit, delete, quote reply — existing
- ✓ Vim-style navigation (j/k/g/G, Ctrl+N/P) — existing
- ✓ GitHub GraphQL API integration with PAT auth — existing
- ✓ YAML-based configuration with platform-specific paths — existing
- ✓ External editor integration ($EDITOR) — existing
- ✓ Filter-based search for issues — existing
- ✓ Markdown preview in terminal — existing

### Active

<!-- Current milestone scope. -->

- [ ] GitHub Actions: list workflow runs
- [ ] GitHub Actions: view workflow run logs
- [ ] GitHub Actions: re-run failed/specific jobs
- [ ] GitHub Actions: new top-level tab in TUI
- [ ] Projects V2: list user/org projects
- [ ] Projects V2: view project items (issues, PRs, drafts)
- [ ] Projects V2: navigate to linked issues/PRs from project view
- [ ] Projects V2: open project items in browser

### Out of Scope

- Pull Request support — deferred to future milestone
- Issue metadata management (assignees, labels, projects, milestone) — deferred to future milestone
- File tree browsing — deferred to future milestone
- Custom keybinding configuration — deferred to future milestone
- Custom editor configuration — deferred to future milestone
- Projects V2 mutations (add/remove/move items, change status) — read-only first, mutations in future milestone

## Context

- Brownfield Go TUI project originally by skanehira, using tview + tcell for rendering
- GitHub API accessed via shurcooL/githubv4 (GraphQL v4 API)
- Architecture: layered MVC — `ui/` (presentation), `github/` (API client), `domain/` (entities), `config/` (settings)
- UI pattern: SelectUI (tables) + ViewUI (markdown preview) + FilterUI (search), connected via channel-based updater
- All existing UI follows the same pattern: list → select → preview/action
- Go version recently updated; dependencies may need updating too
- Projects V2 uses a different GraphQL schema than classic projects — requires new queries
- GitHub Actions API has both REST and GraphQL endpoints; logs typically require REST API

## Constraints

- **Tech stack**: Go + tview/tcell — must stay consistent with existing codebase
- **API**: GitHub GraphQL v4 preferred; REST only where GraphQL doesn't cover (e.g., Actions logs)
- **Auth**: Single PAT token from config.yaml — no OAuth flow changes needed
- **UX pattern**: New features should follow the existing SelectUI/ViewUI/FilterUI pattern

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| GitHub Actions as new top-level tab | Keeps it discoverable, separate from issues | — Pending |
| Projects V2 (not classic) | Classic projects deprecated by GitHub | — Pending |
| Projects read-only for v1 | Ship navigation first, mutations later | — Pending |
| REST API for Actions logs | GraphQL doesn't expose log content | — Pending |

---
*Last updated: 2026-02-16 after initialization*
