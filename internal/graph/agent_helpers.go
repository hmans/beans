package graph

import (
	"fmt"

	"github.com/hmans/beans/internal/agent"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/pkg/forge"
)

// agentSessionToModel converts an agent.Session to the GraphQL model type.
func agentSessionToModel(s *agent.Session) *model.AgentSession {
	msgs := make([]*model.AgentMessage, len(s.Messages))
	for i, m := range s.Messages {
		var role model.AgentMessageRole
		switch m.Role {
		case agent.RoleUser:
			role = model.AgentMessageRoleUser
		case agent.RoleAssistant:
			role = model.AgentMessageRoleAssistant
		case agent.RoleTool:
			role = model.AgentMessageRoleTool
		case agent.RoleInfo:
			role = model.AgentMessageRoleInfo
		}
		images := make([]*model.AgentMessageImage, 0, len(m.Images))
		for _, img := range m.Images {
			images = append(images, &model.AgentMessageImage{
				URL:       fmt.Sprintf("/api/attachments/%s/%s", s.ID, img.ID),
				MediaType: img.MediaType,
			})
		}
		var diff *string
		if m.Diff != "" {
			diff = &m.Diff
		}
		msgs[i] = &model.AgentMessage{
			Role:    role,
			Content: m.Content,
			Images:  images,
			Diff:    diff,
		}
	}

	var status model.AgentSessionStatus
	switch s.Status {
	case agent.StatusIdle:
		status = model.AgentSessionStatusIdle
	case agent.StatusRunning:
		status = model.AgentSessionStatusRunning
	case agent.StatusError:
		status = model.AgentSessionStatusError
	}

	var errPtr *string
	if s.Error != "" {
		errPtr = &s.Error
	}

	var pending *model.PendingInteraction
	if s.PendingInteraction != nil {
		var itype model.InteractionType
		switch s.PendingInteraction.Type {
		case agent.InteractionExitPlan:
			itype = model.InteractionTypeExitPlan
		case agent.InteractionEnterPlan:
			itype = model.InteractionTypeEnterPlan
		case agent.InteractionAskUser:
			itype = model.InteractionTypeAskUser
		}
		var planContent *string
		if s.PendingInteraction.PlanContent != "" {
			planContent = &s.PendingInteraction.PlanContent
		}
		var questions []*model.AskUserQuestion
		for _, q := range s.PendingInteraction.Questions {
			opts := make([]*model.AskUserOption, len(q.Options))
			for j, o := range q.Options {
				opts[j] = &model.AskUserOption{Label: o.Label, Description: o.Description}
			}
			questions = append(questions, &model.AskUserQuestion{
				Header:      q.Header,
				Question:    q.Question,
				MultiSelect: q.MultiSelect,
				Options:     opts,
			})
		}
		pending = &model.PendingInteraction{Type: itype, PlanContent: planContent, Questions: questions}
	}

	var sysStatus *string
	if s.SystemStatus != "" {
		sysStatus = &s.SystemStatus
	}

	var effort *string
	if s.Effort != "" {
		effort = &s.Effort
	}

	var workDir *string
	if s.WorkDir != "" {
		workDir = &s.WorkDir
	}

	subagents := make([]*model.SubagentActivity, len(s.SubagentActivities))
	for i, sa := range s.SubagentActivities {
		subagents[i] = &model.SubagentActivity{
			TaskID:      sa.TaskID,
			Index:       sa.Index,
			Description: sa.Description,
			CurrentTool: sa.CurrentTool,
		}
	}

	quickReplies := make([]string, len(s.QuickReplies))
	copy(quickReplies, s.QuickReplies)

	return &model.AgentSession{
		BeanID:             s.ID,
		AgentType:          s.AgentType,
		Status:             status,
		Messages:           msgs,
		Error:              errPtr,
		Effort:             effort,
		PlanMode:           s.PlanMode,
		ActMode:            s.ActMode,
		SystemStatus:       sysStatus,
		PendingInteraction: pending,
		WorkDir:            workDir,
		SubagentActivities: subagents,
		QuickReplies:       quickReplies,
	}
}

// activeAgentsToModel converts a slice of agent.ActiveAgent to the GraphQL model type.
func activeAgentsToModel(agents []agent.ActiveAgent) []*model.ActiveAgentStatus {
	result := make([]*model.ActiveAgentStatus, len(agents))
	for i, a := range agents {
		var status model.AgentSessionStatus
		switch a.Status {
		case agent.StatusIdle:
			status = model.AgentSessionStatusIdle
		case agent.StatusRunning:
			status = model.AgentSessionStatusRunning
		case agent.StatusError:
			status = model.AgentSessionStatusError
		}
		result[i] = &model.ActiveAgentStatus{
			BeanID: a.BeanID,
			Status: status,
		}
	}
	return result
}

// actionContext provides context for action visibility filtering and prompt generation.
type actionContext struct {
	WorktreeID         string
	WorkDir            string // working directory (worktree path or project root)
	HasChanges         bool   // uncommitted changes or untracked files
	HasNewCommits      bool   // commits ahead of the base branch
	HasUnpushedCommits bool   // commits ahead of the remote tracking branch
	HasConflicts       bool   // rebasing onto base branch would produce conflicts
	MainRepoHasChanges bool   // main repo has uncommitted changes
	MainRepoPath       string // absolute path to the main repo working directory
	PullRequest        *forge.PullRequest
	ForgeCLI           string // "gh", "glab", or "" if no forge detected
	ForgeLoading       bool   // true when forge is detected but PR state hasn't been fetched yet
	IntegrateMode      string // "local" or "pr" — controls which integration buttons are visible
}

// agentActionDef defines a single agent action with its metadata and prompt.
type agentActionDef struct {
	ID          string
	Label       string
	Description string
	// LabelFunc returns a dynamic label based on context. Takes precedence over Label if set.
	LabelFunc func(ctx actionContext) string
	// PromptFunc generates the prompt from the full action context.
	PromptFunc func(ctx actionContext) string
	// Visible determines whether this action should appear. If nil, always visible.
	Visible func(ctx actionContext) bool
	// Disabled returns a reason string if the action should be shown but not executable.
	// If nil or returns "", the action is enabled.
	Disabled func(ctx actionContext) string
}

// agentActions is the single registry of all available agent actions.
var agentActions = []agentActionDef{
	{
		ID:          "commit",
		Label:       "Commit",
		Description: "Create a git commit",
		PromptFunc:  commitPrompt,
	},
	{
		ID:          "review",
		Label:       "Review",
		Description: "Ask for a code review",
		PromptFunc: func(_ actionContext) string {
			return "Ask a subagent for a thorough code review."
		},
	},
	{
		ID:          "tests",
		Label:       "Tests",
		Description: "Create or update tests for the changes in this branch",
		PromptFunc: func(_ actionContext) string {
			return "Create or amend tests for the changes we've made in this branch. Then run the project's test suite and fix any failures."
		},
		Visible: func(ctx actionContext) bool {
			return ctx.HasChanges || ctx.HasNewCommits
		},
	},
	{
		ID:          "learn",
		Label:       "Learn",
		Description: "Identify learnings to add to repository rules",
		PromptFunc: func(_ actionContext) string {
			return "Identify things that we learned during this session and that should be added to this repository's rules files."
		},
		Visible: func(ctx actionContext) bool {
			return ctx.HasChanges || ctx.HasNewCommits
		},
	},
	{
		ID:          "integrate",
		Label:       "Integrate",
		Description: "Commit, complete any associated beans, and squash-merge into main",
		PromptFunc: func(ctx actionContext) string {
			return fmt.Sprintf(`Squash-merge this worktree's work into main. All commits from this branch must be combined into a single commit on main. Follow these steps in order:

CRITICAL SAFETY RULES — READ BEFORE DOING ANYTHING:
- NEVER run "git push" to origin/main or any remote. This is a LOCAL-ONLY operation.
- NEVER run "git push --force" or "git push -f" on ANY branch, especially not main.
- NEVER run any push command that targets the main branch on any remote.
- If something goes wrong, STOP and report the error. Do NOT attempt destructive recovery.

1. If there are associated beans, mark them as completed.
2. If there are uncommitted changes, create a commit (following the usual commit guidelines).
3. Squash-merge onto main:
   a. Rebase onto main to incorporate any prior integrations: git rebase main
   b. Squash all commits into one: git reset --soft main && git commit -m "<your message>"
      - Write a single, well-crafted conventional commit message that summarizes all the work done in this branch. Include relevant bean IDs.
   c. Record the squashed commit SHA: SQUASH_SHA=$(git rev-parse HEAD)
   d. Fast-forward main to the squashed commit (this updates main's ref, index, AND working tree):
      git -C %s merge --ff-only $SQUASH_SHA
   e. If the merge fails (e.g. main moved), go back to step (a) and retry.
4. Reset this branch to main so it doesn't appear to diverge: git reset --hard main

REMINDER: Do NOT push anything to any remote. The integrate action is purely local.`, ctx.MainRepoPath)
		},
		Visible: func(ctx actionContext) bool {
			if ctx.IntegrateMode == "pr" {
				return false
			}
			if ctx.PullRequest != nil {
				return false
			}
			return ctx.HasChanges || ctx.HasNewCommits
		},
		Disabled: func(ctx actionContext) string {
			if ctx.HasConflicts {
				return "Branch has merge conflicts with base branch"
			}
			if ctx.MainRepoHasChanges {
				return "Main workspace has uncommitted changes"
			}
			return ""
		},
	},
	{
		ID:          "create-pr",
		Label:       "Create PR",
		Description: "Push branch and create a pull request",
		LabelFunc: func(ctx actionContext) string {
			if ctx.ForgeLoading {
				return "Loading..."
			}
			if ctx.PullRequest == nil {
				return "Create PR"
			}
			if ctx.PullRequest.State == "merged" {
				return "PR Merged" // not visible, but keep a label for completeness
			}
			if ctx.HasChanges || ctx.HasUnpushedCommits {
				return "Update PR"
			}
			// Nothing to push — label reflects PR check state
			switch ctx.PullRequest.Checks {
			case forge.CheckStatusPass:
				if ctx.PullRequest.CanMerge() {
					return "Merge PR"
				}
				return "Merge PR"
			case forge.CheckStatusPending:
				return "Checks Running"
			case forge.CheckStatusFail:
				return "Fix Tests"
			default:
				return "Merge PR"
			}
		},
		PromptFunc: prPrompt,
		Visible: func(ctx actionContext) bool {
			if ctx.IntegrateMode == "local" {
				return false
			}
			if ctx.ForgeCLI == "" {
				return false
			}
			if ctx.PullRequest == nil {
				return ctx.HasChanges || ctx.HasNewCommits
			}
			// PR merged — hide; the destroy worktree button handles cleanup
			if ctx.PullRequest.State == "merged" {
				return false
			}
			// PR exists and not merged — show
			return true
		},
		Disabled: func(ctx actionContext) string {
			if ctx.ForgeLoading {
				return "Checking PR status..."
			}
			if ctx.PullRequest != nil && !ctx.HasChanges && !ctx.HasUnpushedCommits {
				switch ctx.PullRequest.Checks {
				case forge.CheckStatusPending:
					return "CI checks are still running"
				case forge.CheckStatusPass:
					if !ctx.PullRequest.Mergeable {
						return "PR has merge conflicts or branch protection requirements not met"
					}
				}
			}
			return ""
		},
	},
}

// prPrompt generates a state-specific prompt for the PR action based on the current context.
// Instead of a single catch-all prompt that could lead to unintended merges, each state
// gets a focused prompt that only instructs the agent to perform the relevant action.
func prPrompt(ctx actionContext) string {
	cli := ctx.ForgeCLI

	// No PR exists yet — create one
	if ctx.PullRequest == nil {
		return fmt.Sprintf(`Create a pull request for this branch.

1. If there are uncommitted changes, create a commit first (following conventional commit conventions, include relevant bean IDs).
2. Push the branch: git push -u origin HEAD
3. Create the PR: %s pr create
   - Derive the title from commit messages using conventional commit style. Include relevant bean IDs.
   - Write a clear description summarizing the changes.
4. Report the PR URL when done.`, cli)
	}

	// PR exists, has local changes or unpushed commits — update it
	if ctx.HasChanges || ctx.HasUnpushedCommits {
		return fmt.Sprintf(`Update the existing pull request for this branch.

1. If there are uncommitted changes, create a commit (following conventional commit conventions, include relevant bean IDs).
2. Push: git push
3. If the scope of the PR changed significantly, update the PR title/body using %s pr edit.
4. Report the PR URL when done.

IMPORTANT: Do NOT merge the PR. Only commit and push updates.`, cli)
	}

	// PR exists, everything pushed, checks failing — fix them
	if ctx.PullRequest.Checks == forge.CheckStatusFail {
		return fmt.Sprintf(`The CI checks on this PR are failing. Investigate and fix the failures.

1. Inspect the failed checks: %[1]s pr checks
2. View the failure logs: %[1]s run view --log-failed
3. Fix the issue locally.
4. Run the project's test suite to verify the fix.
5. Commit the fix and push.
6. Report the PR URL when done.

IMPORTANT: Do NOT merge the PR. Only fix the failing checks.`, cli)
	}

	// PR exists, everything pushed, checks pass, mergeable — merge it
	return fmt.Sprintf(`Merge this pull request.

1. Determine the merge strategy: %[1]s repo view --json mergeCommitAllowed,squashMergeAllowed,rebaseMergeAllowed
2. Merge using the appropriate strategy: %[1]s pr merge with the right flag (--squash, --merge, or --rebase).
3. Do NOT switch branches or delete the worktree after merging.
4. Report the PR URL when done.`, cli)
}

// commitPrompt generates a commit prompt. The agent will inspect git state itself.
func commitPrompt(_ actionContext) string {
	return "Create a commit. Examine the current git status and diff, then commit with an appropriate message. If there are non-bean changes, make sure there is an associated bean that is up to date. If the only changes are bean files, describe the bean updates in the commit message."
}

// findAgentAction looks up an action by ID, returning nil if not found.
func findAgentAction(id string) *agentActionDef {
	for i := range agentActions {
		if agentActions[i].ID == id {
			return &agentActions[i]
		}
	}
	return nil
}

// findWorktreePath looks up the worktree filesystem path for a bean.
func (r *Resolver) findWorktreePath(beanID string) (string, error) {
	if r.WorktreeMgr == nil {
		return "", fmt.Errorf("worktree manager not available")
	}
	wts, err := r.WorktreeMgr.List()
	if err != nil {
		return "", fmt.Errorf("list worktrees: %w", err)
	}
	for _, wt := range wts {
		if wt.ID == beanID {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no worktree found for bean %s", beanID)
}
