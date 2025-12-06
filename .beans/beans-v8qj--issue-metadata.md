---
title: "Issue metadata"
status: in-progress
type: epic
created_at: 2025-12-06T21:46:00Z
updated_at: 2025-12-06T22:04:09Z
---

We want to add some pieces of metadata to issues to help with planning and tracking.

## Issue type

- Should be mostly a free-form field with some sane default values (e.g. `task`, `bug`, `epic`)
- Allow customization in `beans.toml`, but don't crash when beans use an unknown type.

## Issue relationships

- Issues can reference other issues, with a defined relationship type (e.g., "blocks", "duplicates", "relates to").
- This will help in understanding dependencies and connections between different tasks.
- Optional: relationship types should be configurable by the user, with `beans init` providing sensible defaults.
- Optional: The interpretation of these relationships could be left to the agent. For example, instead of having a `beans ready` command that finds issues that aren't blocks, the agent could use `beans list --filter "!blocked_by"`
- Relationship types should be able to provide their "opposite" relationship. E.g., if issue A "blocks" issue B, then issue B is "blocked by" issue A. When reading the issue store, we can infer the opposite relationship automatically.
- But maybe hard-coded relationship types are better for discoverability and usability, and reduced complexity.

## Epics / Parent Issues

- Introduce the concept of "epics" or "parent issues" that can group related issues together.
- Often these will be "Epics", high-level tasks that encompass multiple smaller issues.
- Another type might be "Milestones", with a specific milestone issue representing a significant point in the project timeline.
- But sometimes a task is just so complex it needs to have sub-tasks.
- Thought: we could express this on the file system with a directory structure, but this will require us to switch individual beans to being directories rather than files.
- Otherwise, we'll just have to use frontmatter to express parent-child relationships.

## Custom Labels

- Allow users to define custom labels for issues to categorize and filter them based on project-specific criteria.
- These should be mostly free-form, but we shouldn't require the user to define them up-front.
- But configuration may be added to `beans.toml` to define colors (or possibly other properties) for specific labels.

## Priorities

- Introduce a priority system (e.g., Low, Medium, High, Critical) to help prioritize issues.
- Similarly to some of the above, we will either hard-code possible values, or allow user configuration.
