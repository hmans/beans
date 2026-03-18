package graph

import (
	"context"
	"fmt"

	"github.com/hmans/beans/internal/agent"
	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/portalloc"
	"github.com/hmans/beans/internal/terminal"
	"github.com/hmans/beans/internal/worktree"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/forge"
)

//go:generate go tool gqlgen generate

// CentralSessionID is the special session identifier for the central agent chat
// that runs in the project root (not a worktree).
const CentralSessionID = "__central__"

// Resolver is the root resolver for the GraphQL schema.
// It holds a reference to beancore.Core for data access.
type Resolver struct {
	Core        *beancore.Core
	WorktreeMgr *worktree.Manager
	AgentMgr    *agent.Manager
	TerminalMgr *terminal.Manager
	PortAlloc   *portalloc.Allocator
	Forge       forge.Provider       // git forge provider (GitHub, GitLab, etc.) — nil if not detected
	ProjectRoot string               // absolute path to the project root (parent of .beans)
}

// ETagMismatchError is returned when an ETag validation fails.
// This allows callers to distinguish concurrency conflicts from other errors.
type ETagMismatchError struct {
	Provided string
	Current  string
}

func (e *ETagMismatchError) Error() string {
	return fmt.Sprintf("etag mismatch: provided %s, current is %s", e.Provided, e.Current)
}

// ETagRequiredError is returned when require_if_match is enabled and no ETag is provided.
type ETagRequiredError struct{}

func (e *ETagRequiredError) Error() string {
	return "if-match etag is required (set require_if_match: false in config to disable)"
}

// validateETag checks if the provided ifMatch etag matches the bean's current etag.
// Returns an error if validation fails or if require_if_match is enabled and no etag provided.
func (r *Resolver) validateETag(b *bean.Bean, ifMatch *string) error {
	cfg := r.Core.Config()
	requireIfMatch := cfg != nil && cfg.Beans.RequireIfMatch

	// If require_if_match is enabled and no etag provided, reject
	if requireIfMatch && (ifMatch == nil || *ifMatch == "") {
		return &ETagRequiredError{}
	}

	// If ifMatch provided, validate it
	if ifMatch != nil && *ifMatch != "" {
		currentETag := b.ETag()
		if currentETag != *ifMatch {
			return &ETagMismatchError{Provided: *ifMatch, Current: currentETag}
		}
	}

	return nil
}

// validateAndSetParent validates and sets the parent relationship.
func (r *Resolver) validateAndSetParent(b *bean.Bean, parentID string) error {
	if parentID == "" {
		b.Parent = ""
		return nil
	}

	// Normalise short ID to full ID
	normalizedParent, _ := r.Core.NormalizeID(parentID)

	// Validate parent type hierarchy
	if err := r.Core.ValidateParent(b, normalizedParent); err != nil {
		return err
	}

	// Check for cycles
	if cycle := r.Core.DetectCycle(b.ID, "parent", normalizedParent); cycle != nil {
		return fmt.Errorf("setting parent would create cycle: %v", cycle)
	}

	b.Parent = normalizedParent
	return nil
}

// validateAndAddBlocking validates and adds blocking relationships.
func (r *Resolver) validateAndAddBlocking(b *bean.Bean, targetIDs []string) error {
	for _, targetID := range targetIDs {
		// Normalise short ID to full ID
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)

		// Validate: cannot block itself
		if normalizedTargetID == b.ID {
			return fmt.Errorf("bean cannot block itself")
		}

		// Validate: target must exist
		if _, err := r.Core.Get(normalizedTargetID); err != nil {
			return fmt.Errorf("blocking target bean not found: %s", targetID)
		}

		// Check for cycles in both directions
		if cycle := r.Core.DetectCycle(b.ID, "blocking", normalizedTargetID); cycle != nil {
			return fmt.Errorf("adding blocking relationship would create cycle: %v", cycle)
		}
		if cycle := r.Core.DetectCycle(normalizedTargetID, "blocked_by", b.ID); cycle != nil {
			return fmt.Errorf("adding blocking relationship would create cycle: %v", cycle)
		}

		b.AddBlocking(normalizedTargetID)
	}
	return nil
}

// removeBlockingRelationships removes blocking relationships.
func (r *Resolver) removeBlockingRelationships(b *bean.Bean, targetIDs []string) {
	for _, targetID := range targetIDs {
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)
		b.RemoveBlocking(normalizedTargetID)
	}
}

// validateAndAddBlockedBy validates and adds blocked-by relationships.
func (r *Resolver) validateAndAddBlockedBy(b *bean.Bean, targetIDs []string) error {
	for _, targetID := range targetIDs {
		// Normalise short ID to full ID
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)

		// Validate: cannot be blocked by itself
		if normalizedTargetID == b.ID {
			return fmt.Errorf("bean cannot be blocked by itself")
		}

		// Validate: blocker must exist
		if _, err := r.Core.Get(normalizedTargetID); err != nil {
			return fmt.Errorf("blocker bean not found: %s", targetID)
		}

		// Check for cycles in both directions
		if cycle := r.Core.DetectCycle(normalizedTargetID, "blocking", b.ID); cycle != nil {
			return fmt.Errorf("adding blocked-by relationship would create cycle: %v", cycle)
		}
		if cycle := r.Core.DetectCycle(b.ID, "blocked_by", normalizedTargetID); cycle != nil {
			return fmt.Errorf("adding blocked-by relationship would create cycle: %v", cycle)
		}

		b.AddBlockedBy(normalizedTargetID)
	}
	return nil
}

// removeBlockedByRelationships removes blocked-by relationships.
func (r *Resolver) removeBlockedByRelationships(b *bean.Bean, targetIDs []string) {
	for _, targetID := range targetIDs {
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)
		b.RemoveBlockedBy(normalizedTargetID)
	}
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
