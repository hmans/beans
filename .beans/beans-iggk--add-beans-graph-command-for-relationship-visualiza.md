---
# beans-iggk
title: Add beans graph command for relationship visualization
status: draft
type: feature
priority: normal
tags:
    - cli
    - visualization
created_at: 2025-12-07T11:29:37Z
updated_at: 2025-12-13T14:44:14Z
parent: beans-f11p
---


## Summary

Add a `beans graph` command that outputs a visualization of bean relationships.

## Requirements

- Output in DOT format (Graphviz) by default for easy rendering
- Optional ASCII art mode for terminal viewing
- Filter by relationship type (e.g., `--type blocks` to show only blocking relationships)
- Filter by bean (e.g., `beans graph beans-abc` to show only relationships involving that bean)

## Example output

```
beans graph
digraph beans {
  "beans-abc" -> "beans-def" [label="blocks"]
  "beans-ghi" -> "beans-abc" [label="parent"]
}

beans graph --ascii
beans-abc ──blocks──> beans-def
beans-ghi ──parent──> beans-abc
```

## Notes

- DOT output can be piped to `dot -Tpng` for image generation
- Consider coloring nodes by status
- Could integrate with `beans list` as `--graph` flag instead of separate command