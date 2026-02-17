package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	gogithub "github.com/google/go-github/v68/github"

	"github.com/skanehira/ght/domain"
)

var (
	// ansiRegex matches ANSI escape sequences (colors, formatting, cursor movement).
	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

	// timestampRegex matches GitHub Actions log timestamp prefixes like
	// "2024-01-15T10:30:45.1234567Z ".
	timestampRegex = regexp.MustCompile(`(?m)^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z `)

	// maxLogSize caps log downloads at 10MB to prevent OOM.
	maxLogSize int64 = 10 * 1024 * 1024
)

// CleanLog strips ANSI escape sequences and GitHub timestamp prefixes from raw log output.
func CleanLog(raw string) string {
	cleaned := ansiRegex.ReplaceAllString(raw, "")
	cleaned = timestampRegex.ReplaceAllString(cleaned, "")
	return cleaned
}

// ConvertWorkflowRun converts a go-github WorkflowRun to a domain WorkflowRun.
func ConvertWorkflowRun(run *gogithub.WorkflowRun) *domain.WorkflowRun {
	var dur string
	if run.RunStartedAt != nil {
		d := run.GetUpdatedAt().Time.Sub(run.RunStartedAt.Time)
		dur = formatDuration(d)
	}

	return &domain.WorkflowRun{
		ID:         run.GetID(),
		Name:       run.GetName(),
		Title:      run.GetDisplayTitle(),
		Status:     run.GetStatus(),
		Conclusion: run.GetConclusion(),
		HeadBranch: run.GetHeadBranch(),
		Event:      run.GetEvent(),
		RunNumber:  run.GetRunNumber(),
		Duration:   dur,
		CreatedAt:  formatTime(run.GetCreatedAt().Time),
		HTMLURL:    run.GetHTMLURL(),
	}
}

// ConvertWorkflowJob converts a go-github WorkflowJob to a domain WorkflowJob.
func ConvertWorkflowJob(job *gogithub.WorkflowJob) *domain.WorkflowJob {
	var dur string
	if job.StartedAt != nil && job.CompletedAt != nil {
		d := job.CompletedAt.Time.Sub(job.StartedAt.Time)
		dur = formatDuration(d)
	}

	return &domain.WorkflowJob{
		ID:         job.GetID(),
		Name:       job.GetName(),
		Status:     job.GetStatus(),
		Conclusion: job.GetConclusion(),
		Duration:   dur,
		HTMLURL:    job.GetHTMLURL(),
		RunID:      job.GetRunID(),
	}
}

// formatDuration formats a time.Duration as a human-readable string:
// <60s -> "Xs", <1h -> "Xm Ys", >=1h -> "Xh Ym".
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}

// formatTime formats a time.Time for display: "15:04" for today, "Jan 02 15:04" otherwise.
func formatTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("Jan 02 15:04")
}

// ListWorkflowRuns lists workflow runs for a repository.
func ListWorkflowRuns(ctx context.Context, owner, repo string, opts *gogithub.ListWorkflowRunsOptions) (*gogithub.WorkflowRuns, *gogithub.Response, error) {
	client := GetRESTClient()
	if client == nil {
		return nil, nil, fmt.Errorf("REST client not initialized")
	}

	runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}
	return runs, resp, nil
}

// ListWorkflowRunsByWorkflowID lists workflow runs filtered by a specific workflow ID.
func ListWorkflowRunsByWorkflowID(ctx context.Context, owner, repo string, workflowID int64, opts *gogithub.ListWorkflowRunsOptions) (*gogithub.WorkflowRuns, *gogithub.Response, error) {
	client := GetRESTClient()
	if client == nil {
		return nil, nil, fmt.Errorf("REST client not initialized")
	}

	runs, resp, err := client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflow runs for workflow %d: %w", workflowID, err)
	}
	return runs, resp, nil
}

// ListWorkflows lists all workflows for a repository with full pagination.
func ListWorkflows(ctx context.Context, owner, repo string) ([]*gogithub.Workflow, error) {
	client := GetRESTClient()
	if client == nil {
		return nil, fmt.Errorf("REST client not initialized")
	}

	var allWorkflows []*gogithub.Workflow
	opts := &gogithub.ListOptions{PerPage: 100}

	for {
		workflows, resp, err := client.Actions.ListWorkflows(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list workflows: %w", err)
		}
		allWorkflows = append(allWorkflows, workflows.Workflows...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allWorkflows, nil
}

// ListWorkflowJobs lists jobs for a specific workflow run.
func ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64, opts *gogithub.ListWorkflowJobsOptions) (*gogithub.Jobs, error) {
	client := GetRESTClient()
	if client == nil {
		return nil, fmt.Errorf("REST client not initialized")
	}

	jobs, _, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow jobs for run %d: %w", runID, err)
	}
	return jobs, nil
}

// GetWorkflowJobLog fetches the log content for a specific workflow job.
// It follows the redirect URL returned by the API and downloads the log content,
// capped at maxLogSize (10MB) to prevent OOM.
func GetWorkflowJobLog(ctx context.Context, owner, repo string, jobID int64) (string, error) {
	client := GetRESTClient()
	if client == nil {
		return "", fmt.Errorf("REST client not initialized")
	}

	logURL, _, err := client.Actions.GetWorkflowJobLogs(ctx, owner, repo, jobID, 4)
	if err != nil {
		return "", fmt.Errorf("failed to get log URL for job %d: %w", jobID, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create log request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("log download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLogSize))
	if err != nil {
		return "", fmt.Errorf("failed to read log body: %w", err)
	}

	return CleanLog(string(body)), nil
}
