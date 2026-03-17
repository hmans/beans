package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHub implements the Provider interface using the gh CLI.
type GitHub struct{}

func (g *GitHub) Name() string    { return "github" }
func (g *GitHub) CLIName() string { return "gh" }

// ghPRList is the JSON shape returned by `gh pr list --json`.
type ghPRList struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// ghPRView is the JSON shape returned by `gh pr view --json` with full details.
type ghPRView struct {
	Number         int              `json:"number"`
	Title          string           `json:"title"`
	State          string           `json:"state"` // "OPEN", "CLOSED", "MERGED"
	URL            string           `json:"url"`
	IsDraft        bool             `json:"isDraft"`
	MergeStateStatus string         `json:"mergeStateStatus"` // "CLEAN", "BLOCKED", "BEHIND", "DIRTY", "UNKNOWN"
	ReviewDecision string           `json:"reviewDecision"`   // "APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED", ""
	StatusChecks   []ghStatusCheck  `json:"statusCheckRollup"`
}

type ghStatusCheck struct {
	Status     string `json:"status"`     // "COMPLETED", "IN_PROGRESS", "QUEUED", etc.
	Conclusion string `json:"conclusion"` // "SUCCESS", "FAILURE", "NEUTRAL", "SKIPPED", etc.
}

func (g *GitHub) FindPR(ctx context.Context, repoDir string, branch string) (*PullRequest, error) {
	// First, find the PR number for this branch (lightweight query)
	listCmd := exec.CommandContext(ctx, "gh", "pr", "list",
		"--head", branch,
		"--state", "open",
		"--json", "number,url",
		"--limit", "1",
	)
	listCmd.Dir = repoDir
	listOut, err := listCmd.Output()
	if err != nil {
		return nil, nil
	}

	var prs []ghPRList
	if err := json.Unmarshal(listOut, &prs); err != nil {
		return nil, fmt.Errorf("parsing gh pr list output: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}

	// Now fetch full details including merge readiness
	viewCmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", prs[0].Number),
		"--json", "number,title,state,url,isDraft,mergeStateStatus,reviewDecision,statusCheckRollup",
	)
	viewCmd.Dir = repoDir
	viewOut, err := viewCmd.Output()
	if err != nil {
		// Fall back to minimal info from list
		return &PullRequest{
			Number: prs[0].Number,
			URL:    prs[0].URL,
			State:  "open",
		}, nil
	}

	var pr ghPRView
	if err := json.Unmarshal(viewOut, &pr); err != nil {
		return &PullRequest{
			Number: prs[0].Number,
			URL:    prs[0].URL,
			State:  "open",
		}, nil
	}

	return &PullRequest{
		Number:         pr.Number,
		Title:          pr.Title,
		State:          normalizeState(pr.State),
		URL:            pr.URL,
		IsDraft:        pr.IsDraft,
		ChecksPass:     checksPass(pr.StatusChecks),
		ReviewApproved: pr.ReviewDecision == "APPROVED" || pr.ReviewDecision == "",
		Mergeable:      pr.MergeStateStatus == "CLEAN",
	}, nil
}

// checksPass returns true if all status checks have passed (or there are none).
func checksPass(checks []ghStatusCheck) bool {
	for _, c := range checks {
		if c.Status != "COMPLETED" {
			return false
		}
		switch c.Conclusion {
		case "SUCCESS", "NEUTRAL", "SKIPPED":
			// These are fine
		default:
			return false
		}
	}
	return true
}

func (g *GitHub) CreatePR(ctx context.Context, repoDir string, opts CreatePROpts) (*PullRequest, error) {
	args := []string{"pr", "create",
		"--title", opts.Title,
		"--body", opts.Body,
	}
	if opts.BaseBranch != "" {
		args = append(args, "--base", opts.BaseBranch)
	}
	if opts.Draft {
		args = append(args, "--draft")
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh pr create failed: %s", strings.TrimSpace(string(out)))
	}

	// gh pr create outputs the PR URL on success
	url := strings.TrimSpace(string(out))

	// Fetch the created PR details
	return g.findPRByURL(ctx, repoDir, url)
}

func (g *GitHub) findPRByURL(ctx context.Context, repoDir string, url string) (*PullRequest, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", url,
		"--json", "number,title,state,url,isDraft,mergeStateStatus,reviewDecision,statusCheckRollup",
	)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return &PullRequest{URL: url, State: "open"}, nil
	}

	var pr ghPRView
	if err := json.Unmarshal(out, &pr); err != nil {
		return &PullRequest{URL: url, State: "open"}, nil
	}

	return &PullRequest{
		Number:         pr.Number,
		Title:          pr.Title,
		State:          normalizeState(pr.State),
		URL:            pr.URL,
		IsDraft:        pr.IsDraft,
		ChecksPass:     checksPass(pr.StatusChecks),
		ReviewApproved: pr.ReviewDecision == "APPROVED" || pr.ReviewDecision == "",
		Mergeable:      pr.MergeStateStatus == "CLEAN",
	}, nil
}

// normalizeState converts forge-specific states to our canonical form.
func normalizeState(state string) string {
	switch strings.ToUpper(state) {
	case "OPEN":
		return "open"
	case "CLOSED":
		return "closed"
	case "MERGED":
		return "merged"
	default:
		return strings.ToLower(state)
	}
}
