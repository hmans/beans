---
# beans-wcnj
title: Add path traversal guards to file operations
status: todo
type: task
priority: normal
created_at: 2026-03-09T17:01:49Z
updated_at: 2026-03-11T19:54:28Z
order: V7
parent: beans-oe8n
---

Several places use user-influenced paths in filepath.Join() without verifying the result stays within the .beans/ root directory. Locations: (1) tui.go line 479/496 — beanPath from bean objects joined with core.Root(). (2) agent/store.go line 118 — beanID concatenated into conversation file path. (3) worktree.go lines 145/156/166 — beanID used in git branch names. Fix: create a helper function like SafeJoin(root, untrusted) that does filepath.Join + filepath.Clean, then verifies the result has root as a prefix (using filepath.Rel or strings.HasPrefix after cleaning). Apply this everywhere a user-influenced value is used in path construction. For git branch names, validate beanID matches the expected format (alphanumeric + hyphens only) before using it.
