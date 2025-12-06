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

# Bean structure

- Each bean is a markdown file with front matter.
- The front matter contains metadata about the bean, including:
  - `title`: a human-readable, one-line title for the bean
  - `status`: must be one of the statuses defined in `beans.toml`
  - `created_at`: timestamp of when the bean was created
  - `updated_at`: timestamp of the last update to the bean

# Configuration

The `.beans/beans.toml` file configures the project:

```toml
[beans]
prefix = 'myapp-'        # prefix for generated IDs
id_length = 4            # length of the random ID portion
default_status = 'open'  # status for new beans

[[statuses]]
name = 'open'
color = 'green'

[[statuses]]
name = 'in-progress'
color = 'yellow'

[[statuses]]
name = 'done'
color = 'gray'
archive = true  # cleaned up by `beans archive`
```

Colors can be named (`green`, `yellow`, `red`, `gray`, `blue`, `purple`) or hex codes (`#FF6B6B`).

# CLI Commands

- `beans init` - Initialize a `.beans/` directory
- `beans list` - List all beans
- `beans show <id>` - Show a bean's contents
- `beans create "Title"` - Create a new bean (supports `-d`, `-s`, `--no-edit` flags)
- `beans status <id> <status>` - Change a bean's status
- `beans delete <id>` - Delete a bean
- `beans archive` - Delete all beans with an archive status (`archive = true`)
- `beans check` - Validate `beans.toml` configuration
- `beans prompt` - Output instructions for AI coding agents

All commands support `--json` for machine-readable output.

# Building

- `mise build` to build a `./beans` executable

# Testing

- Use `go run .` instead of building the executable first.
- All commands support the `--beans-path` flag to specify a custom path to the `.beans/` directory. Use this for testing (instead of spamming the real `.beans/` directory).

# Releasing

Releases are managed using **svu** (Semantic Version Utility) via mise tasks. svu is installed automatically via mise.

- `mise release:patch` - Bump patch version (e.g., v0.1.4 → v0.1.5)
- `mise release:minor` - Bump minor version (e.g., v0.1.4 → v0.2.0)
- `mise release:major` - Bump major version (e.g., v0.1.4 → v1.0.0)

These tasks create and push the git tag, which triggers goreleaser to build and publish the release.
