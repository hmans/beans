---
title: New beans created outside TUI don't appear automatically
status: completed
type: bug
priority: normal
created_at: 2025-12-12T23:02:58Z
updated_at: 2025-12-13T00:31:29Z
parent: beans-xnp8
---

## Problem

When a modal picker (status, type, or parent) is open in the TUI, changes to beans (new beans created or existing beans updated) are not reflected when returning to the main view.

## Root Cause

The close picker messages (`closeParentPickerMsg`, `closeStatusPickerMsg`, `closeTypePickerMsg`) only restore the previous view state but don't trigger a beans reload. While the picker is open, `beansChangedMsg` may fire but the list isn't refreshed until the user makes a selection (which does trigger reload) or restarts the TUI.

## Fix

Always reload beans when closing any picker modal, regardless of whether a selection was made.