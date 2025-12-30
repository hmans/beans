# TUI Filter Modal Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a unified filter modal to the TUI that allows filtering beans by status, type, and tags with keyboard-driven toggling.

**Architecture:** Replace the single `tagFilter string` in `listModel` with three filter slices (`statusFilter`, `typeFilter`, `tagFilters`). Create a new `filterPickerModel` that renders a three-column modal with toggle keys. Add a persistent filter bar at the bottom of the list view.

**Tech Stack:** Go, Bubble Tea (bubbletea), Lip Gloss (lipgloss)

**Bean:** beans-kviq

---

## Task 1: Add Filter State to listModel

**Files:**

- Modify: `internal/tui/list.go:117-118`

**Step 1: Replace tagFilter with filter slices**

Replace the single `tagFilter string` field with three slices:

```go
// In listModel struct, replace line 117-118:
// OLD:
// tagFilter string // if set, only show beans with this tag

// NEW:
// Active filters (empty slice = no filtering on that dimension)
statusFilter []string // active status filters
typeFilter   []string // active type filters
tagFilters   []string // active tag filters
```

**Step 2: Update setTagFilter to setFilters**

Replace the filter methods:

```go
// Replace setTagFilter, clearFilter, hasActiveFilter methods:

// setFilters sets all filters at once
func (m *listModel) setFilters(statuses, types, tags []string) {
 m.statusFilter = statuses
 m.typeFilter = types
 m.tagFilters = tags
}

// clearFilters clears all active filters
func (m *listModel) clearFilters() {
 m.statusFilter = nil
 m.typeFilter = nil
 m.tagFilters = nil
}

// hasActiveFilter returns true if any filter is active
func (m *listModel) hasActiveFilter() bool {
 return len(m.statusFilter) > 0 || len(m.typeFilter) > 0 || len(m.tagFilters) > 0
}
```

**Step 3: Update loadBeans to use new filters**

```go
// In loadBeans(), replace lines 170-174:
// OLD:
// var filter *model.BeanFilter
// if m.tagFilter != "" {
//  filter = &model.BeanFilter{Tags: []string{m.tagFilter}}
// }

// NEW:
var filter *model.BeanFilter
if m.hasActiveFilter() {
 filter = &model.BeanFilter{
  Status: m.statusFilter,
  Type:   m.typeFilter,
  Tags:   m.tagFilters,
 }
}
```

**Step 4: Run tests to verify no regression**

Run: `go test ./internal/tui/...`
Expected: All existing tests pass

**Step 5: Commit**

```bash
git add internal/tui/list.go
git commit -m "refactor(tui): replace tagFilter with multi-filter state

- Add statusFilter, typeFilter, tagFilters slices to listModel
- Update loadBeans() to construct BeanFilter from all dimensions
- Rename setTagFilter to setFilters, clearFilter to clearFilters

Refs: beans-kviq"
```

---

## Task 2: Create Filter Picker Model Structure

**Files:**

- Create: `internal/tui/filterpicker.go`

**Step 1: Create the file with types and messages**

```go
package tui

import (
 "sort"

 tea "github.com/charmbracelet/bubbletea"
 "github.com/charmbracelet/lipgloss"
 "github.com/hmans/beans/internal/config"
 "github.com/hmans/beans/internal/ui"
)

// filterAppliedMsg is sent when filters are applied
type filterAppliedMsg struct {
 statuses []string
 types    []string
 tags     []string
}

// closeFilterPickerMsg is sent when the filter picker is cancelled
type closeFilterPickerMsg struct{}

// openFilterPickerMsg requests opening the filter picker
type openFilterPickerMsg struct {
 currentStatuses []string
 currentTypes    []string
 currentTags     []string
 availableTags   []string // sorted alphabetically
}

// filterPickerModel is the model for the filter picker modal
type filterPickerModel struct {
 // Current selections (toggled on/off)
 selectedStatuses map[string]bool
 selectedTypes    map[string]bool
 selectedTags     map[string]bool

 // Available options
 statuses []string // ordered: draft, todo, in-progress, completed, scrapped
 types    []string // ordered: milestone, epic, feature, bug, task
 tags     []string // sorted alphabetically

 // Dimensions
 width  int
 height int
}
```

**Step 2: Add constructor**

```go
// Ordered statuses for display (matches key bindings 1-5)
var filterStatusOrder = []string{"draft", "todo", "in-progress", "completed", "scrapped"}

// Ordered types for display (matches key bindings m, e, f, b, t)
var filterTypeOrder = []string{"milestone", "epic", "feature", "bug", "task"}

func newFilterPickerModel(msg openFilterPickerMsg, width, height int) filterPickerModel {
 // Initialize selected maps from current filters
 selectedStatuses := make(map[string]bool)
 for _, s := range msg.currentStatuses {
  selectedStatuses[s] = true
 }

 selectedTypes := make(map[string]bool)
 for _, t := range msg.currentTypes {
  selectedTypes[t] = true
 }

 selectedTags := make(map[string]bool)
 for _, t := range msg.currentTags {
  selectedTags[t] = true
 }

 return filterPickerModel{
  selectedStatuses: selectedStatuses,
  selectedTypes:    selectedTypes,
  selectedTags:     selectedTags,
  statuses:         filterStatusOrder,
  types:            filterTypeOrder,
  tags:             msg.availableTags,
  width:            width,
  height:           height,
 }
}
```

**Step 3: Add Init and basic Update**

```go
func (m filterPickerModel) Init() tea.Cmd {
 return nil
}

func (m filterPickerModel) Update(msg tea.Msg) (filterPickerModel, tea.Cmd) {
 switch msg := msg.(type) {
 case tea.WindowSizeMsg:
  m.width = msg.Width
  m.height = msg.Height

 case tea.KeyMsg:
  switch msg.String() {
  case "esc":
   return m, func() tea.Msg { return closeFilterPickerMsg{} }

  case "enter":
   return m, m.applyFilters

  case "x":
   // Reset all filters
   m.selectedStatuses = make(map[string]bool)
   m.selectedTypes = make(map[string]bool)
   m.selectedTags = make(map[string]bool)

  // Status toggles (1-5)
  case "1":
   m.toggleStatus("draft")
  case "2":
   m.toggleStatus("todo")
  case "3":
   m.toggleStatus("in-progress")
  case "4":
   m.toggleStatus("completed")
  case "5":
   m.toggleStatus("scrapped")

  // Type toggles (m, e, f, b, t)
  case "m":
   m.toggleType("milestone")
  case "e":
   m.toggleType("epic")
  case "f":
   m.toggleType("feature")
  case "b":
   m.toggleType("bug")
  case "t":
   m.toggleType("task")

  default:
   // Check for tag toggles (A-Z)
   if len(msg.String()) == 1 {
    r := rune(msg.String()[0])
    if r >= 'A' && r <= 'Z' {
     idx := int(r - 'A')
     if idx < len(m.tags) {
      m.toggleTag(m.tags[idx])
     }
    }
   }
  }
 }

 return m, nil
}

func (m *filterPickerModel) toggleStatus(status string) {
 m.selectedStatuses[status] = !m.selectedStatuses[status]
 if !m.selectedStatuses[status] {
  delete(m.selectedStatuses, status)
 }
}

func (m *filterPickerModel) toggleType(typ string) {
 m.selectedTypes[typ] = !m.selectedTypes[typ]
 if !m.selectedTypes[typ] {
  delete(m.selectedTypes, typ)
 }
}

func (m *filterPickerModel) toggleTag(tag string) {
 m.selectedTags[tag] = !m.selectedTags[tag]
 if !m.selectedTags[tag] {
  delete(m.selectedTags, tag)
 }
}

func (m filterPickerModel) applyFilters() tea.Msg {
 statuses := make([]string, 0, len(m.selectedStatuses))
 for s := range m.selectedStatuses {
  statuses = append(statuses, s)
 }

 types := make([]string, 0, len(m.selectedTypes))
 for t := range m.selectedTypes {
  types = append(types, t)
 }

 tags := make([]string, 0, len(m.selectedTags))
 for t := range m.selectedTags {
  tags = append(tags, t)
 }

 return filterAppliedMsg{
  statuses: statuses,
  types:    types,
  tags:     tags,
 }
}
```

**Step 4: Run build to verify syntax**

Run: `go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/tui/filterpicker.go
git commit -m "feat(tui): add filter picker model structure

- Add filterPickerModel with status/type/tag selection state
- Implement key bindings: 1-5 for status, m/e/f/b/t for type, A-Z for tags
- Add x to reset, enter to apply, esc to cancel

Refs: beans-kviq"
```

---

## Task 3: Implement Filter Picker View

**Files:**

- Modify: `internal/tui/filterpicker.go`

**Step 1: Add View method**

```go
func (m filterPickerModel) View() string {
 // Calculate modal dimensions
 modalWidth := min(70, m.width-4)

 // Title
 title := lipgloss.NewStyle().Bold(true).Render("Filter Beans")

 // Build columns
 statusCol := m.renderStatusColumn()
 typeCol := m.renderTypeColumn()

 var content string
 if len(m.tags) > 0 {
  tagCol := m.renderTagColumn()
  content = lipgloss.JoinHorizontal(lipgloss.Top, statusCol, "  ", typeCol, "  ", tagCol)
 } else {
  content = lipgloss.JoinHorizontal(lipgloss.Top, statusCol, "    ", typeCol)
 }

 // Footer
 footer := m.renderFooter()

 // Assemble modal
 modalContent := lipgloss.JoinVertical(lipgloss.Left,
  title,
  "",
  content,
  "",
  footer,
 )

 // Border style
 border := lipgloss.NewStyle().
  Border(lipgloss.RoundedBorder()).
  BorderForeground(ui.ColorPrimary).
  Padding(1, 2).
  Width(modalWidth)

 return border.Render(modalContent)
}

func (m filterPickerModel) renderStatusColumn() string {
 header := lipgloss.NewStyle().Bold(true).Render("STATUS")

 var lines []string
 lines = append(lines, header)
 lines = append(lines, "")

 for i, status := range m.statuses {
  key := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(fmt.Sprintf("[%d]", i+1))

  // Get status color from config
  var statusColor string
  for _, s := range config.DefaultStatuses {
   if s.Name == status {
    statusColor = s.Color
    break
   }
  }

  selected := m.selectedStatuses[status]
  label := m.renderLabel(status, statusColor, selected)
  indicator := m.renderIndicator(selected)

  lines = append(lines, fmt.Sprintf("%s %s %s", key, label, indicator))
 }

 return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m filterPickerModel) renderTypeColumn() string {
 header := lipgloss.NewStyle().Bold(true).Render("TYPE")

 typeKeys := map[string]string{
  "milestone": "m",
  "epic":      "e",
  "feature":   "f",
  "bug":       "b",
  "task":      "t",
 }

 var lines []string
 lines = append(lines, header)
 lines = append(lines, "")

 for _, typ := range m.types {
  key := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(fmt.Sprintf("[%s]", typeKeys[typ]))

  // Get type color from config
  var typeColor string
  for _, t := range config.DefaultTypes {
   if t.Name == typ {
    typeColor = t.Color
    break
   }
  }

  selected := m.selectedTypes[typ]
  label := m.renderLabel(typ, typeColor, selected)
  indicator := m.renderIndicator(selected)

  lines = append(lines, fmt.Sprintf("%s %s %s", key, label, indicator))
 }

 return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m filterPickerModel) renderTagColumn() string {
 header := lipgloss.NewStyle().Bold(true).Render("TAGS")

 var lines []string
 lines = append(lines, header)
 lines = append(lines, "")

 // Show up to 26 tags (A-Z)
 maxTags := min(26, len(m.tags))
 for i := 0; i < maxTags; i++ {
  tag := m.tags[i]
  keyChar := string(rune('A' + i))
  key := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(fmt.Sprintf("[%s]", keyChar))

  selected := m.selectedTags[tag]
  label := m.renderLabel(tag, "", selected)
  indicator := m.renderIndicator(selected)

  lines = append(lines, fmt.Sprintf("%s %s %s", key, label, indicator))
 }

 return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m filterPickerModel) renderLabel(name, color string, selected bool) string {
 style := lipgloss.NewStyle()

 if selected {
  style = style.Bold(true)
  if color != "" {
   style = style.Foreground(ui.ResolveColor(color))
  }
 } else {
  style = style.Foreground(lipgloss.Color("#666"))
 }

 // Pad to consistent width
 return style.Render(fmt.Sprintf("%-12s", name))
}

func (m filterPickerModel) renderIndicator(selected bool) string {
 if selected {
  return lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render("●")
 }
 return " "
}

func (m filterPickerModel) renderFooter() string {
 resetKey := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("[x]")
 resetLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#999")).Render(" reset all")

 enterKey := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("[enter]")
 enterLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#999")).Render(" apply")

 escKey := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("[esc]")
 escLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#999")).Render(" cancel")

 return resetKey + resetLabel + "    " + enterKey + enterLabel + "    " + escKey + escLabel
}
```

**Step 2: Add fmt import**

Add `"fmt"` to the imports at the top of the file.

**Step 3: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/tui/filterpicker.go
git commit -m "feat(tui): implement filter picker view rendering

- Add three-column layout for status/type/tags
- Show active selections with bold + color + dot indicator
- Hide tags column when no tags exist
- Render key hints and footer help

Refs: beans-kviq"
```

---

## Task 4: Wire Filter Picker into App

**Files:**

- Modify: `internal/tui/tui.go`

**Step 1: Add viewFilterPicker to viewState enum**

```go
// In the const block around line 24-35, add:
const (
 viewList viewState = iota
 viewDetail
 viewTagPicker
 viewParentPicker
 viewStatusPicker
 viewTypePicker
 viewBlockingPicker
 viewPriorityPicker
 viewCreateModal
 viewHelpOverlay
 viewFilterPicker  // ADD THIS LINE
)
```

**Step 2: Add filterPicker field to App struct**

```go
// In App struct around line 98, add after tagPicker:
type App struct {
 // ... existing fields ...
 tagPicker      tagPickerModel
 filterPicker   filterPickerModel  // ADD THIS LINE
 // ... rest of fields ...
}
```

**Step 3: Change 'g' key from chord to direct filter open**

Replace the key chord handling (around lines 174-194):

```go
// Replace the entire "g" chord block with direct filter open:
// OLD: if a.pendingKey == "g" { ... } and if msg.String() == "g" { a.pendingKey = "g" ... }

// NEW: Remove the pendingKey check for "g" and handle "g" directly:
case tea.KeyMsg:
 // Clear status messages on any keypress
 a.list.statusMessage = ""
 a.detail.statusMessage = ""

 // Handle 'g' key for filter picker (only in list view, not filtering)
 if a.state == viewList && a.list.list.FilterState() != 1 {
  if msg.String() == "g" {
   return a, func() tea.Msg {
    // Collect available tags sorted alphabetically
    tags := a.collectTagsWithCounts()
    sortedTags := make([]string, len(tags))
    for i, t := range tags {
     sortedTags[i] = t.tag
    }
    sort.Strings(sortedTags)

    return openFilterPickerMsg{
     currentStatuses: a.list.statusFilter,
     currentTypes:    a.list.typeFilter,
     currentTags:     a.list.tagFilters,
     availableTags:   sortedTags,
    }
   }
  }
 }

 // Clear pending key on any other key press
 a.pendingKey = ""
 // ... rest of key handling
```

**Step 4: Add sort import**

Add `"sort"` to the imports at the top of the file.

**Step 5: Handle filter picker messages**

Add message handlers after the existing tag picker handlers (around line 273):

```go
// Remove or comment out the openTagPickerMsg and tagSelectedMsg handlers

case openFilterPickerMsg:
 a.filterPicker = newFilterPickerModel(msg, a.width, a.height)
 a.previousState = a.state
 a.state = viewFilterPicker
 return a, a.filterPicker.Init()

case filterAppliedMsg:
 a.state = viewList
 a.list.setFilters(msg.statuses, msg.types, msg.tags)
 return a, a.list.loadBeans

case closeFilterPickerMsg:
 a.state = viewList
 return a, nil
```

**Step 6: Forward messages to filter picker and add View case**

In the Update switch for forwarding to views (around line 580):

```go
case viewFilterPicker:
 var cmd tea.Cmd
 a.filterPicker, cmd = a.filterPicker.Update(msg)
 return a, cmd
```

In the View method, add the filter picker case:

```go
case viewFilterPicker:
 bgView := a.renderListView()
 modal := a.filterPicker.View()
 return overlayModal(bgView, modal, a.width, a.height)
```

**Step 7: Run build and test**

Run: `go build ./... && go test ./internal/tui/...`
Expected: Build and tests pass

**Step 8: Commit**

```bash
git add internal/tui/tui.go internal/tui/filterpicker.go
git commit -m "feat(tui): wire filter picker into app

- Add viewFilterPicker state and filterPicker field
- Change 'g' key to open filter modal directly (remove gt chord)
- Handle openFilterPickerMsg, filterAppliedMsg, closeFilterPickerMsg
- Render filter picker as modal overlay

Refs: beans-kviq"
```

---

## Task 5: Add Filter Bar to List View

**Files:**

- Modify: `internal/tui/list.go`

**Step 1: Add renderFilterBar method**

```go
// Add after the View method:

func (m listModel) renderFilterBar() string {
 // Status section
 var statusParts []string
 for _, status := range filterStatusOrder {
  active := false
  for _, s := range m.statusFilter {
   if s == status {
    active = true
    break
   }
  }
  statusParts = append(statusParts, m.renderFilterItem(status, "status", active))
 }
 statusSection := lipgloss.JoinHorizontal(lipgloss.Left, statusParts...)

 // Type section
 var typeParts []string
 for _, typ := range filterTypeOrder {
  active := false
  for _, t := range m.typeFilter {
   if t == typ {
    active = true
    break
   }
  }
  typeParts = append(typeParts, m.renderFilterItem(typ, "type", active))
 }
 typeSection := lipgloss.JoinHorizontal(lipgloss.Left, typeParts...)

 // Combine with separator
 separator := ui.Muted.Render(" │ ")
 result := statusSection + separator + typeSection

 // Add active tags if any
 if len(m.tagFilters) > 0 {
  var tagParts []string
  for _, tag := range m.tagFilters {
   tagParts = append(tagParts, m.renderFilterItem(tag, "tag", true))
  }
  tagSection := lipgloss.JoinHorizontal(lipgloss.Left, tagParts...)
  result += separator + tagSection
 }

 return result
}

func (m listModel) renderFilterItem(name, itemType string, active bool) string {
 style := lipgloss.NewStyle()

 if active {
  style = style.Bold(true)
  // Get color based on type
  switch itemType {
  case "status":
   for _, s := range config.DefaultStatuses {
    if s.Name == name {
     style = style.Foreground(ui.ResolveColor(s.Color))
     break
    }
   }
  case "type":
   for _, t := range config.DefaultTypes {
    if t.Name == name {
     style = style.Foreground(ui.ResolveColor(t.Color))
     break
    }
   }
  case "tag":
   style = style.Foreground(ui.ColorPrimary)
  }
 } else {
  style = style.Foreground(lipgloss.Color("#555"))
 }

 return style.Render(name) + " "
}
```

**Step 2: Add filterStatusOrder and filterTypeOrder variables**

At the top of list.go (after imports):

```go
// Ordered statuses for filter bar display
var filterStatusOrder = []string{"draft", "todo", "in-progress", "completed", "scrapped"}

// Ordered types for filter bar display
var filterTypeOrder = []string{"milestone", "epic", "feature", "bug", "task"}
```

Note: These duplicate the ones in filterpicker.go. We could move to a shared location later, but for now keep them separate to avoid circular imports.

**Step 3: Modify View to include filter bar**

Find the View method and add the filter bar above the help footer. The View method renders the list with a border. We need to add the filter bar between the list content and the closing border.

```go
// In the View method, before returning, add the filter bar:
// Find where the footer/help is rendered and add filter bar above it

func (m listModel) View() string {
 // ... existing view logic ...

 // Add filter bar
 filterBar := m.renderFilterBar()

 // Combine: list + filter bar + help
 // ... adjust the layout to include filterBar
}
```

The exact modification depends on the current View structure. Look at the current implementation and insert the filter bar appropriately.

**Step 4: Run build and manual test**

Run: `go build ./... && ./beans tui`
Expected: Filter bar appears at bottom of list, all items dimmed when no filters active

**Step 5: Commit**

```bash
git add internal/tui/list.go
git commit -m "feat(tui): add persistent filter bar to list view

- Add renderFilterBar method showing all statuses and types
- Active filters shown bold with configured colors
- Inactive filters shown dimmed
- Active tags shown after type section

Refs: beans-kviq"
```

---

## Task 6: Remove Old Tag Picker

**Files:**

- Delete: `internal/tui/tagpicker.go`
- Modify: `internal/tui/tui.go`

**Step 1: Remove tagpicker.go file**

```bash
rm internal/tui/tagpicker.go
```

**Step 2: Remove tagPicker field from App struct**

In `tui.go`, remove the `tagPicker tagPickerModel` field from the App struct.

**Step 3: Remove viewTagPicker from viewState enum**

Remove `viewTagPicker` from the const block.

**Step 4: Remove tagWithCount type if only used by tag picker**

Check if `tagWithCount` is still needed (it's used by `collectTagsWithCounts`). If still needed, keep it. Otherwise remove.

**Step 5: Remove openTagPickerMsg, tagSelectedMsg, clearFilterMsg types**

These message types are no longer needed since we use the unified filter picker.

**Step 6: Clean up any remaining references**

Search for any remaining references to the removed types and remove them.

**Step 7: Run build and test**

Run: `go build ./... && go test ./internal/tui/...`
Expected: Build and tests pass

**Step 8: Commit**

```bash
git add -A
git commit -m "refactor(tui): remove old tag picker

- Delete tagpicker.go (replaced by unified filter modal)
- Remove viewTagPicker state and tagPicker field
- Remove openTagPickerMsg, tagSelectedMsg message types
- Keep collectTagsWithCounts for filter picker

BREAKING CHANGE: 'gt' key chord no longer works, use 'g' for filter modal

Refs: beans-kviq"
```

---

## Task 7: Update Help Overlay

**Files:**

- Modify: `internal/tui/help.go`

**Step 1: Update help text to show 'g' for filter**

Find the help overlay content and update the key bindings:

```go
// Change:
// "g t" → "filter by tag"
// To:
// "g" → "filter"
```

**Step 2: Run manual test**

Run: `./beans tui`
Press `?` to open help overlay
Expected: Shows "g" for "filter" instead of "g t" for "filter by tag"

**Step 3: Commit**

```bash
git add internal/tui/help.go
git commit -m "docs(tui): update help overlay for new filter key

- Change 'g t' to 'g' for filter modal

Refs: beans-kviq"
```

---

## Task 8: Add Integration Test

**Files:**

- Modify: `internal/tui/list_test.go` or create `internal/tui/filterpicker_test.go`

**Step 1: Write test for filter state**

```go
func TestFilterPickerModel_ToggleStatus(t *testing.T) {
 msg := openFilterPickerMsg{
  currentStatuses: []string{},
  currentTypes:    []string{},
  currentTags:     []string{},
  availableTags:   []string{"frontend", "backend"},
 }
 m := newFilterPickerModel(msg, 80, 24)

 // Initially no statuses selected
 if len(m.selectedStatuses) != 0 {
  t.Errorf("expected 0 selected statuses, got %d", len(m.selectedStatuses))
 }

 // Toggle todo (key "2")
 m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
 if !m.selectedStatuses["todo"] {
  t.Error("expected 'todo' to be selected after pressing '2'")
 }

 // Toggle again to deselect
 m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
 if m.selectedStatuses["todo"] {
  t.Error("expected 'todo' to be deselected after pressing '2' again")
 }
}

func TestFilterPickerModel_ApplyFilters(t *testing.T) {
 msg := openFilterPickerMsg{
  currentStatuses: []string{"todo"},
  currentTypes:    []string{"feature"},
  currentTags:     []string{},
  availableTags:   []string{},
 }
 m := newFilterPickerModel(msg, 80, 24)

 // Verify initial state from msg
 if !m.selectedStatuses["todo"] {
  t.Error("expected 'todo' to be pre-selected")
 }
 if !m.selectedTypes["feature"] {
  t.Error("expected 'feature' to be pre-selected")
 }

 // Apply and check message
 result := m.applyFilters()
 applied, ok := result.(filterAppliedMsg)
 if !ok {
  t.Fatal("expected filterAppliedMsg")
 }

 if len(applied.statuses) != 1 || applied.statuses[0] != "todo" {
  t.Errorf("expected statuses=[todo], got %v", applied.statuses)
 }
 if len(applied.types) != 1 || applied.types[0] != "feature" {
  t.Errorf("expected types=[feature], got %v", applied.types)
 }
}

func TestFilterPickerModel_Reset(t *testing.T) {
 msg := openFilterPickerMsg{
  currentStatuses: []string{"todo", "in-progress"},
  currentTypes:    []string{"bug"},
  currentTags:     []string{"urgent"},
  availableTags:   []string{"urgent", "backend"},
 }
 m := newFilterPickerModel(msg, 80, 24)

 // Verify selections exist
 if len(m.selectedStatuses) != 2 {
  t.Errorf("expected 2 selected statuses, got %d", len(m.selectedStatuses))
 }

 // Press 'x' to reset
 m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

 if len(m.selectedStatuses) != 0 {
  t.Errorf("expected 0 selected statuses after reset, got %d", len(m.selectedStatuses))
 }
 if len(m.selectedTypes) != 0 {
  t.Errorf("expected 0 selected types after reset, got %d", len(m.selectedTypes))
 }
 if len(m.selectedTags) != 0 {
  t.Errorf("expected 0 selected tags after reset, got %d", len(m.selectedTags))
 }
}
```

**Step 2: Run tests**

Run: `go test ./internal/tui/... -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/tui/filterpicker_test.go
git commit -m "test(tui): add filter picker unit tests

- Test toggle status/type selection
- Test apply filters returns correct message
- Test reset clears all selections

Refs: beans-kviq"
```

---

## Task 9: Manual Testing & Polish

**Step 1: Build and run TUI**

```bash
go build -o beans . && ./beans tui
```

**Step 2: Test filter workflow**

1. Press `g` to open filter modal
2. Press `2` to toggle "todo", verify dot appears
3. Press `3` to toggle "in-progress", verify dot appears
4. Press `f` to toggle "feature", verify dot appears
5. Press `Enter` to apply
6. Verify list shows only todo/in-progress features
7. Verify filter bar shows active filters bold/colored
8. Press `g` again, verify selections are preserved
9. Press `x` to reset, press `Enter`
10. Verify all beans visible again, filter bar all dimmed

**Step 3: Test edge cases**

1. Open filter with no tags in project → tags column should be hidden
2. Apply filter with nothing selected → should show all beans
3. Press `Esc` to cancel → should not change filters

**Step 4: Fix any issues found**

Address any visual or functional issues discovered during testing.

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore(tui): polish filter modal implementation

Refs: beans-kviq"
```

---

## Task 10: Update Bean Status

**Step 1: Mark bean as completed**

```bash
beans update beans-kviq -s completed
```

**Step 2: Commit bean update**

```bash
git add .beans/
git commit -m "chore: mark beans-kviq as completed

TUI filter modal implementation complete.

Refs: beans-kviq"
```
