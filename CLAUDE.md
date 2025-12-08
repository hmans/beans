# What we're building

This is a small CLI app that interacts with a .beans/ directory that stores "issues" (like in an issue tracker) as markdown files with front matter. It is meant to be used as part of an AI-first coding workflow.

- This is an agentic-first issue tracker. Issues are called beans.
- Projects configure beans via a `.beans.yml` file at the project root.
- Bean data is stored in a `.beans/` directory (configurable via `beans.path` in `.beans.yml`).
- The executable built from this project here is called `beans` and interacts with said directory.
- The `beans` command is designed to be used by a coding agent (Claude, OpenCode, etc.) to interact with the project's issues.
- `.beans/` contains markdown files that represent individual beans (flat structure, no subdirectories).

# Rules

- ONLY make commits when I explicitly tell you to do so.
- Use conventional commit messages ("feat", "fix", "chore", etc.) when making commits.
- Mark commits as "breaking" using the `!` notation when applicable (e.g., `feat!: ...`).
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.
- When we're working in a PR branch, make separate commits, and update the PR description to reflect the changes made.
- When making changes to the GraphQL schema, run `mise codegen` to regenerate the code.

# GraphQL

- The `internal/graph/` package provides a GraphQL resolver that can be used to query and mutate beans.
- All CLI commands that interact with beans should internally use GraphQL queries/mutations.

# Extra rules for our own beans/issues

- Use the `idea` tag for ideas and proposals.

# Building

- `mise build` to build a `./beans` executable

# Testing

## Unit Tests

- Always write or update tests for the changes you make.
- Run all tests: `go test ./...`
- Run specific package: `go test ./internal/bean/`
- Verbose output: `go test -v ./...`
- Use table-driven tests following Go conventions

## Manual CLI Testing

- Use `go run .` instead of building the executable first.
- When testing read-only functionality, feel free to use this project's own `.beans/` directory. But for anything that modifies data, create a separate test project directory. All commands support the `--beans-path` flag to specify a custom path.
