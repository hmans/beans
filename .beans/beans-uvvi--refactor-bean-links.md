---
title: Refactor bean links
status: completed
type: task
priority: normal
created_at: 2025-12-12T10:45:22Z
updated_at: 2025-12-12T11:27:52Z
---

At the moment, bean links are expressed as an array of objects underneat the `links` key in the bean frontmatter. Each object consists of a single key-value pair, where the key is the link type and the value is the target bean ID. This design was chosen because we initially wanted to let users define their own bean link types.

But we've moved away from that, and settled on the following link types:

- parent (single)
- related (multiple)
- blocks (multiple)
- duplicates (multiple)

All the CLI commands and GraphQL interactions have been built around this free-form link structure. Let's lock it down! In the new design, let's have the following root-level frontmatter keys:

- `milestone`: optional, single value, target bean ID (target must be a milestone)
- `epic`: optional, single value, target bean ID (target must be an epic)
- `feature`: optional single value, target bean ID (bean must be a task, target must be a feature)
- `related`: optional, multiple values, array of target bean IDs
- `blocks`: optional, multiple values, array of target bean IDs
- `duplicates`: optional, multiple values, array of target bean IDs

Both the CLI commands as well as the GraphQL schema and resolvers must be updated to reflect this new design.

## Design Decisions

### Hierarchy vs. Generic Parent

The old `parent` link type is replaced by three specific hierarchy fields:

- `milestone` - assigns a bean to a milestone (release target)
- `epic` - assigns a bean to an epic (thematic grouping)
- `feature` - assigns a task to a feature (only valid for tasks)

A bean can have **all three** set simultaneously, enabling multi-dimensional organization:
- "This bug (task) is part of the Login feature, under the Auth epic, targeting the v2.0 milestone"

### Validation Constraints

| Field | Bean Type Constraint | Target Type Constraint |
|-------|---------------------|----------------------|
| `milestone` | any | must be `milestone` |
| `epic` | any | must be `epic` |
| `feature` | must be `task` or `bug` | must be `feature` |
| `related` | any | any |
| `blocks` | any | any |
| `duplicates` | any | any |

### YAML Format Examples

**Old format:**
```yaml
links:
    - parent: beans-abc
    - blocks: beans-xyz
    - related: beans-123
```

**New format:**
```yaml
epic: beans-abc
blocks:
    - beans-xyz
related:
    - beans-123
```

### GraphQL Computed Fields

The computed fields (`parent`, `children`, `blockedBy`, etc.) will be updated:

- `parent` → **removed** (replaced by `milestone`, `epic`, `feature` references)
- `children` → **removed** (replaced by `milestoneItems`, `epicItems`, `featureItems`)
- `blocks`, `blockedBy`, `duplicates`, `related` → **unchanged** semantically

## Checklist

### Phase 1: Core Data Model (`internal/bean/`)

- [x] Update `Bean` struct to replace `Links` with new fields:
  - `Milestone string` (single, optional)
  - `Epic string` (single, optional)
  - `Feature string` (single, optional)
  - `Related []string` (multiple)
  - `Blocks []string` (multiple)
  - `Duplicates []string` (multiple)
- [x] Remove `Link` struct and `Links` type (along with marshal/unmarshal methods)
- [x] Update `frontMatter` struct for parsing (handles yaml.v2 from frontmatter lib)
- [x] Update `renderFrontMatter` struct for serialization (yaml.v3)
- [x] Update `Parse()` function to handle new field structure
- [x] Update `Render()` function to output new YAML format
- [x] Add helper methods on `Bean`:
  - `HasBlock(target string) bool`
  - `AddBlock(target string)`
  - `RemoveBlock(target string)`
  - `HasRelated(target string) bool`
  - `AddRelated(target string)`
  - `RemoveRelated(target string)`
  - `HasDuplicate(target string) bool`
  - `AddDuplicate(target string)`
  - `RemoveDuplicate(target string)`
- [x] Update/add tests in `bean_test.go` for new format

### Phase 2: Link Validation (`internal/beancore/`)

- [x] Update `KnownLinkTypes` in `core.go` (or remove if no longer needed)
- [x] Refactor `links.go` to work with new field structure:
  - Update `IncomingLink` struct if needed
  - Update `FindIncomingLinks()` to check `Blocks`, `Milestone`, `Epic`, `Feature` fields
  - Update `DetectCycle()` for `Blocks` (no longer need parent cycles)
  - Update `CheckAllLinks()` for new validation:
    - Validate milestone targets are type=milestone
    - Validate epic targets are type=epic
    - Validate feature field only on tasks, targets type=feature
    - Check for broken links across all fields
    - Check for self-references
  - Update `FixBrokenLinks()` to clean all new fields
  - Update `RemoveLinksTo()` to clean all new fields when a bean is deleted
- [x] Add new validation functions:
  - `ValidateHierarchyLinks(bean, allBeans)` - checks type constraints
- [x] Update tests in `links_test.go`

### Phase 3: GraphQL Schema (`internal/graph/`)

- [x] Update `schema.graphqls`:
  - Update `Bean` type:
    - Remove `links: [Link!]!`
    - Add `milestone: Bean` (resolved target)
    - Add `epic: Bean` (resolved target)
    - Add `feature: Bean` (resolved target)
    - Add `milestoneId: String` (raw ID for mutations)
    - Add `epicId: String` (raw ID)
    - Add `featureId: String` (raw ID)
    - Keep `blocks: [Bean!]!`, `blockedBy: [Bean!]!`
    - Keep `duplicates: [Bean!]!`, `related: [Bean!]!`
    - Replace `children` with:
      - `milestoneItems: [Bean!]!` (beans with this as milestone)
      - `epicItems: [Bean!]!` (beans with this as epic)
      - `featureItems: [Bean!]!` (beans with this as feature, only for features)
  - Remove `Link` type (no longer needed externally)
  - Update `LinkInput` → rename to `LinkInput` or split into specific inputs
  - Update `CreateBeanInput`:
    - Remove `links: [LinkInput!]`
    - Add `milestone: String`, `epic: String`, `feature: String`
    - Add `blocks: [String!]`, `related: [String!]`, `duplicates: [String!]`
  - Update `UpdateBeanInput` similarly
  - Update `LinkFilter` for querying (may need redesign)
  - Replace `addLink` and `removeLink` mutations with explicit mutations:
    - `setMilestone(id: ID!, target: String): Bean!` (set or clear with empty string)
    - `setEpic(id: ID!, target: String): Bean!`
    - `setFeature(id: ID!, target: String): Bean!`
    - `addBlock(id: ID!, target: String!): Bean!`
    - `removeBlock(id: ID!, target: String!): Bean!`
    - `addRelated(id: ID!, target: String!): Bean!`
    - `removeRelated(id: ID!, target: String!): Bean!`
    - `addDuplicate(id: ID!, target: String!): Bean!`
    - `removeDuplicate(id: ID!, target: String!): Bean!`
- [x] Run `mise codegen` to regenerate resolver stubs
- [x] Update resolvers in `schema.resolvers.go`:
  - Implement new field resolvers (`Milestone()`, `Epic()`, `Feature()`, etc.)
  - Implement inverse resolvers (`MilestoneItems()`, `EpicItems()`, `FeatureItems()`)
  - Update `BlockedBy()`, `Blocks()`, `Duplicates()`, `Related()` for new data model
  - Update `CreateBean` mutation handler
  - Update `UpdateBean` mutation handler
  - Update/remove `AddLink`/`RemoveLink` mutation handlers
- [x] Update query filtering in `query.resolvers.go` for new link structure

### Phase 4: CLI Commands (`cmd/`)

- [x] Update `create.go`:
  - Replace `--link type:id` with specific flags:
    - `--milestone <id>`
    - `--epic <id>`
    - `--feature <id>` (validate bean type is task or bug)
    - `--block <id>` (repeatable)
    - `--related <id>` (repeatable)
    - `--duplicate <id>` (repeatable)
  - Update GraphQL mutation call
- [x] Update `update.go`:
  - Replace `--link`/`--unlink` with specific flags:
    - `--milestone <id>` / `--no-milestone`
    - `--epic <id>` / `--no-epic`
    - `--feature <id>` / `--no-feature`
    - `--block <id>` (repeatable) / `--unblock <id>` (repeatable)
    - `--related <id>` (repeatable) / `--unrelated <id>` (repeatable)
    - `--duplicate <id>` (repeatable) / `--unduplicate <id>` (repeatable)
  - Update GraphQL mutation calls (use the explicit mutations)
- [x] Update `content.go`:
  - Remove `parseLink()` function (no longer needed)
  - Remove `isKnownLinkType()` function (or adapt)
  - Update `applyLinks()` → split into specific validation functions
  - Update cycle detection for new structure
- [x] Update `show.go` output formatting for new fields
- [x] Update any other commands that reference links

### Phase 5: Documentation & Cleanup

- [x] Update README.md with new link structure
- [x] Update `.beans.yml` example/docs if needed
- [x] Update the agent prompt in `cmd/prime.go` with new link syntax
- [x] Clean up any deprecated code
- [x] Run full test suite and fix any failures

## Breaking Changes

This is a **breaking change** affecting:

1. **YAML format** - old `links` array format no longer supported
2. **GraphQL schema** - `addLink`/`removeLink` replaced with explicit mutations
3. **CLI flags** - `--link` and `--unlink` replaced with specific flags

No migration of existing data is needed. Use `feat!:` commit prefix to signal the breaking change.
