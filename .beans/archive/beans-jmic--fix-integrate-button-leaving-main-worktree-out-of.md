---
# beans-jmic
title: Fix integrate button leaving main worktree out of sync
status: completed
type: bug
created_at: 2026-03-14T15:18:17Z
updated_at: 2026-03-14T15:18:17Z
---

git update-ref moves the ref but does not update main's working directory or index, causing phantom uncommitted changes after integration. Fix: use git merge --ff-only from main's directory instead.
