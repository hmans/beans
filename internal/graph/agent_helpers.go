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
		msgs[i] = &model.AgentMessage{
			Role:    role,
			Content: m.Content,
			Images:  images,
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

// actionContext provides context about the bean for action visibility filtering
// and prompt generation.
type actionContext struct {
	BeanID        string
	BeanStatus    string
	InWorktree    bool
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
		ID:          "start-work",
		Label:       "Start Work",
		Description: "Mark the bean as in-progress and start implementing it",
		PromptFunc: func(ctx actionContext) string {
			return fmt.Sprintf("Mark the bean %s as in-progress and start implementing it.", ctx.BeanID)
		},
		Visible: func(ctx actionContext) bool {
			return ctx.InWorktree && ctx.BeanStatus != "in-progress"
		},
	},
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
		ID:          "integrate",
		Label:       "Integrate",
		Description: "Commit, complete the bean, and rebase onto main",
		PromptFunc: func(ctx actionContext) string {
			return fmt.Sprintf(`Integrate this worktree's work into main. Follow these steps in order:

1. Mark bean %s as completed (update its status).
2. If there are uncommitted changes, create a commit (following the usual commit guidelines).
3. Rebase onto main and fast-forward push — do NOT merge. Do NOT switch to or modify main's working directory (another agent may be working there). Do this from the worktree:
   - Ensure the main repo accepts pushes to checked-out branches: git -C "$(git rev-parse --git-common-dir)/.." config receive.denyCurrentBranch updateInstead
   - Rebase this branch onto main: git rebase main
   - Resolve any rebase conflicts if they arise, then continue with: git rebase --continue
   - Then fast-forward main's branch pointer: git push . HEAD:main
   - This push is fast-forward-only — if another agent integrated first, it will fail safely.
   - If it fails, rebase again onto main (which now includes the other agent's work) and retry the push.`, ctx.BeanID)
		},
		Visible: func(ctx actionContext) bool {
			return ctx.InWorktree && (ctx.HasChanges || ctx.HasNewCommits)
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
	var paths []string
	for _, c := range changes {
		paths = append(paths, c.Path)
		if !strings.HasPrefix(c.Path, ".beans/") {
			allBeans = false
		}
	}

	if allBeans {
		return fmt.Sprintf("Create a commit. The only uncommitted changes are bean files:\n%s\n\nCommit them with an appropriate message describing the bean updates (e.g. status changes, new beans, updated descriptions).",
			strings.Join(paths, "\n"))
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
		if wt.BeanID == beanID {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no worktree found for bean %s", beanID)
}
