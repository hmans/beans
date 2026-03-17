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

	// CreatePR creates a new pull/merge request and returns it.
	CreatePR(ctx context.Context, repoDir string, opts CreatePROpts) (*PullRequest, error)
}

// PullRequest represents a pull/merge request on a git forge.
type PullRequest struct {
	Number         int
	Title          string
	State          string // "open", "closed", "merged", "draft"
	URL            string
	IsDraft        bool
	ChecksPass     bool // all CI checks are passing (or no checks required)
	ReviewApproved bool // review requirements are met (approved or no reviews required)
	Mergeable      bool // forge reports the PR can be merged (no conflicts, branch protections met)
}

// CanMerge returns true if the PR is in a mergeable state:
// not a draft, checks pass, review approved, and forge says it's mergeable.
func (pr *PullRequest) CanMerge() bool {
	return !pr.IsDraft && pr.ChecksPass && pr.Mergeable
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
