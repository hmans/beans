---
name: issue-tracking-with-beans
description: Use when starting work, tracking tasks, or deciding where to record discovered work - clarifies when to use TodoWrite vs Beans
---

# Task Tracking Hierarchy

Two systems serve different purposes. Use the right tool for the job.

| System        | Purpose                    | Persistence  | Audience                |
| ------------- | -------------------------- | ------------ | ----------------------- |
| **TodoWrite** | Live progress visibility   | Session only | User                    |
| **Beans**     | Agent memory & audit trail | Git-tracked  | Agents, future sessions |

## When to Use Each System

**TodoWrite** — User-facing progress indicator for the current session:

- Multi-step work (3+ steps) where the user benefits from seeing progress
- Skip for background/non-user-facing work
- Skip for trivial single-step tasks

**Beans** — Persistent agent memory:

- All non-trivial work (3+ steps)
- Work that may span sessions or context boundaries
- Discovered work during implementation
- Anything needing an audit trail
- Skip for trivial single-step tasks (typo fixes, quick lookups)

## Rule: Use Both TodoWrite and Beans Together

For user-facing, non-trivial work:

1. Create a bean first (`beans create ... -s in-progress`)
2. Create a TodoWrite list for live user visibility
3. Update both as you progress
4. TodoWrite items should mirror bean checklist items

For non-user-facing work (background agents, audit-only):

- Use Beans only
- Skip TodoWrite

## Rule: Update Bean Checklists Immediately

After completing each checklist item in a bean:

1. Edit the bean file: `- [ ]` → `- [x]`
2. This creates a recoverable checkpoint if context is lost
3. The I/O overhead is acceptable for persistence

## Rule: Commit Bean Changes With Code

Every code commit includes its associated bean file updates:

```bash
git commit -m "[TYPE] Description" -- src/file.ts .beans/issue-abc123.md
```

This keeps bean state synchronized with codebase state.

## Git Commit Messages

When closing a Beans issue, reference it in the commit:

```
<descriptive message>

Closes beans-1234.
```

## Rule: Discovered Work Goes to Beans

When you discover work during implementation:

1. Create a bean immediately (`--tag discovered --link related:<current-bean>`)
2. Never ignore discovered work due to context pressure
3. Label discovered issues appropriately for later triage

For epic-level discovered work, create the bean with `--type epic`.

## Querying Work

- `beans list --status open` — Find unblocked work to do next
- `beans show <id>` — View issue details including dependencies
