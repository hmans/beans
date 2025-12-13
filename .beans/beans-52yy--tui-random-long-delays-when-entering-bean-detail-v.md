---
title: 'TUI: Random long delays when entering bean detail view'
status: todo
type: bug
priority: normal
created_at: 2025-12-13T01:27:14Z
updated_at: 2025-12-13T01:27:34Z
---

## Description

The TUI sometimes experiences very long delays (10-20 seconds) when navigating into a bean's detail view. The UI appears to hang until another key is pressed.

## Reproduction

1. Open the TUI with `beans tui`
2. Navigate to a bean and press Enter to view details
3. Occasionally, the view will hang for 10-20 seconds before rendering

## Environment

**Affected terminals:**
- Terminal.app
- Rio Terminal
- Ghostty
- VS Code integrated terminal

## Notes

This appears to affect all tested terminals, making it likely an issue within the TUI itself rather than terminal-specific behavior.