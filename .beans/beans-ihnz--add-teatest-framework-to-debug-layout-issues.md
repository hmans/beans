---
# beans-ihnz
title: Add teatest framework to debug layout issues
status: completed
type: task
priority: normal
created_at: 2025-12-30T20:22:22Z
updated_at: 2025-12-30T20:57:26Z
parent: beans-pn6z
---

## Problem

The linked beans overflow bug (beans-6ljm) is difficult to debug because:
1. Layout calculations involve multiple components (header, links section, body viewport)
2. The bubbles list component has complex internal height calculations
3. Visual inspection requires manual testing at different terminal sizes
4. Regressions are easy to introduce and hard to catch

## Proposal

Add the `teatest` library to create automated tests that:

1. **Capture exact rendered output** at specific terminal dimensions
2. **Golden file comparisons** to detect layout regressions
3. **Programmatic assertions** on layout properties

## Implementation

### 1. Add dependency

```bash
go get github.com/charmbracelet/x/exp/teatest
```

### 2. Create test fixtures

Create a test helper that sets up an App with known test beans:
- Bean with no links (simple case)
- Bean with 1-2 links (fits without scrolling)
- Bean with 5+ links (requires scrolling/pagination)
- Bean with very long titles (tests truncation)

### 3. Write layout tests

```go
func TestDetailLayoutWithManyLinks(t *testing.T) {
    app := createTestAppWithManyLinks(t)
    tm := teatest.NewTestModel(t, app, 
        teatest.WithInitialTermSize(120, 40))
    
    // Navigate to detail view
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Wait for render
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return strings.Contains(string(bts), "Linked Beans")
    })
    
    tm.Send(tea.QuitMsg{})
    
    // Golden file comparison - will fail if layout breaks
    teatest.RequireEqualOutput(t, tm.FinalOutput(t))
}
```

### 4. Test multiple resolutions

- 80x24 (narrow - single pane mode)
- 120x40 (wide - two column mode)
- 160x50 (extra wide)

### 5. Debug workflow

When layout breaks:
1. Run tests with `-update` to capture current (broken) output
2. Inspect the golden file to see exactly what's rendered
3. Fix the calculation
4. Run tests again - golden file shows the fix worked
5. Commit the corrected golden file

## Benefits

- **Reproducible**: Same test, same output, every time
- **Visual diffing**: Golden files show exactly what changed
- **Regression prevention**: CI catches layout breaks before merge
- **Documentation**: Test fixtures document expected behavior

## Files to create/modify

- `internal/tui/tui_test.go` - Add teatest-based layout tests
- `internal/tui/testdata/` - Golden files for expected output
- `go.mod` - Add teatest dependency

## Success criteria

- [ ] teatest dependency added
- [ ] Test helper creates App with configurable test beans
- [ ] Layout tests for narrow and wide terminal modes
- [ ] Tests for beans with 0, few, and many linked beans
- [ ] Golden files committed for all test cases
- [ ] beans-6ljm can be debugged using the test output