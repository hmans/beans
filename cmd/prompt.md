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

Outgoing (active) links - use `--links`:
- `beans list --links blocks` - Show beans that block something
- `beans list --links blocks:<id>` - Show beans that block `<id>`
- `beans list --links parent` - Show beans that have a parent

Incoming (passive) links - use `--linked-as`:
- `beans list --linked-as blocks` - Show beans that are blocked by something
- `beans list --linked-as blocks:<id>` - Show beans that `<id>` blocks
- `beans list --linked-as parent:<id>` - Show beans that have `<id>` as parent

Both support comma-separated values: `--links blocks,parent` (OR logic)

## Creating new beans

- `beans create --help`
- Example: `beans create "Fix login bug" -d "Users cannot log in when..." -s open --no-edit`
- When creating new beans, include a useful description. If you're not sure what to write, ask the user.
- Make the description as detailed as possible, similar to a plan that you would create for yourself.
- If possible, split the work into a checklist of GitHub-Formatted-Markdown tasks. Use a `## Checklist` header to precede it.

## Cleaning up beans

- `beans archive` will archive (delete) beans marked as done.

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
  blocks:
    - other-bean-id
  parent:
    - parent-bean-id
---

The body/description goes here as markdown content.
```

**Front matter fields:**
- `title` (required): A human-readable, one-line title
- `status` (required): Must be one of the statuses defined in `config.yaml`
- `type` (optional): Must be one of the types defined in `config.yaml`
- `created_at`: Timestamp of creation (auto-managed)
- `updated_at`: Timestamp of last update (auto-managed)
- `links`: Relationships to other beans (see Relationships section)

## Configuration

The `.beans/config.yaml` file configures the project:

```yaml
beans:
  prefix: myapp-      # prefix for generated IDs
  id_length: 4        # length of the random ID portion
  default_status: open # status for new beans

statuses:
  - name: open
    color: green
  - name: in-progress
    color: yellow
  - name: done
    color: gray
    archive: true     # cleaned up by `beans archive`

types:
  - name: task
    color: blue
  - name: feature
    color: green
  - name: bug
    color: red
```

Colors can be named (`green`, `yellow`, `red`, `gray`, `blue`, `purple`) or hex codes (`#FF6B6B`).
