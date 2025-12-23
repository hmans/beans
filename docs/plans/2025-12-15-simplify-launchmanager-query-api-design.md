# Simplify LaunchManager Query API

**Date:** 2025-12-15  
**Status:** Approved  
**Bean:** beans-359x

## Problem

LaunchManager currently exposes 5 separate query methods:
- `GetStatus()` - Returns snapshot of all launches
- `IsComplete()` - Checks if all launches finished
- `AllSuccessful()` - Checks if all launches succeeded
- `GetFirstError()` - Returns first failed launch
- `GetCounts()` - Returns counts by status

Each method acquires the lock separately, leading to:
1. **Multiple lock acquisitions** when callers need multiple pieces of information
2. **Potential race conditions** - state can change between calls
3. **Verbose calling code** - callers make 3-4 separate calls in typical usage
4. **Performance overhead** - unnecessary lock contention

Example from `launchprogress.go`:
```go
if m.manager.IsComplete() {           // Lock #1
    if failedLaunch, err := m.manager.GetFirstError(); err != nil {  // Lock #2
        // ...
    }
}
// Later...
pending, running, success, failed := m.manager.GetCounts()  // Lock #3
launches := m.manager.GetStatus()                            // Lock #4
```

## Solution

Replace all query methods with a single `GetSummary()` method that returns a comprehensive snapshot under one lock acquisition.

### Type Definitions

```go
// LaunchCounts represents counts of launches by status
type LaunchCounts struct {
    Pending int
    Running int
    Success int
    Failed  int
    Total   int  // Convenience: pending + running + success + failed
}

// LaunchSummary provides a complete snapshot of launch manager state
type LaunchSummary struct {
    Launches      []*BeanLaunch  // Deep copy of all launches
    Complete      bool           // True if all launches finished
    AllSuccessful bool           // True if all launches succeeded
    FirstError    *BeanLaunch    // First failed launch (nil if none)
    Counts        LaunchCounts   // Status counts
}
```

### Implementation

```go
func (m *LaunchManager) GetSummary() LaunchSummary {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    summary := LaunchSummary{
        Complete:      true,
        AllSuccessful: true,
        Counts: LaunchCounts{
            Total: len(m.launches),
        },
    }
    
    // Deep copy launches and compute all fields in single pass
    summary.Launches = make([]*BeanLaunch, len(m.launches))
    for i, launch := range m.launches {
        // Deep copy
        launchCopy := *launch
        summary.Launches[i] = &launchCopy
        
        // Update counts
        switch launch.Status {
        case LaunchPending:
            summary.Counts.Pending++
            summary.Complete = false
            summary.AllSuccessful = false
        case LaunchRunning:
            summary.Counts.Running++
            summary.Complete = false
            summary.AllSuccessful = false
        case LaunchSuccess:
            summary.Counts.Success++
        case LaunchFailed:
            summary.Counts.Failed++
            summary.AllSuccessful = false
            // Capture first error
            if summary.FirstError == nil && launch.Error != nil {
                summary.FirstError = &launchCopy
            }
        }
    }
    
    return summary
}
```

### Usage Example

Before:
```go
if m.manager.IsComplete() {
    if failedLaunch, err := m.manager.GetFirstError(); err != nil {
        // Handle error
    }
}
pending, running, success, failed := m.manager.GetCounts()
launches := m.manager.GetStatus()
```

After:
```go
summary := m.manager.GetSummary()
if summary.Complete {
    if summary.FirstError != nil {
        // Handle error: summary.FirstError.Error
    }
}
// Use summary.Counts.Pending, .Running, etc.
// Use summary.Launches
```

## Benefits

1. **Single lock acquisition** - All data retrieved under one lock
2. **Consistent snapshot** - Impossible to have inconsistent state between queries
3. **Clearer API** - Single method, obvious what you get
4. **Better performance** - Reduced lock contention
5. **Convenient Total field** - No need to sum counts manually
6. **Simpler calling code** - One call instead of 3-4

## Trade-offs

1. **More computation per call** - Computes all values even if caller only needs one
   - Mitigation: Single loop through launches is fast; callers typically need most values anyway
2. **Slightly more memory allocation** - Returns full struct instead of individual values
   - Mitigation: Struct is small (few pointers + ints); deep copy already happens in GetStatus()

## Migration Strategy

Since this code hasn't been merged yet (still in PR), we can make a clean break:

1. Add new `LaunchCounts` and `LaunchSummary` types
2. Add new `GetSummary()` method
3. Remove old methods entirely (GetStatus, IsComplete, AllSuccessful, GetFirstError, GetCounts)
4. Update all callers to use `GetSummary()`

### Files to Update

- `internal/launcher/manager.go` - Add new types/method, remove old ones
- `internal/tui/launchprogress.go` - Update to use `GetSummary()`
- `internal/launcher/manager_test.go` - Update tests (if they exist)

## Implementation Notes

- Deep copy behavior maintained (same as current `GetStatus()`)
- FirstError returns `*BeanLaunch` (nil if no errors) - caller accesses `.Error` field
- All computation done in single loop for efficiency
- No behavioral changes to existing functionality, just API simplification
