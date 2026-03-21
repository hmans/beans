package graph

import (
	"context"

	"github.com/hmans/beans/internal/agent"
	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/internal/portalloc"
	"github.com/hmans/beans/internal/terminal"
	"github.com/hmans/beans/internal/worktree"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/forge"
)

//go:generate go tool gqlgen generate

// CentralSessionID is the special session identifier for the central agent chat
// that runs in the project root (not a worktree).
const CentralSessionID = "__central__"

// RunSessionSuffix is appended to workspace IDs to form the terminal session ID
// for run command sessions (e.g., "worktree-abc__run").
const RunSessionSuffix = "__run"

// Resolver is the root resolver for the GraphQL schema.
// It embeds CoreResolver for bean CRUD operations and adds UI-specific
// concerns (agents, worktrees, terminals, git operations).
type Resolver struct {
	*beangraph.CoreResolver
	WorktreeMgr *worktree.Manager
	AgentMgr    *agent.Manager
	TerminalMgr *terminal.Manager
	PortAlloc   *portalloc.Allocator
	Forge       forge.Provider // git forge provider (GitHub, GitLab, etc.) — nil if not detected
	ProjectRoot string         // absolute path to the project root (parent of .beans)
}

// worktreeToModel converts an internal worktree to a GraphQL model.
// It takes an optional beancore.Core to resolve BeanIDs into full Bean objects.
// When computeGitStatus is true, it shells out to git to compute hasChanges and
// hasUnmergedCommits; otherwise these default to false to avoid expensive subprocess
// calls on hot paths like subscriptions.
func worktreeToModel(wt *worktree.Worktree, core *beancore.Core, baseRef string, computeGitStatus bool) *model.Worktree {
	m := &model.Worktree{
		ID:     wt.ID,
		Branch: wt.Branch,
		Path:   wt.Path,
		Beans:  []*bean.Bean{},
	}
	if computeGitStatus {
		m.HasChanges = gitutil.HasChanges(wt.Path)
		m.HasUnmergedCommits = gitutil.HasUnmergedCommits(wt.Path, baseRef)
		m.CommitsBehind = gitutil.CommitsBehind(wt.Path, baseRef)
		m.HasConflicts = gitutil.HasConflicts(wt.Path, baseRef)
	}
	if wt.Name != "" {
		m.Name = &wt.Name
	}
	if wt.Description != "" {
		m.Description = &wt.Description
	}
	if core != nil {
		for _, id := range wt.BeanIDs {
			if b, err := core.Get(id); err == nil {
				m.Beans = append(m.Beans, b)
			}
		}
	}
	// Map setup status
	switch wt.Setup {
	case worktree.SetupRunning:
		s := model.WorktreeSetupStatusRunning
		m.SetupStatus = &s
	case worktree.SetupDone:
		s := model.WorktreeSetupStatusDone
		m.SetupStatus = &s
	case worktree.SetupFailed:
		s := model.WorktreeSetupStatusFailed
		m.SetupStatus = &s
	}
	if wt.SetupError != "" {
		m.SetupError = &wt.SetupError
	}
	return m
}

// populatePRsBatch fetches PR data for multiple worktrees in a single batch query
// and sets the results on the corresponding models.
func populatePRsBatch(ctx context.Context, worktrees []*model.Worktree, forgeProvider forge.Provider, repoDir string) {
	if forgeProvider == nil || len(worktrees) == 0 {
		return
	}

	branches := make([]string, len(worktrees))
	for i, wt := range worktrees {
		branches[i] = wt.Branch
	}

	prs, err := forgeProvider.FindPRs(ctx, repoDir, branches)
	if err != nil {
		return
	}

	for _, wt := range worktrees {
		if pr, ok := prs[wt.Branch]; ok {
			wt.PullRequest = forgePRToModel(pr)
		}
	}
}

// forgePRToModel converts a forge PullRequest to a GraphQL model PullRequest.
func forgePRToModel(pr *forge.PullRequest) *model.PullRequest {
	return &model.PullRequest{
		Number:         pr.Number,
		Title:          pr.Title,
		State:          pr.State,
		URL:            pr.URL,
		IsDraft:        pr.IsDraft,
		CheckStatus:    string(pr.Checks),
		ReviewApproved: pr.ReviewApproved,
		Mergeable:      pr.Mergeable,
	}
}
