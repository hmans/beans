# What we're building

You already know what beans is. This is the beans repository.

# Commits

- Use conventional commit messages ("feat", "fix", "chore", etc.) when making commits.
- Include the relevant bean ID(s) in the commit message (a trailing `(bean-xxxx)` is sufficient).
- Mark commits as "breaking" using the `!` notation when applicable (e.g., `feat!: ...`).
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.

# Pull Requests

- When we're working in a PR branch, make separate commits, and update the PR description to reflect the changes made.
- Include the relevant bean ID(s) in the PR title (a trailing `(bean-xxxx)` is sufficient).

# Project Specific

- When making changes to the GraphQL schema, run `mise codegen` to regenerate the code.
- The `internal/graph/` package provides a GraphQL resolver that can be used to query and mutate beans.
- All CLI commands that interact with beans should internally use GraphQL queries/mutations.
- `mise build` to build a `./beans` executable

# Extra rules for our own beans/issues

- Use the `idea` tag for ideas and proposals.

# Testing

## Unit Tests

- Always write or update tests for the changes you make.
- Run all tests: `mise test`
- Run specific package: `go test ./internal/bean/`
- Use table-driven tests following Go conventions

## Manual CLI Testing

- `mise beans` will compile and run the beans CLI. Use it instead of building and running `./beans` manually.
- When testing read-only functionality, feel free to use this project's own `.beans/` directory. But for anything that modifies data, create a separate test project directory. All commands support the `--beans-path` flag to specify a custom path.
