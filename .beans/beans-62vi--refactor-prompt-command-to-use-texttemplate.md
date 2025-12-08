---
title: Refactor prompt command to use text/template
status: todo
type: task
created_at: 2025-12-08T20:16:08Z
updated_at: 2025-12-08T20:16:08Z
---

## Summary

Refactor the `beans prompt` generation to use Go's `text/template` package instead of
string builders and manual string concatenation.

## Motivation

The current implementation likely uses `strings.Builder` or similar manual string
construction, which can be harder to read, maintain, and modify. Using `text/template` would:

- Make the output format more declarative and easier to understand
- Separate the template structure from the data logic
- Make it easier to customize or extend the prompt format in the future
- Follow Go best practices for text generation

## Checklist

- [ ] Review the current `beans prompt` implementation to understand its structure
- [ ] Create a template file or embedded template string for the prompt output
- [ ] Refactor the command to use `text/template` with appropriate data structures
- [ ] Ensure the output remains identical to the current implementation
- [ ] Add/update tests to verify the refactored code produces correct output
