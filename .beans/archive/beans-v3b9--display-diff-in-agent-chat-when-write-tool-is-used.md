---
# beans-v3b9
title: Display diff in agent chat when Write tool is used
status: completed
type: feature
priority: normal
created_at: 2026-03-11T22:33:29Z
updated_at: 2026-03-12T07:33:29Z
order: zzV
---

When the agent uses the Write tool in the beans-serve UI, display a collapsible unified diff of the change inline in the agent chat messages.

## Summary of Changes

### Backend
- Added `Diff` field to `Message` struct (`internal/agent/types.go`)
- Created `internal/agent/diff.go` with `computeUnifiedDiff()` — shells out to `diff -u` for reliable unified diff output
- Added `extractFilePath()` and `extractFileContent()` helpers to `internal/agent/parse.go`
- Modified `internal/agent/claude.go` to capture old file content when Write tool input streams in, then compute diff when tool input completes
- Updated JSONL persistence (`internal/agent/store.go`) to serialize/deserialize the `diff` field
- Added `diff` field to `AgentMessage` GraphQL type and wired it through `agent_helpers.go`

### Frontend
- Added `diff` field to `AgentMessage` TypeScript interface and GraphQL subscription query
- Updated `AgentMessages.svelte`: tool messages with diffs show a clickable expand/collapse toggle (▸/▾) that reveals a color-coded unified diff block

### Tests
- `diff_test.go`: tests for identical content, new files, and modified content
- `parse_test.go`: tests for `extractFilePath` and `extractFileContent`
