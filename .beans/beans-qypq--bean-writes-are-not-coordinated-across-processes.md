---
# beans-qypq
title: Bean writes are not coordinated across processes
status: todo
type: bug
priority: high
created_at: 2026-10-12T08:42:53Z
updated_at: 2026-10-13T12:01:18Z
parent: beans-5txp
blocked_by:
    - beans-26we
---

Bean writes are protected by `Core.mu`, a `sync.RWMutex`. It orders goroutines inside one process only. Several processes write the same `.beans/` files, and none of them coordinates with the others: there is no file lock and no single writer process. A bean can therefore change on disk between the moment a process reads it and the moment it writes it, and the compare-and-swap in `Core.Update` (beans-o1if) does not notice.

## Who writes the files

| Writer | Own `Core` | Learns of other writers by |
|---|---|---|
| every CLI call | yes: `beancore.New` + `Load()` in `internal/commands/root.go:56` | nothing; it loads once at start |
| `beans serve` | yes, long-lived | file watcher, 100 ms debounce (`pkg/beancore/watcher.go:15`) |
| TUI | yes | `core.Load()` after the editor closes |
| editors, `git checkout`/`rebase`/`merge`, worktree agents | no | — |

The CLI calls `CoreResolver` in its own process; it does not send mutations to a running `beans serve`.

`Core.Update` compares `base` with the bean in its own memory, never with the file. `saveToDisk` writes with `os.WriteFile` (`pkg/beancore/core.go:618`), which truncates and rewrites the file in place.

## Drift cases

Inferred from the code, not reproduced.

1. **Two CLI processes.** A and B load bean X. Each compares against its own load, both swaps succeed, the second write wins. `--if-match` does not help: the check runs against the load, not the file.
2. **CLI and `beans serve`.** The CLI writes X. Before the watcher reloads X, a UI mutation's compare-and-swap succeeds against the stale in-memory X and overwrites the CLI's change on disk.
3. **Torn reads.** The watcher, or another process's `Load()`, reads a file while `os.WriteFile` is still writing it. Parsing fails or reads partial content.
4. **Hand edits and git.** These ignore any lock beans could take. They can only be detected.

## Design

All of it lives in `beancore`, so the CLI, the TUI and `beans serve` get it alike.

1. **Lock.** Every bean write takes an exclusive `flock` on one lock file per `.beans/` directory, not per bean file, so that invariants spanning beans hold too: the cycle check (beans-6we9), the checks of beans-26we, `RemoveLinksTo`, `FixBrokenLinks`.
2. **Refresh the other beans.** A lock alone does not keep cross-bean checks valid. Example: process B checks for a cycle against its in-memory bean Y, process A changed Y on disk earlier, and B stores a cycle although it holds the lock. So under the lock, each process brings its view of the other beans up to date before it checks: (a) `stat` all bean files and reload those whose modification time or size changed, or (b) keep a write counter in the lock file, which every write increments, and reload everything when the counter differs from the last value this process saw. (b) is cheaper but misses hand edits and git; step 3 still detects those for the written bean.
3. **Compare against the file.** Under the lock, re-read the bean file and compare its ETag with `base`. On mismatch return `ETagMismatchError` and refresh the in-memory bean from the file. This applies to a write that persists to the main `.beans/`. A dirty or worktree-linked bean differs from its main file on purpose, so the check must compare against the file the write replaces (the worktree file) or fall back to the in-memory compare.
4. **Atomic write.** Write a temp file in the same directory, then `rename` it over the bean file. The temp file name must not end in `.md`, because both watchers load every `.md` file they see. A rename into the watched directory arrives as `Create`, which the watchers already handle.
5. **Update memory, then release the lock.**

Lock order: `Core.mu`, then the file lock. A process that holds `Core.mu` while it waits for the file lock stalls every goroutine of `beans serve`, readers included, as long as a CLI process holds the file lock. The wait therefore needs a timeout, or the file lock must be taken before `Core.mu`.

Lock file location: `.beans/.lock` is not covered by the `.gitignore` that `beans init` writes, and existing repositories never rewrite that file. The lock file must live where git ignores it, e.g. in a dot-directory that `.gitignore` already lists, or outside the repository.

Windows: `.goreleaser` builds for Windows, which has no `flock`. Use `LockFileEx` behind a build tag, or a library such as `gofrs/flock`. On Windows, `rename` over a file that another process holds open can fail.

Limits:

- `flock` is advisory. Editors and git ignore it; step 3 detects their changes but cannot prevent them.
- A worktree has its own `.beans/`, hence its own lock. `beans serve`'s merged view stays as consistent as the worktree watcher keeps it.
- `flock` is unreliable on some network filesystems.

Alternative: `beans serve` as the single writer, the CLI sending its mutations over GraphQL (as `CLAUDE.md` describes). It needs a running server, and it still does not cover hand edits and git, so it does not replace the lock.

## Todo

- [ ] Reproduce case 1 with two concurrent CLI processes, and case 3 with a reader racing `saveToDisk`
- [ ] Decide the lock granularity: one lock per `.beans/` (proposed) or one per bean file
- [ ] Decide the refresh of step 2: `stat` scan or write counter
- [ ] Decide the lock order and the timeout
- [ ] Decide the lock file location, so that git ignores it in existing repositories
- [ ] Decide the Windows implementation
- [ ] Decide what step 3 compares for dirty and worktree-linked beans
- [ ] Decide what the ETag of step 3 hashes: the rendered form of the re-read bean, or the file bytes (beans-o1if chose the rendered form for reads)
- [ ] `saveToDisk` and `saveToWorktree` write atomically (temp file + `rename`)
- [ ] `Core.Update` takes the lock and compares against the file
- [ ] `Create`, `Delete`, `Archive`, `Unarchive`, `LoadAndUnarchive`, `RemoveLinksTo`, `FixBrokenLinks`, `SaveDirty`, `SaveBean` take the lock
- [ ] Tests: two `Core`s on one directory, concurrent updates with the same `base`, exactly one succeeds; two `Core`s adding crossing links cannot store a cycle; reader never sees a partial file
- [ ] `go test -race ./...`
