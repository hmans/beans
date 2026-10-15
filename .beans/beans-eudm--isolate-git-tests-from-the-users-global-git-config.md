---
# beans-eudm
title: Git tests can escape their temp dirs and modify the real repository
status: completed
type: bug
priority: critical
created_at: 2026-10-02T09:13:45Z
updated_at: 2026-10-09T17:12:50Z
---

The git-based Go tests choose their repository only through the working directory (`cmd.Dir = tmpDir` or `git -C dir`). Git ranks environment variables above the working directory, so the tests are not hermetic in two directions.

## Redirection out of the temp dirs (critical)

When `GIT_DIR` (or `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_COMMON_DIR`, ... - see `git rev-parse --local-env-vars`) is set, every git command of the tests acts on that repository instead of its temp dir. Git sets `GIT_DIR` for hooks, `git rebase --exec` and `!`-aliases.

Observed on 2026-10-09: `git rebase --exec 'go test ./...'` ran the tests against this repository. They

- set `core.bare = true` (`git init --bare` in `initBranchedTestRepo`), breaking every checkout,
- added `core.sshCommand = sleep 60` and remote `hangremote` (`TestFetchBaseRefTimesOut`),
- replaced the repository user with `Test <test@test.com>`,
- moved `main` through 17 test commits, created 21 branches and 16 worktree entries,
- tried `git push -u origin HEAD` against GitHub; only a credential helper reading `.env` from the cwd stopped it.

## Config leaking into the tests

`~/.gitconfig` and `/etc/gitconfig` apply to the tests. With `push.pushOption` set, 10 tests in `internal/gitutil` fail ("the receiving end does not support push options"): TestMergeBase, TestMergeBase_FallbackToRemote, TestAllChangesVsUpstream_{CommittedOnly,CommittedPlusUnstaged,WithUntracked}, TestAllFileDiff_{CommittedFile,UntrackedFile}, TestCommitsBehind_{Zero,NonZero}, TestHasConflicts_NoConflict.

## Fix

One shared test-support function, called from `TestMain` in every package whose tests run git (`internal/gitutil`, `internal/worktree`, `internal/graph`):

- [x] Unset every variable listed by `git rev-parse --local-env-vars`
- [x] Set `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`
- [x] Set `GIT_CEILING_DIRECTORIES` to the temp root, so git never walks up from a temp dir into an enclosing repo
- [x] Set `GIT_ALLOW_PROTOCOL=file`, so tests cannot reach the network
- [x] Wire it into `TestMain` of gitutil, worktree, graph
- [x] Test helpers verify after `git init` that `git rev-parse --absolute-git-dir` lies inside their temp dir, else `t.Fatal`
- [x] `initBranchedTestRepo` pushes to the bare repo by path, not by the name `origin`
- [x] `TestFetchBaseRefTimesOut` allows `ssh` for itself (its `core.sshCommand` is a local `sleep`)
- [x] Regression test: a sentinel repo behind `GIT_DIR` stays unchanged
- [x] E2E fixtures (`frontend/e2e/fixtures.ts`) pass a cleaned environment to git and `beans serve`

## Summary of Changes

- New package `internal/gittest`:
  - `Isolate()` unsets every variable from `git rev-parse --local-env-vars` and sets `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CEILING_DIRECTORIES=<temp root>`, `GIT_ALLOW_PROTOCOL=file`.
  - `RequireGitDirInside(t, dir)` fails the test when git resolves a git directory outside `dir`.
  - Tests for each guarantee; each test turns red when its part of `Isolate` is removed.
- `TestMain` calling `gittest.Isolate()` in `internal/gitutil`, `internal/worktree` and `internal/graph`. Only these three packages run git in tests: a logging `git` on `PATH` recorded 0 calls from `internal/commands`, `internal/tui` and `pkg/forge`.
- Repo helpers (`initTestRepo` in gitutil and worktree, `initBranchedTestRepo`, `TestListFiles`) call `RequireGitDirInside` right after creating a repository.
- `initBranchedTestRepo` pushes to the bare repo by path and then sets the upstream from `origin`.
- `TestFetchBaseRefTimesOut`:
  - allows `ssh` for itself;
  - turns its `core.sshCommand` into `sleep 60 #`, because git appends host and command, and `sleep` rejected them, so the fetch never hung;
  - now also fails when the fetch returns before the timeout.
- `frontend/e2e/fixtures.ts`: `isolateGit()` applies the same environment to the Playwright process at import, so specs, the `beans` CLI and `beans-serve` inherit it.

Verification:

- `go test ./...` passes with the user's global config, which sets `push.pushOption`.
- With `GIT_DIR`/`GIT_WORK_TREE` pointing at a sentinel clone of this repo:
  - `go test ./...` left the sentinel's refs, worktrees and config unchanged;
  - the workspace e2e specs (12 tests) also left it unchanged.
- Control without `isolateGit()`: one e2e spec committed into the sentinel and created branches there.
- Full e2e suite: 62 passed.
