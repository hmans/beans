---
title: Add version bumping tool for releases
status: done
created_at: 2025-12-06T16:56:48Z
updated_at: 2025-12-06T17:15:02Z
---



## Context

Currently, releasing a new version requires manually creating and pushing git tags:
```bash
git tag v0.1.4
git push --tags
```

This is error-prone (typos, forgetting the 'v' prefix, wrong version number).

## Solution

Add **svu** (Semantic Version Utility) from the goreleaser team to streamline releases.

## Checklist

- [ ] Document svu installation in README or CLAUDE.md
- [ ] Add mise tasks for releasing (e.g., `mise release:patch`, `mise release:minor`, `mise release:major`)
- [ ] Test the workflow by doing a patch release