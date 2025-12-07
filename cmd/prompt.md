# Beans - Agentic Issue Tracker

This project uses **beans**, an agentic-first issue tracker. Issues are called "beans", and you can
use the "beans" CLI to manage them.

All commands support --json for machine-readable output. Use this flag to parse responses easily.

## CRITICAL: Track All Work With Beans

**BEFORE starting any task the user asks you to do:**

1. FIRST: Create a bean with `beans create "Title" -t <type> -d "Description..." -s in-progress --no-edit`
2. THEN: Do the work
3. FINALLY: Mark done with `beans update <bean-id> --status done`
4. IF and WHEN you COMMIT: Include both your code changes AND the bean file(s) in the commit!

**Do NOT use the TodoWrite tool or markdown TODOs.** Use beans for all task tracking.

If you identify something that should be tracked during your work, create a bean for it.

## Core Rules

- After compaction or clear, run `beans prompt` to re-sync
- All bean commands support the `--json` flag for machine-readable output.
- Lean towards using sub-agents for interacting with beans.

## Finding work

- `beans list --no-status done --no-linked-as blocks --json` to find actionable beans (not done, not blocked)
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

**Relationship types:** `blocks`, `duplicates`, `parent`, `related`

**Filtering by relationship:**

Outgoing (active) links - use `--links`:

- `beans list --links blocks` - Show beans that block something
- `beans list --links blocks:<id>` - Show beans that block `<id>`
- `beans list --links parent` - Show beans that have a parent

Incoming (passive) links - use `--linked-as`:

- `beans list --linked-as blocks` - Show beans that are blocked by something
- `beans list --linked-as blocks:<id>` - Show beans that `<id>` blocks
- `beans list --linked-as parent:<id>` - Show beans that have `<id>` as parent

Use repeated flags for multiple values: `--links blocks --links parent` (OR logic)

**Excluding by relationship:**

Use `--no-links` and `--no-linked-as` to exclude beans matching a relationship:

- `beans list --no-linked-as blocks` - Show beans NOT blocked by anything (actionable work)
- `beans list --no-links parent` - Show beans without a parent (top-level items)

**Excluding by status:**

Use `--no-status` to exclude beans with specific statuses:

- `beans list --no-status done` - Show beans that are not done
- `beans list --no-status done --no-status archived` - Exclude multiple statuses
- `beans list --no-status done --no-linked-as blocks` - Actionable beans (not done, not blocked)

## Creating new beans

- `beans create --help`
- Example: `beans create "Fix login bug" -t bug -d "Users cannot log in when..." -s open --no-edit`
- **Always specify a type with `-t`**. See the "Issue Types" section below for available types and their descriptions.
- When creating new beans, include a useful description. If you're not sure what to write, ask the user.
- Make the description as detailed as possible, similar to a plan that you would create for yourself.
- If possible, split the work into a checklist of GitHub-Formatted-Markdown tasks. Use a `## Checklist` header to precede it.

## Cleaning up beans

- `beans archive` will archive (delete) beans marked as done. ONLY run this when I explicitly tell you to.

## Bean file structure

Each bean is a markdown file with YAML front matter:

```markdown
---
title: Short descriptive title
status: open
type: feature
created_at: 2024-01-15T10:30:00Z
updated_at: 2024-01-16T14:45:00Z
links:
  - blocks: other-bean-id
  - parent: parent-bean-id
  - blocks: another-blocked-id
---

The body/description goes here as markdown content.
```

**Front matter fields:**

- `title` (required): A human-readable, one-line title
- `status` (required): Must be one of the statuses listed in the "Statuses" section below
- `type` (optional): Must be one of the types listed in the "Issue Types" section below
- `created_at`: Timestamp of creation (auto-managed)
- `updated_at`: Timestamp of last update (auto-managed)
- `links`: Array of relationships to other beans, each as `- type: target-id` (see Relationships section)
