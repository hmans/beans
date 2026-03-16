---
# beans-3ijy
title: Plan not shown after ExitPlanMode when ActMode is true
status: completed
type: bug
priority: normal
created_at: 2026-03-16T18:36:01Z
updated_at: 2026-03-16T18:36:11Z
---

When auto-approving EnterPlanMode, ActMode was not cleared, causing the process to respawn with --dangerously-skip-permissions instead of --permission-mode plan. Later ExitPlanMode was treated as a no-op because blockingInteraction returns nil when ActMode is true.

## Summary of Changes

Fixed autoApproveModeSwitch to toggle ActMode alongside PlanMode. When entering plan mode, ActMode is cleared so the process respawns with --permission-mode plan. When exiting plan mode, ActMode is restored. This ensures blockingInteraction properly intercepts ExitPlanMode and shows the plan for approval.
