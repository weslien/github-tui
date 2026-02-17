package domain

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// WorkflowJob represents a GitHub Actions workflow job.
type WorkflowJob struct {
	ID         int64
	Name       string
	Status     string
	Conclusion string
	Duration   string
	HTMLURL    string
	RunID      int64
}

func (j *WorkflowJob) Key() string {
	return fmt.Sprintf("%d", j.ID)
}

func (j *WorkflowJob) Fields() []Field {
	statusText, color := statusDisplay(j.Status, j.Conclusion)

	return []Field{
		{Text: statusText, Color: color},
		{Text: j.Name, Color: tcell.ColorWhite},
		{Text: j.Duration, Color: tcell.ColorWhite},
	}
}
