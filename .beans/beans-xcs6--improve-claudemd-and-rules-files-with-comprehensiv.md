---
# beans-xcs6
title: Improve CLAUDE.md and rules files with comprehensive project documentation
status: completed
type: task
priority: normal
created_at: 2026-03-13T20:27:46Z
updated_at: 2026-03-13T20:27:55Z
---

Flesh out CLAUDE.md with a proper project description and architecture overview. Add path-scoped rules files for backend Go conventions. Expand frontend.md with state management, theme system, and markdown rendering docs. Fix frontmatter to use 'paths' (correct Claude Code syntax) instead of 'globs' (Cursor syntax).

## Summary of Changes

- **CLAUDE.md**: Replaced vague intro with actual project description; expanded 'Project Specific' into architecture map with package overview, GraphQL workflow, and build instructions
- **backend.md** (new): Package layering diagram, resolver pattern, agent/worktree manager docs, concurrency rules, error handling conventions; scoped to Go files
- **frontend.md**: Added architecture section (routing, state management stores, GraphQL client, optimistic updates), theme system docs, icon usage, reusable UI classes, markdown rendering pipeline; removed redundant Shiki example
- **beans.md**: Added relationship validation, ETag concurrency control, dirty beans docs; updated sorting to mention frontend mirror
- **tools.md**: Expanded from 2 to full mise command inventory
- **claude-cli.md**: Added path scoping
- **All rules files**: Fixed frontmatter from `globs:` (Cursor syntax) to `paths:` (correct Claude Code syntax)
