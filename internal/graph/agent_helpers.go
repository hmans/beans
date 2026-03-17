package graph

import (
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/agent"
	"github.com/hmans/beans/internal/gitutil"
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
	MainRepoHasChanges bool   // main repo has uncommitted changes
	MainRepoPath       string // absolute path to the main repo working directory
	PullRequest        *forge.PullRequest
	ForgeCLI           string // "gh", "glab", or "" if no forge detected
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
			return ctx.HasChanges || ctx.HasNewCommits
		},
		Disabled: func(ctx actionContext) string {
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
			if ctx.PullRequest != nil {
				if !ctx.HasChanges && !ctx.HasUnpushedCommits && ctx.PullRequest.CanMerge() {
					return "Merge PR"
				}
				return "Update PR"
			}
			return "Create PR"
		},
		PromptFunc: func(ctx actionContext) string {
			cli := ctx.ForgeCLI
			if ctx.PullRequest != nil {
				// Merge PR — everything is pushed, checks pass, ready to merge
				if !ctx.HasChanges && !ctx.HasUnpushedCommits && ctx.PullRequest.CanMerge() {
					return fmt.Sprintf(`The pull request %s is ready to merge. All checks are passing and the PR is approved.

Merge the PR using: %s pr merge %d --squash --delete-branch

After merging, report the merge result.`, ctx.PullRequest.URL, cli, ctx.PullRequest.Number)
				}

				// Update PR — push latest commits
				return fmt.Sprintf(`A pull request already exists for this branch: %s

Push the latest changes to update it:

1. If there are uncommitted changes, create a commit first (following the usual commit guidelines).
2. Push to the remote: git push
3. If the push fails because the remote is ahead, pull with rebase first: git pull --rebase && git push
4. Optionally update the PR title/body if the scope has changed: %s pr edit --title "..." --body "..."`, ctx.PullRequest.URL, cli)
			}

			// No PR yet — create one
			return fmt.Sprintf(`Create a pull request for this branch. Follow these steps:

1. If there are uncommitted changes, create a commit first (following the usual commit guidelines).
2. Push the branch to the remote: git push -u origin HEAD
3. Create the PR using: %s pr create --title "..." --body "..."
   - Derive the PR title from the branch name and commit messages. Use a conventional commit style prefix.
   - Write a meaningful PR body summarizing the changes.
   - Include any relevant bean IDs.
4. Report the PR URL when done.`, cli)
		},
		Visible: func(ctx actionContext) bool {
			return ctx.ForgeCLI != "" && (ctx.HasChanges || ctx.HasNewCommits || (ctx.PullRequest != nil && ctx.PullRequest.CanMerge()))
		},
		Disabled: func(ctx actionContext) string {
			return ""
		},
	},
}

// commitPrompt inspects the working directory to generate an appropriate commit prompt.
func commitPrompt(ctx actionContext) string {
	changes, err := gitutil.FileChanges(ctx.WorkDir)
	if err != nil || len(changes) == 0 {
		return "Create a commit. Examine the git diff and commit the changes with an appropriate message."
	}

	allBeans := true
	for _, c := range changes {
		if !strings.HasPrefix(c.Path, ".beans/") {
			allBeans = false
			break
		}
	}

	if allBeans {
		return "Create a commit. The only uncommitted changes are bean files. Examine them and commit with an appropriate message describing the bean updates (e.g. status changes, new beans, archived beans)."
	}

	return "Create a commit. Make sure there is an associated bean that is up to date, and possibly even marked as completed if you are done with the change. Then only commit changes related to that change."
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
