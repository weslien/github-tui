package ui

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	gogithub "github.com/google/go-github/v68/github"
	"github.com/rivo/tview"
	"github.com/shurcooL/githubv4"
	"github.com/skanehira/ght/config"
	"github.com/skanehira/ght/domain"
	"github.com/skanehira/ght/github"
	"github.com/skanehira/ght/utils"
)

var (
	WorkflowRunsUI *SelectUI
	WorkflowJobsUI *SelectUI

	actionsStatusLine *tview.TextView
	actionsPages      *tview.Pages

	actionsStatusFilter string
	actionsWorkflowID   int64
	actionsWorkflowName string
	actionsWorkflows    []*gogithub.Workflow

	currentRunID   int64
	currentRunName string

	logCancelFunc context.CancelFunc

	// statusFilterCycle defines the order for cycling through status filters.
	statusFilterCycle = []string{"", "success", "failure", "in_progress", "queued"}
)

// NewActionsUI creates the Actions tab with a workflow runs list, jobs drill-down, and status line.
func NewActionsUI() tview.Primitive {
	// --- Workflow Runs SelectUI ---
	runOpt := func(ui *SelectUI) {
		ui.header = []string{
			"",
			"Status",
			"Workflow",
			"Branch",
			"Event",
			"Duration",
		}
		ui.hasHeader = true

		ui.getList = func(cursor *string) ([]domain.Item, *github.PageInfo) {
			ctx := context.Background()
			owner := config.GitHub.Owner
			repo := config.GitHub.Repo

			opts := &gogithub.ListWorkflowRunsOptions{
				ListOptions: gogithub.ListOptions{PerPage: 30},
			}

			if actionsStatusFilter != "" {
				opts.Status = actionsStatusFilter
			}

			// Convert cursor string to page number
			if cursor != nil {
				page, err := strconv.Atoi(*cursor)
				if err == nil {
					opts.ListOptions.Page = page
				}
			}

			var runs *gogithub.WorkflowRuns
			var resp *gogithub.Response
			var err error

			if actionsWorkflowID > 0 {
				runs, resp, err = github.ListWorkflowRunsByWorkflowID(ctx, owner, repo, actionsWorkflowID, opts)
			} else {
				runs, resp, err = github.ListWorkflowRuns(ctx, owner, repo, opts)
			}
			if err != nil {
				log.Println(err)
				return nil, nil
			}

			items := make([]domain.Item, len(runs.WorkflowRuns))
			for i, run := range runs.WorkflowRuns {
				items[i] = github.ConvertWorkflowRun(run)
			}

			// Convert REST pagination to PageInfo
			pageInfo := &github.PageInfo{}
			if resp != nil && resp.NextPage > 0 {
				pageInfo.HasNextPage = true
				pageInfo.EndCursor = githubv4.String(fmt.Sprintf("%d", resp.NextPage))
			}

			return items, pageInfo
		}

		ui.capture = func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyCtrlO:
				item := WorkflowRunsUI.GetSelect()
				if item != nil {
					run := item.(*domain.WorkflowRun)
					if err := utils.Open(run.HTMLURL); err != nil {
						log.Println(err)
					}
				}
			case tcell.KeyEnter:
				item := WorkflowRunsUI.GetSelect()
				if item != nil {
					run := item.(*domain.WorkflowRun)
					currentRunID = run.ID
					currentRunName = fmt.Sprintf("#%d - %s", run.RunNumber, run.Name)
					actionsPages.SwitchToPage("jobs-view")
					WorkflowJobsUI.focus()
					UI.app.SetFocus(WorkflowJobsUI)
					updateActionsStatusLine()
					go WorkflowJobsUI.GetList()
				}
				return nil
			}

			switch event.Rune() {
			case 'r':
				go WorkflowRunsUI.GetList()
			case 's':
				cycleStatusFilter()
			case 'w':
				showWorkflowSelector()
			}

			return event
		}
	}

	WorkflowRunsUI = NewSelectListUI(UIKind("actions"), tcell.ColorDarkCyan, runOpt)

	// --- Workflow Jobs SelectUI ---
	jobOpt := func(ui *SelectUI) {
		ui.header = []string{
			"",
			"Status",
			"Job",
			"Duration",
		}
		ui.hasHeader = true

		ui.getList = func(cursor *string) ([]domain.Item, *github.PageInfo) {
			ctx := context.Background()
			owner := config.GitHub.Owner
			repo := config.GitHub.Repo

			jobs, err := github.ListWorkflowJobs(ctx, owner, repo, currentRunID, nil)
			if err != nil {
				log.Println(err)
				return nil, nil
			}

			items := make([]domain.Item, len(jobs.Jobs))
			for i, job := range jobs.Jobs {
				items[i] = github.ConvertWorkflowJob(job)
			}

			// Jobs are not paginated (all returned in one response)
			pageInfo := &github.PageInfo{HasNextPage: false}
			return items, pageInfo
		}

		ui.capture = func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape:
				switchToRunsView()
				return nil
			case tcell.KeyCtrlO:
				item := WorkflowJobsUI.GetSelect()
				if item != nil {
					job := item.(*domain.WorkflowJob)
					if err := utils.Open(job.HTMLURL); err != nil {
						log.Println(err)
					}
				}
			case tcell.KeyEnter:
				item := WorkflowJobsUI.GetSelect()
				if item != nil {
					job := item.(*domain.WorkflowJob)
					fetchAndDisplayJobLog(job)
				}
				return nil
			}

			switch event.Rune() {
			case 'r':
				go WorkflowJobsUI.GetList()
			}

			return event
		}
	}

	WorkflowJobsUI = NewSelectListUI(UIKind("jobs"), tcell.ColorYellow, jobOpt)

	// --- Layout ---
	actionsStatusLine = tview.NewTextView().
		SetDynamicColors(true).
		SetText("Actions | Status: all | Workflow: all | [s]tatus [w]orkflow [r]efresh")

	actionsPages = tview.NewPages().
		AddAndSwitchToPage("runs-view", WorkflowRunsUI, true).
		AddPage("jobs-view", WorkflowJobsUI, true, false)

	grid := tview.NewGrid().SetRows(1, 0).
		AddItem(actionsStatusLine, 0, 0, 1, 1, 0, 0, false).
		AddItem(actionsPages, 1, 0, 1, 1, 0, 0, true)

	return grid
}

// switchToRunsView returns from the jobs view to the runs view.
func switchToRunsView() {
	// Cancel any in-progress log download
	if logCancelFunc != nil {
		logCancelFunc()
		logCancelFunc = nil
	}
	actionsPages.SwitchToPage("runs-view")
	WorkflowRunsUI.focus()
	UI.app.SetFocus(WorkflowRunsUI)
	updateActionsStatusLine()
}

// fetchAndDisplayJobLog fetches a job's log and displays it in full-screen preview.
func fetchAndDisplayJobLog(job *domain.WorkflowJob) {
	// Cancel any previous log download
	if logCancelFunc != nil {
		logCancelFunc()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	logCancelFunc = cancel

	actionsStatusLine.SetText(fmt.Sprintf("Loading log for: %s...", job.Name))

	go func() {
		owner := config.GitHub.Owner
		repo := config.GitHub.Repo

		logContent, err := github.GetWorkflowJobLog(ctx, owner, repo, job.ID)
		if err != nil {
			ctxErr := ctx.Err()
			// If context was cancelled by user navigation (not timeout), silently return
			if ctxErr == context.Canceled {
				return
			}
			UI.updater <- func() {
				var msg string
				if isNotFoundError(err) {
					msg = "Log not available. The job may still be running or logs may have expired."
				} else if ctxErr == context.DeadlineExceeded {
					msg = "Log download timed out. The log may be very large. Press Ctrl+O to view in browser."
				} else {
					msg = err.Error()
				}
				UI.Message(msg, func() {
					UI.app.SetFocus(WorkflowJobsUI)
				})
			}
			return
		}

		// Check if log was truncated (10MB limit)
		if int64(len(logContent)) >= 10*1024*1024-1024 {
			logContent += "\n\n--- Log truncated at 10MB. Press Ctrl+O on the job to view full log in browser. ---"
		}

		UI.updater <- func() {
			actionsStatusLine.SetText(fmt.Sprintf("Log: %s | 'o' close | '/' search", job.Name))
			UI.FullScreenPreview(logContent, func() {
				UI.app.SetFocus(WorkflowJobsUI)
			})
		}
	}()
}

// cycleStatusFilter advances to the next status filter and refreshes the list.
func cycleStatusFilter() {
	// Find current position in cycle
	current := 0
	for i, s := range statusFilterCycle {
		if s == actionsStatusFilter {
			current = i
			break
		}
	}
	// Advance to next
	next := (current + 1) % len(statusFilterCycle)
	actionsStatusFilter = statusFilterCycle[next]

	updateActionsStatusLine()
	go WorkflowRunsUI.GetList()
}

// showWorkflowSelector opens a modal list of workflows for the user to select.
func showWorkflowSelector() {
	go func() {
		// Fetch workflows if not cached
		if actionsWorkflows == nil {
			ctx := context.Background()
			owner := config.GitHub.Owner
			repo := config.GitHub.Repo

			workflows, err := github.ListWorkflows(ctx, owner, repo)
			if err != nil {
				log.Println(err)
				return
			}
			actionsWorkflows = workflows
		}

		UI.app.QueueUpdateDraw(func() {
			list := tview.NewList().ShowSecondaryText(false)
			list.SetBorder(true).SetTitle("Select Workflow").SetTitleAlign(tview.AlignLeft)

			// Add "All workflows" option first
			list.AddItem("All workflows", "", 0, nil)

			for _, w := range actionsWorkflows {
				list.AddItem(w.GetName(), "", 0, nil)
			}

			list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
				if index == 0 {
					// "All workflows" selected
					actionsWorkflowID = 0
					actionsWorkflowName = ""
				} else {
					w := actionsWorkflows[index-1]
					actionsWorkflowID = w.GetID()
					actionsWorkflowName = w.GetName()
				}

				UI.pages.RemovePage("workflow-selector")
				UI.app.SetFocus(WorkflowRunsUI)
				updateActionsStatusLine()
				go WorkflowRunsUI.GetList()
			})

			list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEsc {
					UI.pages.RemovePage("workflow-selector")
					UI.app.SetFocus(WorkflowRunsUI)
					return nil
				}
				return event
			})

			modal := UI.Modal(list, 60, 20)
			UI.pages.AddAndSwitchToPage("workflow-selector", modal, true).ShowPage("actions")
		})
	}()
}

// updateActionsStatusLine refreshes the status line text with current filter state.
func updateActionsStatusLine() {
	// Check if we're in jobs view
	name, _ := actionsPages.GetFrontPage()
	if name == "jobs-view" {
		actionsStatusLine.SetText(fmt.Sprintf(
			"Run: %s | Esc: back | Ctrl+O: browser | [r]efresh",
			currentRunName,
		))
		return
	}

	statusText := "all"
	if actionsStatusFilter != "" {
		statusText = actionsStatusFilter
	}
	workflowText := "all"
	if actionsWorkflowName != "" {
		workflowText = actionsWorkflowName
	}
	actionsStatusLine.SetText(fmt.Sprintf(
		"Actions | Status: %s | Workflow: %s | [s]tatus [w]orkflow [r]efresh",
		statusText, workflowText,
	))
}

// isNotFoundError checks if an error represents a 404 Not Found response.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found")
}
