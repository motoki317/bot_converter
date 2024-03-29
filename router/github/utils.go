package github

import (
	"strings"
	"time"

	"github.com/go-playground/webhooks/v6/github"
)

func formatTime(from string, format string) (string, error) {
	layouts := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
	}

	var err error
	for _, layout := range layouts {
		var t time.Time
		t, err = time.Parse(layout, from)
		if err == nil {
			return t.Format(format), nil
		}
	}
	return "", err
}

func getAssigneeNames(payload interface{}) (ret string) {
	var assignees []*github.Assignee
	switch payload.(type) {
	case github.IssuesPayload:
		payload := payload.(github.IssuesPayload)
		assignees = payload.Issue.Assignees
	case github.IssueCommentPayload:
		payload := payload.(github.IssueCommentPayload)
		assignees = payload.Issue.Assignees
	case github.PullRequestPayload:
		payload := payload.(github.PullRequestPayload)
		assignees = payload.PullRequest.Assignees
	case github.PullRequestReviewPayload:
		payload := payload.(github.PullRequestReviewPayload)
		// not a slice of pointers
		reviewAssignees := payload.PullRequest.Assignees
		assignees = make([]*github.Assignee, 0, len(reviewAssignees))
		for _, assignee := range reviewAssignees {
			assignee := assignee
			assignees = append(assignees, &assignee)
		}
	case github.PullRequestReviewCommentPayload:
		payload := payload.(github.PullRequestReviewCommentPayload)
		assignees = payload.PullRequest.Assignees
	default:
		return
	}

	if assignees == nil {
		return
	}

	formatted := make([]string, 0, len(assignees))
	for _, assignee := range assignees {
		formatted = append(formatted, "`"+assignee.Login+"`")
	}
	return strings.Join(formatted, ", ")
}

func getRequestedReviewers(payload github.PullRequestPayload) string {
	formatted := make([]string, 0, len(payload.PullRequest.RequestedReviewers))
	for _, reviewer := range payload.PullRequest.RequestedReviewers {
		formatted = append(formatted, "`"+reviewer.Login+"`")
	}
	return strings.Join(formatted, ", ")
}

func getLabelNames(payload interface{}) (ret string) {
	// github.com/go-playground/webhooks/v6/github/payload.go
	var labels []struct {
		ID          int64  `json:"id"`
		NodeID      string `json:"node_id"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Name        string `json:"name"`
		Color       string `json:"color"`
		Default     bool   `json:"default"`
	}
	switch payload := payload.(type) {
	case github.IssuesPayload:
		labels = payload.Issue.Labels
	case github.IssueCommentPayload:
		labels = payload.Issue.Labels
	case github.PullRequestPayload:
		labels = payload.PullRequest.Labels
	// case github.PullRequestReviewPayload:
	// case github.PullRequestReviewCommentPayload:
	// no labels
	default:
		return
	}

	if labels == nil {
		return
	}

	formatted := make([]string, 0, len(labels))
	for _, label := range labels {
		formatted = append(formatted, ":0x"+label.Color+": `"+label.Name+"`")
	}
	return strings.Join(formatted, ", ")
}

// rmOGP removes OGP display in traQ UI.
func rmOGP(url string) string {
	return strings.TrimPrefix(url, "https:")
}

// stripCommitMessage cuts off the string to first line.
func stripCommitMessage(s string) string {
	line, rest, _ := strings.Cut(s, "\n")
	if rest != "" {
		line += " [truncated]"
	}
	return line
}
