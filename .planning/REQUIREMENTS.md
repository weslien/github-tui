# Requirements: github-tui (ght)

**Defined:** 2026-02-16
**Core Value:** Developers can interact with their GitHub repositories without leaving the terminal

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Foundation

- [ ] **FOUND-01**: App initializes REST API client alongside existing GraphQL client sharing the same PAT token
- [ ] **FOUND-02**: App tracks REST and GraphQL rate limits independently and warns user when approaching limits
- [ ] **FOUND-03**: App validates PAT token scopes on startup and displays clear message if required scopes are missing

### Actions

- [ ] **ACT-01**: User can view a list of workflow runs showing status, workflow name, branch, event, and timing
- [ ] **ACT-02**: User can filter workflow runs by status (success, failure, in_progress)
- [ ] **ACT-03**: User can filter workflow runs by workflow name
- [ ] **ACT-04**: User can select a workflow run and view its jobs
- [ ] **ACT-05**: User can view logs for a specific job with ANSI codes stripped
- [ ] **ACT-06**: User can navigate to a workflow run on GitHub (open in browser)
- [ ] **ACT-07**: Actions appear as a new top-level tab accessible via keybinding

### Projects V2

- [ ] **PROJ-01**: User can view a list of their Projects V2 (user and org-scoped)
- [ ] **PROJ-02**: User can select a project and view its items (issues, PRs, drafts)
- [ ] **PROJ-03**: User can see custom field values (status, text, date, number, single-select) displayed in the items table
- [ ] **PROJ-04**: User can open a project item in the browser

## v2 Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Actions Write Operations

- **ACT-W01**: User can re-run a failed workflow
- **ACT-W02**: User can re-run specific failed jobs
- **ACT-W03**: User can cancel a running workflow
- **ACT-W04**: User can dispatch a workflow with input parameters

### Actions Advanced

- **ACT-A01**: User can see live-updating log output for in-progress runs
- **ACT-A02**: User can filter runs by branch name

### Projects V2 Navigation

- **PROJ-N01**: User can navigate from a project item directly to the linked issue/PR in the existing TUI view
- **PROJ-N02**: User can filter project items by status field value
- **PROJ-N03**: User can sort project items by custom field values

### Projects V2 Advanced

- **PROJ-A01**: User can switch between project views (Table, Board)
- **PROJ-A02**: User can add/remove items from a project
- **PROJ-A03**: User can change item status from within the TUI

## Out of Scope

| Feature | Reason |
|---------|--------|
| Pull Request support | Deferred to separate milestone |
| Issue metadata management (assignees, labels) | Deferred to separate milestone |
| File tree browsing | Deferred to separate milestone |
| Local workflow execution | Different tool (`act`) handles this |
| Custom workflow YAML builder | IDE extensions are better suited |
| Full Projects V2 mutations (drag-drop, field editing) | Web UI is better for heavy editing |
| Projects V2 Board/Kanban layout | High complexity, table view sufficient for TUI |
| Real-time notifications | GitHub already provides these natively |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Phase ? | Pending |
| FOUND-02 | Phase ? | Pending |
| FOUND-03 | Phase ? | Pending |
| ACT-01 | Phase ? | Pending |
| ACT-02 | Phase ? | Pending |
| ACT-03 | Phase ? | Pending |
| ACT-04 | Phase ? | Pending |
| ACT-05 | Phase ? | Pending |
| ACT-06 | Phase ? | Pending |
| ACT-07 | Phase ? | Pending |
| PROJ-01 | Phase ? | Pending |
| PROJ-02 | Phase ? | Pending |
| PROJ-03 | Phase ? | Pending |
| PROJ-04 | Phase ? | Pending |

**Coverage:**
- v1 requirements: 14 total
- Mapped to phases: 0
- Unmapped: 14

---
*Requirements defined: 2026-02-16*
*Last updated: 2026-02-16 after initial definition*
