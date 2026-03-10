---
# beans-buwe
title: Activate bean filter input on Cmd-/ or Ctrl-/
status: completed
type: feature
priority: normal
created_at: 2026-03-10T15:02:46Z
updated_at: 2026-03-10T15:16:29Z
---

When the user presses Cmd-/ (macOS) or Ctrl-/ (other platforms) in the web UI, focus the bean filter input field. This provides a quick keyboard shortcut for filtering beans.

## Summary of Changes

Added Cmd+/ (macOS) and Ctrl+/ (other platforms) as keyboard shortcuts to focus the bean filter input in `+page.svelte`. This was a one-line change extending the existing Cmd/Ctrl+F handler to also match the `/` key.
