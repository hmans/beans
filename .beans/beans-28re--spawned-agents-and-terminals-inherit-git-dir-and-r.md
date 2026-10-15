---
# beans-28re
title: Spawned agents and terminals inherit GIT_DIR and related variables from beans serve
status: scrapped
type: bug
priority: high
created_at: 2026-10-09T14:49:33Z
updated_at: 2026-10-12T07:58:21Z
blocked_by:
    - beans-eudm
---

`beans serve` passes its own environment to the processes it spawns: Claude agents (`internal/agent/claude.go:119`), terminal PTYs (`internal/terminal/terminal.go:286`, `:340`), and the git commands in `internal/worktree`. They select their repository through `cmd.Dir`. If the server starts with `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE` or another variable from `git rev-parse --local-env-vars` set (for example from a hook, `git rebase --exec` or a `!`-alias), an agent working in a worktree acts on that repository instead.

- [ ] Remove the `--local-env-vars` variables from the environment of spawned agents, terminals and git commands
- [ ] Test: with `GIT_DIR` set, a spawned process's `git rev-parse --absolute-git-dir` reports the worktree

## Reasons for Scrapping

The bean was scoped from the test-sandbox incident in beans-eudm by analogy, not from an observed production failure. Analysis on 2026-10-12 found no realistic production trigger.

**What beans does with git.** The CLI and TUI read and watch `.beans/*.md` through the filesystem only; they never call git, apart from one read in `beans init` and in the `mainBranch` query. Beans never commits bean files; the user or an agent does. `beans serve` is the only component that runs git on its own:

- read-only: worktree listing, diffs and status badges, forge detection, linking beans to worktrees (`DetectBeanIDs`)
- writes: `git worktree add` and `git worktree remove --force`, the discard-change action (`checkout`, `clean -f`, `reset`), and `git fetch` of the base ref
- spawned agents and terminals inherit its environment; agent prompts ask for commit, rebase, `reset --hard` and push

The exposure described above is therefore real: with `GIT_DIR` set, every one of these acts on the repository that `GIT_DIR` names.

**When git sets the variables.** Checked with git 2.43:

| Context | Variables set |
|---|---|
| `!`-alias | `GIT_PREFIX`, `GIT_CONFIG_PARAMETERS`; `GIT_DIR` only with `git --git-dir=...` |
| hook in the main worktree | `GIT_INDEX_FILE` |
| hook or `rebase --exec` in a linked worktree | `GIT_DIR`, and `GIT_INDEX_FILE` for hooks |

These variables reach a process only if it starts inside a git operation. `beans serve` is a long-running server that people start from a shell, not from a hook or from `rebase --exec`. The one path that did start it from git, the e2e suite under `rebase --exec`, is closed by `isolateGit()` in beans-eudm.

**What remains.** A user who exports `GIT_DIR` in their shell, for example for a bare-repo dotfiles setup, has asked git for that repository on purpose. If beans silently stripped the variable, that would surprise them as well. This case does not justify the change.
