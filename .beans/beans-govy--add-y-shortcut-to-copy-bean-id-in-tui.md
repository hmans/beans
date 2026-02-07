---
# beans-govy
title: Add 'y' shortcut to copy bean ID in TUI
status: completed
type: feature
priority: normal
created_at: 2025-12-22T19:57:32Z
updated_at: 2025-12-28T11:30:55Z
---

Implement a 'y' keyboard shortcut in the bean TUI to yank/copy the current bean ID to the clipboard.

## Requirements

- The shortcut should be available in all TUI views where a bean is selected
- Should copy the full bean ID (e.g., "beans-abc123") to the system clipboard
- Provide visual feedback when the ID is copied (e.g., flash message or status bar update)
- Follow Bubbletea conventions for clipboard operations

## Implementation Notes

- Use appropriate clipboard library for cross-platform support
- Consider using the same clipboard mechanism as other copy operations in the TUI (if any exist)
- Update the help/shortcuts display to show the new 'y' shortcut

## Full Implementation Plan

# Implementation Plan: Add 'y' Shortcut to Copy Bean ID in TUI

## Summary

Add a keyboard shortcut 'y' to copy the current bean ID to the clipboard in the TUI's list and detail views. The shortcut will copy the full bean ID (with "beans-" prefix) and show confirmation in the footer status line.

## User Requirements

- Copy only the cursor bean ID (ignore multi-select)
- Include full ID with "beans-" prefix (e.g., "beans-abc123")
- Show footer status message as feedback
- Implement only in list and detail views (not pickers)

## Critical Files

- `internal/tui/list.go` - List view keyboard handling
- `internal/tui/detail.go` - Detail view keyboard handling
- `internal/tui/tui.go` - Main app model and message handling
- `internal/tui/keys.go` - Keyboard binding definitions
- `internal/tui/help.go` - Help overlay text

## Implementation Steps

### 1. Add clipboard dependency ✅

**File:** `go.mod` (verification only)

The `github.com/atotto/clipboard` package is already available as an indirect dependency. Verify it's accessible and import it where needed.

### 2. Define copy message type ✅

**File:** `internal/tui/tui.go` (around line 45-60 with other message types)

Add a new message type to communicate copy actions:

```go
type copyBeanIDMsg struct {
    id string
}
```

### 3. Update list view keyboard handling ✅

**File:** `internal/tui/list.go`

**Location:** In the `Update` method around line 396-425 where keyboard shortcuts are handled

Add 'y' key handling (insert after line 410 with the "e" case):

```go
case "y":
    // Copy bean ID to clipboard
    if item, ok := m.list.SelectedItem().(beanItem); ok {
        return m, func() tea.Msg {
            return copyBeanIDMsg{id: item.bean.ID}
        }
    }
```

**Note:** Unlike other shortcuts, 'y' should work even during filtering - users might want to copy a bean ID they found via search.

**Location:** In the footer `View` method around line 480-515 where help text is rendered

Add 'y' to all three help text variations:

1. Line 483-488 (when beans are selected): Add after "toggle" or before "status"
2. Line 490-501 (when filter is active): Add after "edit" or before "status"
3. Line 503-514 (default state): Add after "edit" or before "status"

Example for default state (around line 506):

```go
helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
```

### 4. Update detail view keyboard handling ✅

**File:** `internal/tui/detail.go`

**Location:** In the `Update` method around line 320-379 where keyboard shortcuts are handled

Add 'y' key handling (insert after the "P" case around line 359):

```go
case "y":
    // Copy bean ID to clipboard
    return m, func() tea.Msg {
        return copyBeanIDMsg{id: m.bean.ID}
    }
```

**Location:** In the footer `View` method around line 447-456 where help text is rendered

Add 'y' to the help text (after "edit" around line 447):

```go
footer += helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
    helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
    helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
```

### 5. Handle copy message in main app ✅

**File:** `internal/tui/tui.go`

**Location:** Add clipboard import at top of file (around line 3-15):

```go
import (
    // ... existing imports ...
    "github.com/atotto/clipboard"
)
```

**Location:** In the `Update` method around line 250-350 where other messages are handled

Add clipboard operation and set status message on the appropriate view model:

```go
case copyBeanIDMsg:
    var statusMsg string
    if err := clipboard.WriteAll(msg.id); err != nil {
        statusMsg = fmt.Sprintf("Failed to copy: %v", err)
    } else {
        statusMsg = fmt.Sprintf("Copied %s to clipboard", msg.id)
    }

    // Set status on current view
    if m.state == viewList {
        m.list.statusMessage = statusMsg
    } else if m.state == viewDetail {
        m.detail.statusMessage = statusMsg
    }

    return m, nil
```

### 6. Clear status messages on keypress ✅

**File:** `internal/tui/tui.go`

**Location:** In the `Update` method, add status clearing in the keyboard handling section (around line 150-200 where tea.KeyMsg is handled)

```go
case tea.KeyMsg:
    // Clear status messages on any keypress
    m.list.statusMessage = ""
    m.detail.statusMessage = ""

    // ... rest of keyboard handling ...
```

### 7. Add status message field to list model and render it ✅

**File:** `internal/tui/list.go`

**Location 1:** Add field to `listModel` struct (around line 20-40):

```go
type listModel struct {
    // ... existing fields ...
    statusMessage string  // Status message to display in footer
}
```

**Location 2:** In the `View` method around line 517, modify the footer to show status message when present:

```go
footer := selectionPrefix
if m.statusMessage != "" {
    // Show status message in place of help text when present
    statusStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
    footer += statusStyle.Render(m.statusMessage)
} else {
    footer += help
}
return content + "\n" + footer
```

### 8. Add status message field to detail model and render it ✅

**File:** `internal/tui/detail.go`

**Location 1:** Add field to `detailModel` struct (around line 20-40):

```go
type detailModel struct {
    // ... existing fields ...
    statusMessage string  // Status message to display in footer
}
```

**Location 2:** In the `View` method around line 453-456, modify the footer to show status message:

```go
footer += helpKeyStyle.Render("esc") + " " + helpStyle.Render("back") + "  " +
    helpKeyStyle.Render("?") + " " + helpStyle.Render("help")

// Prepend status message if present
if m.statusMessage != "" {
    statusStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
    footer = statusStyle.Render(m.statusMessage) + "  " + footer
}

return lipgloss.JoinVertical(lipgloss.Left, header, linksSection, body, footer)
```

### 9. Update help overlay ✅

**File:** `internal/tui/help.go`

**Location:** Around line 82-91 where key bindings are displayed in columns

Add the 'y' shortcut to the appropriate column in the help text:

```go
"y           copy bean id",
```

Place it logically with other single-letter shortcuts.

### 10. Ensure alphabetical ordering ✅

Verified that 'y' is placed alphabetically in all help text displays (list view, detail view, help overlay).

## Testing Plan

### Manual Testing

1. **List view:**
   - Navigate to a bean and press 'y'
   - Verify footer shows "Copied beans-XXXXX to clipboard"
   - Paste in terminal to verify correct ID was copied
   - Test with multi-select active (space bar) - should still copy cursor bean

2. **Detail view:**
   - Open a bean detail with enter
   - Press 'y'
   - Verify footer shows copy confirmation
   - Paste to verify

3. **Help overlay:**
   - Press '?' in both views
   - Verify 'y copy bean id' appears in the help

4. **Edge cases:**
   - Empty list (no beans) - should not crash when pressing 'y'
   - During filtering (/) - 'y' should work and copy the bean under cursor
   - Filtering with no results - 'y' should be safe (no-op if no bean selected)

### Cross-platform Clipboard

The `atotto/clipboard` library should work across Linux, macOS, and Windows. Test on Stefan's Linux system initially.

## Implementation Notes

- Follow existing patterns in the codebase (message-based communication, FilterState checks)
- Keep status messages concise and consistent with existing TUI tone
- The clipboard library might require X11/Wayland on Linux - this should work in Stefan's environment
- Consider whether status message should clear on next keypress or persist until next action
- Match the style and formatting of existing keyboard shortcut handling

## Rollout

After implementation:

1. ✅ Update beans-govy checklist with completed items
2. ✅ Test thoroughly in both views
3. ⏳ Verify cross-platform clipboard works (at minimum on Linux)
   - linux works
4. ⏳ Consider documenting in project README or help docs if there's a shortcuts section

## Files Modified

- `internal/tui/tui.go` - Message handling, clipboard operations, status message clearing
- `internal/tui/list.go` - List view keyboard handling and status display
- `internal/tui/detail.go` - Detail view keyboard handling and status display
- `internal/tui/help.go` - Help overlay text

