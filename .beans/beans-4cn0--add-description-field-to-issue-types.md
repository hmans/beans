---
title: Add description field to issue types
status: done
type: feature
created_at: 2025-12-07T16:34:42Z
updated_at: 2025-12-07T16:35:40Z
---

Add a 'description' field to each type in config.yaml that provides a one-line explanation of what each issue type represents. This helps agents understand when to use each type.

## Changes Required

- [ ] Update config struct to include Description field for types
- [ ] Update `beans init` to generate descriptions for default types
- [ ] Update prompt.md to instruct agents to consider type descriptions
- [ ] Update any type-related documentation

## Example

```yaml
types:
  - name: task
    color: blue
    description: A concrete piece of work that needs to be done
  - name: feature
    color: green
    description: A new capability or enhancement to add
  - name: bug
    color: red
    description: Something that is broken and needs fixing
```