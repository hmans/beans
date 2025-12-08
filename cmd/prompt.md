# EXTREMELY IMPORTANT: Beans Usage Guide for Agents

This project uses **beans**, an agentic-first issue tracker. Issues are called "beans", and you can
use the "beans" CLI to manage them. **DO NOT USE the TodoWrite tool or markdown TODOs; use bean instead**.

All commands support --json for machine-readable output. Use this flag to parse responses easily.

## CRITICAL: Track All Work With Beans

**BEFORE starting any task the user asks you to do:**

1. FIRST: Create a bean with `beans create "Title" -t <type> -d "Description..." -s in-progress`
2. THEN: Do the work
3. FINALLY: Mark completed with `beans update <bean-id> --status completed`
4. IF and WHEN you COMMIT: Include both your code changes AND the bean file(s) in the commit!

If you identify something that should be changed or fixed after completing the user's request, create a new bean for that work instead of doing it immediately.

## Core Rules

- After compaction or clear, run `beans prompt` to re-sync
- All `bean` commands support the `--json` flag for machine-readable output.
- Lean towards using sub-agents for interacting with beans.

## Finding work

- `beans list --no-status completed --no-linked-as blocks --json` to find actionable beans (not completed, not blocked)
- `beans list --json` to list all beans (descriptions not included by default)
- `beans list --json --full` to include full description content

## Working on a bean

- `beans update <bean-id> --status in-progress --json` to mark a bean as in-progress
- `beans show <bean-id> --json` to see full details including description
- Adhere to the instructions in the bean's description when working on it

**If the bean has a checklist:**

1. Work through items in order (unless dependencies require otherwise)
2. **After completing each checklist item**, immediately update the bean file to mark it done:
   - Change `- [ ]` to `- [x]` for the completed item
3. When committing code changes, include the updated bean file with checked-off items
4. Re-read the bean periodically to stay aware of remaining work

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

- `beans list --no-status completed` - Show beans that are not completed
- `beans list --no-status completed --no-status scrapped` - Exclude multiple statuses
- `beans list --no-status completed --no-linked-as blocks` - Actionable beans (not completed, not blocked)

**Filtering by type:**

Use `-t/--type` to filter by bean type (OR logic when repeated):

- `beans list -t bug` - Show only bugs
- `beans list -t bug -t feature` - Show bugs OR features
- `beans list --no-type epic` - Exclude epics
- `beans list -t task --no-status completed` - Show open tasks

## Creating new beans

- `beans create --help`
- Example: `beans create "Fix login bug" -t bug -d "Users cannot log in when..." -s todo`
- **Always specify a type with `-t`**. See the "Issue Types" section below for available types and their descriptions.
- When creating a new bean, first see if a similar bean already exists.
- When creating new beans, include a useful description. If you're not sure what to write, ask the user.
- Make the description as detailed as possible, similar to a plan that you would create for yourself.
- If possible, split the work into a checklist of GitHub-Formatted-Markdown tasks. Use a `## Checklist` header to precede it.

## Cleaning up beans

- `beans archive` will archive (delete) beans marked as completed or scrapped. ONLY run this when I explicitly tell you to.

## GraphQL API

Beans provides a GraphQL API for flexible querying. Use `beans query` to execute GraphQL queries.

**Basic usage:**

```bash
# List all beans
beans query '{ beans { id title status } }'

# Get a specific bean
beans query '{ bean(id: "abc") { title status body } }'

# Filter beans
beans query '{ beans(filter: { status: ["todo", "in-progress"] }) { id title } }'

# Get relationships
beans query '{ bean(id: "abc") { title parent { title } children { id title } } }'

# Read from stdin (useful for complex queries or shell escaping issues)
cat query.graphql | beans query
beans query <<'EOF'
query GetBean($id: ID!) { bean(id: $id) { title } }
EOF

# Print schema
beans query --schema
```

**Flags:**

- `--json`: Output raw JSON (no formatting)
- `--schema`: Print the GraphQL schema
- `-v, --variables`: Query variables as JSON string
- `-o, --operation`: Operation name for multi-operation documents

**GraphQL Schema:**

```graphql
type Query {
  bean(id: ID!): Bean
  beans(filter: BeanFilter): [Bean!]!
}

type Bean {
  id: ID!
  slug: String
  path: String!
  title: String!
  status: String!
  type: String
  priority: String
  tags: [String!]
  createdAt: Time
  updatedAt: Time
  body: String
  links: [Link!]

  # Computed relationship fields
  blockedBy: [Bean!]!   # beans that block this one
  blocks: [Bean!]!      # beans this one blocks
  parent: Bean          # parent bean if linked
  children: [Bean!]!    # beans with this as parent
}

type Link {
  type: String!
  target: String!
  targetBean: Bean      # resolved bean (null if broken)
}

input BeanFilter {
  status: [String!]
  excludeStatus: [String!]
  type: [String!]
  excludeType: [String!]
  priority: [String!]
  excludePriority: [String!]
  tags: [String!]
  excludeTags: [String!]
  hasLinks: [String!]   # has outgoing links of type
  linkedAs: [String!]   # is target of links of type
  noLinks: [String!]    # exclude if has outgoing links
  noLinkedAs: [String!] # exclude if is target of links
}

scalar Time
```
