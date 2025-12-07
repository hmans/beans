---
title: Use beans as a release management tool
status: open
type: idea
created_at: 2025-12-07T12:30:19Z
updated_at: 2025-12-07T12:30:19Z
---

## Overview

Beans could be extended to serve as a release management tool, automatically generating changelog entries from completed beans.

## Proposed Features

### 1. Release frontmatter property
- Add a `release` property to bean frontmatter
- Valid values: `patch`, `minor`, `major`
- This indicates the bean should be included in changelog generation

### 2. `beans release` command
The command would:
- Find all beans with status=done that have a `release` property set
- Group them by release type (major/minor/patch)
- Generate/update the project's CHANGELOG.md with entries from these beans
- Archive the consumed beans after processing

### 3. Optional: Git integration (possibly out of scope)
- Create a git commit with the changelog changes
- Create a git tag for the release
- Push to remote

## Checklist

- [ ] Add `release` field support to bean frontmatter (patch/minor/major)
- [ ] Add validation for release field values
- [ ] Implement `beans release` command
- [ ] Detect completed beans with release property
- [ ] Generate CHANGELOG.md entries (follow Keep a Changelog format?)
- [ ] Archive processed beans after release
- [ ] Consider: version number detection/bumping
- [ ] Consider: git commit/tag creation (may be out of scope)

## Open Questions

- Should this replace changie or complement it?
- What changelog format to use? (Keep a Changelog, Conventional Changelog, etc.)
- How to determine the next version number? (read from package.json, go.mod, or prompt?)
- Should git operations be included or left to the user?