# What we're building

Beans is an agentic-first issue tracker. Issues ("beans") live as markdown files in a `.beans/` directory inside a project repo. The system has three interfaces:

- **CLI** (`beans` binary) — create, list, update, and query beans from the terminal
- **Terminal TUI** — Bubbletea-based interactive interface
- **Beans UI** (`beans serve`) — SvelteKit SPA served by an embedded Go HTTP server, communicating via GraphQL (queries, mutations, subscriptions over WebSocket)

The Beans UI is the primary development focus. It includes a backlog board, per-bean agent chat (spawning Claude Code processes), git worktree management, file change diffs, and terminal sessions.

# Commits

- Use conventional commit messages ("feat", "fix", "chore", etc.) when making commits.
- Include the relevant bean ID(s) in the commit message (please follow conventional commit conventions, e.g. `Refs: bean-xxxx`).
- Mark commits as "breaking" using the `!` notation when applicable (e.g., `feat!: ...`).
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.

# Pull Requests

- When we're working in a PR branch, make separate commits, and update the PR description to reflect the changes made.
- Include the relevant bean ID(s) in the PR title (please follow conventional commit conventions, e.g. `Refs: bean-xxxx`).

# Project Structure

Key packages:

- `pkg/bean/` — Bean data model, parsing, sorting, validation (no I/O)
- `pkg/beancore/` — Core engine: disk I/O, file watching, search indexing, worktree watching
- `internal/graph/` — GraphQL schema and resolvers (the API layer for both Beans UI and CLI)
- `internal/agent/` — Agent session manager: spawns Claude Code processes, parses JSONL output, pub/sub for real-time updates
- `internal/worktree/` — Git worktree lifecycle management
- `internal/terminal/` — PTY session management for embedded terminals
- `internal/commands/` — CLI command implementations (Cobra)
- `frontend/` — SvelteKit SPA (embedded into the Go binary via `//go:embed`)

## GraphQL

- When making changes to the GraphQL schema (`internal/graph/schema.graphqls`), run `mise codegen` to regenerate both backend (`generated.go`) and frontend (`frontend/src/lib/graphql/generated.ts`) types.
- When adding or changing frontend GraphQL operations (queries, mutations, subscriptions), update `frontend/src/lib/graphql/operations.graphql` and run `mise codegen`. Do NOT use inline `gql` strings — all operations must go through codegen for type safety.
- All CLI commands that interact with beans should internally use GraphQL queries/mutations against the local server.
- Subscriptions use WebSocket transport. The `beanChanged` subscription supports `includeInitial: true` to send a full snapshot on connect, avoiding race conditions between initial load and live updates.

## Build

- `mise build` to build a `./beans` executable
- The frontend is built and embedded into the Go binary at compile time

# GraphQL Subscriptions

- When a mutation removes or clears state (e.g., deleting a session), the subscription resolver must still send an explicit "empty" payload to the frontend. Never skip `nil` results with `continue` — the frontend needs to know the state changed.

# Worktree State Architecture

- Git worktrees are created **outside** the main repo, in `~/.beans/worktrees/<project-name>/`. This avoids nested repo confusion and accidental tool/search interference. The location is configurable via `worktree.path` in `.beans.yml`.
- `beans-serve` holds **runtime state** as the authoritative view of all beans. It initializes from main repo disk, then merges in changes from worktrees and the GraphQL API.
- The CLI in a worktree uses the **worktree's local `.beans/`** directory — it does NOT redirect to the main repo. This means worktree agents' bean changes travel with their PR.
- `beans-serve` watches active worktrees' `.beans/` dirs and merges file changes into runtime state as "dirty" (not persisted to main disk).
- The `startWork` mutation uses `WithPersist(false)` — status changes are runtime-only until the PR merges.
- When a PR merges and the bean file lands on main, the main watcher picks it up and the dirty flag clears.
- Each worktree has a **metadata file** (`<id>.meta.json`) stored as a sibling in the worktree root directory (e.g. `~/.beans/worktrees/<project>/<id>.meta.json`). This file persists per-worktree state that must survive server restarts: name, description, allocated port, and last-active timestamp. Use `worktree.Manager.SavePort`/`GetPort` etc. to read and write fields — don't access the file directly.

# Agent Architecture

- The central (main workspace) agent session uses ID `__central__` (defined as `CentralSessionID` in `internal/graph/resolver.go` and `MAIN_WORKSPACE_ID` in `frontend/src/lib/worktrees.svelte.ts`). These must stay in sync — the backend uses this ID to determine work directory and system prompt.
- Worktree agent sessions use the worktree ID as their session ID.

# Extra rules for our own beans/issues

- Use the `idea` tag for ideas and proposals.

# Common failure modes when invoked by agents

A failure analysis of 307 agent-driven `beans` invocations surfaced these recurring traps. Avoid them.

## Multi-line GraphQL mutations break under shell quoting

The single most common failure. Bodies containing `"`, `\n`, `()`, or `'` consistently break zsh's parser inside a `'...'` literal — surfacing as `unmatched "`, `parse error near ')'`, or (when an embedded `\n` splits the argv) `Error: accepts at most 1 argument`.

```bash
# BAD — embedded ) and " trip zsh before the query is ever sent
beans query 'mutation { updateBean(id: "x", input: { bodyMod: { append: "Don'\''t..." } }) { id } }'
```

Use one of these instead:

```bash
# 1. Pipe from stdin (beans query reads stdin when no arg is given)
cat <<'EOF' | beans query
mutation { updateBean(id: "x", input: { bodyMod: { append: "..." } }) { id } }
EOF

# 2. Use --variables for the dynamic parts; keep the mutation string trivial
beans query --variables '{"id":"x","body":"..."}' \
  'mutation($id: ID!, $body: String!) { updateBean(id: $id, input: { bodyMod: { append: $body } }) { id } }'

# 3. Prefer dedicated subcommands (beans update, beans create) — their flags sidestep GraphQL entirely.
```

## `bodyMod.replace` is exact-match — read the body first

Replacements fail with `text not found in body` whenever the on-disk form has drifted (e.g., `- [ ]` vs `[ ]`, rewrapped task lines). Fetch the current body before scripting a replacement:

```bash
beans show <id> --body-only
```

…and copy `old` strings verbatim from that output.

## `beans update` has no `--body-replace-old`/`--body-replace-new` flags

Those are GraphQL `bodyMod` fields, not CLI flags. Real `beans update` body flags: `--body`, `--body-append`, `--body-file`. Run `beans update --help` rather than guessing.

## Don't guess GraphQL filter field names

The `BeanFilter` type uses `parentId` (not `parent`), and it takes a single `String`, not `[String]`. When unsure, check the schema:

```bash
beans query --schema | grep -A 30 'input BeanFilter'
```

## `beans show <id>` errors hard on unknown IDs

If you don't have the exact ID, list first (`beans list --json --ready`, `beans list --json -S "<keyword>"`) rather than guessing.

# Testing

- Always write or update tests for the changes you make.

## Unit Tests

- Run all tests: `mise test`
- Run specific package: `go test ./internal/bean/`
- Use table-driven tests following Go conventions

## E2E Tests

- Write or update Playwright e2e tests for any web UI changes.
- Run e2e tests: `mise test:e2e`
- See `frontend/e2e/` for fixtures, page objects, and specs.

## Manual CLI Testing

- `mise beans` will compile and run the beans CLI. Use it instead of building and running `./beans` manually.
- When testing read-only functionality, feel free to use this project's own `.beans/` directory. But for anything that modifies data, create a separate test project directory. All commands support the `--beans-path` flag to specify a custom path.
