package graph

import (
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/agent"
	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/internal/graph/model"
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
	WorktreeID    string
	WorkDir       string // working directory (worktree path or project root)
	HasChanges    bool   // uncommitted changes or untracked files
	HasNewCommits bool   // commits ahead of the base branch
}

// agentActionDef defines a single agent action with its metadata and prompt.
type agentActionDef struct {
	ID          string
	Label       string
	Description string
	// PromptFunc generates the prompt from the full action context.
	PromptFunc  func(ctx actionContext) string
	// Visible determines whether this action should appear. If nil, always visible.
	Visible     func(ctx actionContext) bool
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
		Description: "Commit, complete any associated beans, and rebase onto main",
		PromptFunc: func(_ actionContext) string {
			return `Integrate this worktree's work into main. Follow these steps in order:

1. If there are associated beans, mark them as completed.
2. If there are uncommitted changes, create a commit (following the usual commit guidelines).
3. Rebase onto main and fast-forward push — do NOT merge. Do NOT switch to or modify main's working directory (another agent may be working there). Do this from the worktree:
   - Ensure the main repo accepts pushes to checked-out branches: git -C "$(git rev-parse --git-common-dir)/.." config receive.denyCurrentBranch updateInstead
   - Rebase this branch onto main: git rebase main
   - Resolve any rebase conflicts if they arise, then continue with: git rebase --continue
   - Then fast-forward main's branch pointer: git push . HEAD:main
   - This push is fast-forward-only — if another agent integrated first, it will fail safely.
   - If it fails, rebase again onto main (which now includes the other agent's work) and retry the push.`
		},
		Visible: func(ctx actionContext) bool {
			return ctx.HasChanges || ctx.HasNewCommits
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
