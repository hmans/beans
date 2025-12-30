---
# beans-ozhq
title: 'Task 4: Update detail.go to accept focus parameters'
status: todo
type: task
created_at: 2025-12-30T16:36:18Z
updated_at: 2025-12-30T16:36:18Z
parent: beans-pn6z
---

## Overview

Update detailModel to accept focus parameters for border rendering, and remove internal linksActive handling (will be controlled by App viewState).

## Files

- Modify: `internal/tui/detail.go`

## Steps

### Step 1: Add focus parameters to detailModel struct

Add fields to track which section should appear focused:

```go
type detailModel struct {
	// ... existing fields ...
	linksFocused bool // true = links section has primary border
	bodyFocused  bool // true = body section has primary border
}
```

### Step 2: Update newDetailModel to accept focus parameters

```go
func newDetailModel(b *bean.Bean, resolver *graph.Resolver, cfg *config.Config, width, height int, linksFocused, bodyFocused bool) detailModel {
	m := detailModel{
		bean:         b,
		resolver:     resolver,
		config:       cfg,
		width:        width,
		height:       height,
		ready:        true,
		linksFocused: linksFocused,
		bodyFocused:  bodyFocused,
	}
	// ... rest unchanged, but remove linksActive initialization ...
```

### Step 3: Update View() to use focus parameters for borders

Replace the border color logic in `View()`:

```go
// Links section (if any)
var linksSection string
if len(m.links) > 0 {
	linksBorderColor := ui.ColorMuted
	if m.linksFocused {
		linksBorderColor = ui.ColorPrimary
	}
	// ... rest unchanged
}

// Body
bodyBorderColor := ui.ColorMuted
if m.bodyFocused {
	bodyBorderColor = ui.ColorPrimary
}
```

### Step 4: Remove linksActive from keyboard handling

Remove the `linksActive` field entirely. The detail model no longer handles Tab internally - the App will control which section is focused via the viewState.

Remove from Update():
- The `case "tab":` handler that toggles linksActive
- Any references to `m.linksActive`

The App will recreate the detailModel with appropriate focus parameters when the user presses Tab.

### Step 5: Update all newDetailModel call sites

Update each call in `tui.go` to pass the new focus parameters. For now, pass `false, false` - correct values will be set when integrating:

**Line ~256 (beansChangedMsg handler):**
```go
a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height, false, false)
```

**Line ~325 (statusSelectedMsg handler):**
```go
a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height, false, false)
```

**Line ~359 (typeSelectedMsg handler):**
```go
a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height, false, false)
```

**Line ~393 (prioritySelectedMsg handler):**
```go
a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height, false, false)
```

**Line ~440 (blockingConfirmedMsg handler):**
```go
a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height, false, false)
```

**Line ~535 (parentSelectedMsg handler):**
```go
a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height, false, false)
```

**Line ~570 (selectBeanMsg handler):**
```go
a.detail = newDetailModel(msg.bean, a.resolver, a.config, a.width, a.height, false, false)
```

### Step 6: Build and verify

Run: `mise build`
Expected: Build succeeds

### Step 7: Commit

```bash
git add internal/tui/detail.go internal/tui/tui.go
git commit -m "refactor(tui): detail accepts focus params, remove linksActive

- Add linksFocused/bodyFocused parameters to detailModel
- Border colors controlled by focus params
- Remove internal Tab handling (App controls focus via viewState)

Refs: beans-pn6z"
```