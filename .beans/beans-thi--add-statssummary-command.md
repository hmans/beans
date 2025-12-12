---
title: Add stats/summary command
status: scrapped
type: feature
priority: normal
tags:
    - cli
created_at: 2025-12-07T17:08:36Z
updated_at: 2025-12-12T07:52:19Z
parent: beans-7lmv
---

Add a `beans stats` command that shows a quick summary of project beans.

Implementation:
- Show count by status (open: X, in-progress: Y, done: Z)
- Show count by path/subdirectory
- Show total count
- Support `--json` output for agents
- Optionally show recent activity (beans created/updated recently)

Example output:
```
Status:
  open:        12
  in-progress:  3
  done:        25
  
Total: 40 beans
```