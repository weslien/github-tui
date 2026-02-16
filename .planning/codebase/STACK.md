# Technology Stack

**Analysis Date:** 2026-02-16

## Languages

**Primary:**
- Go 1.22+ - Entire application, TUI client for GitHub

## Runtime

**Environment:**
- Go 1.22 (minimum, specified in go.mod)

**Package Manager:**
- Go modules (go.mod/go.sum)
- Lockfile: Present (go.sum)

## Frameworks

**Core TUI:**
- github.com/rivo/tview v0.0.0-20210312174852-ae9464cc3598 - Terminal UI framework for building interactive TUI components
- github.com/gdamore/tcell/v2 v2.2.0 - Low-level terminal abstraction, terminal input/output handling

**GitHub API Client:**
- github.com/shurcooL/githubv4 v0.0.0-20200928013246-d292edc3691b - GraphQL client for GitHub API v4
- golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 - OAuth2 authentication client

**Utilities:**
- github.com/goccy/go-yaml v1.8.3 - YAML parsing for configuration files
- github.com/atotto/clipboard v0.1.2 - Clipboard operations (copy to/from system clipboard)

## Key Dependencies

**Critical:**
- github.com/shurcooL/githubv4 - Provides GraphQL client for all GitHub API interactions (queries, mutations)
- golang.org/x/oauth2 - Enables token-based authentication with GitHub API
- github.com/rivo/tview - Powers all terminal UI rendering and components
- github.com/gdamore/tcell/v2 - Terminal input/output control, keyboard event handling

**Supporting:**
- github.com/shurcooL/graphql v0.0.0-20200928012149-18c5c3165e3a - Low-level GraphQL support (indirect dependency)
- golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 - Synchronization primitives
- golang.org/x/net v0.0.0-20201026091529-146b70c837a4 - Network utilities
- golang.org/x/sys, golang.org/x/term - System and terminal control

**Color/Display:**
- github.com/fatih/color v1.7.0 - ANSI color output (indirect)
- github.com/lucasb-eyer/go-colorful v1.2.0 - Color manipulation (indirect)
- github.com/mattn/go-colorable v0.1.6 - Color terminal support (indirect)
- github.com/mattn/go-runewidth v0.0.10 - Unicode rune width calculations (indirect)
- github.com/rivo/uniseg v0.2.0 - Text segmentation (indirect)

## Configuration

**Environment:**
- Configuration via `config.yaml` file
- Location platform-specific:
  - Windows: `%AppData%\ght\config.yaml`
  - macOS: `$HOME/Library/Application Support/ght/config.yaml`
  - Linux/Unix: `$HOME/.config/ght/config.yaml`
- Key configuration:
  - `github.token` - Required GitHub Personal Access Token

**Build:**
- Compiled from source using `go build` or `go install ./cmd/ght`
- Entry point: `cmd/ght/main.go`

## Platform Requirements

**Development:**
- Go 1.22 or higher installed
- Git (for extracting repository owner/repo information)
- EDITOR environment variable (defaults to vim if unset)

**Production:**
- Compiled binary requires no runtime dependencies beyond system libraries
- POSIX-compliant terminal (Linux, macOS, Unix)
- Windows support via WSL or Windows terminal emulator (tcell/tview compatible)

---

*Stack analysis: 2026-02-16*
