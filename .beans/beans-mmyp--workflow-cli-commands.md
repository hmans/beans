---
# beans-mmyp
title: Workflow CLI commands
status: todo
type: epic
created_at: 2025-12-27T21:43:38Z
updated_at: 2025-12-27T21:43:38Z
---

Add explicit workflow-style CLI commands that provide intuitive shortcuts for common operations. These commands wrap existing functionality with cleaner, more memorable interfaces.

## Rationale

Currently, users need to use `beans update <id> -s completed` to complete a bean. Workflow commands like `beans complete <id>` are more intuitive and can enforce best practices (like requiring a reason when scrapping).

## Proposed Commands

- `beans complete` - Complete a bean with optional summary
- `beans scrap` - Scrap a bean with required reason  
- `beans start` - Start working on a bean
- `beans ready` - Find beans ready to work on
- `beans next` - Show the next bean to work on
- `beans milestones` - List planned milestones
- `beans blocked` - Show blocked beans
- `beans progress` - Show work progress summary