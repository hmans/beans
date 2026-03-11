# What we're building

You already know what beans is. This is the beans repository.

# Commits

- Use conventional commit messages ("feat", "fix", "chore", etc.) when making commits.
- Include the relevant bean ID(s) in the commit message (please follow conventional commit conventions, e.g. `Refs: bean-xxxx`).
- Mark commits as "breaking" using the `!` notation when applicable (e.g., `feat!: ...`).
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.

# Pull Requests

- When we're working in a PR branch, make separate commits, and update the PR description to reflect the changes made.
- Include the relevant bean ID(s) in the PR title (please follow conventional commit conventions, e.g. `Refs: bean-xxxx`).

# Project Specific

- When making changes to the GraphQL schema, run `mise codegen` to regenerate the code.
- The `internal/graph/` package provides a GraphQL resolver that can be used to query and mutate beans.
- All CLI commands that interact with beans should internally use GraphQL queries/mutations.
- `mise build` to build a `./beans` executable

# GraphQL Subscriptions

- When a mutation removes or clears state (e.g., deleting a session), the subscription resolver must still send an explicit "empty" payload to the frontend. Never skip `nil` results with `continue` — the frontend needs to know the state changed.

# Worktree State Architecture

- `beans-serve` holds **runtime state** as the authoritative view of all beans. It initializes from main repo disk, then merges in changes from worktrees and the GraphQL API.
- The CLI in a worktree uses the **worktree's local `.beans/`** directory — it does NOT redirect to the main repo. This means worktree agents' bean changes travel with their PR.
- `beans-serve` watches active worktrees' `.beans/` dirs and merges file changes into runtime state as "dirty" (not persisted to main disk).
- The `startWork` mutation uses `WithPersist(false)` — status changes are runtime-only until the PR merges.
- When a PR merges and the bean file lands on main, the main watcher picks it up and the dirty flag clears.

# Extra rules for our own beans/issues

- Use the `idea` tag for ideas and proposals.

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
