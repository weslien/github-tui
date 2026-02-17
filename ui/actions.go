package ui

import (
	"context"
	"fmt"
	"log"
	"strconv"

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
	WorkflowRunsUI    *SelectUI
	actionsStatusLine *tview.TextView

	actionsStatusFilter string
	actionsWorkflowID   int64
	actionsWorkflowName string
	actionsWorkflows    []*gogithub.Workflow

	// statusFilterCycle defines the order for cycling through status filters.
	statusFilterCycle = []string{"", "success", "failure", "in_progress", "queued"}
)

// NewActionsUI creates the Actions tab with a workflow runs list and status line.
func NewActionsUI() tview.Primitive {
	opt := func(ui *SelectUI) {
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

	WorkflowRunsUI = NewSelectListUI(UIKind("actions"), tcell.ColorDarkCyan, opt)

	actionsStatusLine = tview.NewTextView().
		SetDynamicColors(true).
		SetText("Actions | Status: all | Workflow: all | [s]tatus [w]orkflow [r]efresh")

	grid := tview.NewGrid().SetRows(1, 0).
		AddItem(actionsStatusLine, 0, 0, 1, 1, 0, 0, false).
		AddItem(WorkflowRunsUI, 1, 0, 1, 1, 0, 0, true)

	return grid
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
