# Beans - Agentic Issue Tracker

This project uses **beans**, an agentic-first issue tracker. Issues are called "beans", and you can
use the "beans" CLI to manage them.

All commands support --json for machine-readable output. Use this flag to parse responses easily.

## Core Rules

- After compaction or clear, run `beans prompt` to re-sync
- Track ALL work using beans (no TodoWrite tool, no markdown TODOs). If you identify something that should be tracked, create a bean for it.
- When I ask you to do something, create a bean for it, give it a useful body/description, keep it updated while you're working on it, and include it in the commit when you make one.
- All bean commands support the `--json` flag for machine-readable output.
- Use `beans create` to create issues, not TodoWrite tool
- When completing work, mark the bean as done using `beans update <bean-id> --status done`. If you're about to create a commit, do this first.
- Lean towards using sub-agents for interacting with beans.
- You can inspect `.beans/config.yaml` to learn about the different issue types and statuses configured for this project.

## Finding work

- `beans list --json` to list all beans (descriptions not included by default)
- `beans list --json --full` to include full description content

## Working on a bean

- `beans update <bean-id> --status in-progress --json` to mark a bean as in-progress
- `beans show <bean-id> --json` to see full details including description
- Adhere to the instructions in the bean's description when working on it

## Relationships

Beans can have relationships to other beans. Use these to express dependencies and connections.

**Adding/removing relationships:**
- `beans update <bean-id> --link blocks:<other-id>` - This bean blocks another
- `beans update <bean-id> --link parent:<other-id>` - This bean has a parent
- `beans update <bean-id> --unlink blocks:<other-id>` - Remove a relationship

**Relationship types:** `blocks`, `duplicates`, `parent`, `relates-to`

**Filtering by relationship:**
- `beans list --linked blocks` - Show all beans that are blocked by something
- `beans list --linked blocks:<id>` - Show beans that `<id>` blocks
- `beans list --linked parent:<id>` - Show beans that have `<id>` as parent
- `beans list --linked blocks,parent` - Comma-separated for multiple filters (OR logic)

## Creating new beans

- `beans create --help`
- Example: `beans create "Fix login bug" -d "Users cannot log in when..." -s open --no-edit`
- When creating new beans, include a useful description. If you're not sure what to write, ask the user.
- Make the description as detailed as possible, similar to a plan that you would create for yourself.
- If possible, split the work into a checklist of GitHub-Formatted-Markdown tasks. Use a `## Checklist` header to precede it.

## Cleaning up beans

- `beans archive` will archive (delete) beans marked as done.
