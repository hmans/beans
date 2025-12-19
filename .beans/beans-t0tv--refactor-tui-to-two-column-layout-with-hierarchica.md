---
# beans-t0tv
title: Refactor TUI to two-column layout with hierarchical navigation
status: todo
type: feature
created_at: 2025-12-14T15:37:22Z
updated_at: 2025-12-14T15:37:22Z
parent: beans-f11p
---

## Summary

Refactor the TUI to a two-column format:

- **Left pane**: List of beans (filterable, navigable)
- **Right pane**: Detail view of the currently highlighted bean

Navigation allows drilling down into bean hierarchies:
- Press Enter on a bean to make it the new "root" - the left pane then shows only that bean's children (and their descendants)
- Some key (Escape? Backspace?) to navigate back up the hierarchy

## Motivation

The current single-list view doesn't provide enough context about individual beans without opening an editor. A two-column layout allows:
- Quick scanning of bean details without leaving the list
- Better understanding of bean hierarchies
- More efficient triage and review workflows

## Requirements

### Layout
- Left pane: Bean list (similar to current view)
- Right pane: Full bean details (title, status, type, priority, tags, body, relationships)
- Responsive sizing (handle terminal width gracefully)

### Navigation
- Up/Down: Move cursor in the list
- Enter: Drill into selected bean (make it the root, show children)
- Escape/Backspace: Navigate back up to parent scope
- Show breadcrumb or indicator of current hierarchy position

### Preserved Functionality
- All filtering (by status, type, priority, tags)
- Status changes (keyboard shortcuts)
- Batch selection and editing
- Opening bean in editor

## Checklist

- [ ] Design the two-column layout structure with Bubbletea
- [ ] Implement left pane (bean list) component
- [ ] Implement right pane (bean detail) component
- [ ] Add responsive width handling between panes
- [ ] Implement Enter to drill into bean hierarchy
- [ ] Implement back navigation (Escape/Backspace)
- [ ] Add breadcrumb/path indicator showing current root
- [ ] Preserve filtering functionality
- [ ] Preserve status change shortcuts
- [ ] Preserve batch selection and editing
- [ ] Preserve editor integration
- [ ] Handle edge cases (no children, deep nesting, narrow terminals)
- [ ] Update help overlay with new keybindings