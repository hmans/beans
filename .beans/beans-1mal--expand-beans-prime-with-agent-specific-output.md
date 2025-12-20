---
# beans-1mal
title: Expand beans prime with agent-specific output
status: todo
type: feature
created_at: 2025-12-20T15:53:29Z
updated_at: 2025-12-20T15:53:29Z
---

Add an optional argument to `beans prime` that allows specifying an agent (e.g., `claude`, `opencode`), which appends agent-specific text to the output.

## Context

Currently `beans prime` outputs generic priming text. Different AI agents may benefit from agent-specific instructions or context that helps them work more effectively with beans.

## Proposed Behavior

- `beans prime` - works as before, outputs generic priming text
- `beans prime claude` - outputs generic text + Claude-specific instructions
- `beans prime opencode` - outputs generic text + OpenCode-specific instructions

## Checklist

- [ ] Add optional positional argument for agent name
- [ ] Define storage location for agent-specific templates (e.g., `extras/` or a new config location)
- [ ] Implement template loading and appending logic
- [ ] Add initial templates for `claude` and `opencode`
- [ ] Update help text and documentation
- [ ] Add tests for the new functionality