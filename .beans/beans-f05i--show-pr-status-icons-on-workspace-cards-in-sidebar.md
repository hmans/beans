---
# beans-f05i
title: Show PR status icons on workspace cards in sidebar
status: completed
type: feature
priority: normal
created_at: 2026-03-17T18:18:15Z
updated_at: 2026-03-17T18:19:28Z
---

In integrate: pr mode, replace the green checkmark on workspace cards with PR status icons: no PR = no icon, checks pending = orange, checks failed = red, checks passed = green, merged = purple.

## Summary of Changes

Modified `frontend/src/lib/components/Sidebar.svelte` to show PR status icons on workspace cards in the sidebar when in `integrate: pr` mode:

- Added `pullRequest` data to the `WorkspaceItem` interface, piped from the worktree subscription
- New PR status icon branch in the status icon logic (before the existing "ready to integrate" check):
  - **Merged**: purple rotated branch icon (`icon-[uil--code-branch]`)
  - **Checks failed**: red X circle (`icon-[uil--times-circle]`)
  - **Checks pending**: orange clock (`icon-[uil--clock]`)
  - **Checks passed**: green check circle (`icon-[uil--check-circle]`)
  - **No PR**: no icon shown (falls through to existing logic)
- Icons are clickable links to the PR URL
- On hover, the destroy button still appears (same pattern as the existing checkmark)
- In `integrate: local` mode, behavior is completely unchanged
