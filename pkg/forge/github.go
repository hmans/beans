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

// ghGraphQLPR represents a PR node from the GitHub GraphQL API response.
type ghGraphQLPR struct {
	Number           int    `json:"number"`
	Title            string `json:"title"`
	State            string `json:"state"`
	URL              string `json:"url"`
	IsDraft          bool   `json:"isDraft"`
	MergeStateStatus string `json:"mergeStateStatus"`
	ReviewDecision   string `json:"reviewDecision"`
	Commits          struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					Contexts struct {
						Nodes []ghGraphQLCheckNode `json:"nodes"`
					} `json:"contexts"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

type ghGraphQLCheckNode struct {
	TypeName   string `json:"__typename"`
	Status     string `json:"status"`     // CheckRun: COMPLETED, IN_PROGRESS, QUEUED, etc.
	Conclusion string `json:"conclusion"` // CheckRun: SUCCESS, FAILURE, NEUTRAL, SKIPPED, etc.
	State      string `json:"state"`      // StatusContext: SUCCESS, PENDING, FAILURE, ERROR, EXPECTED
}

// graphQLCheckToStatusCheck converts a GitHub GraphQL check node to our internal format.
func graphQLCheckToStatusCheck(node ghGraphQLCheckNode) ghStatusCheck {
	if node.TypeName == "StatusContext" {
		switch node.State {
		case "SUCCESS":
			return ghStatusCheck{Status: "COMPLETED", Conclusion: "SUCCESS"}
		case "PENDING", "EXPECTED":
			return ghStatusCheck{Status: "IN_PROGRESS", Conclusion: ""}
		default: // ERROR, FAILURE
			return ghStatusCheck{Status: "COMPLETED", Conclusion: "FAILURE"}
		}
	}
	return ghStatusCheck{Status: node.Status, Conclusion: node.Conclusion}
}

// graphQLPRToForge converts a GitHub GraphQL PR response to our PullRequest type.
func graphQLPRToForge(pr ghGraphQLPR) *PullRequest {
	var checks []ghStatusCheck
	if len(pr.Commits.Nodes) > 0 {
		rollup := pr.Commits.Nodes[0].Commit.StatusCheckRollup
		if rollup != nil {
			for _, node := range rollup.Contexts.Nodes {
				checks = append(checks, graphQLCheckToStatusCheck(node))
			}
		}
	}

	return &PullRequest{
		Number:         pr.Number,
		Title:          pr.Title,
		State:          normalizeState(pr.State),
		URL:            pr.URL,
		IsDraft:        pr.IsDraft,
		Checks:         computeCheckStatus(checks),
		ReviewApproved: pr.ReviewDecision == "APPROVED" || pr.ReviewDecision == "",
		Mergeable:      pr.MergeStateStatus == "CLEAN",
	}
}

const prGraphQLFields = `nodes {
	number title state url isDraft mergeStateStatus reviewDecision
	commits(last: 1) { nodes { commit { statusCheckRollup { contexts(first: 100) {
		nodes { __typename ... on CheckRun { status conclusion } ... on StatusContext { state } }
	} } } } }
}`

func (g *GitHub) FindPRs(ctx context.Context, repoDir string, branches []string) (map[string]*PullRequest, error) {
	if len(branches) == 0 {
		return map[string]*PullRequest{}, nil
	}

	owner, repo, ok := ParseOwnerRepo(getOriginURL(repoDir))
	if !ok {
		return nil, fmt.Errorf("cannot parse GitHub owner/repo from remote URL")
	}

	// Build a single GraphQL query with two aliases per branch (open + merged).
	var queryParts []string
	for i, branch := range branches {
		queryParts = append(queryParts,
			fmt.Sprintf(`open%d: pullRequests(headRefName: %q, first: 1, states: [OPEN]) { %s }`, i, branch, prGraphQLFields),
			fmt.Sprintf(`merged%d: pullRequests(headRefName: %q, first: 1, states: [MERGED], orderBy: {field: CREATED_AT, direction: DESC}) { %s }`, i, branch, prGraphQLFields),
		)
	}
	query := fmt.Sprintf(`{ repository(owner: %q, name: %q) { %s } }`,
		owner, repo, strings.Join(queryParts, "\n"))

	cmd := exec.CommandContext(ctx, "gh", "api", "graphql", "-f", "query="+query)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api graphql: %w", err)
	}

	// Parse the dynamic response.
	var envelope struct {
		Data struct {
			Repository json.RawMessage `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		return nil, fmt.Errorf("parsing graphql response: %w", err)
	}

	var repoData map[string]struct {
		Nodes []ghGraphQLPR `json:"nodes"`
	}
	if err := json.Unmarshal(envelope.Data.Repository, &repoData); err != nil {
		return nil, fmt.Errorf("parsing repository data: %w", err)
	}

	// Resolve results: prefer open PR, fall back to merged.
	result := make(map[string]*PullRequest, len(branches))
	for i, branch := range branches {
		openKey := fmt.Sprintf("open%d", i)
		mergedKey := fmt.Sprintf("merged%d", i)

		if conn, ok := repoData[openKey]; ok && len(conn.Nodes) > 0 {
			result[branch] = graphQLPRToForge(conn.Nodes[0])
		} else if conn, ok := repoData[mergedKey]; ok && len(conn.Nodes) > 0 {
			result[branch] = graphQLPRToForge(conn.Nodes[0])
		}
	}

	return result, nil
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
