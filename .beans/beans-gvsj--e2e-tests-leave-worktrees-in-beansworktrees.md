---
# beans-gvsj
title: E2E tests leave worktrees in ~/.beans/worktrees
status: todo
type: bug
priority: low
created_at: 2026-10-09T18:52:28Z
updated_at: 2026-10-09T18:52:28Z
---

Each e2e test starts `beans-serve` in a temp project without `worktree.path`, so worktrees go to the default `~/.beans/worktrees/<project>/` (`internal/commands/serve.go:121`, `pkg/config/config.go:108`). The test removes its project dir but never this worktree root. Every run leaves `beans-e2e-template-*` directories with worktrees and `.meta.json` files in the user's home; 25 exist from runs on 2026-10-06 and 2026-10-09.

## Fix

- [ ] `frontend/e2e/fixtures.ts` sets `worktree.path` in each test's `.beans.yml` to a dir inside the test's temp project, so the existing `rmSync` removes it
- [ ] Check after an e2e run that `~/.beans/worktrees/` gained no entries
- [ ] Offer the user to delete the existing `beans-e2e-template-*` dirs (outside the repo; do not delete without asking)
