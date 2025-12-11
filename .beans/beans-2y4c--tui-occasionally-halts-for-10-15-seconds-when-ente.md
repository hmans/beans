---
title: TUI occasionally halts for 10-15 seconds when entering bean detail view
status: completed
type: bug
priority: critical
created_at: 2025-12-08T18:56:20Z
updated_at: 2025-12-11T20:32:19Z
---

## Problem

The TUI occasionally halts for 10-15 seconds, or sometimes until the escape key is pressed, when entering the bean detail view.

## Expected Behavior

Entering the bean detail view should be instantaneous and responsive.

## Actual Behavior

- TUI freezes for 10-15 seconds when navigating to bean detail view
- Sometimes the freeze persists until the escape key is pressed
- This appears to be intermittent/occasional rather than consistent

## Investigation Needed

- [ ] Review the bean detail view initialization code in the TUI
- [ ] Check for blocking operations (file I/O, parsing) on the main thread
- [ ] Look for potential deadlocks or race conditions
- [ ] Investigate any async operations that might be blocking the UI
- [ ] Check if glamour markdown rendering is causing delays
- [ ] Test with various bean files to identify patterns