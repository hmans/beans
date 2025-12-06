# Beans - Agentic Issue Tracker

This project uses **beans**, an agentic-first issue tracker. Issues are called "beans", and you can
use the "beans" CLI to manage them.

All commands support --json for machine-readable output. Use this flag to parse responses easily.

## Core Rules

- Track ALL work using beans (no TodoWrite tool, no markdown TODOs)
- Use `beans create` to create issues, not TodoWrite tool
- Never interact with the data inside the `.beans/` directory directly
- After compaction or clear, run `beans prompt` to re-sync
- When completing work, mark the bean as done using `beans status <bean-id> done`

## Finding work

- `beans list --json` to list all beans

## Working on a bean

- `bean status <bean-id> in-progress` to mark a bean as in-progress
- `bean show <bean-id>` to see details about a bean
- Adhere to the instructions in the bean's descriptions when working on it

## Creating new beans

- `beans create --help`
- Example: `beans create "Fix login bug" -d "Users cannot log in when..." -s open --no-edit`
- When creating new beans, include a useful description. If you're not sure what to write, ask the user.
- Make the description as detailed as possible, similar to a plan that you would create for yourself.
- If possible, split the work into a checklist of GitHub-Formatted-Markdown tasks. Use a `## Checklist` header to precede it.
