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
		// No open PR — check for a recently merged one
		mergedCmd := exec.CommandContext(ctx, "gh", "pr", "list",
			"--head", branch,
			"--state", "merged",
			"--json", "number,url",
			"--limit", "1",
		)
		mergedCmd.Dir = repoDir
		mergedOut, err := mergedCmd.Output()
		if err != nil {
			return nil, nil
		}
		var merged []ghPRList
		if err := json.Unmarshal(mergedOut, &merged); err != nil || len(merged) == 0 {
			return nil, nil
		}
		// Return merged PR with minimal details
		return g.fetchPRDetails(ctx, repoDir, merged[0].Number, merged[0].URL)
	}

	return g.fetchPRDetails(ctx, repoDir, prs[0].Number, prs[0].URL)
}

// fetchPRDetails fetches full PR details by number, falling back to minimal info on failure.
func (g *GitHub) fetchPRDetails(ctx context.Context, repoDir string, number int, fallbackURL string) (*PullRequest, error) {
	viewCmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", number),
		"--json", "number,title,state,url,isDraft,mergeStateStatus,reviewDecision,statusCheckRollup",
	)
	viewCmd.Dir = repoDir
	viewOut, err := viewCmd.Output()
	if err != nil {
		return &PullRequest{
			Number: number,
			URL:    fallbackURL,
			State:  "open",
		}, nil
	}

	var pr ghPRView
	if err := json.Unmarshal(viewOut, &pr); err != nil {
		return &PullRequest{
			Number: number,
			URL:    fallbackURL,
			State:  "open",
		}, nil
	}

	return &PullRequest{
		Number:         pr.Number,
		Title:          pr.Title,
		State:          normalizeState(pr.State),
		URL:            pr.URL,
		IsDraft:        pr.IsDraft,
		Checks:         computeCheckStatus(pr.StatusChecks),
		ReviewApproved: pr.ReviewDecision == "APPROVED" || pr.ReviewDecision == "",
		Mergeable:      pr.MergeStateStatus == "CLEAN",
	}, nil
}

// computeCheckStatus determines the aggregate check status from individual checks.
func computeCheckStatus(checks []ghStatusCheck) CheckStatus {
	if len(checks) == 0 {
		return CheckStatusPass
	}
	for _, c := range checks {
		if c.Status != "COMPLETED" {
			return CheckStatusPending
		}
	}
	// All completed — check conclusions
	for _, c := range checks {
		switch c.Conclusion {
		case "SUCCESS", "NEUTRAL", "SKIPPED":
			// fine
		default:
			return CheckStatusFail
		}
	}
	return CheckStatusPass
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
		Checks:         computeCheckStatus(pr.StatusChecks),
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
