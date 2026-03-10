package graph

import (
	"encoding/json"
	"fmt"

	"github.com/hmans/beans/internal/agent"
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
		msgs[i] = &model.AgentMessage{
			Role:    role,
			Content: m.Content,
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
		case agent.InteractionPermission:
			itype = model.InteractionTypePermissionRequest
		}
		var planContent *string
		if s.PendingInteraction.PlanContent != "" {
			planContent = &s.PendingInteraction.PlanContent
		}
		pending = &model.PendingInteraction{Type: itype, PlanContent: planContent}

		// Populate permission-specific fields from denied tools
		if len(s.PendingInteraction.PermissionDenials) > 0 {
			first := s.PendingInteraction.PermissionDenials[0]
			pending.ToolName = &first.ToolName
			if first.ToolInput != nil {
				inputJSON, _ := json.Marshal(first.ToolInput)
				inputStr := string(inputJSON)
				pending.ToolInput = &inputStr
			}
		}
	}

	var sysStatus *string
	if s.SystemStatus != "" {
		sysStatus = &s.SystemStatus
	}

	return &model.AgentSession{
		BeanID:             s.ID,
		AgentType:          s.AgentType,
		Status:             status,
		Messages:           msgs,
		Error:              errPtr,
		PlanMode:           s.PlanMode,
		YoloMode:           s.YoloMode,
		SystemStatus:       sysStatus,
		PendingInteraction: pending,
	}
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
