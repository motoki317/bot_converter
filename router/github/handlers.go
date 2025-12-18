package github

import (
	"fmt"
	"strings"

	"github.com/go-playground/webhooks/v6/github"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"git.trap.jp/toki/bot_converter/router/github/icons"
	"git.trap.jp/toki/bot_converter/utils"
)

// toTitle converts a string to title case using golang.org/x/text/cases.
// Creates a new Caser for each call to ensure thread safety.
func toTitle(s string) string {
	return cases.Title(language.English).String(s)
}

func (c *converter) checkRunHandler(payload github.CheckRunPayload) (string, error) {
	if payload.Action != "completed" {
		return "", nil
	}
	var (
		icon       string
		conclusion string
	)
	switch payload.CheckRun.Conclusion {
	case "success":
		icon = icons.CheckSuccess
		conclusion = "succeeded"
	case "failure":
		icon = icons.CheckFail
		conclusion = "failed"
	case "startup_failure":
		icon = icons.CheckFail
		conclusion = "startup failed"
	case "timed_out":
		icon = icons.CheckFail
		conclusion = "timed out"
	case "skipped":
		icon = icons.CheckSkip
		conclusion = "skipped"
	case "cancelled":
		icon = icons.CheckCancel
		conclusion = "cancelled"
	default:
		return "", nil
	}
	res := fmt.Sprintf(
		"### :%s: [[%s](%s)] Check [%s](%s) %s",
		icon,
		payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
		payload.CheckRun.Name,
		rmOGP(payload.CheckRun.HTMLURL),
		conclusion,
	)
	return res, nil
}

func (c *converter) issuesHandler(payload github.IssuesPayload) (string, error) {
	var icon string
	switch payload.Action {
	case "opened":
		icon = icons.IssueOpened
	case "edited":
		icon = icons.Edit
	case "deleted":
		icon = icons.IssueClosed
	case "closed":
		icon = icons.IssueClosed
	case "reopened":
		icon = icons.IssueOpened
	case "pinned":
		icon = icons.Pin
	case "unpinned":
		icon = icons.Pin
	case "labeled":
		icon = icons.Tag
	case "unlabeled":
		icon = icons.Tag
	case "locked":
		icon = icons.Lock
	case "unlocked":
		icon = icons.Unlock
	case "transferred":
		icon = icons.Transfer
	case "milestoned":
		icon = icons.Milestone
	case "demilestoned":
		icon = icons.Milestone
	case "assigned":
		icon = icons.Assignment
	case "unassigned":
		icon = icons.Assignment
	default:
		return "", nil
	}

	issueName := fmt.Sprintf("[#%d %s](%s)", payload.Issue.Number, payload.Issue.Title, rmOGP(payload.Issue.HTMLURL))
	var m strings.Builder
	switch payload.Action {
	case "assigned":
		fallthrough
	case "unassigned":
		m.WriteString(fmt.Sprintf(
			"### :%s: [[%s](%s)] Issue %s %s to `%s` by `%s`\n",
			icon,
			payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
			issueName,
			toTitle(payload.Action),
			payload.Assignee.Login,
			payload.Sender.Login))
	default:
		m.WriteString(fmt.Sprintf(
			"### :%s: [[%s](%s)] Issue %s %s by `%s`\n",
			icon,
			payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
			issueName,
			toTitle(payload.Action),
			payload.Sender.Login))
	}

	if assignees := getAssigneeNames(payload); assignees != "" {
		m.WriteString("Assignees: ")
		m.WriteString(assignees)
		m.WriteString("\n")
	}
	if labels := getLabelNames(payload); labels != "" {
		m.WriteString("Labels: ")
		m.WriteString(labels)
		m.WriteString("\n")
	}

	isBot := strings.Contains(payload.Sender.Login, "[bot]") || strings.Contains(payload.Issue.User.Login, "[bot]")
	hideBody := isBot || payload.Issue.Body == ""
	if !hideBody {
		if payload.Action == "opened" || payload.Action == "edited" {
			m.WriteString("\n---\n")
			m.WriteString(payload.Issue.Body)
		}
	}

	return m.String(), nil
}

func (c *converter) issueCommentHandler(payload github.IssueCommentPayload) (string, error) {
	isBot := strings.Contains(payload.Sender.Login, "[bot]") || strings.Contains(payload.Comment.User.Login, "[bot]")
	if isBot {
		return "", nil
	}

	var icon string
	switch payload.Action {
	case "created":
		icon = icons.Comment
	case "edited":
		icon = icons.Edit
	case "deleted":
		icon = icons.Retrieved
	default:
		return "", nil
	}

	issueName := fmt.Sprintf("[#%d %s](%s)", payload.Issue.Number, payload.Issue.Title, rmOGP(payload.Issue.HTMLURL))
	var m strings.Builder
	m.WriteString(fmt.Sprintf(
		"### :%s: [[%s](%s)] [Comment](%s) %s by `%s`\n"+
			"%s\n",
		icon,
		payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
		rmOGP(payload.Comment.HTMLURL),
		toTitle(payload.Action),
		payload.Sender.Login,
		issueName))

	if assignees := getAssigneeNames(payload); assignees != "" {
		m.WriteString("Assignees: ")
		m.WriteString(assignees)
		m.WriteString("\n")
	}
	if labels := getLabelNames(payload); labels != "" {
		m.WriteString("Labels: ")
		m.WriteString(labels)
		m.WriteString("\n")
	}

	hideBody := isBot || payload.Comment.Body == ""
	if !hideBody {
		if payload.Action == "created" || payload.Action == "edited" {
			m.WriteString("\n---\n")
			m.WriteString(payload.Comment.Body)
		}
	}

	return m.String(), nil
}

func (c *converter) canProcessPush(payload github.PushPayload) bool {
	if len(c.config.PushBranchFilter) == 0 {
		return true
	}
	return utils.FilterByRegexp(c.config.PushBranchFilter, payload.Ref)
}

func (c *converter) pushHandler(payload github.PushPayload) (string, error) {
	if len(payload.Commits) == 0 {
		return "", nil
	}
	if !c.canProcessPush(payload) {
		return "", nil
	}

	var m strings.Builder
	m.WriteString(fmt.Sprintf(
		"### :%s: [[%s](%s)] %v New",
		icons.Pushed,
		payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
		len(payload.Commits)))

	if len(payload.Commits) == 1 {
		m.WriteString(" Commit")
	} else {
		m.WriteString(" Commits")
	}
	m.WriteString(fmt.Sprintf(
		" to `%s` by `%s`\n",
		payload.Ref,
		payload.Sender.Login))

	m.WriteString("\n---\n")
	for _, commit := range payload.Commits {
		formattedTime, err := formatTime(commit.Timestamp, "2006/01/02 15:04:05")
		if err != nil {
			return "", err
		}
		m.WriteString(fmt.Sprintf(
			":0x%s: [`%s`](%s) : %s - `%s` @ %s\n",
			commit.ID[:6], commit.ID[:6],
			rmOGP(commit.URL),
			stripCommitMessage(commit.Message),
			commit.Author.Name,
			formattedTime))
	}

	return m.String(), nil
}

func (c *converter) releaseHandler(payload github.ReleasePayload) (string, error) {
	if payload.Action != "published" {
		return "", nil
	}

	var m strings.Builder
	releaseType := "Release"
	if payload.Release.Prerelease {
		releaseType = "Prerelease"
	}
	var releaseName string
	if payload.Release.Name != nil {
		releaseName = " " + *payload.Release.Name
	}
	m.WriteString(fmt.Sprintf(
		"### :%s: [[%s](%s)] %s%s %s by %s\n",
		icons.Tag,
		payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
		releaseType, releaseName, toTitle(payload.Action),
		payload.Release.Author.Login))

	m.WriteString(fmt.Sprintf("Tag: %s\n", payload.Release.TagName))

	if payload.Release.Body != nil && *payload.Release.Body != "" {
		m.WriteString("\n---\n")
		m.WriteString(*payload.Release.Body)
	}

	return m.String(), nil
}

func (c *converter) canProcessPR(payload github.PullRequestPayload) bool {
	if len(c.config.PREventTypesFilter) == 0 {
		return true
	}
	return utils.FilterByRegexp(c.config.PREventTypesFilter, payload.Action)
}

func (c *converter) pullRequestHandler(payload github.PullRequestPayload) (string, error) {
	if !c.canProcessPR(payload) {
		return "", nil
	}

	// If action == "closed" and Merged is true, then the pull request was merged
	action := payload.Action
	var icon string
	switch payload.Action {
	case "opened":
		icon = icons.PullRequestOpened
	case "edited":
		icon = icons.Edit
	case "closed":
		if payload.PullRequest.Merged {
			action = "merged"
			icon = icons.PullRequestMerged
		} else {
			action = "closed"
			icon = icons.PullRequestClosed
		}
	case "reopened":
		icon = icons.PullRequestOpened
	case "assigned":
		icon = icons.Assignment
	case "unassigned":
		icon = icons.Assignment
	case "review_requested":
		action = "review requested"
		icon = icons.Assignment
	case "review_request_removed":
		action = "review request removed"
		icon = icons.Assignment
	case "ready_for_review":
		action = "marked as ready for review"
		icon = icons.Assignment
	case "labeled":
		icon = icons.Tag
	case "unlabeled":
		icon = icons.Tag
	// case "synchronize": on push event
	case "locked":
		icon = icons.Lock
	case "unlocked":
		icon = icons.Unlock
	default:
		return "", nil
	}

	prName := fmt.Sprintf("[#%d %s](%s)", payload.PullRequest.Number, payload.PullRequest.Title, rmOGP(payload.PullRequest.HTMLURL))

	var m strings.Builder
	switch payload.Action {
	case "assigned":
		fallthrough
	case "unassigned":
		m.WriteString(fmt.Sprintf(
			"### :%s: [[%s](%s)] Pull Request %s %s to `%s` by `%s`\n",
			icon,
			payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
			prName,
			toTitle(action),
			payload.Assignee.Login,
			payload.Sender.Login))
	case "review_requested":
		m.WriteString(fmt.Sprintf(
			"### :%s: [[%s](%s)] Pull Request %s %s to `%s` by `%s`\n",
			icon,
			payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
			prName,
			toTitle(action),
			payload.RequestedReviewer.Login,
			payload.Sender.Login))
	default:
		m.WriteString(fmt.Sprintf(
			"### :%s: [[%s](%s)] Pull Request %s %s by `%s`\n",
			icon,
			payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
			prName,
			toTitle(action),
			payload.Sender.Login))
	}

	if assignees := getAssigneeNames(payload); assignees != "" {
		m.WriteString("Assignees: ")
		m.WriteString(assignees)
		m.WriteString("\n")
	}
	if reviewers := getRequestedReviewers(payload); reviewers != "" {
		m.WriteString("Reviewers: ")
		m.WriteString(reviewers)
		m.WriteString("\n")
	}
	if labels := getLabelNames(payload); labels != "" {
		m.WriteString("Labels: ")
		m.WriteString(labels)
		m.WriteString("\n")
	}

	isBot := strings.Contains(payload.Sender.Login, "[bot]") || strings.Contains(payload.PullRequest.User.Login, "[bot]")
	hideBody := isBot || payload.PullRequest.Body == ""
	if !hideBody {
		// send pull request body only on the first open or on edited
		if payload.Action == "opened" || payload.Action == "edited" {
			m.WriteString("\n---\n")
			m.WriteString(payload.PullRequest.Body)
		}
	}

	return m.String(), nil
}

func (c *converter) pullRequestReviewHandler(payload github.PullRequestReviewPayload) (string, error) {
	if payload.Action != "submitted" {
		return "", nil
	}

	var action string
	var icon string
	switch payload.Review.State {
	case "approved":
		action = "approved"
		icon = icons.PullRequestApproved
	case "commented":
		action = "commented"
		icon = icons.Comment
	case "changes_requested":
		action = "changes requested"
		icon = icons.Comment
	default:
		return "", nil
	}

	prName := fmt.Sprintf("[#%d %s](%s)", payload.PullRequest.Number, payload.PullRequest.Title, rmOGP(payload.PullRequest.HTMLURL))
	var m strings.Builder
	m.WriteString(fmt.Sprintf(
		"### :%s: [[%s](%s)] Pull Request %s %s by `%s`\n",
		icon,
		payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
		prName,
		toTitle(action),
		payload.Sender.Login))

	if assignees := getAssigneeNames(payload); assignees != "" {
		m.WriteString("Assignees: ")
		m.WriteString(assignees)
		m.WriteString("\n")
	}

	isBot := strings.Contains(payload.Sender.Login, "[bot]")
	hideBody := isBot || payload.Review.Body == ""
	if !hideBody {
		m.WriteString("\n---\n")
		m.WriteString(payload.Review.Body)
	}

	// Review comment event usually follows with actual comment body
	if hideBody && action == "commented" {
		return "", nil
	}

	return m.String(), nil
}

func (c *converter) pullRequestReviewCommentHandler(payload github.PullRequestReviewCommentPayload) (string, error) {
	isBot := strings.Contains(payload.Sender.Login, "[bot]") || strings.Contains(payload.Comment.User.Login, "[bot]")
	if isBot {
		return "", nil
	}

	var icon string
	switch payload.Action {
	case "created":
		icon = icons.Comment
	case "edited":
		icon = icons.Edit
	case "deleted":
		icon = icons.Retrieved
	default:
		return "", nil
	}

	prName := fmt.Sprintf("[#%d %s](%s)", payload.PullRequest.Number, payload.PullRequest.Title, rmOGP(payload.PullRequest.HTMLURL))
	var m strings.Builder
	m.WriteString(fmt.Sprintf(
		"### :%s: [[%s](%s)] [Review Comment](%s) %s by `%s`\n"+
			"%s\n",
		icon,
		payload.Repository.Name, rmOGP(payload.Repository.HTMLURL),
		rmOGP(payload.Comment.HTMLURL),
		toTitle(payload.Action),
		payload.Sender.Login,
		prName))

	if assignees := getAssigneeNames(payload); assignees != "" {
		m.WriteString("Assignees: ")
		m.WriteString(assignees)
		m.WriteString("\n")
	}

	hideBody := isBot || payload.Comment.Body == ""
	if !hideBody {
		m.WriteString("\n---\n")
		m.WriteString(payload.Comment.Body)
	}

	return m.String(), nil
}
