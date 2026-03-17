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

// ghPR is the JSON shape returned by `gh pr list --json`.
type ghPR struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`   // "OPEN", "CLOSED", "MERGED"
	URL     string `json:"url"`
	IsDraft bool   `json:"isDraft"`
}

func (g *GitHub) FindPR(ctx context.Context, repoDir string, branch string) (*PullRequest, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "list",
		"--head", branch,
		"--state", "open",
		"--json", "number,title,state,url,isDraft",
		"--limit", "1",
	)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		// gh returns exit code 1 when not in a repo or no auth — treat as "no PR"
		return nil, nil
	}

	var prs []ghPR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}

	pr := &prs[0]
	return &PullRequest{
		Number:  pr.Number,
		Title:   pr.Title,
		State:   normalizeState(pr.State),
		URL:     pr.URL,
		IsDraft: pr.IsDraft,
	}, nil
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
		"--json", "number,title,state,url,isDraft",
	)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		// Return a minimal PR with just the URL if view fails
		return &PullRequest{URL: url, State: "open"}, nil
	}

	var pr ghPR
	if err := json.Unmarshal(out, &pr); err != nil {
		return &PullRequest{URL: url, State: "open"}, nil
	}

	return &PullRequest{
		Number:  pr.Number,
		Title:   pr.Title,
		State:   normalizeState(pr.State),
		URL:     pr.URL,
		IsDraft: pr.IsDraft,
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
