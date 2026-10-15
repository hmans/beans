---
# beans-b5a0
title: 'justfile: add dev and release recipes, drop mise from agent rules and worktree config'
status: completed
type: task
priority: normal
created_at: 2026-10-12T10:25:50Z
updated_at: 2026-10-12T10:41:11Z
---

The justfile lacks the mise tasks dev, dev:serve, dev:kill and release:{patch,minor,major}. .claude/rules/tools.md, .claude/rules/frontend.md and .beans.yml still call mise.

The watcher and svu must work for any contributor who has Go, without a separate install: run both with `go run module@version`, versions pinned in the justfile.

## Todo

- [x] Pin wgo and svu versions in the justfile (`go run`), not in go.mod
- [x] Recipes: dev, dev-serve, dev-kill, release-patch, release-minor, release-major
- [x] Update .claude/rules/tools.md, .claude/rules/frontend.md, .beans.yml
- [x] Verify: just --list, just dev-serve rebuilds on a .go change, release recipe in a throwaway clone, go build ./...
- [x] Review fixes: trap loop and caller kill in `dev`, Run-session ports, fresh-checkout embed, tool versions for Go 1.24, `dev-kill` matching the server

## Summary of Changes

- `justfile`:
  - `dev`: runs `setup`, then `dev-serve` and `dev-frontend` in parallel. One exiting, Ctrl-C, or the Run terminal closing stops both. Each job runs in its own process group, so the caller is never signalled.
  - Ports for `dev`: `BEANS_PORT`/`VITE_PORT`; else, in a beans-serve Run session, `BEANS_WORKSPACE_PORT` for the frontend (the Open link's target) and that port + 1 for the backend; else 22880/5173. `beans-serve`, `dev-serve`, `dev-frontend` read the same variables as defaults, as the mise tasks did. `dev-frontend` uses `--strictPort`, so a taken port fails instead of moving.
  - `dev-serve`: wgo watches non-test `.go` files outside `frontend/` and rebuilds to `/tmp/beans-serve-dev-<port>`, then restarts it; no temporary binary per run.
  - `dev-kill`: kills the wgo watchers and `/tmp/beans-serve-dev*` of every checkout; an installed `beans-serve` keeps running.
  - `release-patch`/`-minor`/`-major`: svu tag, as in mise (no guard for a dirty tree or an already tagged HEAD, also as in mise).
- wgo v0.6.2 and svu v3.2.4 run through `go run module@version`, pinned in the justfile, not as `tool` in go.mod, so they cannot change the beans binary's dependencies. They are the newest versions whose go.mod accepts the Go version beans requires (1.24); newer ones need Go 1.25.
- `.gitignore`: `dist/` → `/dist/`, so the existing `!/internal/web/dist/.gitkeep` exception takes effect and `internal/web/dist/.gitkeep` is committed. A fresh clone or worktree now compiles `beans-serve` (`go:embed dist/*` needs one file) before the frontend is built.
- `.claude/rules/tools.md`, `.claude/rules/frontend.md`: `just` instead of `mise`.
- `.beans.yml`: worktree `setup: just setup`, `run: just dev`.

Verified:
- `just dev` from a non-interactive `bash -c`: both servers up, Vite proxy reaches `/api/graphql`, frontend exit stops the backend, the caller survives.
- In a PTY with `BEANS_WORKSPACE_PORT=23530`: frontend on 23530, backend on 23531; SIGKILL of the session leader (as `Session.Close` does) stops both, no leftovers.
- In an interactive bash: a `.go` edit restarts the server within 2 s, Ctrl-C stops both, the shell survives.
- A fresh clone with only `.gitkeep` in `internal/web/dist` compiles and serves; `dev-kill` stops the dev servers of two checkouts; no `/tmp/wgo_*.exe` is left.
- The pinned tools build with Go 1.24.6 (`GOTOOLCHAIN=go1.24.6`).

Not changed: `.github/workflows/*.yml`, `conductor.json`, `mise.toml` and the README's mise option still use mise; `frontend/e2e/run.sh` prefers mise when present and falls back to just.
