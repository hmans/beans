# What we're building

This is going to be a small CLI app that interacts with a .beans/ directory that stores "issues" (like in an issue tracker) as markdown files with front matter. It is meant to be used as part of an AI-first coding workflow.

- This is an agentic-first issue tracker. Issues are called beans.
- Projects can store beans (issues) in a `.beans/` subdirectory.
- The executable built from this project here is called `beans` and interacts with said `.beans/` directory.
- The `beans` command is designed to be used by a coding agent (Claude, OpenCode, etc.) to interact with the project's issues.
- `.beans/` contains markdown files that represent individual beans (flat structure, no subdirectories).
- The individual bean filenames start with a string-based ID (use 3-character NanoID here so things stay mergable), optionally followed by a dash and a short description
  (mostly used to keep things human-editable). Examples for valid names: `f7g.md`, `f7g-user-registration.md`.

# Rules

- ONLY make commits when I explicitly tell you to do so.
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.
- After making a meaningful change that should be mentioned in the changelog, create a change file using `changie new`. (See `changie new --help` for options.)

# Building

- `mise build` to build a `./beans` executable

# Testing

## Unit Tests

- Run all tests: `go test ./...`
- Run specific package: `go test ./internal/bean/`
- Verbose output: `go test -v ./...`
- Use table-driven tests following Go conventions

## Manual CLI Testing

- Use `go run .` instead of building the executable first.
- All commands support the `--beans-path` flag to specify a custom path to the `.beans/` directory. Use this for testing (instead of spamming the real `.beans/` directory).

# Releasing

Releases are managed using **changie** for changelog generation and automatic version detection.

- `mise release` - Release with auto-detected version (prompts for confirmation)

This task detects the appropriate version bump based on changelog entries, shows the version to be released, and asks for confirmation before proceeding. It then creates and pushes the git tag, which triggers goreleaser to build and publish the release.
