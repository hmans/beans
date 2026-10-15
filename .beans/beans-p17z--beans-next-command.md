---
# beans-p17z
title: beans next command
status: completed
type: task
priority: normal
created_at: 2025-12-27T21:44:04Z
updated_at: 2026-10-14T16:09:17Z
order: VVg
parent: beans-mmyp
---

Add `beans next`: a ranked list of beans to start next, with the lane and one reason line per row.

## Candidates

The `--ready` set (`beangraph.AddReadyFilter`) plus unblocked drafts, restricted to workable beans: no epics, no milestones, no beans with open children. A child is open unless it is completed or scrapped; a draft child counts as open.

## Fronts

A front is a milestone or epic whose children get focus.

- **Explicit focus** (`next.focus` set): that bean is the only front and replaces the in-progress ones.
- **Otherwise:** every in-progress milestone or epic is a front.
- The nearest front ancestor of a bean decides (an in-progress epic under an in-progress milestone uses the epic).

## Lanes

Each candidate goes into the first lane that matches:

1. **interrupt** – non-draft bug with priority high or critical, anywhere in the tree
2. **focus** – non-draft bean with a front among its ancestors
3. **refine** – draft with a front among its ancestors; the action is to refine it (reason e.g. `refine: draft under epic beans-5txp`). Ranked after focus, so drafts rise to the top once a front has no ready work left.
4. **rest** – everything else, drafts included; shown only with `--all`

## Ordering inside a lane

1. front priority
2. front manual `order`
3. front `updated_at`, most recent first
4. bean priority
5. bean type: bug, feature, task
6. bean manual `order`
7. bean `created_at`, oldest first

Steps 1–3 apply only to beans that have a front. In the interrupt lane, bean priority (step 4) comes first, then steps 1–3, then 5–7, so a critical bug always outranks a high one. Each level uses one attribute: the priority shown on a bean is the priority used for it.

## Output

- Header line names the scope source, e.g. `scope: focus (.beans.local.yml)` or `scope: in-progress fronts`.
- One row per bean: lane, ID, title, reason (e.g. `focus: under epic beans-5txp (in-progress)`).
- Hint when more than one in-progress front exists and no focus is set: `3 active fronts – consider "beans focus set"`.
- If nothing is ready: suggest `beans list --is-blocked` or `beans list`.

Flags:

- `--limit N` – total rows, cut after lane order; default 10, `0` means unlimited.
- `--all` – also show the rest lane.
- `--in-progress` – list in-progress leaf beans to resume instead of the ready candidates: lane `in-progress`, same ordering; exclusive with `--all`.
- `--json` – one object `{scope, hint, warning, rows}`, following `roadmap --json`: `scope` names the scope source and its file, `hint` is the multi-front hint or absent, `warning` names an invalid focus or is absent, each row is `output.Bean` plus `lane` and `reason`.

## Focus setting

- Key `next.focus: <id>`: one ID, which must be a milestone or epic.
- Read from `.beans.yml` and `.beans.local.yml`; the local value replaces the shared one. `.beans.local.yml` carries only `next.focus`; no other key is read from it. It is read also when no `.beans.yml` exists.
- `.beans.local.yml` is per user and gitignored. `beans init` appends `.beans.local.yml` to the project root `.gitignore`: it creates the file when absent, and skips the line when it is already present. With `--beans-path`, init knows no project dir and adds nothing, as it writes no `.beans.yml` there. (`.beans/.gitignore` cannot ignore a file in its parent.)
- Invalid focus: the ID does not exist, is completed or scrapped (also through an ancestor), or is not a milestone or epic. `focus set` refuses it. `next` checks again, because a shared focus can be stale in a branch: when the focus is invalid it falls back to in-progress fronts, with the header `scope: in-progress fronts (focus beans-x invalid: completed)` and the JSON `warning`. `beans focus` shows the same state.
- In a worktree, `.beans.local.yml` is absent (git copies no ignored files), so only the `.beans.yml` focus or the in-progress fronts apply; the scope header shows which.
- CLI writes only `.beans.local.yml`: `beans focus` (show, with source), `beans focus set <id>` (replaces the value), `beans focus clear` (deletes the key from `.beans.local.yml`, so a value in `.beans.yml` applies again).

## Layout

- `pkg/bean/next.go` – ranking as a pure function (no I/O)
- `pkg/config` – read `next.focus` from `.beans.local.yml` over `.beans.yml`, remembering which file it came from
- `internal/commands/next.go`, `internal/commands/focus.go` – CLI, calling the core resolver directly like `beans list`

## Todo

- [x] Ranking in `pkg/bean/next.go` with table-driven tests (lanes incl. refine, front selection, ordering, nested fronts, parent with only a draft child)
- [x] Config: `next.focus` and `.beans.local.yml` merge, with tests
- [x] `beans init` appends `.beans.local.yml` to the root `.gitignore` (append-if-missing), with tests; add the line to this repo's `.gitignore`
- [x] `beans next` command with flags, scope header and multi-front hint
- [x] `beans focus` / `focus set` / `focus clear`
- [x] `prompt.tmpl`: "what to work on next" uses `beans next --json`; lines claiming `ready`/`next`/`start` commands name only existing commands; add a `prime_test.go` string check
- [x] Manual check on this repo: the 3 high bugs under `beans-5txp` rank first

## Out of scope

TUI focus key and GraphQL exposure: see follow-up bean.

## Summary of Changes

- `pkg/bean/next.go`: `RankNext` (lanes interrupt, focus, refine, rest, in-progress; front selection; ordering) and `FocusProblem`, with table-driven tests.
- `pkg/config`: `next.focus` key; `.beans.local.yml` read next to the config dir (only `next.focus`); `Focus()` returns value and source; `SetLocalFocus` / `ClearLocalFocus`; `IgnoreLocalConfig`.
- `beans init` appends `.beans.local.yml` to the project `.gitignore`; this repo's `.gitignore` has the entry.
- `beans next` (`--limit`, `--all`, `--in-progress`, `--json`) and `beans focus` / `focus set` / `focus clear`, with command tests.
- `prompt.tmpl` points agents at `beans next --json`; `prime_test.go` checks that every command the prompt names exists.
- JSON rows also carry `front` (the front's ID), beyond the spec's lane and reason.

## Review (oc-review, ses_ec4e72108ffemt69EKWa47SrOj)

- R1 no front hides all ready work: apply option A (rest lane shown when no front exists)
- R2 `beans focus` lacks `--json` although the prompt promises it: apply, add `--json`
- R3 prime_test misses bare command names and subcommands: apply
- R4 next_test ordering depends on same-second CreatedAt: apply
- R5 prime_test panics with -count=2 (flags registered twice): apply
- R6 `--config` to a missing file ignores and misplaces .beans.local.yml: apply
- R7 multi-front hint counts fronts under a closed ancestor: apply
