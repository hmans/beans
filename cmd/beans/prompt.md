# Beans - Agentic Issue Tracker

This project uses **beans**, an agentic-first issue tracker. Issues are called "beans", and you can
use the "beans" CLI to manage them.

All commands support --json for machine-readable output. Use this flag to parse responses easily.

## Core Rules

- Track ALL work using beans (no TodoWrite tool, no markdown TODOs). If you identify something that should be tracked, create a bean for it.
- All bean commands support the `--json` flag for machine-readable output.
- Use `beans create` to create issues, not TodoWrite tool
- Never interact with the data inside the `.beans/` directory directly, except for updating an existing bean.
- After compaction or clear, run `beans prompt` to re-sync
- When completing work, mark the bean as done using `beans update <bean-id> --status done`. If you're about to create a commit, do this first.
- Lean towards using sub-agents for interacting with beans

## Finding work

- `beans list --json` to list all beans

## Working on a bean

- `beans update <bean-id> --status in-progress --json` to mark a bean as in-progress
- `beans show <bean-id> --json` to see details about a bean
- Adhere to the instructions in the bean's descriptions when working on it

## Creating new beans

- `beans create --help`
- Example: `beans create "Fix login bug" -d "Users cannot log in when..." -s open --no-edit`
- When creating new beans, include a useful description. If you're not sure what to write, ask the user.
- Make the description as detailed as possible, similar to a plan that you would create for yourself.
- If possible, split the work into a checklist of GitHub-Formatted-Markdown tasks. Use a `## Checklist` header to precede it.
