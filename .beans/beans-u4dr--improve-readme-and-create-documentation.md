---
# beans-u4dr
title: Improve README and create documentation
status: todo
type: task
created_at: 2025-12-30T17:28:32Z
updated_at: 2025-12-30T17:28:32Z
---

Create dedicated documentation for beans.

## Design

### Create `docs/beans.md`

A single reference document covering bean file format, configuration, and concepts.

#### Structure

1. **Bean File Format**
   - Location: `.beans/` directory
   - Filename: `<id>--<slug>.md` (slug is optional)
   - YAML frontmatter + markdown body

2. **Frontmatter Fields**
   - `title` (required) - The bean's title
   - `status` (required) - Current lifecycle status
   - `type` - What kind of work this represents
   - `priority` - Urgency level
   - `tags` - List of tags for categorization
   - `parent` - Parent bean ID for hierarchy
   - `blocking` - List of bean IDs this bean blocks
   - `created_at` / `updated_at` - Timestamps

3. **Statuses** (lifecycle states)

   | Status | Description | Archivable |
   |--------|-------------|------------|
   | in-progress | Currently being worked on | No |
   | todo | Ready to be worked on | No |
   | draft | Needs refinement before it can be worked on | No |
   | completed | Finished successfully | Yes |
   | scrapped | Will not be done | Yes |

   - New beans typically start as `todo` or `draft`
   - Move to `in-progress` when work begins
   - End at `completed` or `scrapped`
   - Archivable statuses can be cleaned up with `beans archive`

4. **Types** (what kind of work)

   | Type | Description | Typical Use |
   |------|-------------|-------------|
   | milestone | A target release or checkpoint | Group work that should ship together |
   | epic | A thematic container for related work | Should have child beans, not worked on directly |
   | feature | A user-facing capability or enhancement | Deliverable functionality |
   | bug | Something that is broken and needs fixing | Defects, regressions |
   | task | A concrete piece of work to complete | Chores, sub-tasks for features |

   Hierarchy: `milestone → epic → feature → task/bug`

5. **Priorities** (urgency levels)

   | Priority | Description |
   |----------|-------------|
   | critical | Urgent, blocking work - address immediately |
   | high | Important, should be done before normal work |
   | normal | Standard priority |
   | low | Less important, can be delayed |
   | deferred | Explicitly pushed back |

6. **Relationships**
   - Parent/child hierarchy via `parent` field
   - Blocking relationships via `blocking` field

7. **Configuration** (`.beans.yml`)
   - `beans.path` - Directory for bean files (default: `.beans`)
   - `beans.prefix` - ID prefix (e.g., `myproj-`)
   - `beans.id_length` - Length of generated IDs (default: 4)
   - `beans.default_status` - Default status for new beans (default: `todo`)
   - `beans.default_type` - Default type for new beans (default: `task`)

8. **Tags**
   - Format: lowercase, letters/numbers/hyphens
   - Must start with a letter
   - No consecutive hyphens or trailing hyphens

### Update README

- Add brief list of statuses and types (names only)
- Link to `docs/beans.md` for full reference
- Keep README focused on quick start / overview