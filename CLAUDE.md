# What we're building

You already know what beans is. This is the beans repository.

# Work Parallelization

ABSOLUTELY CRITICAL: We're aiming to parallelize multiple agents. To this end, please follow these rules:

1. YOU MUST create a git worktree for each agent working on a bean. This way, each agent has its own isolated working directory. Do this inside the `.worktrees/` directory.
2. YOU MUST mark the bean as "in-progress" in the main worktree when an agent starts working on it. This prevents multiple agents from working on the same bean simultaneously.
3. Once the agent completes the bean, YOU MUST mark it as "completed" in the main worktree.

# Commits

- Use conventional commit messages ("feat", "fix", "chore", etc.) when making commits.
- Mark commits as "breaking" using the `!` notation when applicable (e.g., `feat!: ...`).
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.

# Pull Requests

- When we're working in a PR branch, make separate commits, and update the PR description to reflect the changes made.

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
- Run all tests: `go test ./...`
- Run specific package: `go test ./internal/bean/`
- Verbose output: `go test -v ./...`
- Use table-driven tests following Go conventions

## Manual CLI Testing

- Use `go run .` instead of building the executable first.
- When testing read-only functionality, feel free to use this project's own `.beans/` directory. But for anything that modifies data, create a separate test project directory. All commands support the `--beans-path` flag to specify a custom path.
