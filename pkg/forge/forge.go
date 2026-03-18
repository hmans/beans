// Package forge provides an abstraction over git hosting providers (GitHub, GitLab, etc.)
// for pull/merge request operations. It uses the provider's CLI tool (gh, glab) under the hood.
package forge

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Provider abstracts pull/merge request operations across git forges.
type Provider interface {
	// Name returns the forge name (e.g., "github", "gitlab").
	Name() string

	// CLIName returns the CLI tool name (e.g., "gh", "glab").
	CLIName() string

	// FindPR returns the open pull/merge request for the given branch, or nil if none exists.
	FindPR(ctx context.Context, repoDir string, branch string) (*PullRequest, error)

	// FindPRs returns pull requests for multiple branches in a single batch query.
	// The returned map is keyed by branch name. Branches with no PR are omitted.
	FindPRs(ctx context.Context, repoDir string, branches []string) (map[string]*PullRequest, error)

	// CreatePR creates a new pull/merge request and returns it.
	CreatePR(ctx context.Context, repoDir string, opts CreatePROpts) (*PullRequest, error)
}

// CheckStatus represents the aggregate state of CI checks on a PR.
type CheckStatus string

const (
	CheckStatusPass    CheckStatus = "pass"    // all checks completed successfully (or no checks)
	CheckStatusFail    CheckStatus = "fail"    // at least one check failed, none still running
	CheckStatusPending CheckStatus = "pending" // at least one check is still running/queued
)

// PullRequest represents a pull/merge request on a git forge.
type PullRequest struct {
	Number         int
	Title          string
	State          string // "open", "closed", "merged", "draft"
	URL            string
	IsDraft        bool
	Checks         CheckStatus // aggregate CI check status
	ReviewApproved bool        // review requirements are met (approved or no reviews required)
	Mergeable      bool        // forge reports the PR can be merged (no conflicts, branch protections met)
}

// CanMerge returns true if the PR is in a mergeable state:
// not a draft, checks pass, review approved, and forge says it's mergeable.
func (pr *PullRequest) CanMerge() bool {
	return !pr.IsDraft && pr.Checks == CheckStatusPass && pr.Mergeable
}

// CreatePROpts are the options for creating a pull/merge request.
type CreatePROpts struct {
	Title      string
	Body       string
	BaseBranch string
	Draft      bool
}

// Detect auto-detects the forge provider from the git remote URL in the given repo directory.
// Returns nil if no supported forge is detected or the corresponding CLI tool is not installed.
func Detect(repoDir string) Provider {
	remoteURL := getOriginURL(repoDir)
	if remoteURL == "" {
		return nil
	}

	switch {
	case isGitHub(remoteURL):
		if !hasCLI("gh") {
			return nil
		}
		return &GitHub{}
	// Future: case isGitLab(remoteURL): return &GitLab{}
	default:
		return nil
	}
}

// ParseOwnerRepo extracts the owner and repository name from a git remote URL.
// Supports SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git) formats.
func ParseOwnerRepo(remoteURL string) (owner, repo string, ok bool) {
	url := strings.TrimSpace(remoteURL)
	url = strings.TrimSuffix(url, ".git")

	// SSH format: git@github.com:owner/repo
	if idx := strings.Index(url, ":"); idx != -1 && !strings.Contains(url[:idx], "/") {
		path := url[idx+1:]
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], true
		}
	}

	// HTTPS format: https://github.com/owner/repo
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.Split(url, "/")
	if len(parts) >= 3 && parts[1] != "" && parts[2] != "" {
		return parts[1], parts[2], true
	}

	return "", "", false
}

// getOriginURL returns the URL of the "origin" git remote.
func getOriginURL(repoDir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// isGitHub returns true if the remote URL points to a GitHub instance.
// Supports github.com and GitHub Enterprise (any host with "github" in the name).
func isGitHub(remoteURL string) bool {
	lower := strings.ToLower(remoteURL)
	return strings.Contains(lower, "github.com") || strings.Contains(lower, "github")
}

// hasCLI checks if a CLI tool is available on PATH.
func hasCLI(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// FormatPRRef returns a human-readable reference for a PR (e.g., "#42").
func FormatPRRef(pr *PullRequest) string {
	return fmt.Sprintf("#%d", pr.Number)
}
