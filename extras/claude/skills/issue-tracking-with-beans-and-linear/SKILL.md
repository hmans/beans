---
name: issue-tracking-with-beans-and-linear
description: Use when starting work, tracking tasks, or deciding where to record discovered work - clarifies when to use TodoWrite vs Beans vs Linear
---

# Task Tracking Hierarchy

Three systems serve different purposes. Use the right tool for the job.

| System        | Purpose                    | Persistence  | Audience                |
| ------------- | -------------------------- | ------------ | ----------------------- |
| **TodoWrite** | Live progress visibility   | Session only | User                    |
| **Beans**     | Agent memory & audit trail | Git-tracked  | Agents, future sessions |
| **Linear**    | Project tracking           | External     | Humans                  |

Linear is for human-visible project tracking. Beans is for agent implementation memory. Both systems work together with bidirectional linking.

## When to Use Each System

**TodoWrite** — User-facing progress indicator for the current session:

- Multi-step work (3+ steps) where the user benefits from seeing progress
- Skip for background/non-user-facing work
- Skip for trivial single-step tasks

**Beans** — Persistent agent memory (only if the project uses Beans):

- All non-trivial work (3+ steps)
- Work that may span sessions or context boundaries
- Discovered work during implementation
- Anything needing an audit trail
- Skip for trivial single-step tasks (typo fixes, quick lookups)

**Linear** — Human-level tracking:

- Epics and milestones
- User-facing features
- Scope/timeline changes
- Decisions requiring human input
- Security concerns

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

## Starting Work on a Linear Ticket

When beginning work on a Linear ticket (e.g., ZCO-123):

1. Run `beans list --type epic | rg --fixed-strings '<linear-ticket-id>'` to find an existing Beans epic
2. If none exists, create one automatically:
   ```
   beans create "<linear-ticket-id>: <design-name>" --type epic --body "<description>" --no-edit
   ```
3. All implementation sub-tasks go under this epic as child issues using `--link parent:<epic-id>`

## Git Commit Messages

All commits related to a Linear ticket MUST reference it:

```
<descriptive message>

Part of ZCO-123.
```

When also closing a Beans issue:

```
<message>

Part of ZCO-123. Closes beans-1234.
```

This ensures Linear ticket traceability in git history even after Beans cleanup.

## Rule: Discovered Work Goes to Beans First

When you discover work during implementation:

1. Create a bean immediately (`--tag discovered --link related:<current-bean>`)
2. Assess if it needs Linear escalation
3. Never ignore discovered work due to context pressure

## Rule: Escalate Discovered Work to Linear

Create a Linear ticket for discovered work IF it:

- Affects scope or timeline of current work
- Requires human decision or approval
- Represents user-facing changes
- Is a security concern
- Is significant enough to track at project level

For purely technical implementation details (refactoring, test fixes, code cleanup), keep them in Beans only with `--tag implementation-detail`.

When Beans work reveals an epic-level concern:

1. Create the bean with `--type epic`
2. Immediately create a corresponding Linear ticket
3. Cross-reference both directions

## Querying Work

- `beans list --status open` — Find unblocked work to do next
- `beans list | rg --fixed-strings "<linear-ticket-id>"` — All Beans issues for a Linear ticket
- `beans show <id>` — View issue details including dependencies

## Provenance for Context

When revisiting a Linear ticket that seems vague, use Beans to trace its origin:

- Find the Beans epic: `beans list --type epic | rg --fixed-strings '<linear-ticket-id>'`
- Use `--tag discovered --link:<epic-ticket-id>` to understand why work was filed
- This provides context that Linear alone cannot

## Completing a Linear Ticket

When all Beans issues under an epic are closed:

1. [ ] Update the Linear ticket with a summary:
   - What was implemented
   - Any discovered work filed as separate Linear tickets
   - Notable decisions or deviations from original scope
2. [ ] Move the Linear ticket to appropriate status

The Linear ticket becomes the permanent record; Beans issues are ephemeral working memory.
