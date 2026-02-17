package domain

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// WorkflowRun represents a GitHub Actions workflow run.
type WorkflowRun struct {
	ID         int64
	Name       string
	Title      string
	Status     string
	Conclusion string
	HeadBranch string
	Event      string
	RunNumber  int
	CreatedAt  string
	Duration   string
	HTMLURL    string
	RunID      int64
}

func (w *WorkflowRun) Key() string {
	return fmt.Sprintf("%d", w.ID)
}

func (w *WorkflowRun) Fields() []Field {
	statusText, color := statusDisplay(w.Status, w.Conclusion)

	return []Field{
		{Text: statusText, Color: color},
		{Text: w.Name, Color: tcell.ColorWhite},
		{Text: w.HeadBranch, Color: tcell.ColorBlue},
		{Text: w.Event, Color: tcell.ColorYellow},
		{Text: w.Duration, Color: tcell.ColorWhite},
	}
}

// statusDisplay returns the display text and color for a workflow status/conclusion pair.
// For completed runs, the conclusion text is displayed; for non-completed runs, the status text.
func statusDisplay(status, conclusion string) (string, tcell.Color) {
	switch status {
	case "completed":
		switch conclusion {
		case "success":
			return conclusion, tcell.ColorGreen
		case "failure":
			return conclusion, tcell.ColorRed
		default:
			// cancelled, skipped, etc.
			return conclusion, tcell.ColorGray
		}
	case "in_progress":
		return status, tcell.ColorYellow
	default:
		// queued, waiting, etc.
		return status, tcell.ColorGray
	}
}
